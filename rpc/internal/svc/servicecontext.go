package svc

import (
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/config"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/appConfigServer"
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
