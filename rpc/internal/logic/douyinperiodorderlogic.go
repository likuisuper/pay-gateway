package logic

import (
	"context"

	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DouyinPeriodOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	payDyPeriodOrderModel *model.PmDyPeriodOrderModel
}

func NewDouyinPeriodOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DouyinPeriodOrderLogic {
	return &DouyinPeriodOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		payDyPeriodOrderModel: model.NewPmDyPeriodOrderModel(define.DbPayGateway),
	}
}

// 抖音周期代扣相关查询和修改
func (l *DouyinPeriodOrderLogic) DouyinPeriodOrder(in *pb.DouyinPeriodOrderReq) (*pb.DouyinPeriodOrderResp, error) {
	resp := pb.DouyinPeriodOrderResp{
		UserId: in.GetUserId(),
		IsSign: 0,
		Msg:    "未签约",
	}

	if in.GetAction() == pb.DouyinPeriodOrderReqAction_DyPeriodActionQuery {
		// 查询
		periodModel, err := l.payDyPeriodOrderModel.GetSignedByUserIdAndPkg(int(in.GetUserId()), in.GetPkg())
		if err != nil || periodModel == nil {
			l.Slowf("DouyinPeriodOrder error: %v, pkg: %s, userId :%d", err, in.GetPkg(), in.GetUserId())

			resp.Msg = err.Error()
			return &resp, nil
		}

		// 已签约
		resp.IsSign = 1
		resp.Msg = "已签约"
		resp.ExpireDate = periodModel.ExpireDate.Format("2006-01-02 15:04:05")
		resp.NextDecuctionTime = periodModel.NextDecuctionTime.Format("2006-01-02 15:04:05")
		resp.DeductionAmount = int64(periodModel.Amount) // 单位分
	}

	return &resp, nil
}
