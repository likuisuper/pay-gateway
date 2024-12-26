package client

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/common/utils"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	jsoniter "github.com/json-iterator/go"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	ksHttpRequestErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "ksHttpRequestErr", nil, "快手请求错误", nil})}
)

// 快手开发接口文档
// https://open.kuaishou.com/docs/develop/server/epay/open-api-new/prePay-new.html

// =============================IOS支付说明=============================
// https://open.kuaishou.com/docs/develop/server/iosEpayAbility/iosEpayGuide.html
// 快手小程序接入iOS的IAP支付，相较于担保支付(单次) (opens new window)，有两处变动改造点
// 使用/openapi/mp/developer/epay/iap/create_order预下单接口
// ks.pay接口增加苹果内购标识
//
// 苹果内购不支持线上退款，因此没有退款请求接口。若用户有退款请求，可使用以下方式
// 商家可联系用户线下协商退款
// 引导用户至苹果客服侧申请退款
// 1.3 关于订单同步
// IAP支付的订单需要进行订单同步，使用订单信息同步接口(opens new window)
//
// 1.4 关于账单查询
// 与担保支付的账单查询接口保持一致：担保支付|账单查询能力(opens new window)
//
// 支付方式为APPLE_PAY的订单，即为苹果支付的订单
//
// IAP订单会在支付账单里面添加
//
// 订单原价
// 用户支付金额
// 平台补贴金额
// 支付渠道：APPLE_PAY
// 在退款账单新增：
//
// 支付渠道：APPLE_PAY
// 1.5 版本说明
// 在iOS系统中，快手11.6.50版本开始封禁小程序支付能力，无法通过微信或者支付宝支付。
//
// 在12.0.20版本中支持苹果IAP支付，因此在接入IAP支付时，需要把测试机版本升级到12.0.20及以上
// =====================================================================

// 请求地址
const (
	ksAccessToken            = "https://open.kuaishou.com/oauth2/access_token"                                 // 获取accessToken
	KsCreateOrderWithChannel = "https://open.kuaishou.com/openapi/mp/developer/epay/create_order_with_channel" // 预下单接口安卓（无收银台版）
	KsCreateOrder            = "https://open.kuaishou.com/openapi/mp/developer/epay/create_order"              // 预下单接口安卓（有收银台版）
	KsCreateOrderIos         = "https://open.kuaishou.com/openapi/mp/developer/epay/iap/create_order"          // 预下单接口苹果（有收银台版） 注：IAP支付接入，仅因对iOS系统内，安卓系统无需关注，使用老的担保支付即可
	KsCancelChannel          = "https://open.kuaishou.com/openapi/mp/developer/epay/cancel_channel"            // 取消支付方式
	KsQueryOrder             = "https://open.kuaishou.com/openapi/mp/developer/epay/query_order"               // 查询订单
)

type KsPayConfig struct {
	AppId     string
	AppSecret string
	NotifyUrl string //支付回调
}

type KsCreateOrderWithChannelReq struct {
	OutOrderNo  string     `json:"out_order_no"` //商户系统内部订单号
	OpenId      string     `json:"open_id"`      //快手用户在当前小程序的open_id
	TotalAmount int        `json:"total_amount"` //用户支付金额，单位为[分]
	Subject     string     `json:"subject"`      //商品描述
	Detail      string     `json:"detail"`       //商品详情
	Type        int        `json:"type"`         //商品类型，不同商品类目的编号
	ExpireTime  int        `json:"expire_time"`  //订单过期时间，单位秒，300s - 172800s
	Sign        string     `json:"sign"`         //签名
	NotifyUrl   string     `json:"notify_url"`   //通知URL，不允许携带查询串
	Provider    KsProvider `json:"provider"`     //无收银台支付 支付方式 json
}

type KsCreateOrderReq struct {
	OutOrderNo  string `json:"out_order_no"` //商户系统内部订单号
	OpenId      string `json:"open_id"`      //快手用户在当前小程序的open_id
	TotalAmount int    `json:"total_amount"` //用户支付金额，单位为[分]
	Subject     string `json:"subject"`      //商品描述
	Detail      string `json:"detail"`       //商品详情
	Type        int    `json:"type"`         //商品类型，不同商品类目的编号
	ExpireTime  int    `json:"expire_time"`  //订单过期时间，单位秒，300s - 172800s
	Sign        string `json:"sign"`         //签名
	NotifyUrl   string `json:"notify_url"`   //通知URL，不允许携带查询串
}

