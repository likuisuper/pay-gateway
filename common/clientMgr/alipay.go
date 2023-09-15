package clientMgr

import (
	"fmt"
	alipay2 "gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"sync"
)

const RedisAppConfigKey = "app:config:%s"    //%s:包名
const RedisAliPayConfigKey = "pay:config:%s" //%s:支付宝的app_id
var cliCache sync.Map

func GetAlipayClientByAppPkgWithCache(pkgName string) (payClient *alipay2.Client, appId string, notifyUrl string, err error) {
	return GetAlipayClientWithCache(pkgName, "")
}

func GetAlipayClientByAppIdWithCache(aliAppId string) (payClient *alipay2.Client, appId string, notifyUrl string, err error) {
	return GetAlipayClientWithCache("", aliAppId)
}

func GetAlipayClientWithCache(pkgName string, aliAppId string) (payClient *alipay2.Client, appId string, notifyUrl string, err error) {

	var appConfigModel *model.PmAppConfigModel
	var payConfigAlipayModel *model.PmPayConfigAlipayModel
	var rKeyAppCfg, rKeyPayCfg string

	pkgCfg := &model.PmAppConfigTable{}
	if aliAppId != "" {
		pkgCfg.AlipayAppID = aliAppId
	} else {
		appConfigModel = model.NewPmAppConfigModel(define.DbPayGateway)
		rKeyAppCfg = appConfigModel.RDB.GetRedisKey(RedisAppConfigKey, pkgName)
		appConfigModel.RDB.GetObject(nil, rKeyAppCfg, pkgCfg)
	}

	payCfg := &model.PmPayConfigAlipayTable{}
	payConfigAlipayModel = model.NewPmPayConfigAlipayModel(define.DbPayGateway)
	if pkgCfg.AlipayAppID != "" {
		rKeyPayCfg = payConfigAlipayModel.RDB.GetRedisKey(RedisAliPayConfigKey, pkgCfg.AlipayAppID)
		payConfigAlipayModel.RDB.GetObject(nil, rKeyPayCfg, payCfg)
	}

	if payCfg.ID != 0 && pkgCfg.AlipayAppID != "" {
		config := *payCfg.TransClientConfig()
		if cli, ok := cliCache.Load(config.AppId); ok {
			payClient = cli.(*alipay2.Client)
		}
	}

	if payCfg.ID == 0 || pkgCfg.ID == 0 || payClient == nil {
		pkgCfg, err = appConfigModel.GetOneByPkgName(pkgName)
		if err != nil {
			util.CheckError("pkgName= %s, 读取应用配置失败，err:=%v", pkgName, err)
			return nil, "", "", err
		}

		payCfg, err = payConfigAlipayModel.GetOneByAppID(pkgCfg.AlipayAppID)
		if err != nil {
			err = fmt.Errorf("pkgName= %s, 读取支付宝配置失败，err:=%v", pkgName, err)
			util.CheckError(err.Error())
			return nil, "", "", err
		}

		config := *payCfg.TransClientConfig()
		cliCache.Delete(config.AppId)

		appConfigModel.RDB.Set(nil, rKeyAppCfg, *pkgCfg, 3*3600)

		rKeyPayCfg = payConfigAlipayModel.RDB.GetRedisKey(RedisAliPayConfigKey, pkgCfg.AlipayAppID)
		payConfigAlipayModel.RDB.Set(nil, rKeyPayCfg, *payCfg, 3*3600)

		payClient, err = client.GetAlipayClient(config)
		if err == nil && payClient != nil {
			cliCache.Store(config.AppId, payClient)
		}
	}

	if err != nil {
		err = fmt.Errorf("pkgName= %s, 初始化支付错误，err:=%v", pkgName, err)
		util.CheckError(err.Error())
		return nil, "", "", err
	}

	return payClient, pkgCfg.AlipayAppID, payCfg.NotifyUrl, err
}
