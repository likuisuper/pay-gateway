package thirdApis

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/util"
)

// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/virtual-payment.html#_2-3-%E6%9C%8D%E5%8A%A1%E5%99%A8API
type wechatXPayApi struct{}

var WechatXPayApi = new(wechatXPayApi)

// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/virtual-payment.html#_2-4-%E7%AD%BE%E5%90%8D%E8%AF%A6%E8%A7%A3
// appKey 可通过小程序MP查看：虚拟支付 -> 基本配置 -> 基础配置中的沙箱AppKey和现网AppKey。注意：记得根据env值选择不同AppKey，env = 0对应现网AppKey，env = 1对应沙箱AppKey
func (*wechatXPayApi) createPaySig(appKey, data string) string {
	mac := hmac.New(sha256.New, []byte(appKey))
	_, _ = mac.Write([]byte(data))

	return hex.EncodeToString(mac.Sum(nil))
}

// 虚拟交易- 申请退款
func (this *wechatXPayApi) RefundOrder(param *XPayRefundOrderParam, appKey, token string) (*XPayRefundOrderDTO, error) {
	apiPath := "/xpay/refund_order"

	jsonByte, _ := json.Marshal(param)
	jsonStr := string(jsonByte)

	paySig := this.createPaySig(appKey, apiPath+"&"+jsonStr)

	uri := fmt.Sprintf("%s%s?access_token=%s&pay_sig=%s",
		WechatXPayHost, apiPath, token, paySig,
	)
	respStr, err := util.HttpPost(uri, param, 5*time.Second)

	var ret XPayRefundOrderDTO
	if err != nil {
		logx.WithContext(context.Background()).
			Error("wechatApi.XPayDownloadBill err:%v, params:%+v, respStr:%+v", err, param, respStr)
		return nil, err
	}
	json.Unmarshal([]byte(respStr), &ret)

	return &ret, err

}

// 虚拟交易 查询创建的订单（现金单，非代币单）
func (this *wechatXPayApi) QueryOrder(param *XPayQueryOrderParam, appKey, token string) (*XPayQueryOrderDTO, error) {
	apiPath := "/xpay/query_order"

	jsonByte, _ := json.Marshal(param)
	jsonStr := string(jsonByte)

	paySig := this.createPaySig(appKey, apiPath+"&"+jsonStr)

	uri := fmt.Sprintf("%s%s?access_token=%s&pay_sig=%s",
		WechatXPayHost, apiPath, token, paySig,
	)
	respStr, err := util.HttpPost(uri, param, 5*time.Second)

	var ret XPayQueryOrderDTO
	if err != nil {
		logx.WithContext(context.Background()).
			Error("wechatApi.XPayDownloadBill err:%v, params:%+v, respStr:%+v", err, param, respStr)
		return nil, err
	}
	json.Unmarshal([]byte(respStr), &ret)

	return &ret, err

}
