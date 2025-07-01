package logic

import (
	"context"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type BindHuaweiPayDataLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	huaweiOrderModel *model.HuaweiOrderModel
}

func NewBindHuaweiPayDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindHuaweiPayDataLogic {
	return &BindHuaweiPayDataLogic{
		ctx:              ctx,
		svcCtx:           svcCtx,
		Logger:           logx.WithContext(ctx),
		huaweiOrderModel: model.NewHuaweiOrderModel(define.DbPayGateway),
	}
}

// 绑定订单号和华为购买token
func (l *BindHuaweiPayDataLogic) BindHuaweiPayData(in *pb.BindHuaweiPayDataReq) (*pb.BindHuaweiPayDataResp, error) {
	l.Sloww("BindHuaweiPayData", logx.Field("purchase token", in.GetHwpayToken()), logx.Field("userId", in.GetUserId()), logx.Field("outOrderNo", in.GetOutOrderNo()))

	// 更新订单号和华为购买token
	err := l.huaweiOrderModel.BindToken(in.GetHwpayToken(), int(in.GetUserId()), in.GetOutOrderNo())
	if err != nil {
		return &pb.BindHuaweiPayDataResp{
			Code: 1,
			Msg:  "failed err: " + err.Error(),
		}, nil
	}

	return &pb.BindHuaweiPayDataResp{
		Code: 0,
		Msg:  "success",
	}, nil
}
