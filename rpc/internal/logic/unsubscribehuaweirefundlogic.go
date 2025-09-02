package logic

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/huawei"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnsubscribeHuaweiRefundLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUnsubscribeHuaweiRefundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnsubscribeHuaweiRefundLogic {
	return &UnsubscribeHuaweiRefundLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 用户主动解除华为订阅后的退款操作
//
// 本接口仅进行最近一次收据的退费操作，不会对订阅产生额外影响，订阅可以继续正常使用，到期后也会进行自动续费。
//
// 当用户意外购买订阅型商品申诉退款时，可以调用返还订阅费用接口退还用户费用，让用户在本周期免费试用后考虑是否留存订阅关系。
//
// 若用户下周期不想继续使用该订阅，您可主动调用取消订阅接口进行取消处理，也可让用户自行在HMS Core（APK）的订阅管理页中取消订阅。
func (l *UnsubscribeHuaweiRefundLogic) UnsubscribeHuaweiRefund(in *pb.UnsubscribeHuaweiReq) (*pb.UnsubscribeHuaweiResp, error) {
	// 记录一下参数
	l.Sloww("UnsubscribeHuaweiRefund", logx.Field("in", in))

	// 获取应用配置
	appConfig, err := model.NewHuaweiAppModel(define.DbPayGateway).GetInfoByPkg(in.GetPkg())
	if appConfig.ID < 1 || err != nil {
		l.Errorf("获取华为应用配置失败 error: %v, pkg: %s", err, in.GetPkg())
		return &pb.UnsubscribeHuaweiResp{
			Code: 1,
			Msg:  "获取华为应用配置失败: " + err.Error(),
		}, nil
	}

	// 发起华为解约后的退款
	// https://developer.huawei.com/consumer/cn/doc/HMSCore-References/api-refund-subscription-fee-0000001050986131
	authHeaderString, err := huawei.NewClient(l.ctx, define.DbPayGateway, appConfig.ClientId, appConfig.ClientSecret, appConfig.AppSecret).BuildAuthorization()
	if err != nil {
		l.Errorf("华为BuildAuthorization失败 error: %v, pkg: %s", err, in.GetPkg())
		return &pb.UnsubscribeHuaweiResp{
			Code: 1,
			Msg:  "华为BuildAuthorization失败: " + err.Error(),
		}, nil
	}

	tmpResult, err := huawei.SubscriptionDemo.ReturnFeeSubscription(authHeaderString, in.GetSubscriptionId(), in.GetPurchaseToken())
	l.Sloww("huawei ReturnFeeSubscription", logx.Field("result", tmpResult), logx.Field("err", err))
	if err != nil {
		l.Errorf("华为订阅后退款失败 err: %v", err)
		return &pb.UnsubscribeHuaweiResp{
			Code: 1,
			Msg:  "华为订阅后退款失败: " + err.Error(),
		}, nil
	}

	var tmpRe HuaweiStopSubscriptionResp
	err = json.Unmarshal([]byte(tmpResult), &tmpRe)
	if err != nil {
		l.Errorf("华为订阅后退款失败 json.Unmarshal err: %v", err)
		return &pb.UnsubscribeHuaweiResp{
			Code: 1,
			Msg:  "华为订阅后退款失败: " + err.Error(),
		}, nil
	}

	if tmpRe.ResponseCode != "0" {
		msg := fmt.Sprintf("华为停止订阅失败 responseCode: %s, responseMessage: %s", tmpRe.ResponseCode, tmpRe.ResponseMessage)
		l.Error(msg)
		return &pb.UnsubscribeHuaweiResp{
			Code: 1,
			Msg:  msg,
		}, nil
	}

	// 成功
	return &pb.UnsubscribeHuaweiResp{
		Code: 0,
		Msg:  tmpResult,
	}, nil
}
