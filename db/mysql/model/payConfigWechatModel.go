package model

import (
	"context"

	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/db"
	"gitee.com/zhuyunkj/zhuyun-core/cache"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

var (
	getPayConfigWechatErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "getPayConfigWechatErr", nil, "获取微信支付配置失败", nil})}
)

// 微信支付配置
type PmPayConfigWechatTable struct {
	ID             int    `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	MchID          string `gorm:"column:mch_id;NOT NULL" json:"mch_id"`                     // 商户id
	AppID          string `gorm:"column:app_id;NOT NULL" json:"app_id"`                     // 应用id
	ApiKey         string `gorm:"column:api_key;NOT NULL" json:"api_key"`                   // apiV3密钥
	ApiKeyV2       string `gorm:"column:api_key_v2;NOT NULL" json:"api_key_v2"`             // apiV2密钥
	NotifyUrl      string `gorm:"column:notify_url" json:"notify_url"`                      // 回调地址
	PrivateKeyPath string `gorm:"column:private_key_path;NOT NULL" json:"private_key_path"` // apiV3密钥
	PublicKeyId    string `gorm:"column:public_key_id;NOT NULL" json:"public_key_id"`       // 公钥id
	PublicKeyPath  string `gorm:"column:public_key_path;NOT NULL" json:"public_key_path"`   // 公钥文件路径
	SerialNumber   string `gorm:"column:serial_number;NOT NULL" json:"serial_number"`       // 商户证书序列号
	Remark         string `gorm:"column:remark;NOT NULL" json:"remark"`                     // 备注信息
	XPayAppKey     string `gorm:"column:xpay_appkey;NOT NULL" json:"xpay_appkey"`           // 虚拟支付现网AppKey
	PlatformNumer  string `gorm:"column:platform_numer;NOT NULL" json:"platform_numer"`     // 微信支付平台证书编号
	WapUrl         string `gorm:"column:wap_url" json:"wap_url"`                            // 支付H5域名
	WapName        string `gorm:"column:wap_name" json:"wap_name"`                          // 支付名称
	// CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	// UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (m *PmPayConfigWechatTable) TableName() string {
	return "pm_pay_config_wechat"
}

func (m *PmPayConfigWechatTable) TransClientConfig() (clientCfg *client.WechatPayConfig) {
	clientCfg = &client.WechatPayConfig{
		AppId:          m.AppID,
		MchId:          m.MchID,
		ApiKey:         m.ApiKey,
		PrivateKeyPath: m.PrivateKeyPath,
		PublicKeyId:    m.PublicKeyId,
		PublicKeyPath:  m.PublicKeyPath,
		SerialNumber:   m.SerialNumber,
		NotifyUrl:      m.NotifyUrl,
		ApiKeyV2:       m.ApiKeyV2,
		WapName:        m.WapName,
		WapUrl:         m.WapUrl,
		PlatformNumer:  m.PlatformNumer,
	}
	return
}

type PmPayConfigWechatModel struct {
	DB  *gorm.DB
	RDB *cache.RedisInstance
}

func NewPmPayConfigWechatModel(dbName string) *PmPayConfigWechatModel {
	return &PmPayConfigWechatModel{
		DB:  db.WithDBContext(dbName),
		RDB: db.WithRedisDBContext(dbName),
	}
}

// 获取应用配置信息
const pm_pay_config_wechat_cache_key = "pm:pay:config:wechat:cache:%s" // %s是appid
func (o *PmPayConfigWechatModel) GetOneByAppID(appID string) (*PmPayConfigWechatTable, error) {
	var cfg PmPayConfigWechatTable

	rkey := o.RDB.GetRedisKey(pm_pay_config_wechat_cache_key, appID)
	err := o.RDB.GetObject(context.Background(), rkey, &cfg)
	if err == nil && cfg.ID > 0 {
		return &cfg, nil
	}

	err = o.DB.Where(" `app_id` = ?", appID).First(&cfg).Error
	if err != nil {
		logx.Errorf("获取wechatPay配置信息失败 err:=%v,appID=%s", err, appID)
		getPayConfigWechatErr.CounterInc()
		return nil, err
	}

	// 设置缓存时间为3分钟
	o.RDB.Set(context.Background(), rkey, cfg, 180)

	return &cfg, nil
}

// 获取微信配置列表
func (o *PmPayConfigWechatModel) GetAllList() (wechatCfgList []*PmPayConfigWechatTable, err error) {
	wechatCfgList = make([]*PmPayConfigWechatTable, 0)

	err = o.DB.Find(&wechatCfgList).Error
	if err != nil {
		logx.Errorf("获取wechatPay配置信息失败，err:=%v,appID=%s", err, "all")
		getPayConfigWechatErr.CounterInc()
		return nil, err
	}
	return
}
