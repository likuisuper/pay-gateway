package client

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/xml"
	"errors"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
	"github.com/zeromicro/go-zero/core/logx"
	"io/ioutil"
	"math/rand"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"
)

var (
	weChatHttpRequestErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "weChatHttpRequestErr", nil, "weChat请求错误", nil})}
	weChatNotifyErr      = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "weChatNotifyErr", nil, "weChat回调通知错误", nil})}
)

const (
	WeChatRequestUri = "https://api.mch.weixin.qq.com/pay/unifiedorder"
	WechatTradeType  = "MWEB"
	WechatSignType   = "MD5"
)

//微信支付参数
type WechatPayConfig struct {
	AppId          string //应用ID
	MchId          string //直连商户号
	ApiKey         string //apiV3密钥
	PrivateKeyPath string //apiV3密钥
	SerialNumber   string //商户证书序列号
	NotifyUrl      string //通知地址
}

//WXOrderParam	微信请求参数
type WXOrderParam struct {
	APPID          string `xml:"appid"`     //公众账号ID
	MchID          string `xml:"mch_id"`    //商户号
	NonceStr       string `xml:"nonce_str"` //随机字符串
	SignType       string `xml:"sign_type"`
	Sign           string `xml:"sign"`             //签名
	Body           string `xml:"body"`             //商品描述
	OutTradeNo     string `xml:"out_trade_no"`     //商户订单号
	TotalFee       string `xml:"total_fee"`        //总金额
	SpbillCreateIP string `xml:"spbill_create_ip"` //终端IP
	NotifyUrl      string `xml:"notify_url"`       //通知地址
	TradeType      string `xml:"trade_type"`       //交易类型
	SceneInfo      string `xml:"scene_info"`       //场景信息
}

//WXOrderReply	微信请求返回结果
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

//nuiApp调起支付参数
type UniAppResp struct {
	OrderInfo string `json:"orderInfo"`
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
	OrderCode string `json:"order_code"` //内部订单号
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

/*
*	WXmd5Sign 微信 md5 签名
*	param  data		interface{}
*	reply	sign	生成的签名
 */
func (w *WeChatCommPay) WXmd5Sign(data interface{}) (sign string) {
	val := make(map[string]string)
	datavalue := reflect.ValueOf(data)
	if datavalue.Kind() != reflect.Struct {
		return ""
	}
	var keys []string
	for i := 0; i < datavalue.NumField(); i++ {
		k := datavalue.Type().Field(i)
		kl := k.Tag.Get("xml")
		v := fmt.Sprintf("%v", datavalue.Field(i))
		if v != "" && v != "0" && kl != "sign" {
			val[kl] = v
			keys = append(keys, kl)
		}
	}
	sort.Strings(keys)
	var stra string
	for _, v := range keys {
		stra = stra + v + "=" + val[v] + "&"
	}
	strb := stra + "key=" + w.Config.ApiKey
	hstr := md5.Sum([]byte(strb))
	sum := fmt.Sprintf("%x", hstr)
	sign = strings.ToUpper(sum)
	return sign
}

/*
*	submitWXOrder	提交微信订单
*	param	data	WXOrderParam
*	reply	prepay_id	预支付交易会话标识
*	reply	mweb_url	支付跳转链接
 */
func submitWXOrder(data WXOrderParam) (res *WXOrderReply, err error) {

	xdata, err := xml.Marshal(data)
	if err != nil {
	}
	xmldata := strings.Replace(string(xdata), "WXOrderParam", "xml", -1)
	body := bytes.NewBufferString(xmldata)
	resp, err := http.Post(WeChatRequestUri, "content-type:text/xml; charset=utf-8", body)
	if err != nil {
		logx.Errorf("发起统一订单支付失败，err:=%v", err)
	}
	defer resp.Body.Close()

	result, _ := ioutil.ReadAll(resp.Body)
	var reply WXOrderReply
	err = xml.Unmarshal(result, &reply)
	if err != nil {
		logx.Errorf("发起统一订单支付失败，err:=%v", err)
		return nil, err
	}
	if reply.ReturnCode == "SUCCESS" && reply.ResultCode == "SUCCESS" {
		return &reply, nil
	}
	return nil, errors.New(reply.ReturnMsg)
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

//获取V3Client
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

//发起微信支付请求V3请求
func (l *WeChatCommPay) WechatPayV3(info *model.PmPayOrderTable, openId string) (uniAppResp *UniAppResp, err error) {
	attach := fmt.Sprintf(`{"order_sn":"%s","type":%d,"value":%d}`, info.OrderSn, info.Amount)
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

//查询支付状态
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
	logx.Slowf("请求微信支付成功！resp = %v,result=%v", resp, result)
	return resp, nil
}

//通知权限验证。及解析内容
func (l *WeChatCommPay) Notify(r *http.Request) (orderInfo *payments.Transaction, err error) {

	//获取私钥
	mchPrivateKey, err := utils.LoadPrivateKeyWithPath(l.Config.PrivateKeyPath)
	if err != nil {
		weChatNotifyErr.CounterInc()
		logx.Errorf("获取私钥发生错误！err=%v", err)
		err = errors.New(`{"code": "FAIL","message": "获取入私钥发生错误"}`)
		return nil, err
	}
	// 1. 使用 `RegisterDownloaderWithPrivateKey` 注册下载器
	err = downloader.MgrInstance().RegisterDownloaderWithPrivateKey(l.Ctx, mchPrivateKey, l.Config.SerialNumber, l.Config.MchId, l.Config.ApiKey)
	if err != nil {
		weChatNotifyErr.CounterInc()
		logx.Errorf("下载解密器失败！err=%v", err)
		err = errors.New(`{"code": "FAIL","message": "下载解密器失败"}`)
		return nil, err
	}
	// 2. 获取商户号对应的微信支付平台证书访问器
	certificateVisitor := downloader.MgrInstance().GetCertificateVisitor(l.Config.MchId)
	// 3. 使用证书访问器初始化 `notify.Handler`
	handler := notify.NewNotifyHandler(l.Config.ApiKey, verifiers.NewSHA256WithRSAVerifier(certificateVisitor))
	transaction := new(payments.Transaction)
	notifyReq, err := handler.ParseNotifyRequest(l.Ctx, r, transaction)
	// 如果验签未通过，或者解密失败
	if err != nil {
		weChatNotifyErr.CounterInc()
		logx.Errorf("验签未通过，或者解密失败！err=%v", err)
		err = errors.New(`{"code": "FAIL","message": "验签未通过，或者解密失败"}`)
		return nil, err
	}
	// 处理通知内容
	logx.Slowf("Wechat notifyReq=%v", notifyReq.Summary)
	logx.Slowf("Wechat content=%v", transaction)

	return transaction, nil
}
