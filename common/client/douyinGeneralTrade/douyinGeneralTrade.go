package douyin

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/bytedance/sonic"
	"net/http"
	"strconv"
	"time"
)

type PayConfig struct {
	AppId             string
	PrivateKey        string // 应用私钥
	KeyVersion        string
	NotifyUrl         string
	PlatformPublicKey string // 平台公钥
	AppSecret         string // 应用密钥
	CustomerImId      string
	GetClientTokenUrl string
}

type PayClient struct {
	config *PayConfig
}

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
}

type SkuType int32

const (
	SkuContentRecharge SkuType = 401
)

type SkuTagGroupId string

const (
	SKuTagGroupIdContentRecharge SkuTagGroupId = "tag_group_7272625659888041996"
)

type Schema struct {
	Path   string `json:"path,omitempty"`   // 小程序xxx详情页跳转路径，没有前导的“/”，路径后不可携带query参数，路径中不可携带『？: & *』等特殊字符，路径只可以是『英文字符、数字、_、/ 』等组成，长度<=512byte
	Params string `json:"params,omitempty"` // xx情页路径参数，自定义的json结构，内部为k-v结构，序列化成字符串存入该字段，平台不限制，但是写入的内容需要能够保证生成访问xx详情的schema能正确跳转到小程序内部的xx详情页，长度须<=512byte，params内key不可重复。
}

func NewDouyinPay(config *PayConfig) *PayClient {
	client := &PayClient{
		config: config,
	}
	return client
}

func (c *PayClient) RequestOrder(data *RequestOrderData) (string, string, error) {
	dataStr, err := sonic.MarshalString(data)
	if err != nil {
		return "", "", err
	}
	byteAuthorization, err := c.GetByteAuthorization("/requestOrder", "POST", dataStr, c.randStr(10), strconv.FormatInt(time.Now().Unix(), 10))
	return dataStr, byteAuthorization, err
}

func ParsePKCS1PrivateKey(data []byte) (key *rsa.PrivateKey, err error) {
	var block *pem.Block
	block, _ = pem.Decode(data)
	if block == nil {
		return nil, errors.New("ErrPrivateKeyFailedToLoad")
	}

	key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return key, err
}

