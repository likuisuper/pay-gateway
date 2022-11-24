package client

import (
	"gitee.com/zhuyunkj/pay-gateway/common/global"
	"testing"
)

func TestKsPay_GetAccessToken(t *testing.T) {
	payCli := NewKsPay(KsPayConfig{
		AppId:     "ks698620895251715795",
		AppSecret: "toS5k0fee7DxfE4LmSPM5g",
		NotifyUrl: "",
	})
	accessToken, err := payCli.HttpGetAccessToken()
	println(accessToken, err)
}

func TestKsPay_GetAccessTokenWithCache(t *testing.T) {
	global.InitMemoryCacheInstance(3)
	payCli := NewKsPay(KsPayConfig{
		AppId:     "ks698620895251715795",
		AppSecret: "toS5k0fee7DxfE4LmSPM5g",
		NotifyUrl: "",
	})
	accessToken, err := payCli.GetAccessTokenWithCache()
	accessToken, err = payCli.GetAccessTokenWithCache()
	println(accessToken, err)
}

func TestKsPay_CreateOrderWithChannel(t *testing.T) {
	payCli := NewKsPay(KsPayConfig{
		AppId:     "ks698620895251715795",
		AppSecret: "toS5k0fee7DxfE4LmSPM5g",
		NotifyUrl: "https://www.baidu.com",
	})
	info := &PayOrder{
		OrderSn: "111",
		Amount:  1,
		Subject: "xhx-test",
	}
	accessToken, err := payCli.CreateOrderWithChannel(info, "")
	println(accessToken, err)
}
