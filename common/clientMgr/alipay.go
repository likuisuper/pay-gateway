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
	return getAlipayClientWithCache(pkgName, "")
}

func GetAlipayClientByAppIdWithCache(aliAppId string) (payClient *alipay2.Client, appId string, notifyUrl string, err error) {
	return getAlipayClientWithCache("", aliAppId)
}

// payCfg: 商户的配置，比如证书、回调地址
// appCfg: 应用的配置，比如使用的商户id
// 一般的场景是'包名->商户id->阿里client`, 但存在极端情况，收到支付宝回调的时候切换了商户, 所以收到回调的时候要使用商户app_id来找client
func getAlipayClientWithCache(pkgName string, aliAppId string) (payClient *alipay2.Client, appId string, notifyUrl string, err error) {

	var appConfigModel *model.PmAppConfigModel
	var payConfigAlipayModel *model.PmPayConfigAlipayModel
	var rKeyAppCfg, rKeyPayCfg string

	pkgCfg := &model.PmAppConfigTable{}
	appConfigModel = model.NewPmAppConfigModel(define.DbPayGateway)
	if aliAppId != "" { // 有传商户app_id，直接使用app_ID
		pkgCfg.AlipayAppID = aliAppId
	} else { // 没有传商户app_id，先根据包名找appPkg的缓存
		rKeyAppCfg = appConfigModel.RDB.GetRedisKey(RedisAppConfigKey, pkgName)
		appConfigModel.RDB.GetObject(nil, rKeyAppCfg, pkgCfg)
	}

	payCfg := &model.PmPayConfigAlipayTable{}
	payConfigAlipayModel = model.NewPmPayConfigAlipayModel(define.DbPayGateway)
	if pkgCfg.AlipayAppID != "" { // 根据商户app_id找商户配置的缓存
		rKeyPayCfg = payConfigAlipayModel.RDB.GetRedisKey(RedisAliPayConfigKey, pkgCfg.AlipayAppID)
		payConfigAlipayModel.RDB.GetObject(nil, rKeyPayCfg, payCfg)
	}

	if payCfg.ID != 0 && pkgCfg.AlipayAppID != "" { // Redis缓存还在，直接从内存缓存中获取客户端
		config := *payCfg.TransClientConfig()
		if cli, ok := cliCache.Load(config.AppId); ok {
			payClient = cli.(*alipay2.Client)
		}
	}

	if payCfg.ID == 0 || pkgCfg.ID == 0 || payClient == nil { // Redis缓存失效，或者内存缓存中没有客户端，再去读取配置还有证书，创建客户端
		if pkgName != "" {
			pkgCfg, err = appConfigModel.GetOneByPkgName(pkgName)
			if err != nil {
				util.CheckError("pkgName= %s, 读取应用配置失败，err:=%v", pkgName, err)
				return nil, "", "", err
			}
			aliAppId = pkgCfg.AlipayAppID
		}

		payCfg, err = payConfigAlipayModel.GetOneByAppID(aliAppId)
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
