package notify

import (
	"context"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyWechatLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNotifyWechatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyWechatLogic {
	return &NotifyWechatLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NotifyWechatLogic) NotifyWechat(req *types.EmptyReq) (resp *types.ResultResp, err error) {
	// todo: add your logic here and delete this line

	return
}
