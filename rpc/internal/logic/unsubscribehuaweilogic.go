package logic

import (
	"context"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/huawei"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnsubscribeHuaweiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUnsubscribeHuaweiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnsubscribeHuaweiLogic {
	return &UnsubscribeHuaweiLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 用户主动解除华为订阅
func (l *UnsubscribeHuaweiLogic) UnsubscribeHuawei(in *pb.UnsubscribeHuaweiReq) (*pb.UnsubscribeHuaweiResp, error) {
	// 记录一下参数
	l.Sloww("UnsubscribeHuawei", logx.Field("in", in))

	// 获取应用配置
	appConfig, err := model.NewHuaweiAppModel(define.DbPayGateway).GetInfoByPkg(in.GetPkg())
	if appConfig.ID < 1 || err != nil {
		l.Errorf("获取华为应用配置失败 error: %v, pkg: %s", err, in.GetPkg())
		return &pb.UnsubscribeHuaweiResp{
			Code: 1,
			Msg:  "获取华为应用配置失败: " + err.Error(),
		}, nil
	}

	// 发起华为解约
	// https://developer.huawei.com/consumer/cn/doc/HMSCore-References/api-cancel-subscription-0000001050746115
	authHeaderString, err := huawei.NewClient(l.ctx, define.DbPayGateway, appConfig.ClientId, appConfig.ClientSecret, appConfig.AppSecret).BuildAuthorization()
	if err != nil {
		l.Errorf("华为BuildAuthorization失败 error: %v, pkg: %s", err, in.GetPkg())
		return &pb.UnsubscribeHuaweiResp{
			Code: 1,
			Msg:  "华为BuildAuthorization失败: " + err.Error(),
		}, nil
	}

	tmpResult, err := huawei.SubscriptionDemo.StopSubscription(authHeaderString, in.GetSubscriptionId(), in.GetPurchaseToken())
	l.Sloww("huawei StopSubscription", logx.Field("result", tmpResult), logx.Field("err", err))
	if err != nil {
		l.Errorf("华为停止订阅失败 err: %v", err)
		return &pb.UnsubscribeHuaweiResp{
			Code: 1,
			Msg:  "华为停止订阅失败: " + err.Error(),
		}, nil
	}

	// 返回 是否成功需要调用方判断 responseCode 是否为 0
	return &pb.UnsubscribeHuaweiResp{
		Code: 0,
		Msg:  tmpResult,
	}, nil
}
