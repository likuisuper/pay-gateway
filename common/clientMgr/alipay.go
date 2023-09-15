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

func GetAlipayClientWithCache(pkgName string) (payClient *alipay2.Client, appId string, notifyUrl string, err error) {

	var appConfigModel *model.PmAppConfigModel
	var payConfigAlipayModel *model.PmPayConfigAlipayModel
	var rKeyAppCfg, rKeyPayCfg string

	pkgCfg := &model.PmAppConfigTable{}
	appConfigModel = model.NewPmAppConfigModel(define.DbPayGateway)
	rKeyAppCfg = appConfigModel.RDB.GetRedisKey(RedisAppConfigKey, pkgName)
	appConfigModel.RDB.GetObject(nil, rKeyAppCfg, pkgCfg)

	payCfg := &model.PmPayConfigAlipayTable{}
	payConfigAlipayModel = model.NewPmPayConfigAlipayModel(define.DbPayGateway)
	if pkgCfg.ID != 0 {
		rKeyPayCfg = payConfigAlipayModel.RDB.GetRedisKey(RedisAliPayConfigKey, pkgCfg.AlipayAppID)
		payConfigAlipayModel.RDB.GetObject(nil, rKeyPayCfg, payCfg)
	}

	if payCfg.ID != 0 && pkgCfg.ID != 0 {
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

	// 临时测试
	payClient, _ = alipay2.New("2021004114603349", "MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCTmzawf+GulVtsOMfWJqg+p0SW8uIQmsjBeLtpJaDLzzbgQzCvNovHep1rL/uy1vKs01I4BWkSWTK1aymW8co5vWgQIC0iv4q7BuJQGVUP1gkwbrwxoQeoFu12L9jqh2oIc8deZC9MtbqK0VrX0q2BA9uFdvyjBoXqM09KMztMD/jeDXderDS+O0/IEnGo8/AEQGxY4kLgJEGh+eGcWqD3WU9BP+KeJxwiAHf/ZhcgXSfS6PfJZU+LqQrJP/rrLEJVfT1yEa58CjFkrWczQ3Eis6wqVap2kK4ycN4O7zaDfYfpsv8YDHpIhV2mpVkqxYtiAbGC+qMaqf2Z8eDuTjOpAgMBAAECggEAWNVm+p5cIof894rMqhOl2d8tJnOSnk+pVtbkY4mj1kUlT57gY/K9+RXQO7wrDRzT/DNKHjETZVmNbSXLZ+6ouEtHn7zdrTX9tkWUWoSEbv1vllhupqe1RfJWg3SUZcGNjPyxFhvRY6dTV0xcEdvXU/gQW6iarzqzyZmLtKpUm5dsjSwKjV6joNK8tvCZEdGbQZCNbQh9PLOjPLfUxPcoN8pNoLkvv6YBulugWsY8qYFH9LcVxogVehqtWSe4C/hR8QktJtrcJXMY0rcVhrr010ctpxiQg8h/n1E6VqeKV6+wYxIONvnav28/sI85rks3vKZGe97xe44kbAIM5+uD6QKBgQDiTIHQpy//C+qe4Qwc7UBLhMeBapWOEs2O4qxywv5EahMxKQ8fMbDdqjNhndEon1dVBou1U92iwzqIptAVOnUvtCY36qhRxJzpgT6tnGo50ilWuqAbYdYnh1gEC4Ho6zjvCYxMmV7guZusQXO8VjfPW13KY1ZfswJqW8gbb7gW1wKBgQCm+rA/KE8eteVGiZq0agHitWxH2idOVaxfsSVqZ4RjNHrZurv+1Ss+S/6AdcN1o4bfNU9DR4ai6oTdhb+bNPN5bn+tWSWbfbR0T5dunJiSsy7605w47FnSIOu+nG+Rfatk16vwk6EP2LXaKDqKw4RCEcp5qoeu4pMumXgUlqM5fwKBgQCRyC2cqAeQazHS9jFidSiFPd10LqB3rP9FPBtRtvIsSpVghw3Zz54bvmhpS0yRucx91sCrqIJQNyp/G89SzZzuhURVo1KZkmpvNraVCv2XkB7XY1R/L1DRmCwINw2Sae38d48tTWREqu1xU5zmSDid2UMbfVEIR36X29aWbisOcwKBgCznTW4ukNhZYgbOCmRp/YfR8gSAjgFq2KgDI2Sx4dAr1L2okdW9zZs7JH23LZD9IM/1rhMRsQsutfw8c4Jxgugs5vje+FYQP+7nWHnOctlAhmm9bk2AgccYQ01HFFmzyduchAh2KuHwDTdViii222JJFoIRcdt94satTrV6rPpRAoGBAMVREudU98vflusrSb46Tao7GOUnlPOlgyvskvFJ6ATndtBQawuEpqQxrAYAGjK43Krfcc0TIdOGw8LFEqk9prUTjjHb+7t87ThVrFCUFcJohpWe3xvhCnuviepXHY7jkroEvZUVHEIdLHjN5u+XDeaErvanBvSrjRCYaB4fZ0zF", true)
	payClient.LoadAliPayPublicKey("MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxKemh4Wc/uq2yYVjM39laJ2RZ1q3EJUD/dTOVe5XnNa5/jKcTfP4kKvcVkkY/gzbCY4JNx3HKNmdECHkDdwHOyr11xUqGFatn4eJ/vWBByRH6p0/n7ZMIcimsG/XePTJAzDqSZI7/YlDoQZNmobJoThrKVeSZyfAgDkCMglmo6B1ragWwcpf6E67t2ZRkrAZ55n8hFBf+qWviYm9VKM2ds9yVFKp+DSVDfWzNF386zOjxPB+LeOq18+ZXU7PFWv/LNuBNZiSwFe2t0a6XxyhHP8LmGCmcrVlS83E3CMoIKGXe2VwE3wKzXMDuwyHxG988eEmIxwSMN333qxT/mCzmwIDAQAB")

	return payClient, pkgCfg.AlipayAppID, notifyUrl, err
}
