package svc

import (
	"gitee.com/zhuyunkj/pay-gateway/api/internal/config"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/middleware"
	"github.com/zeromicro/go-zero/rest"
)

type ServiceContext struct {
	Config config.Config
	Inter  rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		Inter:  middleware.NewInterMiddleware().Handle,
	}
}
