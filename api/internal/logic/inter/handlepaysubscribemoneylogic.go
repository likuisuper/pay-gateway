package inter

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/common/response"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/crontab"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/types"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/code"
)

type HandlePaySubscribeMoneyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHandlePaySubscribeMoneyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HandlePaySubscribeMoneyLogic {
	return &HandlePaySubscribeMoneyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HandlePaySubscribeMoneyLogic) HandlePaySubscribeMoney(req *types.EmptyReq) (resp *types.ResultResp, err error) {
	crontabOrder := crontab.GetCrontabOrder()
	crontabOrder.PayOrder()
	res := response.MakeResult(code.CODE_OK, "操作成功", nil)
	return &res, nil
}
