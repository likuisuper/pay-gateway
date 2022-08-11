package logic

import (
	"context"

	"gitee.com/zhuyunkj/pay-gateway/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type OrderPayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewOrderPayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OrderPayLogic {
	return &OrderPayLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建支付订单
func (l *OrderPayLogic) OrderPay(in *pb.OrderPayReq) (*pb.OrderPayResp, error) {
	// todo: add your logic here and delete this line

	return &pb.OrderPayResp{}, nil
}
