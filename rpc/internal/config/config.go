package config

import (
	"gitee.com/zhuyunkj/zhuyun-core/db"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Mysql []*db.DbConfig `json:"Mysql"`
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

//// 应用包名  对应的配置
//type AppRelConfig struct {
//	AppPkgName     string //应用包名
//	AlipayAppId    string //对应的支付宝appid
//	WechatPayAppId string //对应的微信支付appid
//	TikTokPayAppId string //对应的字节支付appid
//}
