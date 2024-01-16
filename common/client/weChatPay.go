package client

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
	"github.com/zeromicro/go-zero/core/logx"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

var (
	weChatHttpRequestErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "weChatHttpRequestErr", nil, "weChat请求错误", nil})}
	weChatNotifyErr      = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "weChatNotifyErr", nil, "weChat回调通知错误", nil})}
	weChatRefundOrderErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "weChatRefundOrderErr", nil, "weCha退款失败次数", nil})}
	weChatReturnPayErr   = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "weChatReturnPayErr", nil, "微信支付返回错误", nil})}
)

const (
	WeChatRequestUri = "https://api.mch.weixin.qq.com/pay/unifiedorder"
	WechatTradeType  = "MWEB"
	WechatSignType   = "MD5"
	WechatSandboxUri = "https://api.mch.weixin.qq.com/xdc/apiv2sandbox/pay/unifiedorder"
	SandboxUriSign   = "https://api.mch.weixin.qq.com/xdc/apiv2getsignkey/sign/getsignkey"
)

// 微信支付配置
type WechatPayConfig struct {
	AppId          string //应用ID
	MchId          string //直连商户号
	ApiKey         string //apiV3密钥
	PrivateKeyPath string //apiV3密钥
	SerialNumber   string //商户证书序列号
	NotifyUrl      string //通知地址
	ApiKeyV2       string //apiKeyV2密钥
	WapUrl         string // 支付H5域名
	WapName        string // 支付名称
}

// WXOrderParam	微信请求参数
type WXOrderParam struct {
	APPID          string `xml:"appid"`            //公众账号ID
	MchID          string `xml:"mch_id"`           //商户号
	NonceStr       string `xml:"nonce_str"`        //随机字符串
	Attach         string `xml:"attach"`           //附加数据
	Sign           string `xml:"sign"`             //签名
	Body           string `xml:"body"`             //商品描述
	OutTradeNo     string `xml:"out_trade_no"`     //商户订单号
	TotalFee       int    `xml:"total_fee"`        //总金额
	SpbillCreateIP string `xml:"spbill_create_ip"` //终端IP
	NotifyUrl      string `xml:"notify_url"`       //通知地址
	TradeType      string `xml:"trade_type"`       //交易类型
	SceneInfo      string `xml:"scene_info"`       //场景信息
}

// WXOrderReply	微信请求返回结果
type WXOrderReply struct {
	ReturnCode string `xml:"return_code"`  //返回状态码
	ReturnMsg  string `xml:"return_msg"`   //返回信息
	APPID      string `xml:"appid"`        //公众账号ID
	MchID      string `xml:"mch_id"`       //商户号
	DeviceInfo string `xml:"device_info"`  //设备号
	NonceStr   string `xml:"nonce_str"`    //随机字符串
	Sign       string `xml:"sign"`         //签名
	ResultCode string `xml:"result_code"`  //业务结果
	ErrCode    string `xml:"err_code"`     //错误代码
	ErrCodeDes string `xml:"err_code_des"` //错误代码描述
	TradeType  string `xml:"trade_type"`   //交易类型
	PrepayID   string `xml:"prepay_id"`    //预支付交易会话标识
	MwebURL    string `xml:"mweb_url"`     //支付跳转链接

}

// 沙箱请求体
type ShaBoxSignReq struct {
	MchID    string `xml:"mch_id"`    //商户号
	NonceStr string `xml:"nonce_str"` //随机字符串
	Sign     string `xml:"sign"`      //签名
}

//沙箱signKey返回体

type ShaBoxSignResp struct {
	ReturnCode     string `xml:"return_code"`
	ReturnMsg      string `xml:"return_msg"`
	SandboxSignkey string `xml:"sandbox_signkey"`
}

// nuiApp调起支付参数
type UniAppResp struct {
	OrderInfo string `json:"orderInfo"`
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
	OrderCode string `json:"order_code"` //内部订单号
}

