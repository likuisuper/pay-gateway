package main

import (
	"flag"
	"fmt"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/global"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/config"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/server"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/pb/pb"
	nacos2 "gitlab.muchcloud.com/consumer-project/zero-contrib/nacos"
	kv_m "gitlab.muchcloud.com/consumer-project/zhuyun-core/kv_monitor"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/nacos"

	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var nacosConfigFile = flag.String("nacos", "etc/nacos.yaml", "the nacos config file")

// var configFile = flag.String("f", "etc/payment.yaml", "the config file")

func main() {
	kv_m.SetAllMonitorFixLabel("business", "payment.rpc")
	kv_m.InitKvMonitor()
	flag.Parse()

	var c config.Config

	//conf.MustLoad(*configFile, &c)

	//从nacos获取配置
	var nacosConfig nacos.Config
	conf.MustLoad(*nacosConfigFile, &nacosConfig)
	nacosClient, nacosErr := nacos.InitNacosClient(nacosConfig)
	if nacosErr != nil {
		logx.Errorf("初始化nacos客户端失败: " + nacosErr.Error())
		return
	}
	err := nacosClient.GetConfig(nacosConfig.DataId, nacosConfig.GroupId, &c)
	defer nacosClient.CloseClient()
	if err != nil {
		logx.Errorf("获取配置失败：" + err.Error())
		return
	}

	// 初始化数据库
	db.DBInit(c.Mysql, c.RedisConfig)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterPaymentServer(grpcServer, server.NewPaymentServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	var sc []constant.ServerConfig
	for _, v := range c.Nacos.NacosService {
		service := *constant.NewServerConfig(v.Ip, v.Port)
		sc = append(sc, service)
	}

	logRollingConfig := &constant.ClientLogRollingConfig{
		MaxAge: c.Nacos.MaxAge,
	}
	cc := &constant.ClientConfig{
		AppName:             c.Name,
		NamespaceId:         c.Nacos.NamespaceId,
		TimeoutMs:           c.Nacos.TimeoutMs,
		NotLoadCacheAtStart: c.Nacos.NotLoadCacheAtStart,
		LogDir:              c.Nacos.LogDir,
		CacheDir:            c.Nacos.CacheDir,
		Username:            c.Nacos.Username,
		Password:            c.Nacos.Password,
		LogLevel:            c.Nacos.LogLevel,
		LogRollingConfig:    logRollingConfig,
	}

	opts := nacos2.NewNacosConfig(c.Name, c.Nacos.ListenOn, sc, cc)
	err = nacos2.RegisterService(opts)
	if err != nil {
		logx.Errorf("nacosService err:%v", err)
	}

	global.InitMemoryCacheInstance(3)

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
