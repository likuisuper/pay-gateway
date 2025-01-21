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
	APP_NOTIFY_TYPE_PAY                 = "pay"
	APP_NOTIFY_TYPE_REFUND              = "refund"
	APP_NOTIFY_TYPE_SIGN                = "sign"
	APP_NOTIFY_TYPE_UNSIGN              = "unsign"
	APP_NOTIFY_TYPE_SIGN_FEE_FAILED     = "sign_fee_failed"
	APP_NOTIFY_HUAWEI_PRODUCT_SUBSCIRBE = "huawei_product_subscirbe" // 华为商品订阅
	APP_NOTIFY_HUAWEI_PRODUCT_BUY       = "huawei_product_buy"       // 华为商品购买
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
