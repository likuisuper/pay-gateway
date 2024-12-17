package model

import (
	"context"
	"errors"

	"gitee.com/zhuyunkj/pay-gateway/db"
	"gitee.com/zhuyunkj/zhuyun-core/cache"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// 华为配置了订阅的应用信息表
type HuaweiAppTable struct {
	ID                int    `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	AppID             string `gorm:"column:app_id;NOT NULL" json:"app_id"`                         // 华为配置ID
	AppPkg            string `gorm:"column:app_pkg;NOT NULL" json:"app_pkg"`                       // 华为应用包名
	AppSecret         string `gorm:"column:app_secret;NOT NULL" json:"app_secret"`                 // 应用公钥 华为后台:我的项目-常规-开发者-验证公钥, 从后台拷贝的就是base64编码了的
	ClientId          string `gorm:"column:client_id;NOT NULL" json:"client_id"`                   // 应用client_id 华为后台:我的项目-常规-应用-Client ID
	ClientSecret      string `gorm:"column:client_secret;NOT NULL" json:"client_secret"`           // 应用client_secret 华为后台:我的项目-常规-应用-Client Secret
	Sha256Fingerprint string `gorm:"column:sha256_fingerprint;NOT NULL" json:"sha256_fingerprint"` // sha256证书指纹 华为后台:我的项目-常规-应用-SHA256证书指纹
}

func (m *HuaweiAppTable) TableName() string {
	return "huawei_app"
}

type HuaweiAppModel struct {
	DB  *gorm.DB
	RDB *cache.RedisInstance
}

func NewHuaweiAppModel(dbName string) *HuaweiAppModel {
	return &HuaweiAppModel{
		DB:  db.WithDBContext(dbName),
		RDB: db.WithRedisDBContext(dbName),
	}
}

// 查询订阅了的华为应用信息
const huawei_app_info_key = "hw:sub:app:info:%s" // %s是appid
func (o *HuaweiAppModel) GetInfo(appId string) (*HuaweiAppTable, error) {
	var info HuaweiAppTable

	if appId == "" {
		return &info, errors.New("appid cannot be empty")
	}

	rkey := o.RDB.GetRedisKey(huawei_app_info_key, appId)
	err := o.RDB.GetObject(context.TODO(), rkey, &info)
	if err == nil {
		return &info, nil
	}

	err = o.DB.Where("`app_id` = ? ", appId).First(&info).Error
	if err == nil {
		// 缓存一下
		o.RDB.Set(context.TODO(), rkey, info, 300)
	} else {
		logx.Errorf("GetInfo error: %v", err)
	}

	return &info, err
}
