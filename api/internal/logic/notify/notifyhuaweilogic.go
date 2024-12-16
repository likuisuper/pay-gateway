package notify

import (
	"context"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	jsoniter "github.com/json-iterator/go"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyHuaweiLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNotifyHuaweiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyHuaweiLogic {
	return &NotifyHuaweiLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NotifyHuaweiLogic) NotifyHuawei(req *types.HuaweiReq) (resp *types.HuaweiResp, err error) {

	val, _ := jsoniter.Marshal(req)
	logx.Sloww("华为回调记录", logx.Field("data", req), logx.Field("json", string(val)))
	return
}
