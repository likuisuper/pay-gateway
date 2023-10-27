package notify

import (
	"context"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	jsoniter "github.com/json-iterator/go"
	"net/http"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyWechatRefundOrderLogic struct {
	logx.Logger
	ctx                  context.Context
	svcCtx               *svc.ServiceContext
	orderModel           *model.OrderModel
	refundModel          *model.RefundModel
	payConfigWechatModel *model.PmPayConfigWechatModel
}

func NewNotifyWechatRefundOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyWechatRefundOrderLogic {
	return &NotifyWechatRefundOrderLogic{
		Logger:               logx.WithContext(ctx),
		ctx:                  ctx,
		svcCtx:               svcCtx,
		orderModel:           model.NewOrderModel(define.DbPayGateway),
		refundModel:          model.NewRefundModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
	}
}

type wxRefundOrderReply struct {
	EventType    string   `json:"event_type"`
	Id           string   `json:"id"`
	CreateTime   string   `json:"create_time"`
	ResourceType string   `json:"resource_type"`
	Summary      string   `json:"summary"`
	Resource     Resource `json:"resource"`
}
type Resource struct {
	OriginalType   string `json:"original_type"`
	Algorithm      string `json:"algorithm"`
	Ciphertext     string `json:"ciphertext"`
	AssociatedData string `json:"associated_data"`
	Nonce          string `json:"nonce"`
}

func (l *NotifyWechatRefundOrderLogic) NotifyWechatRefundOrder(req *types.WechatRefundReq, r *http.Request) (resp *types.WeChatResp, err error) {
	//退款订单信息
	orderInfo, _ := l.refundModel.GetOneByOutTradeNo(req.OutTradeNo)
	if orderInfo != nil && orderInfo.RefundStatus == 0 {
		//退款成功
		if req.EventType == "REFUND.SUCCESS" {
			orderInfo.RefundStatus = code.ORDER_SUCCESS
		} else {
			//原订单信息
			originOrderInfo, _ := l.orderModel.GetOneByOutTradeNo(req.OutTradeNo)
			payCfg, getPayErr := l.payConfigWechatModel.GetOneByAppID(originOrderInfo.PayAppID)
			if getPayErr != nil {
				err = fmt.Errorf("获取配置失败！err=%v，appid=%s ", err, originOrderInfo.PayAppID)
				logx.Errorf(err.Error())
				return nil, err
			}
			notifyData, jErr := jsoniter.MarshalToString(req)
			if jErr != nil {
				orderInfo.NotifyData = notifyData
			}
			var wxCli *client.WeChatCommPay
			wxCli = client.NewWeChatCommPay(*payCfg.TransClientConfig())
			//查询订单状态
			wxRefundOrderInfo, refundErr := wxCli.GetOrderStatus(req.OutTradeNo)
			if refundErr != nil {
				err = fmt.Errorf("查询订单失败！err=%v ", err)
				logx.Errorf(err.Error())
				return nil, err
			}
			if *wxRefundOrderInfo.TradeState == "REFUND" {
				orderInfo.RefundStatus = code.ORDER_SUCCESS
			}

		}
		//修改订单退款状态
		l.orderModel.UpdateStatusByOutTradeNo(req.OutTradeNo, code.ORDER_REFUNDED)
		//修改退款订单信息
		l.refundModel.Update(orderInfo.OutTradeRefundNo, orderInfo)
		// 回调退款成功
		go func() {
			defer exception.Recover()
			dataMap := make(map[string]interface{})
			dataMap["notify_type"] = code.APP_NOTIFY_TYPE_REFUND
			dataMap["out_trade_refund_no"] = orderInfo.OutTradeRefundNo
			dataMap["out_trade_no"] = orderInfo.OutTradeNo
			dataMap["refund_out_side_app"] = false
			dataMap["refund_status"] = model.REFUND_STATUS_SUCCESS
			dataMap["refund_fee"] = orderInfo.RefundAmount
			_, _ = util.HttpPost(orderInfo.NotifyUrl, dataMap, 5*time.Second)
		}()
	}
	return &types.WeChatResp{
		Code:    "SUCCESS",
		Message: "OK",
	}, nil
}
