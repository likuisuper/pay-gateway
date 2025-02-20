package thirdApis

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/zeromicro/go-zero/core/logx"
	"time"
)

//https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/virtual-payment.html#_2-3-%E6%9C%8D%E5%8A%A1%E5%99%A8API
type wechatXPayApi struct{}

var WechatXPayApi = new(wechatXPayApi)

func (*wechatXPayApi) createPaySig(appKey, data string) string {
	mac := hmac.New(sha256.New, []byte(appKey))
	_, _ = mac.Write([]byte(data))

	return hex.EncodeToString(mac.Sum(nil))
}

//虚拟交易- 申请退款
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

//虚拟交易 查询创建的订单（现金单，非代币单）
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
