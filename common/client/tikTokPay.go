package client

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/zeromicro/go-zero/core/logx"
	kv_m "gitlab.muchcloud.com/consumer-project/zhuyun-core/kv_monitor"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/util"
)

var (
	tikTokHttpRequestErr    = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "tikTokHttpRequestErr", nil, "tikTok请求错误", nil})}
	tikTokNotifyErr         = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "tikTokNotifyErr", nil, "tikTok回调错误", nil})}
	tikTokGetOrderStatusErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "tikTokGetOrderStatusErr", nil, "tikTok回调错误", nil})}
)

const (
	OtherSettleParams = "other_settle_params" // 其他分账方参数 (Other settle params)
	AppId             = "app_id"              // 小程序appID (Applets appID)
	ThirdpartyId      = "thirdparty_id"       // 代小程序进行该笔交易调用的第三方平台服务商 id (The id of the third-party platform service provider that calls the transaction on behalf of the Applets)
	Sign              = "sign"                // 签名 (sign)
)

// 请求地址
const (
	tikTokCreateUri       = "https://developer.toutiao.com/api/apps/ecpay/v1/create_order"
	tikTokQueryUri        = "https://developer.toutiao.com/api/apps/ecpay/v1/query_order"
	tikTokCreateRefundUri = "https://developer.toutiao.com/api/apps/ecpay/v1/create_refund" //发起退款
	tikTokBody            = "充值VIP"
	tikTokValidTime       = 300
)

// 字节支付配置
type TikTokPayConfig struct {
	AppId     string //应用ID
	SALT      string //加密参数
	NotifyUrl string //通知地址
	Token     string //token
}

type TikTokPay struct {
	Config TikTokPayConfig
}

var tikTokPay *TikTokPay

func NewTikTokPay(config TikTokPayConfig) *TikTokPay {
	tikTokPay = &TikTokPay{
		Config: config,
	}
	return tikTokPay
}

// 请求响应
type TikTokReply struct {
	TikTokNotifyResp
	Data TikTokReplyData `json:"data"`
}

type TikTokReplyData struct {
	OrderId    string `json:"order_id"`
	OrderToken string `json:"order_token"`
	OrderCode  string `json:"order_code"`
}

// 回调回复
type TikTokNotifyResp struct {
	ErrNO   int    `json:"err_no"`
	ErrTips string `json:"err_tips"`
}

// 创建支付订单
func (t *TikTokPay) CreateEcPayOrder(info *PayOrder) (result TikTokReply, err error) {

	cpExtra := fmt.Sprintf(`{"amount":%d,"order_code":"%s"`, info.Amount, info.OrderSn)
	body := info.Subject
	//构建请求参数
	data := map[string]interface{}{
		"app_id":       t.Config.AppId,
		"out_order_no": info.OrderSn,
		"total_amount": info.Amount,
		"subject":      info.Subject,
		"body":         body,
		"valid_time":   tikTokValidTime,
		"cp_extra":     cpExtra,
		"notify_url":   t.Config.NotifyUrl,
	}
	data["sign"] = t.getSign(data)
	res, err := util.HttpPost(tikTokCreateUri, data, 5*time.Second)
	dataStr, _ := jsoniter.MarshalToString(data)
	logx.Slow("tikTok请求创建订单 ", dataStr)
	dataStr, _ = jsoniter.MarshalToString(t.Config)
	logx.Slow("tiktok配置 ", dataStr)
	logx.Slow("tikTok请求创建订单返回 ", res)
	if err != nil {
		tikTokHttpRequestErr.CounterInc()
		return
	}
	err = json.Unmarshal([]byte(res), &result)
	if err != nil {
		return
	}
	if result.ErrNO != 0 {
		return result, errors.New(result.ErrTips)
	}
	result.Data.OrderCode = info.OrderSn
	return result, nil
}

// 获取签名
func (t *TikTokPay) getSign(paramsMap map[string]interface{}) string {
	var paramsArr []string
	for k, v := range paramsMap {
		if k == OtherSettleParams || k == AppId || k == ThirdpartyId || k == Sign {
			continue
		}
		value := strings.TrimSpace(fmt.Sprintf("%v", v))
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") && len(value) > 1 {
			value = value[1 : len(value)-1]
		}
		value = strings.TrimSpace(value)
		if value == "" || value == "null" {
			continue
		}
		paramsArr = append(paramsArr, value)
	}

	paramsArr = append(paramsArr, t.Config.SALT)
	sort.Strings(paramsArr)
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(paramsArr, "&"))))
}

