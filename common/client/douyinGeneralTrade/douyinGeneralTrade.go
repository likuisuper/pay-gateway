package douyin

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/bytedance/sonic"
	"github.com/zeromicro/go-zero/core/logx"
)

// 订单查询url
const trade_order_query_url = "https://open.douyin.com/api/trade_basic/v1/developer/order_query/"

// 解约周期代扣
const trade_terminate_sign_url = "https://open.douyin.com/api/trade_auth/v1/developer/terminate_sign/"

// 查询抖音周期代扣签约单的状态
const query_sign_order_url = "https://open.douyin.com/api/trade_auth/v1/developer/query_sign_order/"

type PayConfig struct {
	AppId             string
	PrivateKey        string // 应用私钥
	KeyVersion        string
	NotifyUrl         string
	PlatformPublicKey string // 平台公钥
	CustomerImId      string
	MerchantUid       string // 支付使用的商户号，为空抖音侧会使用默认值
}

type PayClient struct {
	config *PayConfig
}

// 抖音普通商品请求体
// RequestOrderData 请求体 https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/api/industry/general_trade/create_order/requestOrder#33efe69e
type RequestOrderData struct {
	SkuList          []*Sku  `json:"skuList,omitempty"`          // 下单商品信息 必填 支持一个
	OutOrderNo       string  `json:"outOrderNo,omitempty"`       // 外部订单号 必填
	TotalAmount      int32   `json:"totalAmount,omitempty"`      // 订单总金额 默认分 必填
	PayExpireSeconds int32   `json:"payExpireSeconds,omitempty"` // 支付超时时间，单位秒，例如 300 表示 300 秒后过期；不传或传 0 会使用默认值 300，不能超过48小时。非必填
	PayNotifyUrl     string  `json:"payNotifyUrl,omitempty"`     // 支付结果通知地址，必须是 HTTPS 类型，传入后该笔订单将通知到此地址。 非必填
	MerchantUid      string  `json:"merchantUid,omitempty"`      // 开发者自定义收款商户号 非必填
	OrderEntrySchema *Schema `json:"orderEntrySchema,omitempty"` // 订单详情页 必填
	LimitPayWayList  []int32 `json:"limitPayWayList,omitempty"`  // 屏蔽的支付方式，当开发者没有进件某个支付渠道，可在下单时屏蔽对应的支付方式。如：[1, 2]表示屏蔽微信和支付宝 枚举说明： 1-微信 2-支付宝 非必填
	PayScene         string  `json:"payScene,omitempty"`         // 指定支付场景 ios 传IM 安卓不传
	Currency         string  `json:"currency,omitempty"`         // 指定支付币种 ios 钻石支付传DIAMOND 安卓不传
}

// 抖音签约周期代扣商品请求体
// https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/api/industry/credit-products/createSignOrder
type RequestPeriodOrderData struct {
	OutAuthOrderNo     string           `json:"outAuthOrderNo"`               // 开发者侧签约单号，长度<=64byte
	ServiceId          string           `json:"serviceId"`                    // 签约模板ID
	OpenId             string           `json:"openId"`                       // 用户 openId
	ExpireSeconds      int64            `json:"expireSeconds"`                // 签约或签约支付超时时间，单位[秒]，不传默认5分钟，最少30秒，不能超过48小时。建议开发者不要将超时时间设置太短
	NotifyUrl          string           `json:"notifyUrl"`                    // 签约结果回调地址，https开头，长度<=512byte
	FirstDeductionDate *string          `json:"firstDeductionDate,omitempty"` // 首次扣款日期,格式YYYY-MM-DD,纯签约场景需要传入,用于c端展示
	OnBehalfUid        string           `json:"onBehalfUid"`                  // 代签约用户uid，该uid必须由ASCII字母、数字、下划线组成，长度<=64个字符，通常该字段应填入开发者自己系统的uid
	AuthPayOrder       *AuthPayOrderObj `json:"authPayOrder,omitempty"`       // 扣款信息，如果传入该字段则会走签约支付流程，否则走纯签约流程
}

