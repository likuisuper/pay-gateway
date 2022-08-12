package notify

import (
	"context"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyBytedanceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNotifyBytedanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyBytedanceLogic {
	return &NotifyBytedanceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NotifyBytedanceLogic) NotifyBytedance(req *types.ByteDanceReq) (resp *types.ByteDanceResp, err error) {
	// todo: add your logic here and delete this line

	return
}
