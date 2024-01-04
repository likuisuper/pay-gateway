package notify

import (
	"context"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	jsoniter "github.com/json-iterator/go"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"net/http"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyWechatH5OrderLogic struct {
	logx.Logger
	ctx                  context.Context
	svcCtx               *svc.ServiceContext
	payOrderModel        *model.PmPayOrderModel
	payConfigWechatModel *model.PmPayConfigWechatModel
}

func NewNotifyWechatH5OrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyWechatH5OrderLogic {
	return &NotifyWechatH5OrderLogic{
		Logger:               logx.WithContext(ctx),
		ctx:                  ctx,
		svcCtx:               svcCtx,
		payOrderModel:        model.NewPmPayOrderModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
	}
}

func (l *NotifyWechatH5OrderLogic) NotifyWechatH5Order(request *http.Request) (resp *types.WeChatResp, err error) {
	appId := request.Header.Get("AppId")

	payCfg, err := l.payConfigWechatModel.GetOneByAppID(appId)
	if err != nil {
		err = fmt.Errorf("pkgName= %s, 读取微信支付配置失败，err:=%v", "all", err)
		util.CheckError(err.Error())
		return
	}

	var transaction *payments.Transaction
	var wxCli *client.WeChatCommPay
	wxCli = client.NewWeChatCommPay(*payCfg.TransClientConfig())
	transaction, _, err = wxCli.Notify(request)
	if err != nil {
		err = fmt.Errorf("解析及验证内容失败！err=%v ", err)
		logx.Errorf(err.Error())
		return
	}

	if *transaction.TradeState != "SUCCESS" {
		jsonStr, _ := jsoniter.MarshalToString(transaction)
		logx.Slowf("wechat支付回调异常: %s", jsonStr)
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
	orderInfo.PayType = model.PmPayOrderTablePayTypeWechatPayUni
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
