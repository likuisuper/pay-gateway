package client

//支付订单信息
type PayOrder struct {
	OrderSn  string
	Amount   int
	Subject  string
	KsTypeId int
	IP       string
}

//退款订单信息
type RefundOrder struct {
	OutTradeNo    string `json:"out_trade_no"`  //内部订单号
	OutRefundNo   string `json:"out_refund_no"` //商户系统内部的退款单号
	TotalFee      int64  `json:"total_fee"`     //订单总金额，单位为分
	RefundFee     int64  `json:"refund_fee"`    //退款总金额，订单总金额，单位为分
	TransactionId string `json:"transaction_id"`
}
