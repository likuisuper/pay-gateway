package huawei

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/zeromicro/go-zero/core/logx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/cache"
)

// https://developer.huawei.com/consumer/cn/doc/HMSCore-References/obtain-application-level-at-0000001051066052
//
// 获取华为应用token授权地址
const get_token_url = "https://oauth-login.cloud.huawei.com/oauth2/v3/token"

type AtResponse struct {
	AccessToken string `json:"access_token"` // token
	ExpiresIn   int    `json:"expires_in"`   // 过期时间秒
}

type HuaweiAccessTokenClient struct {
	ctx          context.Context
	ClientId     string // ClientId 在AppGallery Connect创建应用之后，系统自动分配的唯一标识符
	ClientSecret string // ClientSecret App secret, 在AppGallery Connect创建应用之后，系统自动分配的公钥
	AppSecret    string // AppSecret 应用公钥, base64编码
	RDB          *cache.RedisInstance
}

func NewClient(ctx context.Context, dbName, ClientId, ClientSecret, AppSecret string) *HuaweiAccessTokenClient {
	return &HuaweiAccessTokenClient{
		ctx:          ctx,
		ClientId:     ClientId,
		ClientSecret: ClientSecret,
		AppSecret:    AppSecret,
		RDB:          db.WithRedisDBContext(dbName),
	}
}

// access token 缓存key
const huawei_access_token_key = "hw:app:access:token:%s" // %s是client id(app id)

// 获取华为应用access token
func (c *HuaweiAccessTokenClient) GetAppAccessToken() (string, error) {
	rkey := c.RDB.GetRedisKey(huawei_access_token_key, c.ClientId)
	tmpAcessToken, err := c.RDB.GetString(context.TODO(), rkey)
	if err == nil {
		return tmpAcessToken, nil
	}

	urlValue := url.Values{"grant_type": {"client_credentials"}, "client_secret": {c.ClientSecret}, "client_id": {c.ClientId}}
	resp, err := RequestHttpClient.PostForm(get_token_url, urlValue)
	if err != nil {
		logx.WithContext(c.ctx).Errorf("RequestHttpClient.PostForm error: %v", err)
		return "", err
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logx.WithContext(c.ctx).Errorf("GetAppAccessToken error: %v", err)
		return "", err
	}

	// 正常时返回
	// {"access_token":"DQEBAP+CjENXLOUNVZP5R9uzLXTD/PWw7xXrYUJOAfnnrGjHE3NPJqGNpgjN9eVJLrVHJoM/9ehRVruNpBb3MTbSldM+ZqRYoWQj1Q==","token_type":"Bearer","expires_in":3600}
	logx.WithContext(c.ctx).Slowf("GetAppAccessToken raw response: %v", string(bodyBytes))

	var atResponse AtResponse
	json.Unmarshal(bodyBytes, &atResponse)
	if atResponse.AccessToken != "" {
		// 设置缓存
		c.RDB.Set(context.TODO(), rkey, atResponse.AccessToken, atResponse.ExpiresIn-300)
		return atResponse.AccessToken, nil
	}

	return "", errors.New("Get token fail, " + string(bodyBytes))
}

// 请求头需要使用Access Token进行鉴权
func (c *HuaweiAccessTokenClient) BuildAuthorization() (string, error) {
	appAt, err := c.GetAppAccessToken()
	if err != nil {
		return "", err
	}

	oriString := fmt.Sprintf("APPAT:%s", appAt)
	var authString = base64.StdEncoding.EncodeToString([]byte(oriString))
	var authHeaderString = fmt.Sprintf("Basic %s", authString)
	return authHeaderString, nil
}
