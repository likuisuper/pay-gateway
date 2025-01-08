package notify

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	jsoniter "github.com/json-iterator/go"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"

	"gitee.com/zhuyunkj/pay-gateway/api/common/notice"
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

func (l *NotifyWechatLogic) NotifyWechat(request *http.Request) (resp *types.WeChatResp, err error) {
	appId := request.Header.Get("AppId")
	logx.Slowf("NotifyWechat %s", appId)

	payCfg, err := l.payConfigWechatModel.GetOneByAppID(appId)
	if err != nil {
		err = fmt.Errorf("微信支付回调 读取微信支付配置失败 appId: %s ,err: %v", appId, err)
		util.CheckError(err.Error())

		DingdingNotify(l.ctx, err.Error())
		return
	}

	var transaction *payments.Transaction
	wxCli := client.NewWeChatCommPay(*payCfg.TransClientConfig())
	transaction, _, err = wxCli.Notify(request)
	if err != nil {
		err = fmt.Errorf("微信支付回调 解析及验证内容失败 err=%v ", err)
		DingdingNotify(l.ctx, err.Error())

		logx.Error(err.Error())
		return
	}

	if *transaction.TradeState != "SUCCESS" {
		jsonStr, _ := jsoniter.MarshalToString(transaction)
		logx.Slowf("wechat支付回调异常: %s", jsonStr)
		return
	}

	//升级为根据订单号和appid查询
	orderInfo, err := l.payOrderModel.GetOneByOrderSnAndAppId(*transaction.OutTradeNo, appId)
	if err != nil || orderInfo == nil || orderInfo.ID < 1 {
		err = fmt.Errorf("微信支付回调 获取订单失败 err:%v, order_code:%s, appId:%s", err, *transaction.OutTradeNo, appId)
		util.CheckError(err.Error())
		DingdingNotify(l.ctx, err.Error())
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
	orderInfo.ThirdOrderNo = *transaction.TransactionId
	//orderInfo.PayType = model.PmPayOrderTablePayTypeWechatPayUni //改为创建订单时指定支付类型，用于补偿机制建设
	err = l.payOrderModel.UpdateNotify(orderInfo)
	if err != nil {
		err = fmt.Errorf("微信支付回调 orderSn: %s UpdateNotify err: %v", orderInfo.OrderSn, err)
		util.CheckError(err.Error())
		DingdingNotify(l.ctx, err.Error())
		return
	}

	//回调业务方接口
	go func() {
		defer exception.Recover()
		headerMap := map[string]string{
			"App-Origin": orderInfo.AppPkgName,
		}
		_, err = util.HttpPostWithHeader(orderInfo.NotifyUrl, transaction, headerMap, 5*time.Second)
		if err != nil {
			util.CheckError("NotifyWechat call business failed orderSn = %s, err:=%v", orderInfo.OrderSn, err)
			CallbackBizFailNum.CounterInc()
		}
	}()

	resp = &types.WeChatResp{
		Code:    "SUCCESS",
		Message: "",
	}

	return
}

// 发送钉钉通知
const DingdingRobot = "https://oapi.dingtalk.com/robot/send?access_token=658d67c2ad4b71bc6ec8d67d947ae158446ea2499a499729cbcdbf118c6d618b"

func DingdingNotify(ctx context.Context, msg string) {
	go util.SafeRun(func() {
		now := time.Now().Format("2006-01-02 15:04:05")
		req := &notice.RobotSendReq{
			Msgtype: "text",
			Text: &notice.Text{
				Content: "[诸云pay-gateway通知] " + msg + ", now:" + now,
			},
		}
		_, err := notice.SendWebhookMsg(ctx, req, DingdingRobot)
		if err != nil {
			logx.WithContext(ctx).Errorf("dingding notify fail, err:%v", err)
		}
	})
}
