package code

// 公共用到的常量
const (
	CODE_OK    = 2000 //成功
	CODE_ERROR = 1005 //操作失败(用户toast)    无上报

)

const (
	PRODUCT_TYPE_COMMON        = 0 // 普通商品
	PRODUCT_TYPE_SUBSCRIBE     = 1 // 订阅商品
	PRODUCT_TYPE_VIP           = 2 // 会员商品
	PRODUCT_TYPE_SUBSCRIBE_FEE = 3 // 订阅商品续费
)

const (
	APP_NOTIFY_TYPE_PAY             = "pay"
	APP_NOTIFY_TYPE_REFUND          = "refund"
	APP_NOTIFY_TYPE_SIGN            = "sign"
	APP_NOTIFY_TYPE_UNSIGN          = "unsign"
	APP_NOTIFY_TYPE_SIGN_FEE_FAILED = "sign_fee_failed"
)

// 订单状态  1:关闭，0:未支付，1:已支付，2:支付失败，3:已退款 4：退款中
const (
	ORDER_CLOSE     = -1
	ORDER_NO_PAY    = 0
	ORDER_SUCCESS   = 1
	ORDER_FAIL      = 2
	ORDER_REFUNDED  = 3
	ORDER_REFUNDING = 4
)

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
	NOTIFICATION_TYPE_INITIAL_BUY            = 0  // 订阅的第一次购买行为
	NOTIFICATION_TYPE_CANCEL                 = 1  // 客服或者App撤销了一个订阅，通过cancellationDate可以获得撤销订阅时间或退款时间
	NOTIFICATION_TYPE_RENEWAL                = 2  // 一个已经过期的订阅自动续期成功，可以通过收据中的“过期时间”获得下次续期时间
	NOTIFICATION_TYPE_INTERACTIVE_RENEWAL    = 3  // 用户主动恢复一个已经过期的订阅，或者用户在一个已经过期的商品订阅上切换到其他选项，成功后服务马上生效
	NOTIFICATION_TYPE_NEW_RENEWAL_PREF       = 4  // 顾客选择组内其他选项并且在当前订阅到期后生效，当前周期不受影响。也就是降级、跨级在下个周期生效的场景。通知会携带上次有效收据，和新的订阅信息，包括商品、订阅ID。
	NOTIFICATION_TYPE_RENEWAL_STOPPED        = 5  // 订阅服务续期被用户、您或者华为停止，已经收费的服务仍然有效。通知内容中包含最近收据、商品、应用、订阅Id和订阅Token等信息。
	NOTIFICATION_TYPE_RENEWAL_RESTORED       = 6  // 用户主动恢复了一个订阅型商品，续期状态恢复正常。通知内容中包含最近收据、商品、应用、订阅Id和订阅Token等信息
	NOTIFICATION_TYPE_RENEWAL_RECURRING      = 7  // 表示一次续期收费成功，包括优惠、免费试用和沙箱。通知内容中包含最近收据、商品、应用、订阅Id和订阅Token等信息
	NOTIFICATION_TYPE_ON_HOLD                = 9  // 表示一个已经到期的订阅进入帐号保留期
	NOTIFICATION_TYPE_PAUSED                 = 10 // 顾客设置暂停续期计划后，到期后订阅进入Paused状态
	NOTIFICATION_TYPE_PAUSE_PLAN_CHANGED     = 11 // 顾客设置了暂停续期计划(包括暂停计划的创建、修改以及在暂停计划生效前的计划终止)
	NOTIFICATION_TYPE_PRICE_CHANGE_CONFIRMED = 12 // 顾客同意了涨价
	NOTIFICATION_TYPE_DEFERRED               = 13 // 订阅的续期时间已经延期
)
