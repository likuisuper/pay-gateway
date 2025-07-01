package svc

import (
	"github.com/zeromicro/go-zero/rest"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/config"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/middleware"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/appConfigServer"
)

type ServiceContext struct {
	Config                 config.Config
	Inter                  rest.Middleware
	BaseAppConfigServerApi *appConfigServer.BaseAppConfigServer
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:                 c,
		Inter:                  middleware.NewInterMiddleware().Handle,
		BaseAppConfigServerApi: appConfigServer.NewBaseAppConfigServer(c.BaseAppConfigServerUrl),
	}
}
