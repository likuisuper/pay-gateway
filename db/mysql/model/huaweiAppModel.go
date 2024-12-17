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
	Id     int    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AppId  string `gorm:"column:app_id" json:"appId"`   // 华为应用app_id
	AppPkg string `gorm:"column:app_pkg" json:"appPkg"` // 华为应用包名
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
func (o *HuaweiAppModel) GetInfo(appId string) (HuaweiAppTable, error) {
	var info HuaweiAppTable

	if appId == "" {
		return info, errors.New("appid cannot be empty")
	}

	rkey := o.RDB.GetRedisKey(huawei_app_info_key, appId)
	err := o.RDB.GetObject(context.TODO(), rkey, &info)
	if err == nil {
		return info, nil
	}

	err = o.DB.Where("`app_id` = ? ", appId).First(&info).Error
	if err == nil {
		// 缓存一下
		o.RDB.Set(context.TODO(), rkey, info, 60)
	} else {
		logx.Errorf("GetInfo error: %v", err)
	}

	return info, err
}
