package health

import (
	"context"
	"time"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/common/response"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type HealthCheckLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHealthCheckLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HealthCheckLogic {
	return &HealthCheckLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HealthCheckLogic) HealthCheck(req *types.HeaderReq) (resp *types.ResultResp, err error) {
	now := time.Now()
	time.Sleep(time.Millisecond)
	data := response.HealthResp{
		Time: time.Since(now).Seconds(),
		Sign: req.Xsign,
	}
	res := response.MakeResult(200, "", data)
	return &res, nil
}
