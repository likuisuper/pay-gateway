package code

// 错误码
const (
	OK  = 2000 //成功
	Err = 1005 //操作失败(用户toast)    无上报

)

// 支付宝
const (
	// 成功
	ALI_PAY_SUCCESS = 2000
	// 失败
	ALI_PAY_FAIL = 1005
	// 达到收款方笔数超限
	ALI_PAY_UP_LIMIT = 1006
)
