package inter

import (
	"context"
	"gitee.com/zhuyunkj/pay-gateway/api/common/response"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/crontab"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"github.com/zeromicro/go-zero/core/logx"
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