//以下回调相关

// 支付回调msg解析结构体
type TikTokNotifyMsgData struct {
	Appid          string `json:"appid"`            //当前交易发起的小程序id
	CpOrderno      string `json:"cp_orderno"`       //开发者侧的订单号
	CpExtra        string `json:"cp_extra"`         //预下单时开发者传入字段
	Way            string `json:"way"`              // 1-微信支付，2-支付宝支付，10-抖音支付
	ChannelNo      string `json:"channel_no"`       //支付渠道侧单号
	PaymentOrderNO string `json:"payment_order_no"` //支付渠道侧PC单号，支付页面可见
	TotalAmount    int    `json:"total_amount"`     //支付金额，单位为分
	Status         string `json:"status"`           //固定SUCCESS
	ItemId         string `json:"item_id"`          //订单来源视频对应视频 id
	SellerUid      string `json:"seller_uid"`       //该笔交易卖家商户号
	PaidAt         int64  `json:"paid_at"`          //支付时间，Unix 时间戳，10 位，整型数
	OrderId        string `json:"order_id"`         //抖音侧订单号
}

// 退款回调msg解析结构体
type TikTokNotifyMsgRefundData struct {
	Appid        string `json:"appid"`          //当前交易发起的小程序id
	CpRefundno   string `json:"cp_refundno"`    //开发者侧的退款订单号
	CpExtra      string `json:"cp_extra"`       //预下单时开发者传入字段
	Status       string `json:"status"`         //状态枚举值： SUCCESS：成功  FAIL：失败
	RefundAmount int    `json:"refund_amount"`  //退款金额，单位为分
	IsAllSettled bool   `json:"is_all_settled"` //是否为分账后退款
	RefundedAt   int64  `json:"refunded_at"`    //退款时间，Unix 时间戳，10 位，整型数，秒级
	Message      string `json:"message"`        //退款失败原因描述，详见发起退款错误码
	OrderId      string `json:"order_id"`       //抖音侧订单号
	RefundNo     string `json:"refund_no"`      //抖音侧退款单号
}

type ByteDanceReq struct {
	Timestamp    string `json:"timestamp,optional"`
	Nonce        string `json:"nonce,optional"`
	Msg          string `json:"msg,optional"`
	Type         string `json:"type,optional"`
	MsgSignature string `json:"msg_signature,optional"`
}

// 回调验证返回
func (t *TikTokPay) Notify(req *ByteDanceReq) (orderInfo *TikTokNotifyMsgData, err error) {
	//签名核对
	timestamp, _ := strconv.Atoi(req.Timestamp)
	notifySing := t.NotifySign(timestamp, req.Nonce, req.Msg)
	if notifySing != req.MsgSignature {
		tikTokNotifyErr.CounterInc()
		logx.Errorf("回调签名错误")
		return nil, errors.New("回调签名错误")
	}
	logx.Infof("字节订单回调信息：msg=%s", req.Msg)
	var orderData TikTokNotifyMsgData
	err = json.Unmarshal([]byte(req.Msg), &orderData)
	if err != nil {
		tikTokNotifyErr.CounterInc()
		logx.Errorf("订单消息解析失败:err=%v", err)
		return nil, errors.New("订单消息解析失败")
	}
	return &orderData, nil
}

// 获取验签
func (t *TikTokPay) NotifySign(timestamp int, nonce, msg string) string {

	sortedString := make([]string, 0)
	sortedString = append(sortedString, t.Config.Token)
	sortedString = append(sortedString, strconv.Itoa(timestamp))
	sortedString = append(sortedString, nonce)
	sortedString = append(sortedString, msg)
	sort.Strings(sortedString)
	h := sha1.New()
	h.Write([]byte(strings.Join(sortedString, "")))
	bs := h.Sum(nil)
	_signature := fmt.Sprintf("%x", bs)
	return _signature
}

//以下为支付结果查询

