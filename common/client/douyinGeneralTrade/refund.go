package douyin

import (
	"time"

	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/bytedance/sonic"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateRefundOrderReq struct {
	OrderId           string             `json:"order_id"`            // 必填 	交易系统订单号
	OutRefundNo       string             `json:"out_refund_no"`       // 必填		开发者侧退款单号
	CpExtra           string             `json:"cp_extra"`            // 非必填 	开发者自定义字段
	OrderEntrySchema  Schema             `json:"order_entry_schema"`  // 必填 	退款单的跳转的 schema
	NotifyUrl         string             `json:"notify_url"`          // 必填 	退款结果通知地址，必须是 HTTPS 类型， 长度 <= 512 byte
	RefundReason      []*RefundReason    `json:"refund_reason"`       // 必填		退款原因，可填多个，不超过10个
	RefundTotalAmount int64              `json:"refund_total_amount"` // 必填		退款总金额 单位分
	ItemOrderDetail   []*ItemOrderDetail `json:"item_order_detail"`   // 非必填 	需要发起退款的商品单信息，数组长度<100，refund_all=false时必填
	RefundAll         bool               `json:"refund_all"`          // 非必填	是否整单退款 当订单未发生任何退款时，可设置refund_all=true，refund_total_amount=订单实付金额，发起整单退款。refund_all=true时不能设置item_order_detail
	Currency          string             `json:"currency"`            // 非必填 	支付币种 钻石订单发起退款必须指定currency=DIAMOND
}

type RefundReason struct {
	Code int64  `json:"code,omitempty"` // 退款原因 必须从以下code中选择:[{"code":101,"text":"不想要了"},{"code":102,"text":"商家服务原因"},{"code":103,"text":"商品质量问题"},{"code":999,"text":"其他"}] 必填
	Text string `json:"text,omitempty"` // 退款原因描述，开发者可自定义，长度<50 必填
}

type ItemOrderDetail struct {
	ItemOrderId  string `json:"item_order_id,omitempty"` // 商品单号 必填
	RefundAmount int64  `json:"refund_amount,omitempty"` // 该item_order 需要退款金额 必填
}

type CreateRefundResp struct {
	ApiCommonResp
	Data CreateRefundRespData `json:"data,omitempty"` // 非必填
}

type CreateRefundRespData struct {
	RefundId            string `json:"refund_id,omitempty"`             // 必填 抖音开放平台交易系统侧退款单号
	RefundAuditDeadline int64  `json:"refund_audit_deadline,omitempty"` // 必填 退款审核的最后期限，13 位 unix 时间戳，精度：毫秒 通常是3天(从退款发起时间开始算)
}

