package notify

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/huawei"
	"gitee.com/zhuyunkj/pay-gateway/common/utils"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"

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
	l.Sloww("华为回调记录", logx.Field("req", req), logx.Field("json", string(jsonByte)))

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
		err := l.handleHuaweiSub(req, hwApp, logModel.Id)
		if err != nil {
			l.Errorf("handleHuaweiSub error: %v", err)
		}
	} else if req.EventType == huawei.HUAWEI_EVENT_TYPE_ORDER {
		// 处理订单
		err := l.handleHuaweiOrder(req, logModel.Id)
		if err != nil {
			l.Errorf("handleHuaweiOrder error: %v", err)
		}
	}

	// 统一返回处理成功 避免华为重试
	return &huawei.NotificationResponse{
		ErrorCode: "0",
	}
}

// 处理订阅流程: https://developer.huawei.com/consumer/cn/doc/HMSCore-Guides/notifications-about-subscription-events-0000001050035037
// 5.校验订阅状态提供商品服务。请根据Subscription服务验证购买Token接口响应中InAppPurchaseData的subIsvalid字段决定是否发货。若subIsvalid为true，则执行发货操作。
func (l *NotifyHuaweiLogic) handleHuaweiSub(req *types.HuaweiReq, hwApp *model.HuaweiAppTable, logId int) error {
	// 验证签名数据
	err := huawei.VerifyRsaSign(req.SubNotification.StatusUpdateNotification, req.SubNotification.NotificationSignature, hwApp.IapPublicKey)
	if err != nil {
		l.Errorf("handleHuaweiSub huawei.VerifyRsaSign error: %v, hwApp : %v", err, hwApp)
		return err
	}

	// 通知描述
	notifyDesc := ""

	// 解析数据
	var info huawei.StatusUpdateNotification
	err = json.Unmarshal([]byte(req.SubNotification.StatusUpdateNotification), &info)
	if err != nil {
		l.Errorf("json.Unmarshal error: %v, raw string: %s", err, req.SubNotification.StatusUpdateNotification)
		return err
	}

	var purchaseData huawei.InAppPurchaseData

	// 取消订阅时间
	cancellationTime := 0

	if info.NotificationType == 1 || info.LatestReceiptInfo == "" {
		// 都是订阅取消
		if info.CancellationDate > 1000 {
			cancellationTime = int(info.CancellationDate) / 1000
		}
	} else {
		if info.SubscriptionId == "" || info.PurchaseToken == "" || info.LatestReceiptInfo == "" {
			l.Errorf("subscription some data is empty, raw data:%v", req.SubNotification.StatusUpdateNotification)
			return errors.New("subscription some data is empty")
		}

		// 将latestReceiptInfo解析成具体InAppPurchaseData数据
		err = json.Unmarshal([]byte(info.LatestReceiptInfo), &purchaseData)
		if err != nil {
			l.Errorf("json.Unmarshal error: %v, raw string: %s", err, info.LatestReceiptInfo)
			return err
		}

		if purchaseData.Kind != 2 {
			// 非订阅型商品 直接返回
			l.Errorf("非订阅型商品 kind not equal 2, req: %v", req)
			return nil
		}

		// 未完成购买或者已经过期，或者购买后已经退款
		if !purchaseData.SubIsvalid {
			// 订阅失效
			l.Sloww("purchaseData is not valid", logx.Field("purchaseData", purchaseData))
			return nil
		}

		// 购买成功
		if purchaseData.PurchaseState == 0 && purchaseData.PurchaseTime > 10000 {
			notifyDesc = "订阅购买成功"
		} else if purchaseData.PurchaseState == 1 {
			notifyDesc = "订阅取消"
		} else if purchaseData.PurchaseState == 2 {
			notifyDesc = "订阅已退款"
		} else {
			notifyDesc = "订阅其他状态"
		}
	}

	// 根据购买token查找订单数据
	hworder, err := l.huaweiOrderModel.GetOneByTokenAndSubId(info.PurchaseToken, info.SubscriptionId, purchaseData.OriSubscriptionId, purchaseData.OrderId)
	if err != nil {
		l.Errorf("GetOneByTokenAndSubId error: %v, token: %s", err, info.PurchaseToken)
		return err
	}

	if hworder == nil || hworder.Id < 1 {
		err = errors.New("获取订单失败")
		l.Error(err.Error() + " 订单为空, token: " + info.PurchaseToken)
		return err
	}

	if hworder.AppId != info.ApplicationId {
		err = errors.New("应用id不一致")
		l.Errorf(err.Error()+" 数据库app id: %s, 回传app id: %s", hworder.AppId, info.ApplicationId)
		return err
	}

	// 购买时间
	var purchaseTime string
	if purchaseData.OriPurchaseTime > 1000 {
		// OriPurchaseTime 原购买时间，UTC时间戳，以毫秒为单位
		purchaseTime = time.Unix(int64(purchaseData.OriPurchaseTime/1000), 0).Format("2006-01-02 15:04:05")
	} else if purchaseData.PurchaseTime > 1000 {
		purchaseTime = time.Unix(int64(purchaseData.PurchaseTime/1000), 0).Format("2006-01-02 15:04:05")
	}

	// 使用回调来 提供服务
	// 通知事件的类型
	notificationType := info.NotificationType

	// switch notificationType {
	// case huawei.NOTIFICATION_TYPE_INITIAL_BUY:
	// case huawei.NOTIFICATION_TYPE_CANCEL:
	// case huawei.NOTIFICATION_TYPE_RENEWAL:
	// case huawei.NOTIFICATION_TYPE_INTERACTIVE_RENEWAL:
	// case huawei.NOTIFICATION_TYPE_NEW_RENEWAL_PREF:
	// case huawei.NOTIFICATION_TYPE_RENEWAL_STOPPED:
	// case huawei.NOTIFICATION_TYPE_RENEWAL_RESTORED:
	// case huawei.NOTIFICATION_TYPE_RENEWAL_RECURRING:
	// case huawei.NOTIFICATION_TYPE_ON_HOLD:
	// case huawei.NOTIFICATION_TYPE_PAUSED:
	// case huawei.NOTIFICATION_TYPE_PAUSE_PLAN_CHANGED:
	// case huawei.NOTIFICATION_TYPE_PRICE_CHANGE_CONFIRMED:
	// case huawei.NOTIFICATION_TYPE_DEFERRED:
	// default:
	// }

	// 更新数据
	var updateData map[string]interface{}
	if hworder.Status == 0 {
		// 未处理过
		updateData = map[string]interface{}{
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
			"status":            1, // 已处理
			"pay_time":          purchaseTime,
			"expiration_date":   int(purchaseData.ExpirationDate / 1000),
		}
	} else {
		// 已经处理过
		updateData = map[string]interface{}{
			"version":           req.Version,
			"event_type":        req.EventType,
			"notify_time":       int(req.NotifyTime / 1000), // 毫秒转成秒级时间戳
			"notification_type": notificationType,
			"auto_renew_status": info.AutoRenewStatus,
		}

		if purchaseTime != "" {
			updateData["pay_time"] = purchaseTime
		}

		if info.Environment != "" {
			updateData["environment"] = strings.ToLower(info.Environment)
		}

		if info.OrderId != "" {
			updateData["pay_order_id"] = info.OrderId
			updateData["platform_trade_no"] = info.OrderId
		}

		if purchaseData.SubscriptionId != "" {
			updateData["subscription_id"] = purchaseData.SubscriptionId
		}

		if purchaseData.ExpirationDate > 1000 {
			updateData["expiration_date"] = int(purchaseData.ExpirationDate / 1000)
		}
	}

	newProductId := ""
	if info.ProductId != "" && hworder.ProductId != info.ProductId {
		// 商品id不一致 更改了订阅商品id
		updateData["product_id"] = info.ProductId
		newProductId = info.ProductId
	}

	if cancellationTime < 1 && purchaseData.CancellationTime > 1000 {
		cancellationTime = int(purchaseData.CancellationTime / 1000)
	}
	if cancellationTime > 0 {
		updateData["cancellation_date"] = cancellationTime
	} else {
		updateData["cancellation_date"] = 0
	}

	err = l.huaweiOrderModel.UpdateData(hworder.Id, updateData)
	if err != nil {
		return err
	}

	// 回调业务方接口 使用回调来 提供服务 回调才是处理实际逻辑的地方
	// 成功操作时 异步回调 把订单号、通知类型、用户id、包名返回
	if hworder.AppNotifyUrl != "" {
		go util.SafeRun(func() {
			// 回调数据
			callbackData := map[string]interface{}{
				"notify_type":       code.APP_NOTIFY_HUAWEI_PRODUCT_SUBSCIRBE,
				"user_id":           hworder.UserId,
				"out_trade_no":      hworder.OutTradeNo,
				"pkg":               hworder.AppPkg,
				"notification_type": notificationType,
				"new_product_id":    newProductId,     // 新商品id
				"notidy_desc":       notifyDesc,       // 通知描述
				"cancellation_time": cancellationTime, // 取消订阅时间
			}

			headerMap := map[string]string{
				"App-Origin": hworder.AppPkg,
			}
			l.Sloww("华为回调app数据", logx.Field("call back data", callbackData))
			tmpErr := utils.CallbackWithRetry(hworder.AppNotifyUrl, headerMap, callbackData, 5*time.Second)
			if tmpErr != nil {
				l.Errorf("callback error: %v, call back data: %v", tmpErr, callbackData)
			}
		})
	} else {
		logx.Errorf("order id:%d, out_trade_no:%s, app notify url is empty", hworder.Id, hworder.OutTradeNo)
	}

	return nil
}

