package client

import (
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	alipay2 "gitlab.muchcloud.com/consumer-project/alipay"
	kv_m "gitlab.muchcloud.com/consumer-project/zhuyun-core/kv_monitor"
)

var (
	aliPayClientInitFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "aliPayClientInitFailNum", nil, "支付宝 client 初始化失败", nil})}
)

// 支付宝配置
type AliPayConfig struct {
	AppId            string
	PrivateKey       string
	PublicKey        string
	AppCertPublicKey string
	PayRootCert      string
	NotifyUrl        string
	IsProduction     bool
}

func GetAlipayClient(config AliPayConfig) (client *alipay2.Client, err error) {
	client, err = alipay2.New(config.AppId, config.PrivateKey, config.IsProduction, func(c *alipay2.Client) {
		transport := &http.Transport{
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   10,
		}
		c.Client = &http.Client{ // 不要使用默认的，默认的没有超时设置
			Transport: transport,
			Timeout:   10 * time.Second,
		}
	})

	// 将 key 的验证调整到初始化阶段
	if err != nil {
		aliPayClientInitFailNum.CounterInc()
		return nil, err
	}
	err = client.LoadAppPublicCertFromFile(config.AppCertPublicKey) // 加载应用公钥证书
	if err != nil {
		logx.Errorf("加载应用公钥证书失败：%v, appId:%s", err.Error(), config.AppId)
		aliPayClientInitFailNum.CounterInc()
		return nil, err
	}

	err = client.LoadAliPayRootCertFromFile(config.PayRootCert) // 加载支付宝根证书
	if err != nil {
		logx.Errorf("加载支付宝根证书：%v, appId:%s", err.Error(), config.AppId)
		aliPayClientInitFailNum.CounterInc()
		return nil, err
	}

	err = client.LoadAliPayPublicCertFromFile(config.PublicKey) // 加载支付宝公钥证书
	if err != nil {
		logx.Errorf("加载支付宝公钥证书：%v, appId:%s", err.Error(), config.AppId)
		aliPayClientInitFailNum.CounterInc()
		return nil, err
	}

	return client, err
}
