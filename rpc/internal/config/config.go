package config

import (
	"gitee.com/zhuyunkj/zhuyun-core/cache"
	"gitee.com/zhuyunkj/zhuyun-core/db"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Mysql       []*db.DbConfig `json:"Mysql"`
	Nacos       NacosConfig
	RedisConfig []*cache.RedisConfigs `json:"RedisConfig"`
	SnowFlake   SnowFlake             `json:"SnowFlake,optional"` //雪花算法参数
}

// mysql配置
type DbConfig struct {
	Name         string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  int
	Debug        bool
	Charset      string
	Domain       string
	Port         int
	Dbname       string
	Username     string
	Passwd       string
	ConnTimeout  int
	ReadTimeout  int
	WriteTimeout int
}

// redis 配置
type RedisConfig struct {
	MaxActive   int
	MaxIdle     int
	IdleTimeout int
	Address     string
	Passwd      string
}

// nacos配置
type NacosConfig struct {
	NacosService        []NacosService
	ListenOn            string
	NamespaceId         string
	TimeoutMs           uint64
	NotLoadCacheAtStart bool
	LogDir              string
	CacheDir            string
	Username            string
	Password            string
	LogLevel            string
	MaxAge              int
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
