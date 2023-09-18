package config

import (
	"gitee.com/zhuyunkj/zhuyun-core/cache"
	"gitee.com/zhuyunkj/zhuyun-core/db"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Mysql       []*db.DbConfig        `json:"Mysql"`
	Nacos       NacosConfig           `json:"Nacos"`
	RedisConfig []*cache.RedisConfigs `json:"RedisConfig"`
}

// nacos配置
type NacosConfig struct {
	NacosService []NacosService
	NamespaceId  string
	TimeoutMs    uint64
	Username     string
	Password     string
}

type NacosService struct {
	Ip   string
	Port uint64
}
