package logic

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/utils"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type WechatRefundOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	orderModel           *model.OrderModel
	refundModel          *model.RefundModel
	appConfigModel       *model.PmAppConfigModel
	payConfigWechatModel *model.PmPayConfigWechatModel
}

func NewWechatRefundOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WechatRefundOrderLogic {
	return &WechatRefundOrderLogic{
		ctx:                  ctx,
		svcCtx:               svcCtx,
		Logger:               logx.WithContext(ctx),
		orderModel:           model.NewOrderModel(define.DbPayGateway),
		refundModel:          model.NewRefundModel(define.DbPayGateway),
		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),

	}
}

// 微信统一支付退款
func (l *WechatRefundOrderLogic) WechatRefundOrder(in *pb.WechatRefundOrderReq) (*pb.CreateRefundResp, error) {
	order, err := l.orderModel.GetOneByOutTradeNo(in.OutTradeNo)
	if err != nil {
		util.CheckError("获取订单失败,订单号=%s，err:=%v", in.OutTradeNo, err)
		err = errors.New("读取应用配置失败")
		return nil, err
	}
	if order.Status != code.ORDER_SUCCESS {
		err = errors.New("订单状态错误")
		return nil, err
	}
	payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(order.PayAppID)
	if cfgErr != nil {
		err = fmt.Errorf("pkgName= %s, 读取微信支付配置失败，err:=%v", order.AppPkg, cfgErr)
		util.CheckError(err.Error())
		return nil, cfgErr
	}
	payClient := client.NewWeChatCommPay(*payCfg.TransClientConfig())

	data := &client.RefundOrder{
		OutTradeNo:  in.OutTradeNo,
		OutRefundNo: "t" + utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
		TotalFee:    in.TotalFee,
		RefundFee:   in.RefundFee,
		TransactionId: order.PlatformTradeNo,
	}

	refundRes, refundErr := payClient.RefundOrder(data)
	if refundErr != nil {
		err = fmt.Errorf("发起退款失败:order_sn = %s .err =%v ", order.OutTradeNo, refundErr)
		util.CheckError(err.Error())
		return nil, refundErr
	}
	//修改订单状态为退款中
	order.Status = code.ORDER_REFUNDING
	l.orderModel.UpdateNotify(order)
	//创建退款订单
	refund := model.RefundTable{
		PayType:          order.PayType,
		OutTradeNo:       order.OutTradeNo,
		OutTradeRefundNo: utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
		Reason:           in.RefundReason,
		RefundAmount:     int(in.RefundFee),
		NotifyUrl:        order.AppNotifyUrl,
		Operator:         "系统",
		AppPkg:           order.AppPkg,
		RefundNo:         *refundRes.OutRefundNo,
		ReviewerComment:  "自动退款",
		RefundedAt:       time.Now(),
	}
	l.refundModel.Create(&refund)

	return &pb.CreateRefundResp{
		OutTradeRefundNo: order.OutTradeNo,
		Desc:             "OK",
	}, nil
}
