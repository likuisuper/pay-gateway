package logic

import (
	"context"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DyPeriodOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	payDyPeriodOrderModel *model.PmDyPeriodOrderModel
}

func NewDyPeriodOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DyPeriodOrderLogic {
	return &DyPeriodOrderLogic{
		ctx:                   ctx,
		svcCtx:                svcCtx,
		Logger:                logx.WithContext(ctx),
		payDyPeriodOrderModel: model.NewPmDyPeriodOrderModel(define.DbPayGateway),
	}
}

// DyPeriodOrder 查询抖音周期代扣订单
func (l *DyPeriodOrderLogic) DyPeriodOrder(in *pb.DyPeriodOrderReq) (*pb.DyPeriodOrderResp, error) {

	dyPeriodOrder, err := l.payDyPeriodOrderModel.GetOneByOrderSnAndPkg(in.OrderSn, in.AppPkg)
	if err != nil || dyPeriodOrder == nil || dyPeriodOrder.ID < 1 {
		CreateDyRefundFailNum.CounterInc()
		l.Errorf("CreateDyPeriodRefund pkgName= %s, order_sn: %v 获取抖音代扣订单失败 err:=%v", in.AppPkg, in.OrderSn, err)
		return nil, err
	}

	return &pb.DyPeriodOrderResp{
		PayStatus:  int64(dyPeriodOrder.PayStatus),
		SignNo:     dyPeriodOrder.SignNo,
		SignStatus: int64(dyPeriodOrder.SignStatus),
	}, nil
}
