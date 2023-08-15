package client

import (
	"gitee.com/zhuyunkj/pay-gateway/common/global"
	"testing"
)

func TestKsPay_GetAccessToken(t *testing.T) {
	payCli := NewKsPay(KsPayConfig{
		AppId:     "ks715227916870121633",
		AppSecret: "xAFjgq_av6sdhhfXBjSZ-w",
		NotifyUrl: "",
	})
	accessToken, err := payCli.HttpGetAccessToken()
	println(accessToken, err)
}

func TestKsPay_GetAccessTokenWithCache(t *testing.T) {
	global.InitMemoryCacheInstance(3)
	payCli := NewKsPay(KsPayConfig{
		AppId:     "ks715227916870121633",
		AppSecret: "xAFjgq_av6sdhhfXBjSZ-w",
		NotifyUrl: "",
	})
	accessToken, err := payCli.GetAccessTokenWithCache()
	accessToken, err = payCli.GetAccessTokenWithCache()
	println(accessToken, err)
}

func TestKsPay_CreateOrderWithChannel(t *testing.T) {
	global.InitMemoryCacheInstance(3)

	payCli := NewKsPay(KsPayConfig{
		AppId:     "ks715227916870121633",
		AppSecret: "xAFjgq_av6sdhhfXBjSZ-w",
		NotifyUrl: "https://test.api.pay-gateway.yunxiacn.com/notify/kspay",
	})
	info := &PayOrder{
		OrderSn:  "111111",
		Amount:   1,
		Subject:  "xhx-test",
		KsTypeId: 1273,
	}
	accessToken, err := payCli.CreateOrderWithChannel(info, "f18edd95958a8bd414bf57246298c1e9")
	println(accessToken, err)
}

func TestKsPay_CreateOrder(t *testing.T) {
	global.InitMemoryCacheInstance(3)

	payCli := NewKsPay(KsPayConfig{
		AppId:     "ks715227916870121633",
		AppSecret: "xAFjgq_av6sdhhfXBjSZ-w",
		NotifyUrl: "https://test.api.pay-gateway.yunxiacn.com/notify/kspay",
	})
	info := &PayOrder{
		OrderSn:  "111111",
		Amount:   1,
		Subject:  "xhx-test",
		KsTypeId: 1273,
	}
	accessToken, err := payCli.CreateOrderWithChannel(info, "f18edd95958a8bd414bf57246298c1e9")
	println(accessToken, err)
}

func TestKsPay_CancelChannel(t *testing.T) {
	global.InitMemoryCacheInstance(3)

	payCli := NewKsPay(KsPayConfig{
		AppId:     "ks715227916870121633",
		AppSecret: "xAFjgq_av6sdhhfXBjSZ-w",
		NotifyUrl: "https://test.api.pay-gateway.yunxiacn.com/notify/kspay",
	})
	err := payCli.CancelChannel("111111")
	println(err)
}

func TestKsPay_QueryOrder(t *testing.T) {
	global.InitMemoryCacheInstance(3)

	payCli := NewKsPay(KsPayConfig{
		AppId:     "ks715227916870121633",
		AppSecret: "xAFjgq_av6sdhhfXBjSZ-w",
		NotifyUrl: "https://test.api.pay-gateway.yunxiacn.com/notify/kspay",
	})
	info, err := payCli.QueryOrder("111111")
	println(info, err)
}