type WxNotifyReq struct {
	Id           string    `json:"id"`
	CreateTime   time.Time `json:"create_time"`
	ResourceType string    `json:"resource_type"`
	EventType    string    `json:"event_type"`
	Summary      string    `json:"summary"`
	Resource     struct {
		OriginalType   string `json:"original_type"`
		Algorithm      string `json:"algorithm"`
		Ciphertext     string `json:"ciphertext"`
		AssociatedData string `json:"associated_data"`
		Nonce          string `json:"nonce"`
	} `json:"resource"`
}

type WeChatCommPay struct {
	Config WechatPayConfig
	Ctx    context.Context
}

func NewWeChatCommPay(config WechatPayConfig) *WeChatCommPay {
	return &WeChatCommPay{
		Config: config,
		Ctx:    context.Background(),
	}
}

func WxPayCalcSign(mReq map[string]interface{}, key string) (sign string) {
	//STEP 1, 对key进行升序排序.
	sorted_keys := make([]string, 0)
	for k, _ := range mReq {
		sorted_keys = append(sorted_keys, k)
	}
	sort.Strings(sorted_keys)

	//STEP2, 对key=value的键值对用&连接起来，略过空值
	var signStrings string
	for _, k := range sorted_keys {
		value := fmt.Sprintf("%v", mReq[k])
		if value != "" {
			signStrings = signStrings + k + "=" + value + "&"
		}
	}

	//STEP3, 在键值对的最后加上key=API_KEY
	if key != "" {
		signStrings = signStrings + "key=" + key
	}

	//STEP4, 进行MD5签名并且将所有字符转为大写.
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(signStrings)) //
	cipherStr := md5Ctx.Sum(nil)
	upperSign := strings.ToUpper(hex.EncodeToString(cipherStr))

	return upperSign
}

/*
*	getRandStr 获取随机字符串
*	param	n	位数
*	reply	随机字符串
 */
func getRandStr(n int) string {
	leterset := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var leteridxbits uint = 6
	var mask int64 = 1<<leteridxbits - 1

	rnd := rand.NewSource(time.Now().UnixNano())
	res := make([]byte, 0, n)

	for i, bits := 0, rnd.Int63(); i < n; i++ {
		if bits == 0 {
			bits = rnd.Int63()
		}
		idx := int(bits & mask)
		if idx < len(leterset) {
			res = append(res, leterset[idx])
		} else {
			i--
		}
		bits >>= leteridxbits
	}
	return string(res[:n])
}

// 获取V3Client
func (l *WeChatCommPay) getClient() (uniAppResp *core.Client, err error) {
	ctx := context.Background()
	// 使用商户私钥等初始化 client，并使它具有自动定时获取微信支付平台证书的能力
	mchPrivateKey, err := utils.LoadPrivateKeyWithPath(l.Config.PrivateKeyPath)
	if err != nil {
		weChatHttpRequestErr.CounterInc()
		logx.Errorf("请求微信支付发生错误,err =%v", err)
		return nil, err
	}
	opts := []core.ClientOption{
		option.WithWechatPayAutoAuthCipher(l.Config.MchId, l.Config.SerialNumber, mchPrivateKey, l.Config.ApiKey),
	}
	client, err := core.NewClient(ctx, opts...)
	if err != nil {
		weChatHttpRequestErr.CounterInc()
		logx.Errorf("请求微信支付发生错误,err =%v", err)
		return nil, err
	}
	return client, nil
}

// 支付请求v3  web
func (l *WeChatCommPay) WechatPayV3Native(info *PayOrder) (resp *native.PrepayResponse, err error) {
	attach := fmt.Sprintf(`{"order_sn":"%s","value":%d}`, info.OrderSn, info.Amount)
	client, err := l.getClient()
	if err != nil {
		logx.Errorf("请求微信支付发生错误,err =%v", err)
		return
	}

	body := info.Subject
	svc := native.NativeApiService{Client: client}
	resp, result, err := svc.Prepay(l.Ctx,
		native.PrepayRequest{
			Appid:       core.String(l.Config.AppId),
			Mchid:       core.String(l.Config.MchId),
			Description: core.String(body),
			OutTradeNo:  core.String(info.OrderSn),
			Attach:      core.String(attach),
			NotifyUrl:   core.String(l.Config.NotifyUrl),
			Amount: &native.Amount{
				Total: core.Int64(int64(info.Amount)),
			},
		},
	)
	if err != nil {
		weChatHttpRequestErr.CounterInc()
		logx.Errorf("请求微信支付发生错误,err =%v", err)
		return
	}
	logx.Infof("请求微信支付成功！result = %v", result)
	return
}

