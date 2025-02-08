package model

import (
	"context"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/db"
	"gitee.com/zhuyunkj/zhuyun-core/cache"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

var (
	getAppConfigErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "getAppConfigErr", nil, "获取app配置信息失败", nil})}
)

type PmAppConfigTable struct {
	ID             int       `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	AppPkgName     string    `gorm:"column:app_pkg_name;NOT NULL" json:"app_pkg_name"`           // 应用包名
	AlipayAppID    string    `gorm:"column:alipay_app_id;NOT NULL" json:"alipay_app_id"`         // 对应的支付宝appid
	WechatPayAppID string    `gorm:"column:wechat_pay_app_id;NOT NULL" json:"wechat_pay_app_id"` // 对应的微信支付appid
	TiktokPayAppID string    `gorm:"column:tiktok_pay_app_id;NOT NULL" json:"tiktok_pay_app_id"` // 对应的字节支付appid
	KsPayAppID     string    `gorm:"column:ks_pay_app_id;NOT NULL" json:"ks_pay_app_id"`         // 对应的快手支付appid
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (m *PmAppConfigTable) TableName() string {
	return "pm_app_config"
}

type PmAppConfigModel struct {
	DB  *gorm.DB
	RDB *cache.RedisInstance
}

func NewPmAppConfigModel(dbName string) *PmAppConfigModel {
	return &PmAppConfigModel{
		DB:  db.WithDBContext(dbName),
		RDB: db.WithRedisDBContext(dbName),
	}
}

// 获取应用配置信息
const pm_app_config_cache_key = "pm:app:config:cache:%s" // %s是包名
func (o *PmAppConfigModel) GetOneByPkgName(pkgName string) (appConfig *PmAppConfigTable, err error) {
	var cfg PmAppConfigTable

	rkey := o.RDB.GetRedisKey(pm_app_config_cache_key, pkgName)
	err = o.RDB.GetObject(context.Background(), rkey, &cfg)
	if err == nil && cfg.ID > 0 {
		return &cfg, nil
	}

	err = o.DB.Where(" `app_pkg_name` = ?", pkgName).First(&cfg).Error
	if err != nil {
		logx.Errorf("获取app配置信息失败，err:=%v,pkg=%s", err, pkgName)
		getPayOrderErr.CounterInc()
		return nil, err
	}

	// 设置缓存时间为3分钟
	o.RDB.Set(context.Background(), rkey, cfg, 180)

	return &cfg, nil
}
