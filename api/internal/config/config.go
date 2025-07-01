package config

import (
	"github.com/zeromicro/go-zero/rest"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/cache"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/db"
)

type Config struct {
	rest.RestConf
	Mysql                  []*db.DbConfig        `json:"Mysql"`
	Nacos                  NacosConfig           `json:"Nacos"`
	RedisConfig            []*cache.RedisConfigs `json:"RedisConfig"`
	SnowFlake              SnowFlake             `json:"SnowFlake,optional"` //雪花算法参数
	Alarm                  Alarm                 //自定义告警
	BaseAppConfigServerUrl string                `json:"BaseAppConfigServerUrl"` // baseAppConfigServer地址
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

// 雪花算法参数
type SnowFlake struct {
	MachineNo int64 //工作ID
	WorkerNo  int64 //数据中心ID
}

type Alarm struct {
	Redis       cache.PublishRedisConfig
	DingDingUrl string
}
