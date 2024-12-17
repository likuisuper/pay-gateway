package notify

import (
	"context"
	"encoding/json"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

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

// https://developer.huawei.com/consumer/cn/doc/HMSCore-References/api-notifications-about-subscription-events-v2-0000001385268541
func (l *NotifyHuaweiLogic) NotifyHuawei(req *types.HuaweiReq) {
	val, _ := json.Marshal(req)
	logx.Sloww("华为回调记录", logx.Field("data", req), logx.Field("json", string(val)))
}
