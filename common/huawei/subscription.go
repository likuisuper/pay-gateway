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
	// 应用ID
	ApplicationId int64 `json:"applicationId"`
	// 消耗型商品或者非消耗型商品：固定为false。
	//
	// 订阅型商品：
	//
	// true：订阅处于活动状态并且将在下一个结算日期自动续订。
	//
	// false：用户已经取消订阅。 用户可以在下一个结算日期之前访问订阅内容，并且在该日期后将无法访问，除非重新启用自动续订。 如果提供了宽限期，只要宽限期未过，此值就会对所有订阅保持设置为true。 下一次结算日期每天都会自动延长，直至宽限期结束，或者用户更改付款方式。
	AutoRenewing bool `json:"autoRenewing"`
	// 订单ID，唯一标识一笔需要收费的收据，由华为应用内支付服务器在创建订单以及订阅型商品续费时生成。
	//
	// 每一笔新的收据都会使用不同的orderId。
	OrderId string `json:"orderId"`
	// 商品类别，取值包括：0：消耗型商品 1：非消耗型商品 2：订阅型商品
	Kind int `json:"kind"`
	// 应用安装包名
	PackageName string `json:"packageName"`
	// 商品ID。每种商品必须有唯一的ID，由应用在PMS中维护，或者应用发起购买时传入。为避免资金损失，您在对支付结果验签成功后，必须对其进行校验。
	ProductId string `json:"productId"`
	// 商品名称
	ProductName string `json:"productName"`
	// 商品购买时间，UTC时间戳，以毫秒为单位。如果没有完成购买，则没有值。
	PurchaseTime int64 `json:"purchaseTime"`
	// 订单交易状态。-1：初始化 0：已购买 1：已取消 2：已退款 3：待处理
	PurchaseState int `json:"purchaseState"`
	// 商户侧保留信息，由您在调用支付接口时传入
	DeveloperPayload string `json:"developerPayload"`
	// 应用发起消耗请求时自定义的挑战字，可唯一标识此次消耗请求，仅一次性商品存在。
	DeveloperChallenge string `json:"developerChallenge"`
	// 消耗状态，仅一次性商品存在，取值包括：0：未消耗 1：已消耗
	ConsumptionState int `json:"consumptionState"`
	// 用于唯一标识商品和用户对应关系的购买令牌，在支付完成时由华为应用内支付服务器生成。
	//
	// 说明 该字段是唯一标识商品和用户对应关系的，在订阅型商品正常续订时不会改变。当前92位，后续存在扩展可能，如要进行存储，建议您预留128位的长度。 如要进行存储，为保证安全，建议加密存储。
	PurchaseToken string `json:"purchaseToken"`
	// 购买类型。0：沙盒环境。1：促销，暂不支持。 正式购买不会返回该参数。
	PurchaseType int `json:"purchaseType"`
	// 定价货币的币种
	//
	// 为避免资金损失，您在对支付结果验签成功后，必须对其进行校验。
	Currency string `json:"currency"`
	// 商品实际价格*100以后的值。商品实际价格精确到小数点后2位，例如此参数值为501，则表示商品实际价格为5.01。
	//
	// 为避免资金损失，您在对支付结果验签成功后，必须对其进行校验。
	Price int64 `json:"price"`
	// 国家/地区码，用于区分国家/地区信息，请参见ISO 3166标准。
	Country string `json:"country"`
	// 支付方式: https://developer.huawei.com/consumer/cn/doc/HMSCore-References/server-data-model-0000001050986133#section135412662210
	PayType string `json:"payType"`
	// 交易单号，用户支付后生成
	PayOrderId string `json:"payOrderId"`
	// ========================以下参数只在订阅场景返回========================
	//
	// 上次续期收款的订单ID，由支付服务器在续期扣费时生成。首次购买订阅型商品时的lastOrderId与orderId数值相同。
	LastOrderId string `json:"lastOrderId"`
	// 订阅型商品所属的订阅组ID
	ProductGroup string `json:"productGroup"`
	// 原购买时间，UTC时间戳，以毫秒为单位
	OriPurchaseTime int64 `json:"oriPurchaseTime"`
	// 订阅ID
	SubscriptionId string `json:"subscriptionId"`
	// 原订阅ID。有值表示当前订阅是从其他商品切换来的，该值可以关联切换前的商品订阅信息
	OriSubscriptionId string `json:"oriSubscriptionId"`
	// 购买数量
	Quantity int `json:"quantity"`
	// 已经付费订阅的天数，免费试用和促销期周期除外。
	DaysLasted int `json:"daysLasted"`
	// 成功标准续期（没有设置促销的续期）的期数，为0或者不存在表示还没有成功续期。
	NumOfPeriods int `json:"numOfPeriods"`
	// 成功促销续期期数
	NumOfDiscount int `json:"numOfDiscount"`
	// 订阅型商品过期时间，UTC时间戳，以毫秒为单位。对于一个成功收费的自动续订收据，该时间表示续期日期或者超期日期。如果商品最近的收据的该时间是一个过去的时间，则订阅已经过期。
	ExpirationDate int64 `json:"expirationDate"`
	// 对于已经过期的订阅，表示过期原因，取值包括：
	// 1：用户取消
	// 2：商品不可用
	// 3：用户签约信息异常
	// 4：Billing错误
	// 5：用户未同意涨价
	// 6：未知错误
	// 同时有多个异常时，优先级为：1 > 2 > 3…
	ExpirationIntent int `json:"expirationIntent"`
	// 一个过期的订阅，系统是否仍然在尝试自动完成续期处理。取值包括：
	// 0：终止尝试
	// 1：仍在尝试完成续期
	RetryFlag int `json:"retryFlag"`
	// 是否处于促销价续期周期内。
	// 1：是
	// 0：否
	IntroductoryFlag int `json:"introductoryFlag"`
	// 是否处于免费试用周期内。
	// 1：是
	// 0：否
	TrialFlag int `json:"trialFlag"`
	// 订阅撤销时间，发生退款且服务立即不可用，UTC时间戳，以毫秒为单位。
	// 在顾客投诉，通过客服撤销订阅，或者顾客升级、跨级到同组其他商品并且立即生效场景下，需要撤销原有订阅的上次收据时有值。
	CancelTime int64 `json:"cancelTime"`
	// 取消原因。
	// 3：您主动发起的退款、撤销等。如果cancelTime同时为空，表示是返还订阅费用场景。
	// 2：顾客升级、跨级等。
	// 1：顾客因为在App内遇到了问题而取消了订阅。
	// 0：其他原因取消，比如顾客错误地订阅了商品。
	// 如果为空且cancelTime有值，表示是升级等操作导致的取消。
	CancelReason int `json:"cancelReason"`
	// App信息，预留
	AppInfo string `json:"appInfo"`
	// 用户是否已经关闭订阅上的通知。
	// 1：是
	// 0：否
	// 关闭状态下，订阅相关的通知均不会发送给用户。
	NotifyClosed int `json:"notifyClosed"`
	// 续期状态。
	// 1：当前周期到期时自动续期
	// 0：用户停止了续期
	// 仅针对自动续期订阅，对有效和过期的订阅均有效，并不代表顾客的订阅状态。通常，取值为0时，应用可以给顾客提供其他的订阅选项，例如推荐一个同组更低级别的商品。该值为0通常代表着顾客主动取消了该订阅。
	RenewStatus int `json:"renewStatus"`
	// 商品提价时的用户意见。
	// 1：用户已经同意提价
	// 0：用户未采取动作，超期后订阅失效
	PriceConsentStatus int `json:"priceConsentStatus"`
	// 下次续期价格。在有priceConsentStatus情况下，供客户端参考，用于提示用户新的续期价格。
	RenewPrice int64 `json:"renewPrice"`
	// true：表示商品已经收费且未过期，也没有发生退款；商品处于宽限期。您可以基于该标志为顾客提供服务。
	// false：未完成购买或者已经过期，或者购买后已经退款。
	// 如果顾客已经取消订阅，在已经购买的商品过期之前，subIsvalid仍然为True。
	SubIsvalid bool `json:"subIsvalid"`
	// 是否延迟结算。
	// 1：是
	// 其他：否
	DeferFlag int `json:"deferFlag"`
	// 取消订阅途径。
	// 0：顾客
	// 1：您
	// 2：华为
	CancelWay int `json:"cancelWay"`
	// 取消订阅时间，UTC时间戳，以毫秒为单位
	CancellationTime int64 `json:"cancellationTime"`
	// 用户取消后订阅关系保留的天数，并不表示本订阅已经取消。
	CancelledSubKeepDays int `json:"cancelledSubKeepDays"`
	// 一个暂停的订阅恢复的时间，UTC时间戳，以毫秒为单位。
	ResumeTime int64 `json:"resumeTime"`
	// 订阅型商品宽限期过期的时间，UTC时间戳，以毫秒为单位。
	GraceExpirationTime int64 `json:"graceExpirationTime"`
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
