package notify

import (
	"context"
	"encoding/json"
	"errors"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/huawei"
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
	authHeaderString     string // 华为请求头需要使用Access Token进行鉴权
}

func NewNotifyHuaweiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyHuaweiLogic {
	return &NotifyHuaweiLogic{
		Logger:               logx.WithContext(ctx),
		ctx:                  ctx,
		svcCtx:               svcCtx,
		notifyHuaweiLogModel: model.NewNotifyHuaweiLogModel(define.DbPayGateway),
		huaweiOrderModel:     model.NewHuaweiOrderModel(define.DbPayGateway),
		huaweiAppModel:       model.NewHuaweiAppModel(define.DbPayGateway),
		authHeaderString:     "",
	}
}

// 华为参考文档: https://developer.huawei.com/consumer/cn/doc/HMSCore-Guides/notifications-about-subscription-events-0000001050035037
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

	if req.EventType != huawei.HUAWEI_EVENT_TYPE_SUBSCRIPTION && req.EventType != huawei.HUAWEI_EVENT_TYPE_ORDER {
		// 参数异常
		l.Errorf("NotifyHuawei param error, unexpected event type:%v", req.EventType)
		return
	}

	// 查询包应用信息
	hwApp, _ := l.huaweiAppModel.GetInfo(req.ApplicationId)
	if hwApp.ID < 1 {
		l.Errorw("huaweiAppModel not found", logx.Field("appId", req.ApplicationId))
		return
	}

	// 获取一下华为token请求头信息
	huaweiAtClient := huawei.NewClient(l.ctx, define.DbPayGateway, hwApp.ClientId, hwApp.ClientSecret, hwApp.AppSecret)
	tmpHeaderStr, err := huaweiAtClient.BuildAuthorization()
	if err != nil {
		l.Errorf("huaweiAtClient.BuildAuthorization err:%v", err)
		return
	}
	l.authHeaderString = tmpHeaderStr

	logx.Infow("NotifyHuawei", logx.Field("authHeaderString", tmpHeaderStr))

	// 记录日志
	logModel := &model.NotifyHuaweiLogTable{
		AppId:  req.ApplicationId,
		AppPkg: hwApp.AppPkg,
		Data:   string(jsonByte),
	}
	l.notifyHuaweiLogModel.Create(logModel)

	// if req.EventType == huawei.HUAWEI_EVENT_TYPE_SUBSCRIPTION {
	// 	// 处理订阅
	// 	l.handleHuaweiSub(req, logModel.Id, hwApp.IapPublicKey)
	// } else if req.EventType == huawei.HUAWEI_EVENT_TYPE_ORDER {
	// 	// 处理订单
	// 	l.handleHuaweiOrder(req, logModel.Id, hwApp.IapPublicKey)
	// }
}

// 处理订阅流程: https://developer.huawei.com/consumer/cn/doc/HMSCore-Guides/notifications-about-subscription-events-0000001050035037
//
// 验证InAppPurchaseData: https://developer.huawei.com/consumer/cn/doc/HMSCore-Guides/verifying-inapppurchasedata-0000001494212281
//
// 3.调用华为IAP服务器提供的Subscription服务验证购买Token接口，查询购买数据及其签名数据。
//
// 4.IAP服务器返回购买数据及其签名数据。为避免资金损失，您在验签成功后，必须校验InAppPurchaseData中的productId、price、currency等信息的一致性。验证方法和公钥获取方式可参见验证InAppPurchaseData。
//
// 5.校验订阅状态提供商品服务。请根据Subscription服务验证购买Token接口响应中InAppPurchaseData的subIsvalid字段决定是否发货。若subIsvalid为true，则执行发货操作。
func (l *NotifyHuaweiLogic) handleHuaweiSub(req *types.HuaweiReq, logId uint64, appPublicKey string) (*huawei.NotificationResponse, error) {
	var info huawei.StatusUpdateNotification
	err := json.Unmarshal([]byte(req.SubNotification.StatusUpdateNotification), &info)
	if err != nil {
		l.Errorf("json.Unmarshal error: %v, raw string: %s", err, req.SubNotification.StatusUpdateNotification)
		return nil, err
	}

	if info.SubscriptionId == "" || info.PurchaseToken == "" {
		l.Errorf("SubscriptionId or PurchaseToken is empty raw data:%v", req.SubNotification.StatusUpdateNotification)
		return nil, errors.New("SubscriptionId or PurchaseToken is empty")
	}

	// 3.调用华为IAP服务器提供的Subscription服务验证购买Token接口，查询购买数据及其签名数据。
	hwCommonResp, err := huawei.SubscriptionDemo.GetSubscription(l.authHeaderString, info.SubscriptionId, info.PurchaseToken)
	if err != nil {
		return nil, err
	}

	// 验证签名数据
	err = huawei.VerifyRsaSign(hwCommonResp.InappPurchaseData, hwCommonResp.DataSignature, appPublicKey)
	if err != nil {
		l.Errorf("handleHuaweiSub huawei.VerifyRsaSign error: %v", err)
		return nil, err
	}

	// https://developer.huawei.com/consumer/cn/doc/HMSCore-Guides/verifying-inapppurchasedata-0000001494212281
	// 4.IAP服务器返回购买数据及其签名数据。为避免资金损失，您在验签成功后，必须校验InAppPurchaseData中的productId、price、currency等信息与下单的一致性。验证方法和公钥获取方式可参见验证InAppPurchaseData。
	// 第4步暂时跳过了
	// todo

	// 5.校验订阅状态提供商品服务。请根据Subscription服务验证购买Token接口响应中InAppPurchaseData的subIsvalid字段决定是否发货。若subIsvalid为true，则执行发货操作。
	var tmpInappPurData huawei.InAppPurchaseData
	err = json.Unmarshal([]byte(hwCommonResp.InappPurchaseData), &tmpInappPurData)
	if err != nil {
		l.Errorf("json.Unmarshal error: %v, raw InappPurchaseData string: %s", err, hwCommonResp.InappPurchaseData)
		return nil, err
	}

	// 通知事件的类型
	notificationType := info.NotificationType
	switch notificationType {
	case huawei.NOTIFICATION_TYPE_INITIAL_BUY:
	case huawei.NOTIFICATION_TYPE_CANCEL:
	case huawei.NOTIFICATION_TYPE_RENEWAL:
	case huawei.NOTIFICATION_TYPE_INTERACTIVE_RENEWAL:
	case huawei.NOTIFICATION_TYPE_NEW_RENEWAL_PREF:
	case huawei.NOTIFICATION_TYPE_RENEWAL_STOPPED:
	case huawei.NOTIFICATION_TYPE_RENEWAL_RESTORED:
	case huawei.NOTIFICATION_TYPE_RENEWAL_RECURRING:
	case huawei.NOTIFICATION_TYPE_ON_HOLD:
	case huawei.NOTIFICATION_TYPE_PAUSED:
	case huawei.NOTIFICATION_TYPE_PAUSE_PLAN_CHANGED:
	case huawei.NOTIFICATION_TYPE_PRICE_CHANGE_CONFIRMED:
	case huawei.NOTIFICATION_TYPE_DEFERRED:
	default:
	}

	response := huawei.NotificationResponse{ErrorCode: "0"}
	return &response, nil
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
func (l *NotifyHuaweiLogic) handleHuaweiOrder(req *types.HuaweiReq, logId uint64, appPublicKey string) {
}
