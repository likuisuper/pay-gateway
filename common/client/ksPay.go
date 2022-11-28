package client

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/global"
	"gitee.com/zhuyunkj/pay-gateway/common/utils"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	jsoniter "github.com/json-iterator/go"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"
)

var (
	ksHttpRequestErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "ksHttpRequestErr", nil, "快手请求错误", nil})}
)

//请求地址
const (
	ksAccessToken            = "https://open.kuaishou.com/oauth2/access_token"                                 //获取accessToken
	KsCreateOrderWithChannel = "https://open.kuaishou.com/openapi/mp/developer/epay/create_order_with_channel" //预下单接口（无收银台版）
	KsCancelChannel          = "https://open.kuaishou.com/openapi/mp/developer/epay/cancel_channel"            //取消支付方式
	KsQueryOrder             = "https://open.kuaishou.com/openapi/mp/developer/epay/query_order"               //查询订单
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

//快手支付
type KsPay struct {
	Config KsPayConfig
}

var ksPay *KsPay

func NewKsPay(config KsPayConfig) *KsPay {
	if ksPay == nil {
		ksPay = &KsPay{
			Config: config,
		}
	}
	return ksPay
}

//获取accessToken
func (p *KsPay) HttpGetAccessToken() (accessToken string, err error) {
	dataMap := map[string]string{
		"app_id":     p.Config.AppId,
		"app_secret": p.Config.AppSecret,
		"grant_type": "client_credentials",
	}
	body, err := p.post(ksAccessToken, dataMap)
	if err != nil {
		util.CheckError("HttpGetAccessToken :%v", err)
		ksHttpRequestErr.CounterInc()
		return
	}
	resultCode := jsoniter.Get(body, "result").ToInt()
	if resultCode != 1 {
		err = errors.New("获取失败")
		return
	}
	accessToken = jsoniter.Get(body, "access_token").ToString()
	return
}

//获取accessToken 有缓存
func (p *KsPay) GetAccessTokenWithCache() (accessToken string, err error) {
	cacheKey := "ks:access:token:" + p.Config.AppId
	source := func() interface{} {
		at, atErr := p.HttpGetAccessToken()
		if atErr != nil {
			return atErr
		}
		return at
	}
	expire := 86400
	err = global.MemoryCacheInstance.GetDataWithCache(cacheKey, expire, &accessToken, source)
	if err != nil {
		return
	}
	return
}

//预下单接口(无收银台版)
func (p *KsPay) CreateOrderWithChannel(info *PayOrder, openId string) (respData *KsCreateOrderWithChannelResp, err error) {
	accessToken, err := p.GetAccessTokenWithCache()
	if err != nil {
		util.CheckError("CreateOrderWithChannel Err:%v", err)
		return
	}
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

//取消支付方式接口
func (p *KsPay) CancelChannel(orderSn string) (err error) {
	accessToken, err := p.GetAccessTokenWithCache()
	if err != nil {
		util.CheckError("CreateOrderWithChannel Err:%v", err)
		return
	}
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

//查询订单
func (p *KsPay) QueryOrder(orderSn string) (paymentInfo *KsQueryOrderResp, err error) {
	accessToken, err := p.GetAccessTokenWithCache()
	if err != nil {
		util.CheckError("QueryOrder Err:%v", err)
		return
	}
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

func (p *KsPay) Sign(param interface{}) (sign string) {
	dataMap := make(map[string]interface{}, 0)
	jsonBytes, _ := jsoniter.Marshal(param)
	_ = jsoniter.Unmarshal(jsonBytes, &dataMap)

	signParam := make(map[string]string, 0)
	signParam["app_id"] = p.Config.AppId
	for k, v := range dataMap {
		if k == "provider" {
			jsonStr, _ := jsoniter.MarshalToString(v)
			signParam[k] = jsonStr
		} else {
			signParam[k] = utils.ToString(v)
		}
	}
	sign = p.makeSign(signParam)
	return
}

//参数生成签名
func (p *KsPay) makeSign(data map[string]string) string {
	str := ""

	//map根据key排序
	var keys = make([]string, 0)
	for k, _ := range data {
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

//请求快手post  x-www-form-urlencoded方式传参
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

//回调签名
func (p *KsPay) NotifySign(bodyStr string) (sign string) {
	str := bodyStr + p.Config.AppSecret
	h := md5.New()
	h.Write([]byte(str))
	md5Str := hex.EncodeToString(h.Sum(nil))
	sign = strings.ToLower(md5Str)
	return
}
