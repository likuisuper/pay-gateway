package model

import (
	"gitee.com/zhuyunkj/pay-gateway/db"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"time"
)

var (
	refundOrderMysqlErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "refundOrderMysqlErr", nil, "退款订单数据库操作失败", nil})}
)

const (
	//退款状态
	PmRefundOrderTableRefundStatusApply   = 0
	PmRefundOrderTableRefundStatusSuccess = 1
	PmRefundOrderTableRefundStatusFail    = 2
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
	RefundedAt   int64     `gorm:"column:refunded_at;default:0;NOT NULL" json:"refunded_at"`     // 退款时间
	NotifyData   string    `gorm:"column:notify_data" json:"notify_data"`                        // 退款回调数据
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
		logx.Errorf("Create, info:%+v, err:=%v", info, err)
		refundOrderMysqlErr.CounterInc()
	}
	return err
}

//更新订单
func (o *PmRefundOrderModel) Update(outRefundNo string, info *PmRefundOrderTable) error {
	err := o.DB.Where("out_refund_no = ?", outRefundNo).Updates(info).Error
	if err != nil {
		logx.Errorf("Update, info:%+v, err:=%v", info, err)
		refundOrderMysqlErr.CounterInc()
	}
	return err
}

//获取订单
func (o *PmRefundOrderModel) GetInfo(outRefundNo string) (info *PmRefundOrderTable, err error) {
	info = new(PmRefundOrderTable)
	err = o.DB.Where("out_refund_no = ?", outRefundNo).Find(info).Error
	if err != nil {
		logx.Errorf("GetInfo, outRefundNo:%s, err:%v", outRefundNo, err)
		refundOrderMysqlErr.CounterInc()
	}
	return
}

func (o *PmRefundOrderModel) GetInfoByRefundNo(refundNo string) (info *PmRefundOrderTable, err error) {
	info = new(PmRefundOrderTable)
	err = o.DB.Where("refund_no = ?", refundNo).Find(info).Error
	if err != nil {
		logx.Errorf("GetInfo, refundNo:%s, err:%v", refundNo, err)
		refundOrderMysqlErr.CounterInc()
	}
	return
}
