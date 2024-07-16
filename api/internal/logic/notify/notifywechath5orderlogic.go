package notify

import (
	"context"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/common/utils"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/alarm"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	jsoniter "github.com/json-iterator/go"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// 流量用

type NotifyWechatH5OrderLogic struct {
	logx.Logger
	ctx                  context.Context
	svcCtx               *svc.ServiceContext
	orderModel           *model.OrderModel
	payConfigWechatModel *model.PmPayConfigWechatModel
}

func NewNotifyWechatH5OrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyWechatH5OrderLogic {
	return &NotifyWechatH5OrderLogic{
		Logger:               logx.WithContext(ctx),
		ctx:                  ctx,
		svcCtx:               svcCtx,
		orderModel:           model.NewOrderModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
	}
}

func (l *NotifyWechatH5OrderLogic) NotifyWechatH5Order(request *http.Request) (resp *types.WeChatResp, err error) {
	var req types.WechatNotifyH5Req
	err = httpx.ParsePath(request, &req)
	if err != nil {
		err = fmt.Errorf("解析path失败！err=%v ", err)
		logx.WithContext(l.ctx).Errorf(err.Error())
		return
	}

	payCfg, err := l.payConfigWechatModel.GetOneByAppID(req.AppID)
	if err != nil {
		err = fmt.Errorf("appid= %s, 读取微信支付配置失败，err:=%v", req.AppID, err)
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
	logx.Infof("weixin h5 pay transaction:%+v", transaction)

	//获取订单信息
	orderInfo, err := l.orderModel.GetOneByOutTradeNo(*transaction.OutTradeNo)
	if err != nil {
		err = fmt.Errorf("获取订单失败！err=%v,order_code = %s", err, transaction.OutTradeNo)
		util.CheckError(err.Error())
		return
	}
	if orderInfo.Status != model.PmPayOrderTablePayStatusNo {
		notifyOrderHasDispose.CounterInc()
		err = fmt.Errorf("订单已处理")
		return
	}
	//修改数据库
	orderInfo.Status = model.PmPayOrderTablePayStatusPaid
	orderInfo.PayType = model.PmPayOrderTablePayTypeWechatPayH5
	orderInfo.PlatformTradeNo = *transaction.TransactionId
	err = l.orderModel.UpdateNotify(orderInfo)
	if err != nil {
		err = fmt.Errorf("trade_no = %s, UpdateNotify，err:=%v", orderInfo.PlatformTradeNo, err)
		util.CheckError(err.Error())
		return
	}

	//回调业务方接口
	go func() {
		defer exception.Recover()
		dataMap := map[string]interface{}{
			"out_trade_no": *transaction.OutTradeNo,
		}
		dataMap["notify_type"] = code.APP_NOTIFY_TYPE_PAY
		err = utils.CallbackWithRetry(orderInfo.AppNotifyUrl, dataMap, 5*time.Second)
		if err != nil {
			desc := fmt.Sprintf("回调通知用户付款成功 异常, app_pkg=%s, user_id=%d, out_trade_no=%s, 报错信息：%v", orderInfo.AppPkg, orderInfo.UserID, orderInfo.OutTradeNo, err)
			alarm.ImmediateAlarm("notifyUserPayErr", desc, alarm.ALARM_LEVEL_FATAL)
		}
	}()

	resp = &types.WeChatResp{
		Code:    "SUCCESS",
		Message: "",
	}

	return
}