const (
	WapUrl  = "https://kuaikanju-h5.yunjuhudong.com"
	WapName = "快看剧"
)

// 支付请求  统一下单
func (l *WeChatCommPay) WechatPayUnified(info *PayOrder, appConfig *WechatPayConfig) (resp *WXOrderReply, err error) {
	requireUri := WeChatRequestUri
	attchByte, _ := json.Marshal(info)
	attach := string(attchByte)
	sceneInfo := `{
	"h5_info": {
		"type": "Wap",
		"wap_url": "%s",
		"wap_name": "%s"
	}
}`
	if appConfig.WapUrl != "" && appConfig.WapName != "" {
		sceneInfo = fmt.Sprintf(sceneInfo, appConfig.WapUrl, appConfig.WapName)
	} else {
		sceneInfo = fmt.Sprintf(sceneInfo, WapUrl, WapName)
	}

	NonceStr := getRandStr(32)
	params := &WXOrderParam{
		APPID:          l.Config.AppId,
		MchID:          l.Config.MchId,
		NonceStr:       NonceStr,
		TradeType:      WechatTradeType,
		Body:           info.Subject,
		OutTradeNo:     info.OrderSn,
		TotalFee:       info.Amount,
		SpbillCreateIP: info.IP,
		NotifyUrl:      l.Config.NotifyUrl,
		SceneInfo:      sceneInfo,
		Attach:         attach,
	}
	var m map[string]interface{}
	m = make(map[string]interface{}, 12)
	m["appid"] = params.APPID
	m["mch_id"] = params.MchID
	m["nonce_str"] = params.NonceStr
	m["trade_type"] = params.TradeType
	m["body"] = params.Body
	m["out_trade_no"] = params.OutTradeNo
	m["total_fee"] = params.TotalFee
	m["spbill_create_ip"] = params.SpbillCreateIP
	m["notify_url"] = params.NotifyUrl
	m["scene_info"] = params.SceneInfo
	m["attach"] = params.Attach
	params.Sign = WxPayCalcSign(m, l.Config.ApiKeyV2)
	//开启沙箱测试
	//	shaBoxSign := WxPayCalcSign(map[string]interface{}{
	//		"mch_id":    params.MchID,
	//		"nonce_str": params.NonceStr,
	//	}, l.Config.ApiKey)
	//	shaBoxReq := fmt.Sprintf(`<xml><mch_id>%s</mch_id><nonce_str>%s</nonce_str><sign>%s</sign></xml>`, params.MchID, params.NonceStr, shaBoxSign)
	//	shaBoxBody, _ := XmlHttpPost(SandboxUriSign, shaBoxReq)
	//	var shaBoxSignResp ShaBoxSignResp
	//	xml.Unmarshal(shaBoxBody, &shaBoxSignResp)
	//	requireUri = WechatSandboxUri
	//	params.Sign = shaBoxSignResp.SandboxSignkey

	bytesReq, err := xml.Marshal(params)
	if err != nil {
		logx.Errorf("以xml形式编码发送错误,原因:%v", err)
		return
	}
	strReq := string(bytesReq)
	strReq = strings.Replace(strReq, "WXOrderParam", "xml", -1)
	resBody, err := XmlHttpPost(requireUri, strReq)
	if err != nil {
		weChatHttpRequestErr.CounterInc()
		return nil, err
	}
	var wechatReply WXOrderReply
	xmlErr := xml.Unmarshal(resBody, &wechatReply)
	if xmlErr != nil {
		logx.Errorf("wechatReply xmlErr,原因:%v", err)
		return nil, xmlErr
	}
	if wechatReply.ResultCode == "FAIL" {
		weChatReturnPayErr.CounterInc()
		logx.Errorf("发起支付错误,原因:%s", wechatReply.ReturnMsg)
		return nil, nil
	}
	return &wechatReply, nil
}

