package model

import (
	"context"

	"gitee.com/zhuyunkj/pay-gateway/db"
	"gitee.com/zhuyunkj/zhuyun-core/cache"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// 支付宝关联app配置
type AppAlipayAppTable struct {
	ID     int    `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	AppID  string `gorm:"column:app_id;NOT NULL" json:"app_id"`   // 关联的配置ID
	AppPkg string `gorm:"column:app_pkg;NOT NULL" json:"app_pkg"` // 关联的应用包名
}

func (m *AppAlipayAppTable) TableName() string {
	return "app_alipay_app"
}

type AppAlipayAppModel struct {
	DB  *gorm.DB
	RDB *cache.RedisInstance
}

func NewAppAlipayAppModel(dbName string) *AppAlipayAppModel {
	return &AppAlipayAppModel{
		DB:  db.WithDBContext(dbName),
		RDB: db.WithRedisDBContext(dbName),
	}
}

// 获取应用配置信息
const app_alipay_app_list_key = "app:alipay:list:%s" // %s是包名
func (o *AppAlipayAppModel) GetValidConfig(appPkg string) (AppAlipayAppTable, error) {
	list := make([]AppAlipayAppTable, 0)

	rkey := o.RDB.GetRedisKey(app_alipay_app_list_key, appPkg)
	o.RDB.GetObject(context.TODO(), rkey, &list)
	if len(list) > 0 {
		// 缓存一下
		return list[0], nil
	}

	var tbl AppAlipayAppTable

	// status 状态（1：正常，2停用）
	// sort_no升序
	// relate_type 关联类型（1：备用，2：兜底）
	err := o.DB.Where("`app_pkg` = ? and `status` = 1", appPkg).Order("sort_no asc").Order("relate_type asc").Find(&list).Error
	if err != nil {
		logx.Errorf("获取关联的支付宝配置信息失败, pkg:%s, err:%v", appPkg, err)
		return tbl, err
	}

	if len(list) > 0 {
		tbl = list[0]
		// 缓存一下 10秒过期
		o.RDB.Set(context.TODO(), rkey, list, 5)
	}

	return tbl, nil
}