type KsCreateOrderReqIos struct {
	OutOrderNo    string `json:"out_order_no"`    //商户系统内部订单号
	OpenId        string `json:"open_id"`         //快手用户在当前小程序的open_id
	UserPayAmount int    `json:"user_pay_amount"` //用户实际金额，单位为[分]
	OrderAmount   int    `json:"order_amount"`    //订单原价，单位为[分]，不允许传非整数的数值。
	Subject       string `json:"subject"`         //商品描述
	Detail        string `json:"detail"`          //商品详情
	Type          int    `json:"type"`            //商品类型，不同商品类目的编号
	ExpireTime    int    `json:"expire_time"`     //订单过期时间，单位秒，300s - 172800s
	Sign          string `json:"sign"`            //签名
	NotifyUrl     string `json:"notify_url"`      //通知URL，不允许携带查询串
}

type KsProvider struct {
	Provider            string `json:"provider"`              //支付方式，枚举值，目前只支持"WECHAT"、"ALIPAY"两种
	ProviderChannelType string `json:"provider_channel_type"` //支付方式子类型，枚举值，目前只支持"NORMAL"
}

type KsCreateOrderWithChannelResp struct {
	OrderNo        string `json:"order_no"`         //订单号
	OrderInfoToken string `json:"order_info_token"` //token
}

type KsQueryOrderResp struct {
	TotalAmount     int    `json:"total_amount"`     //预下单用户支付金额
	PayStatus       string `json:"pay_status"`       // PROCESSING-处理中|SUCCESS-成功|FAILED-失败|TIMEOUT-超时
	PayChannel      string `json:"pay_channel"`      // WECHAT-微信 | ALIPAY-支付宝
	OutOrderNo      string `json:"out_order_no"`     //开发者下单单号
	KsOrderNo       string `json:"ks_order_no"`      //快手小程序平台订单号
	ExtraInfo       string `json:"extra_info"`       //订单来源信息，历史订单为""
	EnablePromotion bool   `json:"enable_promotion"` //是否参与分销，true:分销，false:非分销
	PromotionAmount int    `json:"promotion_amount"` //预计分销金额，单位：分
}

// 快手支付
type KsPay struct {
	Config KsPayConfig
}

func NewKsPay(config KsPayConfig) *KsPay {
	return &KsPay{
		Config: config,
	}
}

// 预下单接口(无收银台版)
func (p *KsPay) CreateOrderWithChannel(info *PayOrder, openId string, accessToken string) (respData *KsCreateOrderWithChannelResp, err error) {
	uri := fmt.Sprintf("%s?app_id=%s&access_token=%s", KsCreateOrderWithChannel, p.Config.AppId, accessToken)
	provider := KsProvider{
		Provider:            "WECHAT",
		ProviderChannelType: "NORMAL",
	}
	//providerJsonStr, _ := jsoniter.MarshalToString(provider)
	param := &KsCreateOrderWithChannelReq{
		OutOrderNo:  info.OrderSn,
		OpenId:      openId,
		TotalAmount: info.Amount,
		Subject:     info.Subject,
		Detail:      info.Subject,
		Type:        info.KsTypeId,
		ExpireTime:  3600,
		NotifyUrl:   p.Config.NotifyUrl,
		Provider:    provider,
	}
	param.Sign = p.Sign(param)

	dataStr, err := util.HttpPost(uri, param, 3*time.Second)
	if err != nil {
		jsonStr, _ := jsoniter.MarshalToString(param)
		util.CheckError("CreateOrderWithChannel Err:%v, dataJson: %s", err, jsonStr)
		ksHttpRequestErr.CounterInc()
		return
	}
	resultCode := jsoniter.Get([]byte(dataStr), "result").ToInt()
	if resultCode != 1 {
		errorMsg := jsoniter.Get([]byte(dataStr), "error_msg").ToString()
		err = errors.New(errorMsg)
		jsonStr, _ := jsoniter.MarshalToString(param)
		util.CheckError("CreateOrderWithChannel Err:%v, dataJson: %s", err, jsonStr)
		ksHttpRequestErr.CounterInc()
		return
	}

	respData = new(KsCreateOrderWithChannelResp)
	jsoniter.Get([]byte(dataStr), "order_info").ToVal(respData)

	return
}