// 微信xmlHttp请求
func XmlHttpPost(uri string, params string) ([]byte, error) {
	logx.Infof("微信支付请求,地址：%s 参数:%s", uri, params)
	req, err := http.NewRequest("POST", uri, strings.NewReader(params))
	if err != nil {
		logx.Errorf("http.NewRequest错误,原因:%v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml;charset=utf-8")
	c := http.Client{}
	httpResp, _err := c.Do(req)
	if _err != nil {
		logx.Errorf("http请求错误错误,原因:%v", err)
		return nil, _err

	}
	defer httpResp.Body.Close()
	body, bodyErr := ioutil.ReadAll(httpResp.Body)
	if bodyErr != nil {
		logx.Errorf("ReaddBody Error,原因:%v", err)
		return nil, err
	}
	logx.Infof("微信支付请求,返回内容：%s", string(body))
	return body, nil
}

// 支付请求v3  h5
func (l *WeChatCommPay) WechatPayV3H5(info *PayOrder) (resp *h5.PrepayResponse, err error) {
	attach := fmt.Sprintf(`{"order_sn":"%s","value":%d}`, info.OrderSn, info.Amount)
	client, err := l.getClient()
	if err != nil {
		logx.Errorf("请求微信支付发生错误,err =%v", err)
		return
	}
	body := info.Subject
	svc := h5.H5ApiService{Client: client}
	total := int64(info.Amount)
	amount := &h5.Amount{
		Total: &total,
	}

	h5InfoType := "h5"
	request := h5.PrepayRequest{
		Appid:       core.String(l.Config.AppId),
		Mchid:       core.String(l.Config.MchId),
		Description: core.String(body),
		OutTradeNo:  core.String(info.OrderSn),
		Attach:      core.String(attach),
		NotifyUrl:   core.String(fmt.Sprintf("%s/%s", l.Config.NotifyUrl, l.Config.AppId)),
		Amount:      amount,
		SceneInfo: &h5.SceneInfo{
			PayerClientIp: &info.IP,
			H5Info: &h5.H5Info{
				Type: &h5InfoType,
			},
		},
	}
	resp, result, err := svc.Prepay(l.Ctx, request)
	if err != nil {
		weChatHttpRequestErr.CounterInc()
		logx.Errorf("请求微信支付发生错误,err =%v", err)
		return
	}
	logx.Infof("请求微信支付成功！result = %v, request:%+v", result, request)
	return
}

// 发起微信支付请求V3请求  jsapi
func (l *WeChatCommPay) WechatPayV3(info *PayOrder, openId string) (uniAppResp *UniAppResp, err error) {
	attach := fmt.Sprintf(`{"order_sn":"%s","value":%d}`, info.OrderSn, info.Amount)
	client, err := l.getClient()
	if err != nil {
		logx.Errorf("请求微信支付发生错误,err =%v", err)
		return nil, err
	}
	body := info.Subject
	svc := jsapi.JsapiApiService{Client: client}
	// 得到prepay_id，以及调起支付所需的参数和签名
	resp, result, err := svc.PrepayWithRequestPayment(l.Ctx,
		jsapi.PrepayRequest{
			Appid:       core.String(l.Config.AppId),
			Mchid:       core.String(l.Config.MchId),
			Description: core.String(body),
			OutTradeNo:  core.String(info.OrderSn),
			Attach:      core.String(attach),
			NotifyUrl:   core.String(l.Config.NotifyUrl),
			Amount: &jsapi.Amount{
				Total: core.Int64(int64(info.Amount)),
			},
			Payer: &jsapi.Payer{
				Openid: core.String(openId),
			},
		},
	)
	if err != nil {
		weChatHttpRequestErr.CounterInc()
		logx.Errorf("请求微信支付发生错误,err =%v", err)
		return nil, err
	}
	logx.Infof("请求微信支付成功！result = %v", result)
	payResult := &UniAppResp{
		TimeStamp: *resp.TimeStamp,
		NonceStr:  *resp.NonceStr,
		Package:   "prepay_id=" + *resp.PrepayId,
		SignType:  *resp.SignType,
		PaySign:   *resp.PaySign,
		OrderCode: info.OrderSn,
	}
	return payResult, nil
}

// 查询支付状态
func (l *WeChatCommPay) GetOrderStatus(codeCode string) (orderInfo *payments.Transaction, err error) {
	client, err := l.getClient()
	if err != nil {
		logx.Errorf("请求微信支付发生错误,err =%v", err)
		return nil, err
	}
	svc := jsapi.JsapiApiService{Client: client}

	resp, result, err := svc.QueryOrderByOutTradeNo(l.Ctx,
		jsapi.QueryOrderByOutTradeNoRequest{
			OutTradeNo: core.String(codeCode),
			Mchid:      core.String(l.Config.MchId),
		},
	)
	if err != nil {
		weChatHttpRequestErr.CounterInc()
		logx.Errorf("请求微信查询订单发生错误,err =%v", err)
		return nil, err
	}
	logx.Infof("请求微信支付成功！resp = %v,result=%v", resp, result)
	return resp, nil
}

// 通知权限验证。及解析内容
func (l *WeChatCommPay) Notify(r *http.Request) (orderInfo *payments.Transaction, data map[string]interface{}, err error) {
	//获取私钥
	mchPrivateKey, err := utils.LoadPrivateKeyWithPath(l.Config.PrivateKeyPath)
	if err != nil {
		weChatNotifyErr.CounterInc()
		logx.Errorf("获取私钥发生错误！err=%v", err)
		err = errors.New(`{"code": "FAIL","message": "获取入私钥发生错误"}`)
		return nil, nil, err
	}
	// 1. 使用 `RegisterDownloaderWithPrivateKey` 注册下载器
	err = downloader.MgrInstance().RegisterDownloaderWithPrivateKey(l.Ctx, mchPrivateKey, l.Config.SerialNumber, l.Config.MchId, l.Config.ApiKey)
	if err != nil {
		weChatNotifyErr.CounterInc()
		logx.Errorf("下载解密器失败！err=%v", err)
		err = errors.New(`{"code": "FAIL","message": "下载解密器失败"}`)
		return nil, nil, err
	}
	// 2. 获取商户号对应的微信支付平台证书访问器
	certificateVisitor := downloader.MgrInstance().GetCertificateVisitor(l.Config.MchId)
	// 3. 使用证书访问器初始化 `notify.Handler`
	handler := notify.NewNotifyHandler(l.Config.ApiKey, verifiers.NewSHA256WithRSAVerifier(certificateVisitor))
	//支付回调
	transaction := new(payments.Transaction)
	notifyReq, err := handler.ParseNotifyRequest(l.Ctx, r, transaction)
	// 如果验签未通过，或者解密失败
	if err != nil {
		weChatNotifyErr.CounterInc()
		err = fmt.Errorf("验签未通过，或者解密失败！err=%v, r:%+v, config:%+v", err, r, l.Config)
		logx.Error(err.Error())
		//err = errors.New(`{"code": "FAIL","message": "验签未通过，或者解密失败"}`)
		return nil, nil, err
	}
	// 处理通知内容
	logx.Slowf("Wechat notifyReq=%v", notifyReq.Summary)
	logx.Slowf("Wechat content=%v", transaction)
	return transaction, nil, nil
}

// 退款解密内容
type RefundOrderReply struct {
	TransactionId string `json:"transaction_id,omitempty"`
}

// 退款支付回调
func (l *WeChatCommPay) RefundNotify(r *http.Request) (orderInfo map[string]interface{}, err error) {

	mchPrivateKey, err := utils.LoadPrivateKeyWithPath(l.Config.PrivateKeyPath)
	if err != nil {
		logx.Errorf("mchPrivateKey！err=%v", err)
	}
	// 1. 使用 `RegisterDownloaderWithPrivateKey` 注册下载器
	err = downloader.MgrInstance().RegisterDownloaderWithPrivateKey(l.Ctx, mchPrivateKey, l.Config.SerialNumber, l.Config.MchId, l.Config.ApiKey)
	if err != nil {
		weChatNotifyErr.CounterInc()
		logx.Errorf("注册下载器失败！err=%v", err)
		err = errors.New(`{"code": "FAIL","message": "注册下载器"}`)
		return nil, err
	}
	// 2. 获取商户号对应的微信支付平台证书访问器
	certificateVisitor := downloader.MgrInstance().GetCertificateVisitor(l.Config.MchId)
	// 3. 使用证书访问器初始化 `notify.Handler`
	handler := notify.NewNotifyHandler(l.Config.ApiKey, verifiers.NewSHA256WithRSAVerifier(certificateVisitor))
	//支付回调
	content := make(map[string]interface{})
	notifyReq, err := handler.ParseNotifyRequest(l.Ctx, r, &content)
	// 如果验签未通过，或者解密失败
	if err != nil {
		weChatNotifyErr.CounterInc()
		err = fmt.Errorf("验签未通过，或者解密失败！err=%w", err)
		logx.Error(err.Error())
		//err = errors.New(`{"code": "FAIL","message": "验签未通过，或者解密失败"}`)
		return nil, err
	}
	jsonData, _ := json.Marshal(content)
	logx.Slowf("Wechat 解密后内容=%s", string(jsonData))
	// 处理通知内容
	logx.Slowf("Wechat notifyReq=%v", notifyReq.Summary)
	logx.Slowf("Wechat content=%v", content)
	return content, nil
}

// 关闭订单
type CloserReq struct {
	Mchid string `json:"mchid"`
}

func (l *WeChatCommPay) CloseOrder(orderCode string) error {
	client, err := l.getClient()
	if err != nil {
		logx.Errorf("关闭订单发生错误,err =%v", err)
		return err
	}
	body := CloserReq{
		Mchid: l.Config.MchId,
	}
	uri := fmt.Sprintf("https://api.mch.weixin.qq.com/v3/pay/transactions/out-trade-no/%s/close", orderCode)
	result, err := client.Post(l.Ctx, uri, body)
	if err != nil {
		return err
	}
	logx.Slowf("关闭订单返回信息状态:，%d", result.Response.StatusCode)
	return nil
}

const refundReason = "用户退款"

// 订单退款
func (l *WeChatCommPay) RefundOrder(refundOrder *RefundOrder) (*refunddomestic.Refund, error) {
	client, err := l.getClient()
	if err != nil {
		weChatRefundOrderErr.CounterInc()
		logx.Errorf("退款发生错误,err =%v", err)
		return nil, err
	}
	params, _ := url.Parse(l.Config.NotifyUrl)
	notifyUri := fmt.Sprintf("%s://%s/notify/refund/wechat/%s", params.Scheme, params.Host, refundOrder.OutTradeNo)
	svc := refunddomestic.RefundsApiService{Client: client}
	resp, result, err := svc.Create(l.Ctx,
		refunddomestic.CreateRequest{
			OutTradeNo:    core.String(refundOrder.OutTradeNo),
			OutRefundNo:   core.String(refundOrder.OutRefundNo),
			TransactionId: core.String(refundOrder.TransactionId),
			Reason:        core.String(refundReason),
			NotifyUrl:     core.String(notifyUri),
			FundsAccount:  refunddomestic.REQFUNDSACCOUNT_AVAILABLE.Ptr(),
			Amount: &refunddomestic.AmountReq{
				Currency: core.String("CNY"),
				Refund:   core.Int64(refundOrder.RefundFee),
				Total:    core.Int64(refundOrder.TotalFee),
			},
		},
	)
	if err != nil {
		// 处理错误
		weChatRefundOrderErr.CounterInc()
		logx.Errorf("退款 call Create err:%s", err)
		return nil, err
	} else {
		// 处理返回结果
		logx.Infof("退款 status=%d resp=%s", result.Response.StatusCode, resp)
	}
	return resp, nil
}
