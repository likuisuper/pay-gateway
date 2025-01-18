package notify

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

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

// 华为参考文档:
// https://developer.huawei.com/consumer/cn/doc/HMSCore-Guides/notifications-about-subscription-events-0000001050035037
//
// https://developer.huawei.com/consumer/cn/doc/HMSCore-References/api-notifications-about-subscription-events-v2-0000001385268541
//
// 测试环境回调地址: https://test.api.pay-gateway.yunxiacn.com/notify/huawei
//
// 线上环境回调地址: https://pay-gw.muchcloud.com/notify/huawei
func (l *NotifyHuaweiLogic) NotifyHuawei(req *types.HuaweiReq) *huawei.NotificationResponse {
	jsonByte, _ := json.Marshal(req)
	l.Sloww("华为回调记录", logx.Field("data", req), logx.Field("json", string(jsonByte)))

	if req.EventType == "" {
		// 配置的时候保存的回调 不处理 记录一下日志就可以了
		return nil
	}

	if req.EventType != huawei.HUAWEI_EVENT_TYPE_SUBSCRIPTION && req.EventType != huawei.HUAWEI_EVENT_TYPE_ORDER {
		// 参数异常
		l.Errorf("NotifyHuawei param error, unexpected event type:%v", req.EventType)
		return nil
	}

	// 查询包应用信息
	hwApp, _ := l.huaweiAppModel.GetInfo(req.ApplicationId)
	if hwApp.ID < 1 {
		l.Errorw("huaweiAppModel not found", logx.Field("appId", req.ApplicationId))
		return nil
	}

	// 记录日志
	logModel := &model.NotifyHuaweiLogTable{
		AppId:  req.ApplicationId,
		AppPkg: hwApp.AppPkg,
		Data:   string(jsonByte),
	}
	l.notifyHuaweiLogModel.Create(logModel)

	if req.EventType == huawei.HUAWEI_EVENT_TYPE_SUBSCRIPTION {
		// 处理订阅
		res, err := l.handleHuaweiSub(req, hwApp, logModel.Id)
		if err == nil {
			return res
		}
		l.Errorf("handleHuaweiSub error: %v", err)
	} else if req.EventType == huawei.HUAWEI_EVENT_TYPE_ORDER {
		// 处理订单
		res, err := l.handleHuaweiOrder(req, hwApp, logModel.Id)
		if err == nil {
			return res
		}

		l.Errorf("handleHuaweiOrder error: %v", err)
	}

	return nil
}