// 预下单接口安卓(有收银台版)
func (p *KsPay) CreateOrder(info *PayOrder, openId string, accessToken string) (respData *KsCreateOrderWithChannelResp, err error) {
	uri := fmt.Sprintf("%s?app_id=%s&access_token=%s", KsCreateOrder, p.Config.AppId, accessToken)

	param := &KsCreateOrderReq{
		OutOrderNo:  info.OrderSn,
		OpenId:      openId,
		TotalAmount: info.Amount,
		Subject:     info.Subject,
		Detail:      info.Subject,
		Type:        info.KsTypeId,
		ExpireTime:  3600, // 订单过期时间，单位秒，300s - 172800s
		NotifyUrl:   p.Config.NotifyUrl,
	}
	param.Sign = p.Sign(param)

	dataStr, err := util.HttpPost(uri, param, 3*time.Second)
	if err != nil {
		jsonStr, _ := jsoniter.MarshalToString(param)
		util.CheckError("CreateOrder Err:%v, dataJson: %s", err, jsonStr)
		ksHttpRequestErr.CounterInc()
		return
	}

	resultCode := jsoniter.Get([]byte(dataStr), "result").ToInt()
	if resultCode != 1 {
		errorMsg := jsoniter.Get([]byte(dataStr), "error_msg").ToString()
		err = errors.New(errorMsg)
		jsonStr, _ := jsoniter.MarshalToString(param)
		util.CheckError("CreateOrderWithChannel Err:%v, dataJson: %s", err, jsonStr)
		ksHttpRequestErr.CounterInc()
		return
	}

	respData = new(KsCreateOrderWithChannelResp)
	jsoniter.Get([]byte(dataStr), "order_info").ToVal(respData)

	logx.Slowf("KsPay-CreateOrder, param:%+v, dataStr:%s", param, dataStr)
	return
}

// 预下单接口苹果ios
// https://open.kuaishou.com/docs/develop/server/iosEpayAbility/iosEpayGuide.html
//
// 苹果价格档位
// https://open.kuaishou.com/docs/develop/server/iosEpayAbility/feeStandards.html
func (p *KsPay) CreateOrderIos(info *PayOrder, openId string, accessToken string) (respData *KsCreateOrderWithChannelResp, err error) {
	uri := fmt.Sprintf("%s?app_id=%s&access_token=%s", KsCreateOrderIos, p.Config.AppId, accessToken)

	param := &KsCreateOrderReqIos{
		OutOrderNo:    info.OrderSn,
		OpenId:        openId,
		UserPayAmount: info.Amount, // 用户实际金额，单位为[分] 如果订单没有补贴，则user_pay_amount=order_amount
		OrderAmount:   info.Amount, // 必填
		Subject:       info.Subject,
		Detail:        info.Subject,
		Type:          info.KsTypeId,
		ExpireTime:    3600, // 订单过期时间，单位秒，300s - 172800s
		NotifyUrl:     p.Config.NotifyUrl,
	}
	param.Sign = p.Sign(param)

	dataStr, err := util.HttpPost(uri, param, 3*time.Second)
	if err != nil {
		jsonStr, _ := jsoniter.MarshalToString(param)
		util.CheckError("CreateOrder Err:%v, dataJson: %s", err, jsonStr)
		ksHttpRequestErr.CounterInc()
		return
	}

	resultCode := jsoniter.Get([]byte(dataStr), "result").ToInt()
	if resultCode != 1 {
		errorMsg := jsoniter.Get([]byte(dataStr), "error_msg").ToString()
		err = errors.New(errorMsg)
		jsonStr, _ := jsoniter.MarshalToString(param)
		util.CheckError("CreateOrderWithChannel Err:%v, dataJson: %s", err, jsonStr)
		ksHttpRequestErr.CounterInc()
		return
	}

	respData = new(KsCreateOrderWithChannelResp)
	jsoniter.Get([]byte(dataStr), "order_info").ToVal(respData)

	logx.Slowf("KsPay-CreateOrder, param:%+v, dataStr:%s", param, dataStr)
	return
}