type AuthPayOrderObj struct {
	OutPayOrderNo string `json:"outPayOrderNo"`           // 开发者侧代扣单的单号，长度<=64byte
	MerchantUid   string `json:"merchantUid"`             // 开发者自定义收款商户号，限定在在小程序绑定的商户号内
	InitialAmount *int64 `json:"initialAmount,omitempty"` // 首期代扣金额，单位[分]，不传则使用模板上的扣款金额，签约模板支持前N（N<=6）期优惠，该字段优先级高于模板的配置的第一期优惠价格，举例：如果当前参数传入扣款金额为10元，而实际模板中配置的第一期优惠价格为20元，那么第一期的实际扣款金额是10元
	NotifyUrl     string `json:"notifyUrl"`               // 支付结果回调地址，https开头，长度<=512byte
}

type Sku struct {
	SkuId       string        `json:"skuId,omitempty"`       // 外部商品id 必填
	Price       int32         `json:"price,omitempty"`       // 价格 单位：分 必填
	Quantity    int32         `json:"quantity,omitempty"`    // 购买数量 0 < quantity <= 100 必填
	Title       string        `json:"title,omitempty"`       // 商品标题，长度 <= 256字节 必填
	ImageList   []string      `json:"imageList,omitempty"`   // 商品图片链接，长度 <= 512 字节 注意：目前只支持传入一项 必填
	Type        SkuType       `json:"type,omitempty"`        // 商品类型 必填
	TagGroupId  SkuTagGroupId `json:"tagGroupId,omitempty"`  // 交易规则标签组 必填 https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/trade-system/general/tag/tag_group_query#2b56d127
	EntrySchema *Schema       `json:"entrySchema,omitempty"` // 商品详情页 非必填
	SkuAttr     string        `json:"skuAttr,omitempty"`     // 商品信息：需要将不同商品类型定义的具体结构，转换成json string​ ​号卡类商品必填，即当前商品类型 type in [101、102、103、104、105、106、107]的商品必填，内部结构请详见下文 ”skuAttr“ 小节说明。​ https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/trade-system/general/order/request-order-data-sign
}

type SkuType int32

const (
	SkuContentRecharge SkuType = 401
)

type SkuTagGroupId string

const (
	SKuTagGroupIdContentRecharge SkuTagGroupId = "tag_group_7272625659888041996"
)

const (
	PaySceneIM      = "IM"      // 支付场景值-im
	CurrencyDiamond = "DIAMOND" // 支付币种-钻石
)

type Schema struct {
	Path   string `json:"path,omitempty"`   // 小程序xxx详情页跳转路径，没有前导的“/”，路径后不可携带query参数，路径中不可携带『？: & *』等特殊字符，路径只可以是『英文字符、数字、_、/ 』等组成，长度<=512byte
	Params string `json:"params,omitempty"` // xx情页路径参数，自定义的json结构，内部为k-v结构，序列化成字符串存入该字段，平台不限制，但是写入的内容需要能够保证生成访问xx详情的schema能正确跳转到小程序内部的xx详情页，长度须<=512byte，params内key不可重复。
}

// 抖音签约授权回调结构体
type DySignCallbackNotify struct {
	AppId          string `json:"app_id"`            // 小程序 app_id
	Status         string `json:"status"`            // 签约结果状态，目前有四种状态： "SUCCESS" （用户签约成功 ） •"TIME_OUT" （用户未签约，订单超时关单） •"CANCEL" (解约成功)	•"DONE" （服务完成，已到期）
	AuthOrderId    string `json:"auth_order_id"`     // 平台侧签约单的单号，长度<=64byte
	OutAuthOrderNo string `json:"out_auth_order_no"` // 开发者侧签约单的单号，长度<=64byte
	EventTime      int64  `json:"event_time"`        // 用户签约成功/签约取消/解约成功的时间戳，单位为毫秒
}

