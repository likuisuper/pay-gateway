package client

import (
	"gitee.com/zhuyunkj/pay-gateway/internal/config"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	alipay2 "github.com/smartwalle/alipay/v3"
)

var (
	aliPayClientInitFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "aliPayClientInitFailNum", nil, "支付宝 client 初始化失败", nil})}
)

func GetAlipayClient(config config.AliPay) (client *alipay2.Client, err error) {
	client, err = alipay2.New(config.AppId, config.PrivateKey, config.IsProduction)
	// 将 key 的验证调整到初始化阶段
	if err != nil {
		aliPayClientInitFailNum.CounterInc()
		return nil, err
	}
	err = client.LoadAppPublicCertFromFile(config.AppCertPublicKey) // 加载应用公钥证书
	if err != nil {
		aliPayClientInitFailNum.CounterInc()
		return nil, err
	}
	err = client.LoadAliPayRootCertFromFile(config.PayRootCert) // 加载支付宝根证书
	if err != nil {
		aliPayClientInitFailNum.CounterInc()
		return nil, err
	}
	err = client.LoadAliPayPublicCertFromFile(config.PublicKey) // 加载支付宝公钥证书
	if err != nil {
		aliPayClientInitFailNum.CounterInc()
		return nil, err
	}
	return client, err
}