// 结果信息返回解析结构体
type TikTokOrderStatusReply struct {
	ErrNo       int               `json:"err_no"`       //返回码，详见错误码
	ErrTips     string            `json:"err_tips"`     //返回码描述，详见错误码描述
	OutOrderNo  string            `json:"out_order_no"` //开发者侧的订单号
	OrderId     string            `json:"order_id"`     //抖音侧的订单号
	PaymentInfo TikTokPaymentInfo `json:"payment_info"` //支付信息
	CpsInfo     string            `json:"cps_info"`     //若该订单为cps订单，该字段会返回该笔订单的达人分佣金额。
}

type TikTokPaymentInfo struct {
	TotalFee         int    `json:"total_fee"`          //支付金额，单位为分
	OrderStatus      string `json:"order_status"`       //支付状态枚举值： SUCCESS：成功 TIMEOUT：超时未支付 PROCESSING：处理中 FAIL：失败
	PayTime          string `json:"pay_time"`           //支付时间， 格式为"yyyy-MM-dd hh:mm:ss"
	Way              int    `json:"way"`                //支付渠道， 1-微信支付，2-支付宝支付，10-抖音支付
	ChannelNo        string `json:"channel_no"`         //支付渠道侧单号
	ChannelGatewayNo string `json:"channel_gateway_no"` //支付渠道侧的商家订单号
	SellerUid        string `json:"seller_uid"`         //该笔交易卖家商户号
	ItemId           string `json:"item_id"`            //订单来源视频对应视频 id
}

// 查询订单结果
func (t *TikTokPay) GetOrderStatus(orderCode string) (orderInfo *TikTokPaymentInfo, err error) {
	data := map[string]interface{}{
		"app_id":       t.Config.AppId,
		"out_order_no": orderCode,
	}
	data["sign"] = t.getSign(data)
	res, err := util.HttpPost(tikTokQueryUri, data, 5*time.Second)
	logx.Infof("tikTok请求订单订单返回 res:=%s", res)
	if err != nil {
		tikTokHttpRequestErr.CounterInc()
		return
	}
	var result TikTokOrderStatusReply
	err = json.Unmarshal([]byte(res), &result)
	if err != nil {
		return nil, err
	}
	if result.ErrNo == 0 {
		return &result.PaymentInfo, nil
	} else {
		tikTokGetOrderStatusErr.CounterInc()
		return nil, errors.New(result.ErrTips)
	}
	return nil, nil
}

// 创建退款订单
type TikTokCreateRefundOrderReq struct {
	AppId        string `json:"app_id"`               //小程序APPID
	OutOrderNo   string `json:"out_order_no"`         //商户分配支付单号，标识进行退款的订单
	OutRefundNo  string `json:"out_refund_no"`        //商户分配退款号，保证在商户中唯一
	Reason       string `json:"reason"`               //退款原因
	RefundAmount int    `json:"refund_amount"`        //退款金额，单位分
	Sign         string `json:"sign,omitempty"`       //签名，详见签名DEMO
	NotifyUrl    string `json:"notify_url,omitempty"` //商户自定义回调地址，必须以 https 开头，支持 443 端口
}

type TikTokCreateRefundOrderResp struct {
	ErrNo    int    `json:"err_no"`    //错误码
	ErrTips  string `json:"err_tips"`  //错误描述
	RefundNo string `json:"refund_no"` //担保交易服务端退款单号
}

// 创建退款订单
func (t *TikTokPay) CreateRefundOrder(refundReq TikTokCreateRefundOrderReq) (resp TikTokCreateRefundOrderResp, err error) {
	refundReq.NotifyUrl = t.Config.NotifyUrl

	paramsMap := make(map[string]interface{}, 0)
	jsonBytes, _ := json.Marshal(refundReq)
	_ = json.Unmarshal(jsonBytes, &paramsMap)

	refundReq.Sign = t.getSign(paramsMap)

	res, err := util.HttpPost(tikTokCreateRefundUri, refundReq, 5*time.Second)
	if err != nil {
		util.CheckError("CreateRefundOrder, config:%+v, refundReq:%+v, err:%v", t.Config, refundReq, err)
		tikTokHttpRequestErr.CounterInc()
		return
	}
	logx.Slowf("CreateRefundOrder, config:%+v, refundReq:%+v, res:%s", t.Config, refundReq, res)

	err = json.Unmarshal([]byte(res), &resp)
	if err != nil {
		util.CheckError("CreateRefundOrder-CreateRefundOrder, res:%s, err:%v", res, err)
		return
	}
	return
}