// 签约回调用户签约状态
const (
	Dy_Sign_Status_SUCCESS  = "SUCCESS"  // 用户签约成功
	Dy_Sign_Status_TIME_OUT = "TIME_OUT" // 用户未签约，订单超时关单
	Dy_Sign_Status_CANCEL   = "CANCEL"   // 解约成功
	Dy_Sign_Status_DONE     = "DONE"     // 服务完成，服务完成，签约已到期
)

// 签约支付状态
const (
	Dy_Sign_Pay_Status_SUCCESS  = "SUCCESS"  // 成功
	Dy_Sign_Pay_Status_TIME_OUT = "TIME_OUT" // 超时未支付 ｜超时未扣款成功
	Dy_Sign_Pay_Status_FAIL     = "FAIL"     // （扣款失败，原因基本都是用户无支付方式（解绑了付款卡）或用户的扣款卡余额不足，建议失败后不要立即重试，隔日再进行重试，若一个月内连续多次扣款均不成功，考虑和用户进行解约）
)

// 签约订单查询返回的用户签约状态
// TOBESERVED: 待服务 SERVING：服务中 CANCEL: 已解约 TIMEOUT: 用户未签约 DONE: 服务完成，签约已到期
const (
	Dy_Sign_Status_Query_SERVING = "SERVING" // 服务中
	Dy_Sign_Status_Query_CANCEL  = "CANCEL"  // 已解约
	Dy_Sign_Status_Query_TIMEOUT = "TIMEOUT" // 用户未签约
	Dy_Sign_Status_Query_DONE    = "DONE"    // 服务完成，签约已到期
)

// 抖音签约支付回调结构体
type DySignPayCallbackNotify struct {
	AppId string `json:"app_id"` // 小程序 app_id
	// 扣款结果状态状态枚举：
	// "SUCCESS" （扣款成功）
	// "TIMEOUT" （超时未支付 ｜超时未扣款成功）
	// "FAIL" （扣款失败，原因基本都是用户无支付方式（解绑了付款卡）或用户的扣款卡余额不足，建议失败后不要立即重试，隔日再进行重试，若一个月内连续多次扣款均不成功，考虑和用户进行解约）
	Status        string `json:"status"`           // 扣款结果状态
	AuthOrderId   string `json:"auth_order_id"`    // 平台侧签约单的单号，长度<=64byte
	PayOrderId    string `json:"pay_order_id"`     // 平台侧代扣单的单号，长度<=64byte
	OutPayOrderNo string `json:"out_pay_order_no"` // 开发者侧代扣单的单号，长度<=64byte
	TotalAmount   int64  `json:"total_amount"`     // 扣款金额，单位[分]
	PayChannel    int32  `json:"pay_channel"`      // 支付渠道枚举（扣款成功时才有）10：抖音支付
	ChannelPayId  string `json:"channel_pay_id"`   // 渠道支付单
	MerchantUid   string `json:"merchant_uid"`     // 该笔交易卖家商户号
	UserBillPayId string `json:"user_bill_pay_id"` // 用户抖音交易单号（账单号），和用户抖音钱包-账单中所展示的交易单号相同
	EventTime     int64  `json:"event_time"`       // 用户签约成功/签约取消/解约成功的时间戳，单位为毫秒
}

func NewDouyinPay(config *PayConfig) *PayClient {
	client := &PayClient{
		config: config,
	}
	return client
}

// 生成普通商品订单签名
func (c *PayClient) RequestOrder(data interface{}) (string, string, error) {
	dataStr, err := sonic.MarshalString(data)
	if err != nil {
		logx.Errorf("RequestOrder error: %v", err)
		return "", "", err
	}

	logx.Sloww("RequestOrder", logx.Field("dataStr", dataStr))

	byteAuthorization, err := c.GetByteAuthorization("/requestOrder", "POST", dataStr, c.randStr(10), strconv.FormatInt(time.Now().Unix(), 10))
	return dataStr, byteAuthorization, err
}

