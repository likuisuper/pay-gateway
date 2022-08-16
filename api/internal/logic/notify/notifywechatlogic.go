package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	notifyOrderHasDispose = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "notifyOrderHasDispose", nil, "回调订单已处理", nil})}
)

type NotifyWechatLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	payOrderModel        *model.PmPayOrderModel
	payConfigWechatModel *model.PmPayConfigWechatModel
}

func NewNotifyWechatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyWechatLogic {
	return &NotifyWechatLogic{
		Logger:               logx.WithContext(ctx),
		ctx:                  ctx,
		svcCtx:               svcCtx,
		payOrderModel:        model.NewPmPayOrderModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
	}
}

func (l *NotifyWechatLogic) NotifyWechat(req *types.EmptyReq, r *http.Request) (resp *types.WeChatResp, err error) {
	appId := r.Header.Get("AppId")
	logx.Info("NotifyWechat", appId)

	d := make(map[string]interface{}, 0)
	if err = httpx.Parse(r, &d); err != nil {
		return
	}
	jsonBytes, _ := json.Marshal(d)
	logx.Info("NotifyWechat", appId, string(jsonBytes))

	payCfg, err := l.payConfigWechatModel.GetOneByAppID(appId)
	if err != nil {
		err = fmt.Errorf("pkgName= %s, 读取微信支付配置失败，err:=%v", "all", err)
		util.CheckError(err.Error())
		return
	}

	var transaction *payments.Transaction
	var wxCli *client.WeChatCommPay
	wxCli = client.NewWeChatCommPay(*payCfg.TransClientConfig())
	transaction, err = wxCli.Notify(r)
	if err != nil {
		err = fmt.Errorf("解析及验证内容失败！err=%v ", err)
		logx.Errorf(err.Error())
		return
	}
	//获取订单信息
	orderInfo, err := l.payOrderModel.GetOneByCode(*transaction.OutTradeNo)
	if err != nil {
		err = fmt.Errorf("获取订单失败！err=%v,order_code = %s", err, transaction.OutTradeNo)
		util.CheckError(err.Error())
		return
	}
	if orderInfo.PayStatus != model.PmPayOrderTablePayStatusNo {
		notifyOrderHasDispose.CounterInc()
		err = fmt.Errorf("订单已处理")
		return
	}
	//修改数据库
	orderInfo.NotifyAmount = int(*transaction.Amount.PayerTotal)
	orderInfo.PayStatus = model.PmPayOrderTablePayStatusPaid
	err = l.payOrderModel.UpdateNotify(orderInfo)
	if err != nil {
		err = fmt.Errorf("orderSn = %s, UpdateNotify，err:=%v", orderInfo.OrderSn, err)
		util.CheckError(err.Error())
		return
	}

	//回调业务方接口
	go func() {
		defer exception.Recover()
		_, _ = util.HttpPost(orderInfo.NotifyUrl, transaction, 5*time.Second)
	}()

	resp = &types.WeChatResp{
		Code:    "SUCCESS",
		Message: "",
	}

	return
}
