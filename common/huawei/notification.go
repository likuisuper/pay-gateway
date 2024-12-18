package huawei

// https://developer.huawei.com/consumer/cn/doc/HMSCore-References/api-notifications-about-subscription-events-v2-0000001385268541
const (
	// 通知类型 ORDER：订单
	HUAWEI_EVENT_TYPE_ORDER = "ORDER"
	// 通知类型 SUBSCRIPTION：订阅
	HUAWEI_EVENT_TYPE_SUBSCRIPTION = "SUBSCRIPTION"
)

// 华为通知类型
//
// https://developer.huawei.com/consumer/cn/doc/HMSCore-References/api-notifications-about-subscription-events-v2-0000001385268541#ZH-CN_TOPIC_0000001050706084__table2818954
//
// 说明 INITIAL_BUY ，RENEWAL，INTERACTIVE_RENEWAL情况下，不会有RENEWAL_RECURRING，因为本身就表示一次成功的续期。
//
// 例如，用户购买一个商品，第一次购买发送INITIAL_BUY通知，下次续期以及以后的每次正常续期，都会发送RENEWAL_RECURRING通知。
const (
	// 订阅的第一次购买行为
	NOTIFICATION_TYPE_INITIAL_BUY = 0
	// 客服或者App撤销了一个订阅，通过cancellationDate可以获得撤销订阅时间或退款时间
	NOTIFICATION_TYPE_CANCEL = 1
	// 一个已经过期的订阅自动续期成功，可以通过收据中的“过期时间”获得下次续期时间
	NOTIFICATION_TYPE_RENEWAL = 2
	// 用户主动恢复一个已经过期的订阅，或者用户在一个已经过期的商品订阅上切换到其他选项，成功后服务马上生效
	NOTIFICATION_TYPE_INTERACTIVE_RENEWAL = 3
	// 顾客选择组内其他选项并且在当前订阅到期后生效，当前周期不受影响。也就是降级、跨级在下个周期生效的场景。通知会携带上次有效收据，和新的订阅信息，包括商品、订阅ID。
	NOTIFICATION_TYPE_NEW_RENEWAL_PREF = 4
	// 订阅服务续期被用户、您或者华为停止，已经收费的服务仍然有效。通知内容中包含最近收据、商品、应用、订阅Id和订阅Token等信息。
	NOTIFICATION_TYPE_RENEWAL_STOPPED = 5
	// 用户主动恢复了一个订阅型商品，续期状态恢复正常。通知内容中包含最近收据、商品、应用、订阅Id和订阅Token等信息
	NOTIFICATION_TYPE_RENEWAL_RESTORED = 6
	// 表示一次续期收费成功，包括优惠、免费试用和沙箱。通知内容中包含最近收据、商品、应用、订阅Id和订阅Token等信息
	NOTIFICATION_TYPE_RENEWAL_RECURRING = 7
	// 表示一个已经到期的订阅进入帐号保留期
	NOTIFICATION_TYPE_ON_HOLD = 9
	// 顾客设置暂停续期计划后，到期后订阅进入Paused状态
	NOTIFICATION_TYPE_PAUSED = 10
	// 顾客设置了暂停续期计划(包括暂停计划的创建、修改以及在暂停计划生效前的计划终止)
	NOTIFICATION_TYPE_PAUSE_PLAN_CHANGED = 11
	// 顾客同意了涨价
	NOTIFICATION_TYPE_PRICE_CHANGE_CONFIRMED = 12
	// 订阅的续期时间已经延期
	NOTIFICATION_TYPE_DEFERRED = 13
)

type NotificationRequest struct {
	StatusUpdateNotification string `json:"statusUpdateNotification"`
	NotificationSignature    string `json:"notifycationSignature"`
}

type NotificationResponse struct {
	ErrorCode string `json:"errorCode"`
	ErrorMsg  string `json:"errorMsg"`
}

