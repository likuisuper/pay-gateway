package main

import (
	"errors"
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/crontab"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/alarm"
	kv_m "gitlab.muchcloud.com/consumer-project/zhuyun-core/kv_monitor"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/nacos"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/util"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/config"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/handler"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var nacosConfigFile = flag.String("nacos", "etc/nacos.yaml", "the nacos config file")

const nacosServerName = "pay-gateway.cron"

func main() {
	kv_m.SetAllMonitorFixLabel("business", "payment.api")
	kv_m.InitKvMonitor()
	flag.Parse()

	var c config.Config
	var nacosConfig nacos.Config
	conf.MustLoad(*nacosConfigFile, &nacosConfig)
	nacosClient, nacosErr := nacos.InitNacosClient(nacosConfig)
	if nacosErr != nil {
		logx.Error("初始化nacos客户端失败: " + nacosErr.Error())
	}

	// 加载一次配置
	err := nacosClient.GetConfig(nacosConfig.DataId, nacosConfig.GroupId, &c)
	if err != nil {
		logx.Error("获取配置失败：" + err.Error())
		return
	}

	// 初始化数据库
	db.DBInit(c.Mysql, c.RedisConfig)

	//初使化钉钉通知服务
	alarm.InitAlarmClient(c.Name, c.Alarm.Redis, c.Alarm.DingDingUrl)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	//注册nacos服务并启用
	go util.SafeRun(func() {
		nacosInstanc, _ := RegisterInstance(&nacosConfig)

		if nacosInstanc != nil {
			namingClient, _ := nacos.InitNamingClient(nacosConfig)
			crontab.InitCrontabOrder(namingClient, nacosServerName, &c, ctx)
		}
	})

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()

}

// 测试nacos服务
func RegisterInstance(nacosConfig *nacos.Config) (*nacos.Instance, error) {
	// 初始化服务注册 nacosclient
	namingClient, nacosErr := nacos.InitNamingClient(*nacosConfig)
	if nacosErr != nil {
		logx.Errorf("初始化 nacos 服务注册客户端失败, err= %v", nacosErr)
		return namingClient, nacosErr
	}

	var registerInsParamm nacos.RegisterInstanceParam
	ip, err := util.ExternalIP()
	if err != nil {
		logx.Errorf("util.ExternalIP err : %v", err)
	}

	registerInsParamm.Ip = ip.String()
	registerInsParamm.Port = 0
	registerInsParamm.ServiceName = nacosServerName
	registerInsParamm.Weight = 10

	// 按照随机值取数
	startTime := strconv.FormatInt(time.Now().Unix(), 10)
	registerInsParamm.Metadata = map[string]string{"startTime": startTime}
	suc, err := namingClient.RegisterInstance(&registerInsParamm)
	if err != nil || !suc {
		logx.Errorf("注册 nacos 服务实例失败, err= %v", err)
		return namingClient, errors.New("注册实例失败")
	}
	return namingClient, nil
}