// 处理订单流程: https://developer.huawei.com/consumer/cn/doc/HMSCore-Guides/notifications-about-subscription-events-0000001050035037
// 5.验证结果成功，处理发货，并记录购买商品的Token。请根据Order服务验证购买Token接口响应中InAppPurchaseData的purchaseState字段决定是否发货。若purchaseState为0，则执行发货操作。
// 6.调用华为IAP服务器提供的Order服务确认购买接口确认购买（即消耗）。
// 7.IAP服务器返回确认购买结果。
func (l *NotifyHuaweiLogic) handleHuaweiOrder(req *types.HuaweiReq, logId int) error {
	// 订单 notificationType 通知事件的类型，取值如下：1：支付成功 2：退款成功
	if req.OrderNotification.NotificationType != 1 {
		// 非支付成功直接返回
		l.Errorw("订单非支付成功", logx.Field("req", req))
		return nil
	}

	purchaseToken := req.OrderNotification.PurchaseToken
	if purchaseToken == "" {
		err := errors.New("订单购买token为空")
		l.Error(err.Error())
		return err
	}

	// 根据购买token查找订单数据
	hworder, err := l.huaweiOrderModel.GetOneByTokenAndSubId(purchaseToken, "", "", "")
	if err != nil {
		l.Errorf("GetOneByTokenAndSubId error: %v, token: %s", err, purchaseToken)
		return err
	}

	if hworder == nil || hworder.Id < 1 {
		err = errors.New("获取订单失败")
		l.Error(err.Error() + " 订单为空, token: " + purchaseToken)
		return err
	}

	// 购买型只处理一次
	if hworder.Status != 0 {
		// 订单已处理
		l.Slowf("订单已处理 purchase_token: %s, order id: %d", purchaseToken, hworder.Id)
		return nil
	}

	// 验证数据
	if hworder.ProductId != req.OrderNotification.ProductId {
		err = errors.New("商品id不一致")
		l.Errorf(err.Error()+" 数据库商品id: %s, 回传商品id: %s", hworder.ProductId, req.OrderNotification.ProductId)
		return err
	}

	if hworder.AppId != req.ApplicationId {
		err = errors.New("应用id不一致")
		l.Errorf(err.Error()+" 数据库app id: %s, 回传app id: %s", hworder.AppId, req.ApplicationId)
		return err
	}

	// 更新数据
	var purchaseTime string
	if req.NotifyTime > 1000 {
		purchaseTime = time.Unix(int64(req.NotifyTime/1000), 0).Format("2006-01-02 15:04:05")
	}
	updateData := map[string]interface{}{
		"log_id":            logId,
		"version":           req.Version,
		"event_type":        req.EventType,
		"notify_time":       int(req.NotifyTime / 1000), // 毫秒转成秒级时间戳
		"notification_type": req.OrderNotification.NotificationType,
		"environment":       "",
		"status":            1,
		"pay_time":          purchaseTime,
	}
	err = l.huaweiOrderModel.UpdateData(hworder.Id, updateData)
	if err != nil {
		return err
	}

	// 回调业务方接口 使用回调来 提供服务 回调才是处理实际逻辑的地方
	// 成功操作时 异步回调 把订单号、通知类型、用户id、包名返回
	if hworder.AppNotifyUrl != "" {
		go util.SafeRun(func() {
			// 回调数据
			callbackData := map[string]interface{}{
				"notify_type":  code.APP_NOTIFY_HUAWEI_PRODUCT_BUY,
				"user_id":      hworder.UserId,
				"out_trade_no": hworder.OutTradeNo,
				"pkg":          hworder.AppPkg,
			}

			headerMap := map[string]string{
				"App-Origin": hworder.AppPkg,
			}
			l.Sloww("华为回调app数据", logx.Field("call back data", callbackData))
			tmpErr := utils.CallbackWithRetry(hworder.AppNotifyUrl, headerMap, callbackData, 5*time.Second)
			if tmpErr != nil {
				l.Errorf("callback error: %v, call back data: %v", tmpErr, callbackData)
			}
		})
	} else {
		logx.Errorf("order id:%d, out_trade_no:%s, app notify url is empty", hworder.Id, hworder.OutTradeNo)
	}

	return nil
}
