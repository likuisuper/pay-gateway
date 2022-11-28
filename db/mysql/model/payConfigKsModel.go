package model

import (
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/db"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"time"
)

var (
	getPayConfigKsErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "getPayConfigKsErr", nil, "获取快手支付配置失败", nil})}
)

// 快手支付配置
type PmPayConfigKsTable struct {
	ID        int       `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	AppID     string    `gorm:"column:app_id;NOT NULL" json:"app_id"`         // 应用id
	AppSecret string    `gorm:"column:app_secret;NOT NULL" json:"app_secret"` // 应用secret
	NotifyUrl string    `gorm:"column:notify_url;NOT NULL" json:"notify_url"` // 回调地址
	Remark    string    `gorm:"column:remark;NOT NULL" json:"remark"`         // 备注信息
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (m *PmPayConfigKsTable) TableName() string {
	return "pm_pay_config_ks"
}

func (m *PmPayConfigKsTable) TransClientConfig() (clientCfg *client.KsPayConfig) {
	clientCfg = &client.KsPayConfig{
		AppId:     m.AppID,
		AppSecret: m.AppSecret,
		NotifyUrl: m.NotifyUrl,
	}
	return
}

type PmPayConfigKsModel struct {
	DB *gorm.DB
}

func NewPmPayConfigKsModel(dbName string) *PmPayConfigKsModel {
	return &PmPayConfigKsModel{
		DB: db.WithDBContext(dbName),
	}
}

//获取应用配置信息
func (o *PmPayConfigKsModel) GetOneByAppID(appID string) (appConfig *PmPayConfigKsTable, err error) {
	var cfg PmPayConfigKsTable
	err = o.DB.Where(" `app_id` = ?", appID).First(&cfg).Error
	if err != nil {
		logx.Errorf("获取app配置信息失败，err:=%v,appID=%s", err, appID)
		getPayConfigKsErr.CounterInc()
		return nil, err
	}
	return &cfg, nil
}