// https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/api/industry/credit-products/createSignOrder
// 生成周期代扣签约下单接口签名
func (c *PayClient) CreateSignOrder(data interface{}) (string, string, error) {
	dataStr, err := sonic.MarshalString(data)
	if err != nil {
		logx.Errorf("CreateSignOrder error: %v", err)
		return "", "", err
	}

	logx.Sloww("CreateSignOrder", logx.Field("dataStr", dataStr))

	byteAuthorization, err := c.GetByteAuthorization("/createSignOrder", "POST", dataStr, c.randStr(10), strconv.FormatInt(time.Now().Unix(), 10))
	return dataStr, byteAuthorization, err
}

func ParsePKCS1And8PrivateKey(data []byte) (key *rsa.PrivateKey, err error) {
	var block *pem.Block
	block, _ = pem.Decode(data)
	if block == nil {
		return nil, errors.New("ErrPrivateKeyFailedToLoad")
	}

	if block.Type == "PRIVATE KEY" {
		tmpkey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		if key, ok := tmpkey.(*rsa.PrivateKey); ok {
			return key, nil
		}

		err = errors.New("can not parse ParsePKCS8PrivateKey -----BEGIN PRIVATE KEY----- data")
		return nil, err
	}

	key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return key, err
}

func (c *PayClient) GetByteAuthorization(url, method, data, nonceStr, timestamp string) (string, error) {
	var byteAuthorization string
	privateKey, err := ParsePKCS1And8PrivateKey([]byte(c.config.PrivateKey))
	if err != nil {
		logx.Errorw("GetByteAuthorization ParsePKCS1And8PrivateKey", logx.Field("url", url), logx.Field("method", method), logx.Field("err", err))
		return "", err
	}

	// 生成签名
	signature, err := c.getSignature(method, url, timestamp, nonceStr, data, privateKey)
	if err != nil {
		logx.Errorw("GetByteAuthorization getSignature", logx.Field("url", url), logx.Field("method", method), logx.Field("err", err))
		return "", err
	}

	// 构造byteAuthorization
	byteAuthorization = fmt.Sprintf("SHA256-RSA2048 appid=%s,nonce_str=%s,timestamp=%s,key_version=%s,signature=%s", c.config.AppId, nonceStr, timestamp, c.config.KeyVersion, signature)
	return byteAuthorization, nil
}

func (c *PayClient) getSignature(method, url, timestamp, nonce, data string, privateKey *rsa.PrivateKey) (string, error) {
	targetStr := method + "\n" + url + "\n" + timestamp + "\n" + nonce + "\n" + data + "\n"
	h := sha256.New()
	h.Write([]byte(targetStr))
	digestBytes := h.Sum(nil)

	signBytes, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digestBytes)
	if err != nil {
		return "", err
	}
	sign := base64.StdEncoding.EncodeToString(signBytes)

	return sign, nil
}

func (c *PayClient) randStr(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		tmpStr := ""
		for i := 0; i < length; i++ {
			tmpStr += "a"
		}
		return tmpStr
	}
	return base64.StdEncoding.EncodeToString(b)
}

// GeneralTradeCallbackData 抖音侧回调通用请求体
type GeneralTradeCallbackData struct {
	Msg     string    `json:"msg,omitempty"`
	Type    EventType `json:"type,omitempty"`
	Version string    `json:"version,omitempty"`
}

type EventType string

const (
	EventPayment         EventType = "payment"           // 支付
	EventRefund          EventType = "refund"            // 退款
	EventPreCreateRefund EventType = "pre_create_refund" // 客服预退款订单
	EventSettle          EventType = "settle"            //
	EventSignCallback    EventType = "sign_callback"     // 抖音周期代扣签约回调
	EventSignPayCallback EventType = "sign_pay_callback" // 抖音周期代扣结果回调通知
)

