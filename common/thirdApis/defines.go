package thirdApis

const (
	//WechatXPayHost 微信虚拟支付host
	WechatXPayHost = "https://api.weixin.qq.com"
)

type XPayRefundOrderParam struct {
	OpenId        string `json:"openid"`          // openid	string	下单时的用户openid
	OrderId       string `json:"order_id"`        // order_id	string	下单时的单号，即jsapi接口传入的OutTradeNo，与wx_order_id字段二选一
	wxOrderId     string `json:"wx_order_id"`     // wx_order_id	string	支付单对应的微信侧单号，与order_id字段二选一
	RefundOrderId string `json:"refund_order_id"` // refund_order_id	string	本次退款时需要传的单号，长度为[8,32]，字符只允许使用字母、数字、'_'、'-'
	LeftFee       int    `json:"left_fee"`        // left_fee	int	当前单剩余可退金额，单位分，可以通过调用query_order接口查到
	RefundFee     int    `json:"refund_fee"`      // refund_fee	int	本次退款金额，单位分，需要(0,left_fee]
	BizMeta       string `json:"biz_meta"`        // biz_meta	string	商家自定义数据，传入后可在query_order接口查询时原样返回，长度需要[0,1024]
	RefundReason  string `json:"refund_reason"`   // refund_reason	string	退款原因，当前仅支持以下值 0-暂无描述 1-产品问题，影响使用或效果不佳 2-售后问题，无法满足需求 3-意愿问题，用户主动退款 4-价格问题 5:其他原因
	ReqFrom       string `json:"req_from"`        // req_from	string	退款来源，当前仅支持以下值 1-人工客服退款，即用户电话给客服，由客服发起退款流程 2-用户自己发起退款流程 3-其它
	Env           int    `json:"env"`             //env	int	0-正式环境 1-沙箱环境
}

type XPayRefundOrderDTO struct {
	ErrCode         int    `json:"errcode"`            //errcode	int	错误码
	ErrMsg          string `json:"errmsg"`             //errmsg	string	错误信息
	RefundOrderId   string `json:"refund_order_id"`    //refund_order_id	string	退款单号
	RefundWxOrderId string `json:"refund_wx_order_id"` //refund_wx_order_id	string	退款单的微信侧单号
	PayOrderId      string `json:"pay_order_id"`       //pay_order_id	string	该退款单对应的支付单单号
	PayWxOrderId    string `json:"pay_wx_order_id"`    //pay_wx_order_id	string	该退款单对应的支付单微信侧单号
}

type XPayQueryOrderParam struct {
	OpenId    string `json:"openid"`      //openid	string	用户的openid
	OrderId   string `json:"order_id"`    //order_id	string	创建的订单号
	WxOrderId string `json:"wx_order_id"` //wx_order_id	string	微信内部单号(与order_id二选一)
	Env       int    `json:"env"`         //env	int	0-正式环境 1-沙箱环境
}

type XPayQueryOrderDTO struct {
	ErrCode int           `json:"errcode"` //errcode	int	错误码
	ErrMsg  string        `json:"errmsg"`  //errmsg	string	错误信息
	Order   XPayOrderItem `json:"order"`   //	order	object	订单信息
}

type XPayOrderItem struct {
	OrderId        string `json:"order_id"`         //order_id	string	订单号
	CreateTime     int64  `json:"create_time"`      //create_time	int	创建时间
	UpdateTime     int64  `json:"update_time"`      //update_time	int	更新时间
	Status         int    `json:"status"`           //status	int	当前状态 0-订单初始化（未创建成功，不可用于支付）1-订单创建成功 2-订单已经支付，待发货 3-订单发货中 4-订单已发货 5-订单已经退款 6-订单已经关闭（不可再使用） 7-订单退款失败 8-用户退款完成 9-回收广告金完成 10-分账回退完成
	BizType        int    `json:"biz_type"`         //biz_type	int	业务类型0-短剧
	OrderFee       int    `json:"order_fee"`        //order_fee	int	订单金额，单位分
	CouponFee      int    `json:"coupon_fee"`       //coupon_fee	int	订单优惠金额，单位分(暂无此字段)
	PaidFee        int    `json:"paid_fee"`         //paid_fee	int	用户支付金额
	OrderType      int    `json:"order_type"`       //order_type	int	订单类型0-支付单 1-退款单
	RefundFee      int    `json:"refund_fee"`       //refund_fee	int	当类型为退款单时表示退款金额，单位分
	PaidTime       int    `json:"paid_time"`        //paid_time	int	支付/退款时间，unix秒级时间戳
	ProvideTime    int    `json:"provide_time"`     //provide_time	int	发货时间
	BizMeta        string `json:"biz_meta"`         //biz_meta	string	订单创建时传的信息
	EnvType        int    `json:"env_type"`         //env_type	int	环境类型1-现网 2-沙箱
	Token          string `json:"token"`            //token	string	下单时米大师返回的token
	LeftFee        int    `json:"left_fee"`         //left_fee	int	支付单类型时表示此单经过退款还剩余的金额，单位分
	WxOrderId      string `json:"wx_order_id"`      //wx_order_id	string	微信内部单号
	ChannelOrderId string `json:"channel_order_id"` //channel_order_id	string	渠道单号，为用户微信支付详情页面上的商户单号
	WxPayOrderId   string `json:"wxpay_order_id"`   //wxpay_order_id	string	微信支付交易单号，为用户微信支付详情页面上的交易单号
	SettTime       int    `json:"sett_time"`        //sett_time	int	结算时间的秒级时间戳，大于0表示结算成功
	SettState      int    `json:"sett_state"`       //sett_state	int	结算状态0-未开始结算 1-结算中 2-结算成功 3-待结算（与0相同）
}