// 支付后拿到的最新的purchaseToken，表示该商品和该用户的对应关系。
//
// latestReceipt表示当前成功收费收据的token；latestExpiredReceipt表示上个周期收据的token。
//
// 如果是续期订阅，purchaseToken与latestReceipt相同；如果是切换订阅，purchaseToken与latestReceipt不同。
//
// 当收到类型为NEW_RENEWAL_PREF(4)的通知时，purchaseToken与latestReceipt不同，subscriptionId为下周期订阅ID，此时这个订阅还未发生扣费，还没有生成扣费的收据，latestReceipt是切换前订阅对应的最后一笔收据的token。
type StatusUpdateNotification struct {
	Environment                       string `json:"environment"`                       // 发送通知的环境。PROD：正式环境 Sandbox：沙盒测试
	NotificationType                  int    `json:"notificationType"`                  // 通知事件的类型
	SubscriptionId                    string `json:"subscriptionId"`                    // 订阅ID
	PurchaseToken                     string `json:"purchaseToken"`                     // 订阅token，与上述订阅ID字段subscriptionId对应
	CancellationDate                  int64  `json:"cancellationDate"`                  // 撤销订阅时间或退款时间，UTC时间戳，以毫秒为单位，仅在notificationType取值为CANCEL的场景下会传入。
	OrderId                           string `json:"orderId"`                           // 订单ID，唯一标识一笔需要收费的收据，由华为应用内支付服务器在创建订单以及订阅型商品续费时生成。每一笔新的收据都会使用不同的orderId。通知类型为NEW_RENEWAL_PREF时不存在。
	LatestReceipt                     string `json:"latestReceipt"`                     // 最近的一笔收据的token，仅在notificationType取值为INITIAL_BUY 、RENEWAL或INTERACTIVE_RENEWAL并且续期成功情况下传入
	LatestReceiptInfo                 string `json:"latestReceiptInfo"`                 // 最近的一笔收据，JSON字符串格式，包含的参数请参见InappPurchaseDetails，在notificationType取值为CANCEL时无值
	LatestReceiptInfoSignature        string `json:"latestReceiptInfoSignature"`        // 对latestReceiptInfo的签名字符串，签名算法为statusUpdateNotification中的signatureAlgorithm。您的服务器在收到签名字符串后，需要参见对返回结果验签使用IAP公钥对latestReceiptInfo的JSON字符串进行验签。公钥获取请参见查询支付服务信息。
	LatestExpiredReceipt              string `json:"latestExpiredReceipt"`              // 最近的一笔过期收据的token
	LatestExpiredReceiptInfo          string `json:"latestExpiredReceiptInfo"`          // 最近的一笔过期收据，JSON字符串格式，在notificationType取值为RENEWAL或INTERACTIVE_RENEWAL时有值
	LatestExpiredReceiptInfoSignature string `json:"latestExpiredReceiptInfoSignature"` // 对latestExpiredReceiptInfo的签名字符串，签名算法为statusUpdateNotification中的signatureAlgorithm。您的服务器在收到签名字符串后，需要参见对返回结果验签使用IAP公钥对latestExpiredReceiptInfo的JSON字符串进行验签。
	SignatureAlgorithm                string `json:"signatureAlgorithm"`                // 签名算法
	AutoRenewStatus                   int    `json:"autoRenewStatus"`                   // 续期状态。取值说明：1：当前周期到期后正常续期 0：用户已终止续期
	RefundPayOrderId                  string `json:"refundPayOrderId"`                  // 退款交易号，在notificationType取值为CANCEL时有值
	ProductId                         string `json:"productId"`                         // 订阅型商品ID
	ApplicationId                     string `json:"applicationId"`                     // 应用ID
	ExpirationIntent                  int    `json:"expirationIntent"`                  // 超期原因，仅在notificationType为RENEWAL或INTERACTIVE_RENEWAL时并且续期失败情况下有值
}
