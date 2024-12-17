package huawei

import (
	"fmt"
	"log"
)

type OrderClient struct {
}

var OrderDemo = &OrderClient{}

// 站点信息 服务

// 站点信息

// Order

// 中国站点：https://orders-drcn.iap.cloud.huawei.com.cn

// 德国站点：https://orders-dre.iap.cloud.huawei.eu

// 新加坡站点：https://orders-dra.iap.cloud.huawei.asia

// 俄罗斯站点：https://orders-drru.iap.cloud.huawei.ru

// https://developer.huawei.com/consumer/cn/doc/HMSCore-References/api-common-statement-0000001050986127
func getOrderUrl(accountFlag int) string {
	return "https://orders-drcn.iap.cloud.huawei.com.cn"
}

func (orderDemo *OrderClient) VerifyToken(authHeaderString, purchaseToken, productId string, accountFlag int) {
	bodyMap := map[string]string{"purchaseToken": purchaseToken, "productId": productId}
	url := getOrderUrl(accountFlag) + "/applications/purchases/tokens/verify"
	bodyBytes, err := SendRequest(authHeaderString, url, bodyMap)
	if err != nil {
		log.Printf("err is %s", err)
		return
	}

	// TODO: display the response as string in console, you can replace it with your business logic.
	log.Printf("%s", bodyBytes)
}

func (orderDemo *OrderClient) CancelledListPurchase(authHeaderString string, endAt int64, startAt int64, maxRows int, productType int, continuationToken string, accountFlag int) {
	bodyMap := map[string]string{
		"endAt":             fmt.Sprintf("%v", endAt),
		"startAt":           fmt.Sprintf("%v", startAt),
		"maxRows":           fmt.Sprintf("%v", maxRows),
		"type":              fmt.Sprintf("%v", productType),
		"continuationToken": continuationToken,
	}
	url := getOrderUrl(accountFlag) + "/applications/v2/purchases/cancelledList"
	bodyBytes, err := SendRequest(authHeaderString, url, bodyMap)
	if err != nil {
		log.Printf("err is %s", err)
	}
	// TODO: display the response as string in console, you can replace it with your business logic.
	log.Printf("%s", bodyBytes)
}

func (orderDemo *OrderClient) ConfirmPurchase(authHeaderString, purchaseToken, productId string, accountFlag int) {
	bodyMap := map[string]string{
		"purchaseToken": purchaseToken,
		"productId":     productId,
	}
	url := getOrderUrl(accountFlag) + "/applications/v2/purchases/confirm"
	bodyBytes, err := SendRequest(authHeaderString, url, bodyMap)
	if err != nil {
		log.Printf("err is %s", err)
	}
	// TODO: display the response as string in console, you can replace it with your business logic.
	log.Printf("%s", bodyBytes)
}
