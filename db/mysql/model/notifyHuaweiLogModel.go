package model

import (
	"time"

	"gitee.com/zhuyunkj/pay-gateway/db"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// NotifyHuaweiLogTable represents a notify_huawei_log struct data.
type NotifyHuaweiLogTable struct {
	Id        int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AppId     string    `gorm:"column:app_id" json:"appId"`         // 华为应用app_id
	AppPkg    string    `gorm:"column:app_pkg" json:"appPkg"`       // 华为应用包名
	Data      string    `gorm:"column:data" json:"data"`            // 华为通知回调内容原始内容
	CreatedAt time.Time `gorm:"column:created_at" json:"createdAt"` // 创建时间
}

func (m *NotifyHuaweiLogTable) TableName() string {
	return "notify_huawei_log"
}

type NotifyHuaweiLogModel struct {
	DB *gorm.DB
}

func NewNotifyHuaweiLogModel(dbName string) *NotifyHuaweiLogModel {
	return &NotifyHuaweiLogModel{
		DB: db.WithDBContext(dbName),
	}
}

// 创建记录
func (o *NotifyHuaweiLogModel) Create(info *NotifyHuaweiLogTable) error {
	err := o.DB.Create(info).Error
	if err != nil {
		logx.Errorf("创建失败 err:%v", err)
	}
	return err
}
