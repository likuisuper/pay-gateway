package logic

import (
	"context"

	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AlipayFundTransUniTransferLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAlipayFundTransUniTransferLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayFundTransUniTransferLogic {
	return &AlipayFundTransUniTransferLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 支付宝转出
func (l *AlipayFundTransUniTransferLogic) AlipayFundTransUniTransfer(in *pb.AlipayFundTransUniTransferReq) (*pb.Empty, error) {
	// todo: add your logic here and delete this line

	return &pb.Empty{}, nil
}