// 处理订阅流程: https://developer.huawei.com/consumer/cn/doc/HMSCore-Guides/notifications-about-subscription-events-0000001050035037
// 5.校验订阅状态提供商品服务。请根据Subscription服务验证购买Token接口响应中InAppPurchaseData的subIsvalid字段决定是否发货。若subIsvalid为true，则执行发货操作。
func (l *NotifyHuaweiLogic) handleHuaweiSub(req *types.HuaweiReq, hwApp *model.HuaweiAppTable, logId int) (*huawei.NotificationResponse, error) {
	// 验证签名数据
	err := huawei.VerifyRsaSign(req.SubNotification.StatusUpdateNotification, req.SubNotification.NotificationSignature, hwApp.IapPublicKey)
	if err != nil {
		l.Errorf("handleHuaweiSub huawei.VerifyRsaSign error: %v, hwApp : %v", err, hwApp)
		return nil, err
	}

	// 解析数据
	var info huawei.StatusUpdateNotification
	err = json.Unmarshal([]byte(req.SubNotification.StatusUpdateNotification), &info)
	if err != nil {
		l.Errorf("json.Unmarshal error: %v, raw string: %s", err, req.SubNotification.StatusUpdateNotification)
		return nil, err
	}

	if info.SubscriptionId == "" || info.PurchaseToken == "" || info.LatestReceiptInfo == "" {
		l.Errorf("subscription some data is empty, raw data:%v", req.SubNotification.StatusUpdateNotification)
		return nil, errors.New("subscription some data is empty")
	}

	// 将latestReceiptInfo解析成具体InAppPurchaseData数据
	var purchaseData huawei.InAppPurchaseData
	err = json.Unmarshal([]byte(info.LatestReceiptInfo), &purchaseData)
	if err != nil {
		l.Errorf("json.Unmarshal error: %v, raw string: %s", err, info.LatestReceiptInfo)
		return nil, err
	}

	// 未完成购买或者已经过期，或者购买后已经退款
	if !purchaseData.SubIsvalid {
		// 订阅失效
		l.Sloww("purchaseData is not valid", logx.Field("purchaseData", purchaseData))
		response := huawei.NotificationResponse{ErrorCode: "0"}
		return &response, nil
	}

	// 未购买成功
	if purchaseData.PurchaseState != 0 || purchaseData.PurchaseTime < 10000 {
		l.Sloww("purchaseData is not success", logx.Field("purchaseData", purchaseData))
		response := huawei.NotificationResponse{ErrorCode: "0"}
		return &response, nil
	}

	// 根据购买token查找订单数据
	hworder, err := l.huaweiOrderModel.GetOneByToken(info.PurchaseToken)
	if err != nil {
		l.Errorf("GetOneByToken error: %v, token: %s", err, info.PurchaseToken)
		return nil, err
	}

	if hworder == nil {
		err = errors.New("获取订单失败")
		l.Error(err.Error() + " 订单为空, token: " + info.PurchaseToken)
		return nil, err
	}

	if hworder.ProductId != info.ProductId {
		err = errors.New("商品id不一致")
		l.Errorf(err.Error()+" 数据库商品id: %s, 回传商品id: %s", hworder.ProductId, info.ProductId)
		return nil, err
	}

	if hworder.AppId != info.ApplicationId {
		err = errors.New("应用id不一致")
		l.Errorf(err.Error()+" 数据库app id: %s, 回传app id: %s", hworder.AppId, info.ApplicationId)
		return nil, err
	}

	// 提供服务
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

	// 购买时间
	purchaseTime := time.Unix(int64(purchaseData.PurchaseTime/1000), 0)

	// 更新数据
	updateData := map[string]interface{}{
		"log_id":            logId,
		"version":           req.Version,
		"event_type":        req.EventType,
		"notify_time":       int(req.NotifyTime / 1000), // 毫秒转成秒级时间戳
		"notification_type": notificationType,
		"environment":       strings.ToLower(info.Environment),
		"pay_order_id":      info.OrderId,
		"platform_trade_no": info.OrderId,
		"subscription_id":   purchaseData.SubscriptionId,
		"auto_renew_status": info.AutoRenewStatus,
		"status":            1, // TODO: 还有退款等其他
		"pay_time":          purchaseTime.Format("2006-01-02 15:04:05"),
		"expiration_date":   int(purchaseData.ExpirationDate / 1000),
	}
	err = l.huaweiOrderModel.UpdateData(hworder.Id, updateData)
	if err != nil {
		return nil, err
	}

	// 续费的时候需要回调
	// TODO:

	response := huawei.NotificationResponse{ErrorCode: "0"}
	return &response, nil
}

// 处理订单流程: https://developer.huawei.com/consumer/cn/doc/HMSCore-Guides/notifications-about-subscription-events-0000001050035037
// 5.验证结果成功，处理发货，并记录购买商品的Token。请根据Order服务验证购买Token接口响应中InAppPurchaseData的purchaseState字段决定是否发货。若purchaseState为0，则执行发货操作。
// 6.调用华为IAP服务器提供的Order服务确认购买接口确认购买（即消耗）。
// 7.IAP服务器返回确认购买结果。
func (l *NotifyHuaweiLogic) handleHuaweiOrder(req *types.HuaweiReq, hwApp *model.HuaweiAppTable, logId int) (*huawei.NotificationResponse, error) {
	response := huawei.NotificationResponse{ErrorCode: "0"}
	return &response, nil
}
