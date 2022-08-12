package config

import (
	"gitee.com/zhuyunkj/zhuyun-core/db"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Mysql []*db.DbConfig `json:"Mysql"`
	////支付配置
	//Alipay    []*AliPay
	//WeChatPay []*WechatPay
	//TikTokPay []*TikTokPay

	//AppRel []*AppRelConfig //包名对应的支付配置
}

////支付宝参数
//type AliPay struct {
//	AppId            string
//	PrivateKey       string
//	PublicKey        string
//	AppCertPublicKey string
//	PayRootCert      string
//	IsProduction     bool
//}
//
////字节支付参数
//type TikTokPay struct {
//	AppId     string //应用ID
//	SALT      string //加密参数
//	NotifyUrl string //通知地址
//	Token     string //token
//}
//
////微信支付参数
//type WechatPay struct {
//	AppId          string //应用ID
//	MchId          string //直连商户号
//	ApiKey         string //apiV3密钥
//	PrivateKeyPath string //apiV3密钥
//	SerialNumber   string //商户证书序列号
//	NotifyUrl      string //通知地址
//}

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
