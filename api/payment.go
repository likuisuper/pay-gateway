package main

import (
	"flag"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/db"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/config"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/handler"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/payment.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	// 初始化数据库
	db.DBInit(c.Mysql)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()

}