type GeneralTradeMsg struct {
	AppId          string `json:"app_id,omitempty"`           // 必填 appId
	OutOrderNo     string `json:"out_order_no,omitempty"`     // 必填 开发者系统订单号
	OrderId        string `json:"order_id,omitempty"`         // 必填 抖音平台侧订单号
	Status         string `json:"status,omitempty"`           // 必填 支付结果状态枚举 "SUCCESS" （支付成功 ） "CANCEL" （支付取消）
	TotalAmount    int64  `json:"total_amount,omitempty"`     // 必填 订单总金额 单位分 当用户以钻石兑换时，会填充为钻石数量
	DiscountAmount int64  `json:"discount_amount,omitempty"`  // 非必填 订单优惠金额 单位分
	PayChannel     int32  `json:"pay_channel,omitempty"`      // 非必填 支付渠道枚举 （支付成功时才有）：1：微信2：支付宝10：抖音支付20:钻石支付
	ChannelPayId   string `json:"channel_pay_id,omitempty"`   // 非必填 渠道支付单号，如微信/支付宝的支付单号，长度 <= 64byte 注：status="SUCCESS"时一定有值
	MerchantUid    string `json:"merchant_uid,omitempty"`     // 非必填 交易卖家商户号 注：status="SUCCESS"时一定有值
	Message        string `json:"message,omitempty"`          // 非必填 交易取消原因 如："USER_CANCEL"：用户取消"TIME_OUT"：超时取消
	EventTime      int64  `json:"event_time,omitempty"`       // 必填 用户支付成功/支付取消时间戳，单位为毫秒
	UserBillPayId  string `json:"user_bill_pay_id,omitempty"` // 非必填 对应用户抖音账单里的"支付单号" 注：status="SUCCESS"时一定有值
	Currency       string `json:"currency,omitempty"`         // 非必填 当用户以钻石兑换时，currency=DIAMOND
}

// VerifyNotify 验签 https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/trade-system/general/order/notify-payment-result
func (c *PayClient) VerifyNotify(req *http.Request, body []byte) error {
	timestamp := req.Header.Get("Byte-Timestamp")
	nonce := req.Header.Get("Byte-Nonce-Str")
	headerSignatureEncode := req.Header.Get("Byte-Signature")

	isPass, err := c.CheckSign(timestamp, nonce, string(body), headerSignatureEncode, c.config.PlatformPublicKey)
	if err != nil {

		return err
	}

	if !isPass {
		return errors.New("sign verify not pass")
	}
	return nil
}

func (c *PayClient) CheckSign(timestamp, nonce, body, signature, pubKeyStr string) (bool, error) {
	pubKey, err := c.pemToRSAPublicKey(pubKeyStr) // 注意验签时publicKey使用平台公钥而非应用公钥
	if err != nil {
		return false, err
	}

	hashed := sha256.Sum256([]byte(timestamp + "\n" + nonce + "\n" + body + "\n"))
	signBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	err = rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashed[:], signBytes)
	return err == nil, nil
}

func (c *PayClient) pemToRSAPublicKey(pemKeyStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemKeyStr))
	if block == nil || len(block.Bytes) == 0 {
		return nil, fmt.Errorf("empty block in pem string")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	switch key := key.(type) {
	case *rsa.PublicKey:
		return key, nil
	default:
		return nil, fmt.Errorf("not rsa public key")
	}
}

type GetClientTokenReq struct {
	ClientKey    string `json:"client_key,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	GrantType    string `json:"grant_type,omitempty"`
}

type GetClientTokenResp struct {
	Data    GetClientTokenData `json:"data"`
	Message string             `json:"message,omitempty"`
}

type GetClientTokenData struct {
	ExpiresIn   int64  `json:"expires_in,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	Description string `json:"description,omitempty"`
	ErrorCode   int64  `json:"error_code,omitempty"`
}

type QueryOrderReq struct {
	OrderId    string `json:"order_id,omitempty"`     // 非必填 交易订单号，order_id 与 out_order_no 二选一
	OutOrderNo string `json:"out_order_no,omitempty"` // 非必填 开发者的单号，order_id 与 out_order_no 二选一
}

type QueryOrderResp struct {
	ApiCommonResp
	Data *QueryOrderData `json:"data,omitempty"`
}