// 取消支付方式接口
func (p *KsPay) CancelChannel(orderSn string, accessToken string) (err error) {
	uri := fmt.Sprintf("%s?app_id=%s&access_token=%s", KsCancelChannel, p.Config.AppId, accessToken)
	params := map[string]interface{}{
		"out_order_no": orderSn,
	}
	params["sign"] = p.Sign(params)

	dataStr, err := util.HttpPost(uri, params, 3*time.Second)
	if err != nil {
		util.CheckError("CreateOrderWithChannel Err:%v, OrderSn:%s", err, orderSn)
		ksHttpRequestErr.CounterInc()
		return
	}
	resultCode := jsoniter.Get([]byte(dataStr), "result").ToInt()
	if resultCode != 1 {
		errorMsg := jsoniter.Get([]byte(dataStr), "error_msg").ToString()
		err = errors.New(errorMsg)
		util.CheckError("CreateOrderWithChannel Err:%v, OrderSn:%s", err, orderSn)
		ksHttpRequestErr.CounterInc()
		return
	}

	return

}

// 查询订单
func (p *KsPay) QueryOrder(orderSn string, accessToken string) (paymentInfo *KsQueryOrderResp, err error) {
	uri := fmt.Sprintf("%s?app_id=%s&access_token=%s", KsQueryOrder, p.Config.AppId, accessToken)
	params := map[string]interface{}{
		"out_order_no": orderSn,
	}
	params["sign"] = p.Sign(params)
	dataStr, err := util.HttpPost(uri, params, 3*time.Second)
	if err != nil {
		util.CheckError("QueryOrder Err:%v, OrderSn:%s", err, orderSn)
		ksHttpRequestErr.CounterInc()
		return
	}

	resultCode := jsoniter.Get([]byte(dataStr), "result").ToInt()
	if resultCode != 1 {
		errorMsg := jsoniter.Get([]byte(dataStr), "error_msg").ToString()
		err = errors.New(errorMsg)
		util.CheckError("QueryOrder Err:%v, code:%d, OrderSn:%s", err, resultCode, orderSn)
		ksHttpRequestErr.CounterInc()
		return
	}

	paymentInfo = new(KsQueryOrderResp)
	jsoniter.Get([]byte(dataStr), "payment_info").ToVal(paymentInfo)
	return
}

//==========================  util  ========================================================================

// 快手支付核心签名
// https://open.kuaishou.com/docs/develop/server/epay/appendix.html
func (p *KsPay) Sign(param interface{}) (sign string) {
	dataMap := make(map[string]interface{}, 0)
	jsonBytes, _ := json.Marshal(param)
	_ = json.Unmarshal(jsonBytes, &dataMap)

	signParam := make(map[string]string, 0)
	signParam["app_id"] = p.Config.AppId
	for k, v := range dataMap {
		if k == "provider" {
			jsonStr, _ := jsoniter.MarshalToString(v)
			signParam[k] = jsonStr
		} else {
			signParam[k] = utils.ToString(v)
		}
		//signParam[k] = utils.ToString(v)
	}
	sign = p.makeSign(signParam)
	return
}

// 参数生成签名
func (p *KsPay) makeSign(data map[string]string) string {
	str := ""

	//map根据key排序
	var keys = make([]string, 0)
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, v := range keys {
		if data[v] == "" {
			continue
		}
		str += "&" + v + "=" + data[v]
	}
	str = strings.Trim(str, "&")
	str += p.Config.AppSecret

	h := md5.New()
	h.Write([]byte(str))
	md5Str := hex.EncodeToString(h.Sum(nil))
	sign := strings.ToLower(md5Str)

	return sign
}

// 请求快手post  x-www-form-urlencoded方式传参
func (p *KsPay) post(url string, postData map[string]string) (body []byte, err error) {
	dataStr := ""
	for k, v := range postData {
		kvStr := fmt.Sprintf("%s=%s", k, v)
		dataStr += "&" + kvStr
	}
	dataStr = strings.Trim(dataStr, "&")
	payload := strings.NewReader(dataStr)

	client := &http.Client{}
	req, err := http.NewRequest("POST", url, payload)

	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	return
}

// 回调签名
func (p *KsPay) NotifySign(bodyStr string) (sign string) {
	str := bodyStr + p.Config.AppSecret
	h := md5.New()
	h.Write([]byte(str))
	md5Str := hex.EncodeToString(h.Sum(nil))
	sign = strings.ToLower(md5Str)
	return
}
