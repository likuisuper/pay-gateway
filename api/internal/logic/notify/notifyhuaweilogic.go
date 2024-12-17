package notify

import (
	"context"
	"encoding/json"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyHuaweiLogic struct {
	logx.Logger
	ctx                  context.Context
	svcCtx               *svc.ServiceContext
	notifyHuaweiLogModel *model.NotifyHuaweiLogModel
	huaweiOrderModel     *model.HuaweiOrderModel
	huaweiAppModel       *model.HuaweiAppModel
}

func NewNotifyHuaweiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyHuaweiLogic {
	return &NotifyHuaweiLogic{
		Logger:               logx.WithContext(ctx),
		ctx:                  ctx,
		svcCtx:               svcCtx,
		notifyHuaweiLogModel: model.NewNotifyHuaweiLogModel(define.DbPayGateway),
		huaweiOrderModel:     model.NewHuaweiOrderModel(define.DbPayGateway),
		huaweiAppModel:       model.NewHuaweiAppModel(define.DbPayGateway),
	}
}

// 华为参考文档
//
// https://developer.huawei.com/consumer/cn/doc/HMSCore-References/api-notifications-about-subscription-events-v2-0000001385268541
//
// 测试环境回调地址: https://test.api.pay-gateway.yunxiacn.com/notify/huawei
//
// 线上环境回调地址: https://pay-gw.muchcloud.com/notify/huawei
func (l *NotifyHuaweiLogic) NotifyHuawei(req *types.HuaweiReq) {
	jsonByte, _ := json.Marshal(req)
	l.Sloww("华为回调记录", logx.Field("data", req), logx.Field("json", string(jsonByte)))

	if req.EventType == "" {
		// 配置的时候保存的回调 不处理 记录一下日志就可以了
		return
	}

	if req.EventType != code.HUAWEI_EVENT_TYPE_SUBSCRIPTION && req.EventType != code.HUAWEI_EVENT_TYPE_ORDER {
		// 参数异常
		l.Errorf("NotifyHuawei param error, unexpected event type:%v", req.EventType)
	}

	// 查询包应用信息
	hwApp, _ := l.huaweiAppModel.GetInfo(req.ApplicationId)
	if hwApp.Id < 1 {
		l.Errorw("huaweiAppModel not found", logx.Field("appId", req.ApplicationId))
	}

	// 记录日志
	logModel := &model.NotifyHuaweiLogTable{
		AppId:  req.ApplicationId,
		AppPkg: hwApp.AppPkg,
		Data:   string(jsonByte),
	}
	l.notifyHuaweiLogModel.Create(logModel)

	if req.EventType == code.HUAWEI_EVENT_TYPE_SUBSCRIPTION {
		// 处理订阅
		l.handleHuaweiSub(req, logModel.Id)
		return
	}

	if req.EventType == code.HUAWEI_EVENT_TYPE_ORDER {
		// 处理订单
		l.handleHuaweiOrder(req, logModel.Id)
		return
	}
}

// 处理订阅流程: https://developer.huawei.com/consumer/cn/doc/HMSCore-Guides/notifications-about-subscription-events-0000001050035037
//
// 3.调用华为IAP服务器提供的Subscription服务验证购买Token接口，查询购买数据及其签名数据。
//
// 4.IAP服务器返回购买数据及其签名数据。为避免资金损失，您在验签成功后，必须校验InAppPurchaseData中的productId、price、currency等信息的一致性。验证方法和公钥获取方式可参见验证InAppPurchaseData。
//
// 5.校验订阅状态提供商品服务。请根据Subscription服务验证购买Token接口响应中InAppPurchaseData的subIsvalid字段决定是否发货。若subIsvalid为true，则执行发货操作。
func (l *NotifyHuaweiLogic) handleHuaweiSub(req *types.HuaweiReq, logId uint64) {
}

// 处理订单流程: https://developer.huawei.com/consumer/cn/doc/HMSCore-Guides/notifications-about-subscription-events-0000001050035037
//
// 3.调用华为IAP服务器提供的Order服务验证购买Token接口，查询购买数据及签名数据。
//
// 4.IAP服务器返回购买数据及签名数据。为避免资金损失，您在验签成功后，必须校验InAppPurchaseData中的productId、price、currency等信息的一致性。验证方法和公钥获取方式可参见验证InAppPurchaseData。
//
// 5.验证结果成功，处理发货，并记录购买商品的Token。请根据Order服务验证购买Token接口响应中InAppPurchaseData的purchaseState字段决定是否发货。若purchaseState为0，则执行发货操作。
//
// 6.调用华为IAP服务器提供的Order服务确认购买接口确认购买（即消耗）。
//
// 7.IAP服务器返回确认购买结果。
func (l *NotifyHuaweiLogic) handleHuaweiOrder(req *types.HuaweiReq, logId uint64) {
}