func (c *PayClient) GetByteAuthorization(url, method, data, nonceStr, timestamp string) (string, error) {
	var byteAuthorization string
	// 读取私钥
	//key, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(c.config.PrivateKey, "\n", ""))
	//if err != nil {
	//	return "", err
	//}
	//privateKey, err := x509.ParsePKCS1PrivateKey(key)
	//if err != nil {
	//	return "", err
	//}
	privateKey, err := ParsePKCS1PrivateKey([]byte(c.config.PrivateKey))
	if err != nil {
		return "", err
	}
	// 生成签名
	signature, err := c.getSignature(method, url, timestamp, nonceStr, data, privateKey)
	if err != nil {
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
		panic(err)
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
	EventPayment         EventType = "payment"
	EventRefund          EventType = "refund"
	EventPreCreateRefund EventType = "pre_create_refund"
)

type GeneralTradeMsg struct {
	AppId          string `json:"app_id,omitempty"`           // 必填 appId
	OutOrderNo     string `json:"out_order_no,omitempty"`     // 必填 开发者系统订单号
	OrderId        string `json:"order_id,omitempty"`         // 必填 抖音开平侧订单号
	Status         string `json:"status,omitempty"`           // 必填 支付结果状态枚举 "SUCCESS" （支付成功 ） "CANCEL" （支付取消）
	TotalAmount    int64  `json:"total_amount,omitempty"`     // 必填 订单总金额 单位分
	DiscountAmount int64  `json:"discount_amount,omitempty"`  // 非必填 订单优惠金额 单位分
	PayChannel     int32  `json:"pay_channel,omitempty"`      // 非必填 支付渠道枚举 （支付成功时才有）：1：微信2：支付宝10：抖音支付
	ChannelPayId   string `json:"channel_pay_id,omitempty"`   // 非必填 渠道支付单号，如微信/支付宝的支付单号，长度 <= 64byte 注：status="SUCCESS"时一定有值
	MerchantUid    string `json:"merchant_uid,omitempty"`     // 非必填 交易卖家商户号 注：status="SUCCESS"时一定有值
	Message        string `json:"message,omitempty"`          // 非必填 交易取消原因 如："USER_CANCEL"：用户取消"TIME_OUT"：超时取消
	EventTime      int64  `json:"event_time,omitempty"`       // 必填 用户支付成功/支付取消时间戳，单位为毫秒
	UserBillPayId  string `json:"user_bill_pay_id,omitempty"` // 非必填 对应用户抖音账单里的"支付单号" 注：status="SUCCESS"时一定有值
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

// GetClientToken 获取接口调用token https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/interface-request-credential/non-user-authorization/get-client_token#739149f2
func (c *PayClient) GetClientToken() (*GetClientTokenResp, error) {
	req := &GetClientTokenReq{
		ClientKey:    c.config.AppId,
		ClientSecret: c.config.AppSecret,
		GrantType:    "client_credential",
	}
	result, err := util.HttpPost("https://open.douyin.com/oauth/client_token/", req, time.Second*3)
	if err != nil {
		return nil, err
	}
	resp := new(GetClientTokenResp)
	err = sonic.UnmarshalString(result, resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
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
	AppId          string                 `json:"app_id,omitempty"`          // 必填 appId
	OrderId        string                 `json:"order_id,omitempty"`        // 必填 开平侧订单号
	OutOrderNo     string                 `json:"out_order_no,omitempty"`    // 必填 开发者系统订单号
	PayStatus      string                 `json:"pay_status,omitempty"`      // 必填 订单支付状态 PROCESS： 订单处理中 支付处理中 SUCCESS：成功 支付成功 FAIL：失败 支付失败 暂无该情况会支付失败 TIMEOUT：用户超时未支付
	TotalAmount    int64                  `json:"total_amount,omitempty"`    // 必填 订单总金额 单位分 支付金额 = total_amount - discount_amount
	TradeTime      int64                  `json:"trade_time,omitempty"`      // 必填 交易下单时间 毫秒
	ChannelPayId   string                 `json:"channel_pay_id,omitempty"`  // 非必填 渠道支付单号，如：微信的支付单号、支付宝支付单号。 只有在支付成功时才会有值。
	DiscountAmount int64                  `json:"discount_amount,omitempty"` // 非必填 订单优惠金额，单位：分，接入营销时请关注这个字段
	ItemOrderList  []*QueryOrderItemOrder `json:"item_order_list,omitempty"` // 非必填 item单信息
	MerchantUid    string                 `json:"merchant_uid,omitempty"`    // 非必填 收款商户号
	PayChannel     int8                   `json:"pay_channel,omitempty"`     // 非必填 支付渠道枚举 （支付成功时才有）：1：微信2：支付宝10：抖音支付
	PayTime        int64                  `json:"pay_time,omitempty"`        // 非必填 支付成功时间 毫秒
}

type QueryOrderItemOrder struct {
	ItemOrderAmount int64  `json:"item_order_amount,omitempty"` // 必填 item单金额 分
	ItemOrderId     string `json:"item_order_id,omitempty"`     // 必填 抖音侧 交易系统商品单号
	SkuId           string `json:"sku_id,omitempty"`            // 必填 用户下单传入的skuId
}

// QueryOrder 查询订单 https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/trade-system/general/order/query_order
func (c *PayClient) QueryOrder(orderId, outOrderId string) (*QueryOrderResp, error) {
	clientToken, err := getClientToken(c.config.GetClientTokenUrl, c.config.AppId)
	if err != nil {
		return nil, err
	}
	header := map[string]string{
		"access-token": clientToken,
	}
	req := &QueryOrderReq{
		OrderId:    orderId,
		OutOrderNo: outOrderId,
	}
	result, err := util.HttpPostWithHeader("https://open.douyin.com/api/trade_basic/v1/developer/order_query/", req, header, time.Second*3)
	if err != nil {
		return nil, err
	}

	resp := new(QueryOrderResp)
	err = sonic.UnmarshalString(result, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type ApiCommonResp struct {
	ErrNo  int64  `json:"err_no,omitempty"`
	ErrMsg string `json:"err_msg,omitempty"`
	LogId  string `json:"log_id,omitempty"`
}