type QueryOrderData struct {
	AppId               string                 `json:"app_id,omitempty"`          // 必填 appId
	OrderId             string                 `json:"order_id,omitempty"`        // 必填 抖音平台侧订单号
	OutOrderNo          string                 `json:"out_order_no,omitempty"`    // 必填 开发者系统订单号
	PayStatus           string                 `json:"pay_status,omitempty"`      // 必填 订单支付状态 PROCESS： 订单处理中 支付处理中 SUCCESS：成功 支付成功 FAIL：失败 支付失败 暂无该情况会支付失败 TIMEOUT：用户超时未支付
	TotalAmount         int64                  `json:"total_amount,omitempty"`    // 必填 订单总金额 单位分 支付金额 = total_amount - discount_amount
	TradeTime           int64                  `json:"trade_time,omitempty"`      // 必填 交易下单时间 毫秒
	ChannelPayId        string                 `json:"channel_pay_id,omitempty"`  // 非必填 渠道支付单号，如：微信的支付单号、支付宝支付单号。 只有在支付成功时才会有值。
	DiscountAmount      int64                  `json:"discount_amount,omitempty"` // 非必填 订单优惠金额，单位：分，接入营销时请关注这个字段
	ItemOrderList       []*QueryOrderItemOrder `json:"item_order_list,omitempty"` // 非必填 item单信息
	MerchantUid         string                 `json:"merchant_uid,omitempty"`    // 非必填 收款商户号
	PayChannel          int8                   `json:"pay_channel,omitempty"`     // 非必填 支付渠道枚举 （支付成功时才有）：1：微信2：支付宝10：抖音支付
	PayTime             int64                  `json:"pay_time,omitempty"`        // 非必填 支付成功时间 毫秒
	Currency            string                 `json:"currency,omitempty"`        // 非必填 支付币种 钻石支付为DIAMOND
	TotalCurrencyAmount int64                  `json:"total_currency_amount"`     // 非必填 当用户以钻石兑换时，该字段会填充对应的钻石数量
}

type QueryOrderItemOrder struct {
	ItemOrderAmount         int64  `json:"item_order_amount,omitempty"`          // 必填 item单金额 分
	ItemOrderId             string `json:"item_order_id,omitempty"`              // 必填 抖音侧 交易系统商品单号
	SkuId                   string `json:"sku_id,omitempty"`                     // 必填 用户下单传入的skuId
	ItemOrderCurrencyAmount int64  `json:"item_order_currency_amount,omitempty"` // 非必填 当用户以钻石兑换时，该字段会填充对应的钻石数量
}

type ApiCommonResp struct {
	ErrNo  int64  `json:"err_no,omitempty"` // 0是正常
	ErrMsg string `json:"err_msg,omitempty"`
	LogId  string `json:"log_id,omitempty"`
}

