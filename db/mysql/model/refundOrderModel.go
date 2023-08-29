package model

import (
	"gitee.com/zhuyunkj/pay-gateway/db"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"time"
)

var (
	createRefundOrderErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "createRefundOrderErr", nil, "创建退款订单失败", nil})}
)

// 退款订单
type PmRefundOrderTable struct {
	ID           int       `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	AppID        string    `gorm:"column:app_id;NOT NULL" json:"app_id"`                         // 小程序APPID
	OutOrderNo   string    `gorm:"column:out_order_no;NOT NULL" json:"out_order_no"`             // 商户分配支付单号，标识进行退款的订单
	OutRefundNo  string    `gorm:"column:out_refund_no;NOT NULL" json:"out_refund_no"`           // 商户分配退款号，保证在商户中唯一
	Reason       string    `gorm:"column:reason;NOT NULL" json:"reason"`                         // 退款原因
	RefundAmount int       `gorm:"column:refund_amount;default:0;NOT NULL" json:"refund_amount"` // 退款金额，单位分
	NotifyUrl    string    `gorm:"column:notify_url;NOT NULL" json:"notify_url"`                 // 回调应用机地址
	RefundNo     string    `gorm:"column:refund_no;NOT NULL" json:"refund_no"`                   // 担保交易服务端退款单号
	RefundStatus int       `gorm:"column:refund_status;default:0;NOT NULL" json:"refund_status"` // 0申请中  1成功  2失败
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (m *PmRefundOrderTable) TableName() string {
	return "pm_refund_order"
}

type PmRefundOrderModel struct {
	DB *gorm.DB
}

func NewPmRefundOrderModel(dbName string) *PmRefundOrderModel {
	return &PmRefundOrderModel{
		DB: db.WithDBContext(dbName),
	}
}

//创建订单
func (o *PmRefundOrderModel) Create(info *PmRefundOrderTable) error {
	err := o.DB.Create(info).Error
	if err != nil {
		logx.Errorf("创建退款订单失败，err:=%v", err)
		createPayOrderErr.CounterInc()
	}
	return err
}
