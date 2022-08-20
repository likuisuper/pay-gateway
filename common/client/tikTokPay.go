package client

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	jsoniter "github.com/json-iterator/go"
	"github.com/zeromicro/go-zero/core/logx"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	tikTokHttpRequestErr    = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "tikTokHttpRequestErr", nil, "tikTok请求错误", nil})}
	tikTokNotifyErr         = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "tikTokNotifyErr", nil, "tikTok回调错误", nil})}
	tikTokGetOrderStatusErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "tikTokGetOrderStatusErr", nil, "tikTok回调错误", nil})}
)

//请求地址
const (
	tikTokCreateUri = "https://developer.toutiao.com/api/apps/ecpay/v1/create_order"
	tikTokQueryUri  = "https://developer.toutiao.com/api/apps/ecpay/v1/query_order"
	tikTokBody      = "充值VIP"
	tikTokValidTime = 300
)

//字节支付配置
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

//请求响应
type TikTokReply struct {
	TikTokNotifyResp
	Data TikTokReplyData `json:"data"`
}

type TikTokReplyData struct {
	OrderId    string `json:"order_id"`
	OrderToken string `json:"order_token"`
	OrderCode  string `json:"order_code"`
}

//回调回复
type TikTokNotifyResp struct {
	ErrNO   int    `json:"err_no"`
	ErrTips string `json:"err_tips"`
}

//创建支付订单
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

//获取签名
func (t *TikTokPay) getSign(paramsMap map[string]interface{}) string {
	var paramsArr []string
	//加入token
	paramsMap["token"] = t.Config.Token
	for k, v := range paramsMap {
		if k == "other_settle_params" {
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
		switch k {
		case "app_id", "thirdparty_id", "sign":
		default:
			paramsArr = append(paramsArr, value)
		}
	}
	paramsArr = append(paramsArr, t.Config.SALT)
	sort.Strings(paramsArr)
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(paramsArr, "&"))))
}

//以下回调相关

//支付回调msg解析结构体
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

type ByteDanceReq struct {
	Timestamp    string `json:"timestamp,optional"`
	Nonce        string `json:"nonce,optional"`
	Msg          string `json:"msg,optional"`
	Type         string `json:"type,optional"`
	MsgSignature string `json:"msg_signature,optional"`
}

//回调验证返回
func (t *TikTokPay) Notify(req *ByteDanceReq) (orderInfo *TikTokNotifyMsgData, err error) {
	//签名核对
	timestamp, _ := strconv.Atoi(req.Timestamp)
	notifySing := t.notifySign(timestamp, req.Nonce, req.Msg)
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

//获取验签
func (t *TikTokPay) notifySign(timestamp int, nonce, msg string) string {

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

//结果信息返回解析结构体
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

//查询订单结果
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