// CreateRefundOrder 创建退款订单 https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/trade-system/general/refund/create_refund
func (c *PayClient) CreateRefundOrder(req *CreateRefundOrderReq, clientToken string) (*CreateRefundResp, error) {
	header := map[string]string{
		"access-token": clientToken,
	}

	result, err := util.HttpPostWithHeader("https://open.douyin.com/api/trade_basic/v1/developer/refund_create/", req, header, time.Second*3)
	logx.Sloww("CreateRefundOrder", logx.Field("err", err), logx.Field("result", result), logx.Field("req", req))

	if err != nil {
		return nil, err
	}

	resp := new(CreateRefundResp)
	err = sonic.UnmarshalString(result, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// RefundMsg 退款回调消息 https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/trade-system/general/refund/refund_notify
type RefundMsg struct {
	AppId             string `json:"app_id"`
	Status            string `json:"status"`              // 必填	退款状态枚举值：SUCCESS：退款成功FAIL：退款失败
	OrderId           string `json:"order_id"`            // 必填	抖音开平侧订单号
	CpExtra           string `json:"cp_extra"`            // 必填	开发者传入字段
	Message           string `json:"message"`             // 非必填 结果描述信息，如失败原因
	EventTime         int64  `json:"event_time"`          // 必填	退款时间戳，单位为毫秒
	RefundId          string `json:"refund_id"`           // 必填	抖音开平侧退款单号
	OutRefundNo       string `json:"out_refund_no"`       // 必填 开发者自定义的退款单号（可能为空)
	RefundTotalAmount int    `json:"refund_total_amount"` // 必填	退款总金额 单位分 钻石订单退款会填充退钻石数量
	IsAllSettled      bool   `json:"is_all_settled"`      // 必填	是否为分账后退款
	RefundItemDetail  struct {
		ItemOrderQuantity int `json:"item_order_quantity"` // 必填	用户退款商品单数量
		ItemOrderDetail   []struct {
			RefundAmount int    `json:"refund_amount"` // 必填	退款金额 单位分 钻石订单退款会填充退钻石数量
			ItemOrderId  string `json:"item_order_id"` // 必填	抖音开平侧商品单id
		} `json:"item_order_detail"`
	} `json:"refund_item_detail"`
	Currency   string `json:"currency,omitempty"`    // 非必填 支付币种 钻石支付为DIAMOND
	RefundType int64  `json:"refund_type,omitempty"` // 非必填 来源类型，枚举值： 1: 用户发起 2：开发者发起 4：抖音客服退款
}

// PreCreateRefundMsg 退款申请回调消息 https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/trade-system/general/refund/refund_callback
type PreCreateRefundMsg struct {
	AppId               string   `json:"app_id"`
	OpenId              string   `json:"open_id"`
	RefundId            string   `json:"refund_id"`
	OrderId             string   `json:"order_id"`
	OutOrderNo          string   `json:"out_order_no"`
	RefundTotalAmount   int64    `json:"refund_total_amount"` // 必填	退款总金额 单位分 钻石订单退款会填充退钻石数量
	NeedRefundAudit     int8     `json:"need_refund_audit"`
	RefundAuditDeadline int64    `json:"refund_audit_deadline"`
	CreateRefundTime    int64    `json:"create_refund_time"`
	RefundSource        int64    `json:"refund_source"`
	RefundReason        []string `json:"refund_reason"`
	RefundDescription   string   `json:"refund_description"`
	CpExtra             string   `json:"cp_extra"`
	RefundItemDetail    struct {
		ItemOrderQuantity int64 `json:"item_order_quantity"`
		ItemOrderDetail   []struct {
			ItemOrderId  string `json:"item_order_id"`
			RefundAmount int    `json:"refund_amount"` // 必填	退款金额 单位分 钻石订单退款会填充退钻石数量
		} `json:"item_order_detail"`
	} `json:"refund_item_detail"`
	Currency string `json:"currency,omitempty"` // 非必填 支付币种 钻石支付为DIAMOND
}

type AuditRefundReq struct {
	RefundId          string `json:"refund_id"`
	RefundAuditStatus int8   `json:"refund_audit_status"`
	DenyMessage       string `json:"deny_message"`
}

// AuditRefund 审核退款订单 https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/trade-system/general/refund/refund_audit
func (c *PayClient) AuditRefund(req *AuditRefundReq, clientToken string) (*ApiCommonResp, error) {
	header := map[string]string{
		"access-token": clientToken,
	}

	result, err := util.HttpPostWithHeader("https://open.douyin.com/api/trade_basic/v1/developer/refund_audit_callback/", req, header, time.Second*3)
	if err != nil {
		return nil, err
	}

	resp := new(ApiCommonResp)
	err = sonic.UnmarshalString(result, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type QueryRefundReq struct {
	RefundId    string `json:"refund_id"` // refund_id , out_refund_no , order_id 三选一，不能都不填。
	OutRefundNo string `json:"out_refund_no"`
	OrderId     string `json:"order_id"`
}

type QueryRefundResp struct {
	ApiCommonResp
	Data interface{} `json:"data"`
}

type QueryRefundRespData struct {
	RefundList []struct {
		MerchantAuditDetail struct {
			AuditStatus         string `json:"audit_status"`
			NeedRefundAudit     int64  `json:"need_refund_audit"`
			RefundAuditDeadline int64  `json:"refund_audit_deadline"`
		} `json:"merchant_audit_detail"`
		CreateAt          int64  `json:"create_at"`
		RefundAt          int64  `json:"refund_at"`
		RefundStatus      string `json:"refund_status"`
		RefundTotalAmount int64  `json:"refund_total_amount"` // 必填	退款总金额 单位分 钻石订单退款会填充退钻石数量
		ItemOrderDetail   []struct {
			ItemOrderId  string `json:"item_order_id"`
			RefundAmount int64  `json:"refund_amount"` // 必填	退款金额 单位分 钻石订单退款会填充退钻石数量
		} `json:"item_order_detail"`
		Message     string `json:"message"`
		OrderId     string `json:"order_id"`
		OutRefundNo string `json:"out_refund_no"`
		RefundId    string `json:"refund_id"`
		Currency    string `json:"currency,omitempty"` // 非必填 支付币种 钻石支付为DIAMOND
	} `json:"refund_list"`
}

// QueryRefund 查询退款订单 https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/trade-system/general/refund/query_refund
func (c *PayClient) QueryRefund(req *QueryRefundReq, clientToken string) (*QueryRefundResp, error) {
	header := map[string]string{
		"access-token": clientToken,
	}

	result, err := util.HttpPostWithHeader("https://open.douyin.com/api/trade_basic/v1/developer/refund_query/", req, header, time.Second*3)
	if err != nil {
		return nil, err
	}

	resp := new(QueryRefundResp)
	err = sonic.UnmarshalString(result, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
