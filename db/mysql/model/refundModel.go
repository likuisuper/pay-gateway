package model

import (
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db"
	kv_m "gitlab.muchcloud.com/consumer-project/zhuyun-core/kv_monitor"
	"gorm.io/gorm"
)

var (
	// createRefundOrderErr       = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "createRefundOrderErr", nil, "创建退款订单失败(新)", nil})}
	updateRefundOrderNotifyErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "updateRefundOrderNotifyErr", nil, "更新回调退款订单失败（新）", nil})}
	getRefundOrderErr          = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "getRefundOrderErr", nil, "获取退款订单失败（新）", nil})}
)

const (
	REFUND_STATUS_NONE    = 0
	REFUND_STATUS_SUCCESS = 1
	REFUND_STATUS_FAILD   = 2
)

type RefundTable struct {
	ID               int       `gorm:"column:id;primary_key;AUTO_INCREMENT"`
	AppPkg           string    `gorm:"column:app_pkg;NOT NULL"`                                 // 包名
	PayType          int       `gorm:"column:pay_type;NOT NULL"`                                // 支付类型（1:支付宝，2微信）
	OutTradeNo       string    `gorm:"column:out_trade_no;NOT NULL"`                            // 商户订单号
	OutTradeRefundNo string    `gorm:"column:out_trade_refund_no;NOT NULL"`                     // 商户退款单号
	Reason           string    `gorm:"column:reason;NOT NULL"`                                  // 退款原因
	RefundAmount     int       `gorm:"column:refund_amount;default:0;NOT NULL"`                 // 退款金额，单位分
	RefundStatus     int       `gorm:"column:refund_status;NOT NULL"`                           // 0申请中  1成功  2失败
	RefundNo         string    `gorm:"column:refund_no;NOT NULL"`                               // 平台退款单号
	RefundedAt       time.Time `gorm:"column:refunded_at;default:0000-00-00 00:00:00;NOT NULL"` // 退款时间
	NotifyUrl        string    `gorm:"column:notify_url;NOT NULL"`                              // 回调地址
	NotifyData       string    `gorm:"column:notify_data;NOT NULL"`                             // 退款回调数据
	Operator         string    `gorm:"column:operator;NOT NULL"`                                // 操作者
	Reviewer         string    `gorm:"column:reviewer;NOT NULL"`                                // 审核人员
	ReviewerComment  string    `gorm:"column:reviewer_comment;NOT NULL"`                        // 审核人员备注
	CreatedAt        time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;NOT NULL"`    // 创建时间
	UpdatedAt        time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP;NOT NULL"`    // 更新时间
}

func (m *RefundTable) TableName() string {
	return "refund"
}

type RefundModel struct {
	DB *gorm.DB
}

func NewRefundModel(dbName string) *RefundModel {
	return &RefundModel{
		DB: db.WithDBContext(dbName),
	}
}

// 创建订单
func (o *RefundModel) Create(info *RefundTable) error {
	err := o.DB.Create(info).Error
	if err != nil {
		logx.Errorf("创建支付订单失败，err:=%v", err)
		getRefundOrderErr.CounterInc()
	}
	return err
}

// 更新订单
func (o *RefundModel) Update(outRefundNo string, info *RefundTable) error {
	err := o.DB.Where("out_trade_refund_no = ?", outRefundNo).Updates(info).Error
	if err != nil {
		logx.Errorf("Update, info:%+v, err:=%v", info, err)
		updateRefundOrderNotifyErr.CounterInc()
	}
	return err
}

// 获取订单信息
func (o *RefundModel) GetOneByOutTradeRefundNo(outTradeRefundNo string) (info *RefundTable, err error) {
	var refundInfo RefundTable
	err = o.DB.Where("`out_trade_refund_no` = ? ", outTradeRefundNo).First(&refundInfo).Error
	if err != nil {
		logx.Errorf("获取退款订单信息失败 err:=%v, out_trade_refund_no=%s", err, outTradeRefundNo)
		getRefundOrderErr.CounterInc()
		return nil, err
	}

	return &refundInfo, nil
}

// 获取订单信息
func (o *RefundModel) GetOneByOutTradeNo(outTradeNo string) (info *RefundTable, err error) {
	var refundInfo RefundTable
	err = o.DB.Where("`out_trade_no` = ? ", outTradeNo).First(&refundInfo).Error
	if err != nil {
		logx.Errorf("获取订单信息失败 err:%v, out_trade_no:%s", err, outTradeNo)
		getRefundOrderErr.CounterInc()
		return nil, err
	}

	return &refundInfo, nil
}

func (o *RefundModel) UpdateNotify(info *RefundTable) error {
	err := o.DB.Save(&info).Error
	if err != nil {
		logx.Errorf("更新回调退款订单失败 err=%v", err)
		updateRefundOrderNotifyErr.CounterInc()
	}
	return err
}
