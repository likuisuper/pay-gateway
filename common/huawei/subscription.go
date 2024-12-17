package huawei

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
)

// https://developer.huawei.com/consumer/cn/doc/HMSCore-References/api-common-statement-0000001050986127
// Subscription
// 中国站点：https://subscr-drcn.iap.cloud.huawei.com.cn
// 德国站点：https://subscr-dre.iap.cloud.huawei.eu
// 新加坡站点：https://subscr-dra.iap.cloud.huawei.asia
// 俄罗斯站点：https://subscr-drru.iap.cloud.huawei.ru

type SubscriptionClient struct {
}

var SubscriptionDemo = &SubscriptionClient{}

// 获取订阅url
const sub_req_url = "https://subscr-drcn.iap.cloud.huawei.com.cn"

type HwCommonResponse struct {
	ResponseCode       string `json:"responseCode"`       // 返回码。0：成功。 其他：失败，具体请参见错误码。
	ResponseMessage    string `json:"responseMessage"`    // 响应描述
	InappPurchaseData  string `json:"inappPurchaseData"`  // 包含购买详情的字符串（JSONString格式），格式请参见InappPurchaseDetails。
	DataSignature      string `json:"dataSignature"`      // inappPurchaseData基于应用RSA IAP私钥的签名信息，签名算法为signatureAlgorithm。应用请参见对返回结果验签使用IAP公钥对inappPurchaseData的JSON字符串进行验签。
	SignatureAlgorithm string `json:"signatureAlgorithm"` // 签名算法。默认为：SHA256WithRSA/PSS
}

// https://developer.huawei.com/consumer/cn/doc/HMSCore-References/json-inapppurchasedata-0000001050986125
type InAppPurchaseData struct {
}

// https://developer.huawei.com/consumer/cn/doc/HMSCore-References/api-subscription-verify-purchase-token-0000001050706080
//
// 本接口只针对订阅型商品
//
// subscriptionId 订阅ID
//
// purchaseToken 商品的购买Token，发起购买和查询订阅信息均会返回
func (subscriptionDemo *SubscriptionClient) GetSubscription(authHeaderString, subscriptionId, purchaseToken string) (*HwCommonResponse, error) {
	bodyMap := map[string]string{
		"subscriptionId": subscriptionId,
		"purchaseToken":  purchaseToken,
	}
	url := sub_req_url + "/sub/applications/v2/purchases/get"
	respStr, err := SendRequest(authHeaderString, url, bodyMap)
	if err != nil {
		return nil, err
	}

	var hwResp HwCommonResponse
	err = json.Unmarshal([]byte(respStr), &hwResp)
	if err != nil {
		logx.Errorf("json.Unmarshal error: %v, raw data:%s", err, respStr)
		return nil, err
	}

	if hwResp.ResponseCode != "0" {
		// 异常
		return nil, errors.New(hwResp.ResponseMessage)
	}

	return &hwResp, nil
}

func (subscriptionDemo *SubscriptionClient) StopSubscription(authHeaderString, subscriptionId, purchaseToken string) (string, error) {
	bodyMap := map[string]string{
		"subscriptionId": subscriptionId,
		"purchaseToken":  purchaseToken,
	}
	url := sub_req_url + "/sub/applications/v2/purchases/stop"
	return SendRequest(authHeaderString, url, bodyMap)
}

func (subscriptionDemo *SubscriptionClient) DelaySubscription(authHeaderString, subscriptionId, purchaseToken string, currentExpirationTime, desiredExpirationTime int64) (string, error) {
	bodyMap := map[string]string{
		"subscriptionId":        subscriptionId,
		"purchaseToken":         purchaseToken,
		"currentExpirationTime": fmt.Sprintf("%v", currentExpirationTime),
		"desiredExpirationTime": fmt.Sprintf("%v", desiredExpirationTime),
	}
	url := sub_req_url + "/sub/applications/v2/purchases/delay"
	return SendRequest(authHeaderString, url, bodyMap)
}

func (subscriptionDemo *SubscriptionClient) ReturnFeeSubscription(authHeaderString, subscriptionId, purchaseToken string) (string, error) {
	bodyMap := map[string]string{
		"subscriptionId": subscriptionId,
		"purchaseToken":  purchaseToken,
	}

	url := sub_req_url + "/sub/applications/v2/purchases/returnFee"
	return SendRequest(authHeaderString, url, bodyMap)
}

func (subscriptionDemo *SubscriptionClient) WithdrawalSubscription(authHeaderString, subscriptionId, purchaseToken string) (string, error) {
	bodyMap := map[string]string{
		"subscriptionId": subscriptionId,
		"purchaseToken":  purchaseToken,
	}
	url := sub_req_url + "/sub/applications/v2/purchases/withdrawal"
	return SendRequest(authHeaderString, url, bodyMap)
}
