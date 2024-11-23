package logic

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	alipay2 "gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/clientMgr"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/utils"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AlipayRefundLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	orderModel  *model.OrderModel
	refundModel *model.RefundModel
}

func NewAlipayRefundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayRefundLogic {
	return &AlipayRefundLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		orderModel:  model.NewOrderModel(define.DbPayGateway),
		refundModel: model.NewRefundModel(define.DbPayGateway),
	}
}

// 支付宝：退款
func (l *AlipayRefundLogic) AlipayRefund(in *pb.AlipayRefundReq) (*pb.AliRefundResp, error) {
	payClient, _, _, err := clientMgr.GetAlipayClientByAppPkgWithCache(in.AppPkgName)
	if err != nil {
		return nil, err
	}

	order, err := l.orderModel.GetOneByOutTradeNo(in.OutTradeNo)
	if err != nil {
		errInfo := fmt.Sprintf("创建退款订单：获取订单失败!!! %s", in.OutTradeNo)
		logx.Errorf(errInfo)
		createRefundErr.CounterInc()
		return nil, errors.New(errInfo)
	}

	refund := model.RefundTable{
		PayType:          order.PayType,
		OutTradeNo:       order.OutTradeNo,
		OutTradeRefundNo: utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
		Reason:           in.RefundReason,
		NotifyUrl:        in.RefundNotifyUrl,
		Operator:         in.Operator,
		AppPkg:           order.AppPkg,
		RefundNo:         in.TradeNo, // 支付宝退款没有退款单号
		ReviewerComment:  "自动退款",
		RefundedAt:       time.Now(),
	}

	tradeRefund := alipay2.TradeRefund{
		OutTradeNo:   in.OutTradeNo,
		RefundAmount: in.RefundAmount,
		RefundReason: in.RefundReason,
		OutRequestNo: refund.OutTradeRefundNo,
	}

	result, err := payClient.TradeRefund(tradeRefund)
	if err != nil {
		logx.Errorf(err.Error())
	}

	if result.Content.Code == alipay2.CodeSuccess && result.Content.FundChange == "Y" {
		floatAmount, _ := strconv.ParseFloat(result.Content.RefundFee, 64)
		intAmount := int(floatAmount * 100)
		refund.RefundAmount = intAmount
		refund.RefundStatus = model.REFUND_STATUS_SUCCESS
		//l.refundModel.Update(in.OutTradeNo, &refund)

		err = l.refundModel.Create(&refund)
		if err != nil {
			errInfo := fmt.Sprintf("创建退款订单失败!!! %s", in.OutTradeNo)
			logx.Errorf(errInfo)
			createRefundErr.CounterInc()
			return nil, errors.New(errInfo)
		}

		order.Status = code.ORDER_REFUNDED
		l.orderModel.UpdateNotify(order)

		return &pb.AliRefundResp{
			Status:           code.ALI_PAY_SUCCESS,
			RefundFee:        result.Content.RefundFee,
			OutTradeRefundNo: refund.OutTradeRefundNo,
		}, nil
	} else {
		return &pb.AliRefundResp{
			Status: code.ALI_PAY_FAIL,
			Desc:   "Msg: " + result.Content.Msg + " SubMsg: " + result.Content.SubMsg,
		}, err
	}
}
