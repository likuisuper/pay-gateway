package logic

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/utils"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	kv_m "gitlab.muchcloud.com/consumer-project/zhuyun-core/kv_monitor"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AlipayCreateRefundLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	refundModel *model.RefundModel
	orderModel  *model.OrderModel
}

var (
	createRefundErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "createRefundErr", nil, "创建退款单失败", nil})}
)

func NewAlipayCreateRefundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayCreateRefundLogic {
	return &AlipayCreateRefundLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		refundModel: model.NewRefundModel(define.DbPayGateway),
		orderModel:  model.NewOrderModel(define.DbPayGateway),
	}
}

// 支付宝：创建退款订单
func (l *AlipayCreateRefundLogic) AlipayCreateRefund(in *pb.AlipayRefundReq) (*pb.CreateRefundResp, error) {
	order, err := l.orderModel.GetOneByOutTradeNo(in.OutTradeNo)
	if err != nil || order == nil || order.ID < 1 {
		errInfo := fmt.Sprintf("创建退款订单：获取订单失败!!! %s", in.OutTradeNo)
		logx.Error(errInfo)
		createRefundErr.CounterInc()
		return nil, errors.New(errInfo)
	}

	if order.Status != model.PmPayOrderTablePayStatusPaid {
		errInfo := fmt.Sprintf("创建退款订单：订单状态错误!!! %s", in.OutTradeNo)
		logx.Error(errInfo)
		createRefundErr.CounterInc()
		return nil, errors.New(errInfo)
	}

	refundAmountFloat, _ := strconv.ParseFloat(in.RefundAmount, 64)
	refundAmount := int(refundAmountFloat * 100)

	if refundAmount > order.Amount {
		errInfo := fmt.Sprintf("创建退款订单：退款金额大于支付金额!!! %s", in.OutTradeNo)
		logx.Error(errInfo)
		createRefundErr.CounterInc()
		return nil, errors.New(errInfo)
	}

	refund := model.RefundTable{
		PayType:          order.PayType,
		OutTradeNo:       order.OutTradeNo,
		OutTradeRefundNo: utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
		Reason:           in.RefundReason,
		RefundAmount:     refundAmount,
		NotifyUrl:        in.RefundNotifyUrl,
		Operator:         in.Operator,
		AppPkg:           order.AppPkg,
		RefundNo:         in.TradeNo, // 支付宝退款没有退款单号
	}
	err = l.refundModel.Create(&refund)
	if err != nil {
		errInfo := fmt.Sprintf("创建退款订单失败!!! %s", in.OutTradeNo)
		logx.Error(errInfo)
		createRefundErr.CounterInc()
		return nil, errors.New(errInfo)
	}

	return &pb.CreateRefundResp{
		OutTradeRefundNo: refund.OutTradeRefundNo,
		Desc:             "创建退款订单成功",
	}, nil
}