// QueryOrder 查询订单 https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/trade-system/general/order/query_order
func (c *PayClient) QueryOrder(orderId, outOrderId, clientToken string) (*QueryOrderResp, error) {
	if orderId == "" && outOrderId == "" {
		return nil, errors.New("OrderId and OutOrderNo can not empty same time")
	}

	header := map[string]string{
		"access-token": clientToken,
	}
	req := &QueryOrderReq{
		OrderId:    orderId,
		OutOrderNo: outOrderId, // 类似1235700313565384704
	}
	result, err := util.HttpPostWithHeader(trade_order_query_url, req, header, time.Second*5)

	// 记录返回日志
	logx.Sloww("QueryOrder", logx.Field("result", result), logx.Field("OrderId", orderId), logx.Field("OutOrderNo", outOrderId), logx.Field("err", err))

	if err != nil {
		return nil, err
	}

	resp := new(QueryOrderResp)
	err = json.Unmarshal([]byte(result), resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// 用户签约返回数据结构体
type UserSignResp struct {
	ApiCommonResp
	UserSignData UserSignDataObj `json:"data,optional"`
}

type UserSignDataObj struct {
	AppId          string `json:"app_id,optional"`            // 小程序 app_id
	AuthOrderId    string `json:"auth_order_id,optional"`     // 平台侧签约单的单号，长度<=64byte
	OutAuthOrderNo string `json:"out_auth_order_no,optional"` // 开发者侧签约单的单号，长度<=64byte
	ServiceId      string `json:"service_id,optional"`        // 签约模板ID
	Status         string `json:"status,optional"`            // 签约单状态 TOBESERVED: 待服务 SERVING：服务中 CANCEL: 已解约 TIMEOUT: 用户未签约 DONE: 服务完成，签约已到期
	CancelSource   int    `json:"cancel_source,optional"`     // 解约来源 1-用户解约 2-商户解约
	OpenId         string `json:"open_id,optional"`           // 用户open id
	SignTime       int64  `json:"sign_time,optional"`         // 用户签约完成时间，时间毫秒
}

// 查询抖音周期代扣签约单的状态
// https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/payment/management-capacity/periodic-deduction/sign/query-sign-order
//
// clientToken appid的access token
//
// outAuthOrderId 开发者侧签约单的单号
func (c *PayClient) QuerySignOrder(clientToken, outAuthOrderId string) (*UserSignResp, error) {
	header := map[string]string{
		"access-token": clientToken,
	}

	// auth_order_idString
	// 示例："ad712312312313213"
	// 平台侧签约单的单号，长度<=64byte，auth_order_id 与 out_auth_order_no 二选一

	// out_auth_order_noString
	// 示例："out_order_1"
	// 开发者侧签约单的单号，长度<=64byte，

	// auth_order_id 与 out_auth_order_no 二选一

	params := map[string]string{
		"out_auth_order_no": outAuthOrderId,
	}
	result, err := util.HttpPostWithHeader(query_sign_order_url, params, header, time.Second*5)

	// 记录返回日志
	logx.Sloww("QuerySignOrder", logx.Field("result", result), logx.Field("outAuthOrderId", outAuthOrderId), logx.Field("err", err))

	if err != nil {
		return nil, err
	}

	// 正常时返回
	// {
	// 	"data": {
	// 	  "app_id": "tt312312313123",
	// 	  "auth_order_id": "ad712312312313213",
	// 	  "out_auth_order_no": "out_order_1",
	// 	  "service_id": "64",
	// 	  "status": "CANCEL",
	// 	  "cancel_source": 1,
	// 	  "open_id": "ffwqeqgyqwe312",
	// 	  "sign_time": 1698128528000,
	// 	  "notify_url": "https://www.asdasd"
	// 	},
	// 	"err_no": 0,
	// 	"err_msg": "success",
	// 	"log_id": "2022092115392201020812109511046"
	//   }

	// 异常时返回
	// {
	// 	"err_no": 10000,
	// 	"err_msg": "参数不合法",
	// 	"log_id": "2022092115392201020812109511046"
	// }

	resp := new(UserSignResp)
	err = json.Unmarshal([]byte(result), resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// 发起解约抖音周期代扣
// 签约单状态只有在 服务中（SERVING）才允许解约
// https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/payment/management-capacity/periodic-deduction/sign/terminate-sign
//
// clientToken appid的access token
//
// authOrderId 平台侧签约单的单号
func (c *PayClient) TerminateSign(clientToken, authOrderId string) (*ApiCommonResp, error) {
	header := map[string]string{
		"access-token": clientToken,
	}

	params := map[string]string{
		"auth_order_id": authOrderId,
	}
	result, err := util.HttpPostWithHeader(trade_terminate_sign_url, params, header, time.Second*5)

	// 记录返回日志
	logx.Sloww("TerminateSign", logx.Field("result", result), logx.Field("authOrderId", authOrderId), logx.Field("err", err))

	if err != nil {
		return nil, err
	}

	// 正常时返回
	// {
	// 	"err_no": 0,
	// 	"err_msg": "success",
	// 	"log_id": "2022092115392201020812109511046"
	// 	}

	// 异常时返回
	// {
	// 	"err_no": 10000,
	// 	"err_msg": "参数不合法",
	// 	"log_id": "2022092115392201020812109511046"
	// }

	resp := new(ApiCommonResp)
	err = json.Unmarshal([]byte(result), resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
