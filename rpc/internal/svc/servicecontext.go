package svc

import (
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/config"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config       config.Config
	AppConfigMap map[string]*AppPkgConfig
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:       c,
		AppConfigMap: initAppsConfig(c),
	}
}

//包名对应的配置
type AppPkgConfig struct {
	AppRel    config.AppRelConfig
	Alipay    config.AliPay
	WechatPay config.WechatPay
	TikTokPay config.TikTokPay
}

//初始化应用包名对应的配置   map[pkgName]config
func initAppsConfig(cfg config.Config) (appConfig map[string]*AppPkgConfig) {
	appConfig = make(map[string]*AppPkgConfig)

	appRelMap := make(map[string]*config.AppRelConfig, 0)
	for _, relCfg := range cfg.AppRel {
		appRelMap[relCfg.AppPkgName] = relCfg
	}
	alipayMap := make(map[string]*config.AliPay, 0)
	for _, payCfg := range cfg.Alipay {
		alipayMap[payCfg.AppId] = payCfg
	}
	wechatPayMap := make(map[string]*config.WechatPay, 0)
	for _, payCfg := range cfg.WeChatPay {
		wechatPayMap[payCfg.AppId] = payCfg
	}
	tiktokPayMap := make(map[string]*config.TikTokPay, 0)
	for _, payCfg := range cfg.TikTokPay {
		tiktokPayMap[payCfg.AppId] = payCfg
	}
	//包配置map组装
	for pkgName, appRelCfg := range appRelMap {
		appConfig[pkgName] = &AppPkgConfig{
			AppRel: *appRelCfg,
		}
		if payCfg, ok := alipayMap[appRelCfg.AlipayAppId]; ok {
			appConfig[pkgName].Alipay = *payCfg
		}
		if payCfg, ok := wechatPayMap[appRelCfg.WechatPayAppId]; ok {
			appConfig[pkgName].WechatPay = *payCfg
		}
		if payCfg, ok := tiktokPayMap[appRelCfg.TikTokPayAppId]; ok {
			appConfig[pkgName].TikTokPay = *payCfg
		}
	}
	//配置检查
	for pkgName, appCfg := range appConfig {
		if appCfg.Alipay.AppId == "" {
			logx.Errorf("initAppsConfig fail [%s] [%s] ...\n", pkgName, "alipay")
		}
		if appCfg.WechatPay.AppId == "" {
			logx.Errorf("initAppsConfig fail [%s] [%s] ...\n", pkgName, "wechatPay")
		}
		if appCfg.TikTokPay.AppId == "" {
			logx.Errorf("initAppsConfig fail [%s] [%s] ...\n", pkgName, "tiktokPay")
		}
	}
	return
}
