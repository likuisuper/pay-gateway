package svc

import (
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/config"
	"gitee.com/zhuyunkj/zhuyun-core/appConfigServer"
)

type ServiceContext struct {
	Config                 config.Config
	BaseAppConfigServerApi *appConfigServer.BaseAppConfigServer
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:                 c,
		BaseAppConfigServerApi: appConfigServer.NewBaseAppConfigServer(c.BaseAppConfigServerUrl),
	}
}
