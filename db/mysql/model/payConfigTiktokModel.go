package model

import (
	"github.com/zeromicro/go-zero/core/logx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/client"
	douyin "gitlab.muchcloud.com/consumer-project/pay-gateway/common/client/douyinGeneralTrade"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db"
	kv_m "gitlab.muchcloud.com/consumer-project/zhuyun-core/kv_monitor"
	"gorm.io/gorm"
)

var (
	getPayConfigTiktokErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "getPayConfigTiktokErr", nil, "获取抖音支付配置失败", nil})}
)

// 字节支付配置
type PmPayConfigTiktokTable struct {
	ID                 int    `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	AppID              string `gorm:"column:app_id;NOT NULL" json:"app_id"`                               // 应用id
	Salt               string `gorm:"column:salt;NOT NULL" json:"salt"`                                   // 加密参数
	NotifyUrl          string `gorm:"column:notify_url;NOT NULL" json:"notify_url"`                       // 回调地址
	Token              string `gorm:"column:token;NOT NULL" json:"token"`                                 // token
	Remark             string `gorm:"column:remark;NOT NULL" json:"remark"`                               // 备注信息
	PrivateKey         string `gorm:"column:private_key" json:"private_key"`                              // 私钥
	KeyVersion         string `gorm:"column:key_version" json:"key_version"`                              // 私钥版本号
	PlatformPublicKey  string `gorm:"column:platform_public_key" json:"platform_public_key"`              // 平台公钥
	CustomerImId       string `gorm:"column:customer_im_id" json:"customer_im_id"`                        // 抖音客服id 用于ios支付
	MerchantUid        string `gorm:"column:merchant_uid;NOT NULL" json:"merchant_uid"`                   // 自定义的商户号
	SignPayMerchantUid string `gorm:"column:sign_pay_merchant_uid;NOT NULL" json:"sign_pay_merchant_uid"` // 抖音代扣收款商户号 一般跟merchant_uid一样
	// CreatedAt          time.Time `gorm:"column:created_at" json:"created_at"`
	// UpdatedAt          time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (m *PmPayConfigTiktokTable) TableName() string {
	return "pm_pay_config_tiktok"
}

func (m *PmPayConfigTiktokTable) TransClientConfig() (clientCfg *client.TikTokPayConfig) {
	clientCfg = &client.TikTokPayConfig{
		AppId:     m.AppID,
		SALT:      m.Salt,
		NotifyUrl: m.NotifyUrl,
		Token:     m.Token,
	}
	return
}

func (m *PmPayConfigTiktokTable) GetGeneralTradeConfig() (clientCfg *douyin.PayConfig) {
	clientCfg = &douyin.PayConfig{
		AppId:             m.AppID,
		PrivateKey:        m.PrivateKey,
		KeyVersion:        m.KeyVersion,
		NotifyUrl:         m.NotifyUrl,
		PlatformPublicKey: m.PlatformPublicKey,
		CustomerImId:      m.CustomerImId,
		MerchantUid:       m.MerchantUid,
	}
	return
}

type PmPayConfigTiktokModel struct {
	DB *gorm.DB
}

func NewPmPayConfigTiktokModel(dbName string) *PmPayConfigTiktokModel {
	return &PmPayConfigTiktokModel{
		DB: db.WithDBContext(dbName),
	}
}

// 获取应用配置信息
func (o *PmPayConfigTiktokModel) GetOneByAppID(appID string) (appConfig *PmPayConfigTiktokTable, err error) {
	var cfg PmPayConfigTiktokTable
	err = o.DB.Where(" `app_id` = ?", appID).First(&cfg).Error
	if err != nil {
		logx.Errorf("获取app配置信息失败，err:=%v,appID=%s", err, appID)
		getPayConfigTiktokErr.CounterInc()
		return nil, err
	}
	return &cfg, nil
}
