package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	Etcd EtcdConfig
}

type EtcdConfig struct {
	Host []string
}
