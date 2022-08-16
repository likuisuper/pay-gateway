package config

import (
	"gitee.com/zhuyunkj/zhuyun-core/db"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Etcd  EtcdConfig
	Mysql []*db.DbConfig `json:"Mysql"`
}

type EtcdConfig struct {
	Host []string
}
