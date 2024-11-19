package model

import (
	"time"

	"gitee.com/zhuyunkj/pay-gateway/db"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// 支付宝订阅成功失败冗余表
type AppAlipayOrderTable struct {
	ID        int       `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	AppPkg    string    `gorm:"column:app_pkg;NOT NULL" json:"app_pkg"`     // 应用包名
	AppId     string    `gorm:"column:app_id;NOT NULL" json:"app_id"`       // 支付APPID
	DeviceId  string    `gorm:"column:device_id;NOT NULL" json:"device_id"` // 用户设备号
	Code      int       `gorm:"column:code;NOT NULL" json:"code"`           // 支付宝返回的状态码
	PayMoney  int       `gorm:"column:pay_money;NOT NULL" json:"pay_money"` // 支付金额
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (m *AppAlipayOrderTable) TableName() string {
	return "app_alipay_order"
}

type AppAlipayOrderModel struct {
	DB *gorm.DB
	// RDB *cache.RedisInstance
}

func NewAppAlipayOrderModel(dbName string) *AppAlipayOrderModel {
	return &AppAlipayOrderModel{
		DB: db.WithDBContext(dbName),
		// RDB: db.WithRedisDBContext(dbName),
	}
}

func (o *AppAlipayOrderModel) Create(info *AppAlipayOrderTable) (err error) {
	err = o.DB.Create(info).Error
	if err != nil {
		logx.Errorf("创建app_alipay_order失败 err=%v", err)
	}

	return err
}
