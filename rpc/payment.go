package main

import (
	"flag"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/global"
	"gitee.com/zhuyunkj/pay-gateway/db"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/config"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/server"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	"gitee.com/zhuyunkj/zero-contrib/nacos"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/payment.yaml", "the config file")

func main() {
	kv_m.SetAllMonitorFixLabel("business", "payment.rpc")
	kv_m.InitKvMonitor()
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	// 初始化数据库
	db.DBInit(c.Mysql)
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

	opts := nacos.NewNacosConfig("payment.rpc", c.Nacos.ListenOn, sc, cc)
	err := nacos.RegisterService(opts)
	if err != nil {
		logx.Errorf("nacosService err:%v", err)
	}

	global.InitMemoryCacheInstance(3)

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
