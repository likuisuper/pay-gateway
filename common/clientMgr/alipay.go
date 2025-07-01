package clientMgr

import (
	"context"
	"errors"
	"fmt"
	"sync"

	alipay2 "gitlab.muchcloud.com/consumer-project/alipay"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/client"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/util"
)

const RedisAppConfigKey = "app:config:%s"    //%s:包名
const RedisAliPayConfigKey = "pay:config:%s" //%s:支付宝的app_id

const RedisAppConfigKey2 = "app:config2:%s"    //%s:包名
const RedisAliPayConfigKey2 = "pay:config2:%s" //%s:支付宝的app_id

var cliCache sync.Map

func GetAlipayClientByAppPkgWithCache(pkgName string) (payClient *alipay2.Client, appId string, notifyUrl string, err error) {
	return getAlipayClientWithCache(pkgName, "")
}

func GetAlipayClientByAppIdWithCache(aliAppId string) (payClient *alipay2.Client, appId string, notifyUrl string, err error) {
	return getAlipayClientWithCache("", aliAppId)
}

// 需要选用不同的支付宝账号
func GetAlipayClienMerchantInfo(pkgName string) (payClient *alipay2.Client, appId string, notifyUrl string, merchantNo string, merchantName string, err error) {
	// 需要根本包名去找不同的支付号 然后筛选出来可用的
	appConfig, err := model.NewAppAlipayAppModel(define.DbPayGateway).GetValidConfig(pkgName)
	if err != nil {
		return
	}

	aliAppId := appConfig.AppID
	payCfg, err := model.NewPmPayConfigAlipayModel(define.DbPayGateway).GetOneByAppID(aliAppId)
	if err != nil {
		err = fmt.Errorf("getAlipayClientChoiceDiff读取支付宝配置失败 pkgName=%s, aliAppId=%s err=%v", pkgName, aliAppId, err)
		util.CheckError(err.Error())
		return nil, "", "", "", "", err
	}

	config := *payCfg.TransClientConfig()
	payClient, err = client.GetAlipayClient(config)
	if err != nil {
		err = fmt.Errorf("getAlipayClientChoiceDiff初始化支付错误 pkgName= %s, err:=%v", pkgName, err)
		util.CheckError(err.Error())
		return nil, "", "", "", "", err
	}

	return payClient, aliAppId, payCfg.NotifyUrl, payCfg.MerchantNo, payCfg.MerchantName, nil
}

// payCfg: 商户的配置，比如证书、回调地址
// appCfg: 应用的配置，比如使用的商户id
// 一般的场景是'包名->商户id->阿里client`, 但存在极端情况，收到支付宝回调的时候切换了商户, 所以收到回调的时候要使用商户app_id来找client
func getAlipayClientWithCache(pkgName string, aliAppId string) (payClient *alipay2.Client, appId string, notifyUrl string, err error) {
	if pkgName == "" && aliAppId == "" {
		return nil, "", "", errors.New("pkg name and aliAppId all empty")
	}

	var appConfigModel *model.PmAppConfigModel
	var payConfigAlipayModel *model.PmPayConfigAlipayModel
	var rKeyAppCfg, rKeyPayCfg string

	pkgCfg := &model.PmAppConfigTable{}
	appConfigModel = model.NewPmAppConfigModel(define.DbPayGateway)
	if aliAppId != "" {
		// 有传商户app_id，直接使用app_ID
		pkgCfg.AlipayAppID = aliAppId
	} else {
		// 没有传商户app_id，先根据包名找appPkg的缓存
		rKeyAppCfg = appConfigModel.RDB.GetRedisKey(RedisAppConfigKey, pkgName)
		appConfigModel.RDB.GetObject(context.TODO(), rKeyAppCfg, pkgCfg)
	}

	payCfg := &model.PmPayConfigAlipayTable{}
	payConfigAlipayModel = model.NewPmPayConfigAlipayModel(define.DbPayGateway)
	if pkgCfg.AlipayAppID != "" {
		// 根据商户app_id找商户配置的缓存
		rKeyPayCfg = payConfigAlipayModel.RDB.GetRedisKey(RedisAliPayConfigKey, pkgCfg.AlipayAppID)
		payConfigAlipayModel.RDB.GetObject(context.TODO(), rKeyPayCfg, payCfg)
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

		appConfigModel.RDB.Set(context.TODO(), rKeyAppCfg, *pkgCfg, 3*3600)

		rKeyPayCfg = payConfigAlipayModel.RDB.GetRedisKey(RedisAliPayConfigKey, pkgCfg.AlipayAppID)
		payConfigAlipayModel.RDB.Set(context.TODO(), rKeyPayCfg, *payCfg, 3*3600)

		payClient, err = client.GetAlipayClient(config)
		if err == nil && payClient != nil {
			cliCache.Store(config.AppId, payClient)
		}
	}

	if err != nil {
		err = fmt.Errorf("pkgName= %s, 初始化支付错误 err:=%v", pkgName, err)
		util.CheckError(err.Error())
		return nil, "", "", err
	}

	return payClient, pkgCfg.AlipayAppID, payCfg.NotifyUrl, err
}
