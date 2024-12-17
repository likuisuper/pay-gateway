package huawei

import (
	"fmt"
	"log"
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
func getSubUrl(accountFlag int) string {
	return "https://subscr-drcn.iap.cloud.huawei.com.cn"
}

func (subscriptionDemo *SubscriptionClient) GetSubscription(authHeaderString, subscriptionId, purchaseToken string, accountFlag int) {
	bodyMap := map[string]string{
		"subscriptionId": subscriptionId,
		"purchaseToken":  purchaseToken,
	}
	url := getSubUrl(accountFlag) + "/sub/applications/v2/purchases/get"
	bodyBytes, err := SendRequest(authHeaderString, url, bodyMap)
	if err != nil {
		log.Printf("err is %s", err)
	}
	// TODO: display the response as string in console, you can replace it with your business logic.
	log.Printf("%s", bodyBytes)
}

func (subscriptionDemo *SubscriptionClient) StopSubscription(authHeaderString, subscriptionId, purchaseToken string, accountFlag int) {
	bodyMap := map[string]string{
		"subscriptionId": subscriptionId,
		"purchaseToken":  purchaseToken,
	}
	url := getSubUrl(accountFlag) + "/sub/applications/v2/purchases/stop"
	bodyBytes, err := SendRequest(authHeaderString, url, bodyMap)
	if err != nil {
		log.Printf("err is %s", err)
	}
	// TODO: display the response as string in console, you can replace it with your business logic.
	log.Printf("%s", bodyBytes)
}

func (subscriptionDemo *SubscriptionClient) DelaySubscription(authHeaderString, subscriptionId, purchaseToken string, currentExpirationTime, desiredExpirationTime int64, accountFlag int) {
	bodyMap := map[string]string{
		"subscriptionId":        subscriptionId,
		"purchaseToken":         purchaseToken,
		"currentExpirationTime": fmt.Sprintf("%v", currentExpirationTime),
		"desiredExpirationTime": fmt.Sprintf("%v", desiredExpirationTime),
	}
	url := getSubUrl(accountFlag) + "/sub/applications/v2/purchases/delay"
	bodyBytes, err := SendRequest(authHeaderString, url, bodyMap)
	if err != nil {
		log.Printf("err is %s", err)
	}
	// TODO: display the response as string in console, you can replace it with your business logic.
	log.Printf("%s", bodyBytes)
}

func (subscriptionDemo *SubscriptionClient) ReturnFeeSubscription(authHeaderString, subscriptionId, purchaseToken string, accountFlag int) {
	bodyMap := map[string]string{
		"subscriptionId": subscriptionId,
		"purchaseToken":  purchaseToken,
	}

	url := getSubUrl(accountFlag) + "/sub/applications/v2/purchases/returnFee"
	bodyBytes, err := SendRequest(authHeaderString, url, bodyMap)
	if err != nil {
		log.Printf("err is %s", err)
	}
	// TODO: display the response as string in console, you can replace it with your business logic.
	log.Printf("%s", bodyBytes)
}

func (subscriptionDemo *SubscriptionClient) WithdrawalSubscription(authHeaderString, subscriptionId, purchaseToken string, accountFlag int) {
	bodyMap := map[string]string{
		"subscriptionId": subscriptionId,
		"purchaseToken":  purchaseToken,
	}
	url := getSubUrl(accountFlag) + "/sub/applications/v2/purchases/withdrawal"
	bodyBytes, err := SendRequest(authHeaderString, url, bodyMap)
	if err != nil {
		log.Printf("err is %s", err)
	}
	// TODO: display the response as string in console, you can replace it with your business logic.
	log.Printf("%s", bodyBytes)
}
