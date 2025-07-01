package model

import (
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/code"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db"
	kv_m "gitlab.muchcloud.com/consumer-project/zhuyun-core/kv_monitor"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/util"
	"gorm.io/gorm"
)

var (
	createOrderErr       = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "createOrderErr", nil, "创建支付订单失败(新)", nil})}
	updateOrderNotifyErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "updateOrderNotifyErr", nil, "更新回调订单失败（新）", nil})}
	getOrderErr          = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "getOrderErr", nil, "获取支付订单失败（新）", nil})}
)

// 用户订单表
type OrderTable struct {
	ID                  int       `gorm:"column:id;primary_key;AUTO_INCREMENT"`
	AppPkg              string    `gorm:"column:app_pkg;NOT NULL"`                                 // 包名
	UserID              int       `gorm:"column:user_id;default:0;NOT NULL"`                       // 业务程序中的用户编号
	OutTradeNo          string    `gorm:"column:out_trade_no;NOT NULL"`                            // 内部订单号
	PlatformTradeNo     string    `gorm:"column:platform_trade_no;NOT NULL"`                       // 支付宝/微信等平台的订单号
	Amount              int       `gorm:"column:amount;default:0;NOT NULL"`                        // 支付金额 单位分
	Status              int       `gorm:"column:status;default:0;NOT NULL"`                        // -1:关闭，0:未支付，1:已支付，2:支付失败，3:已退款
	PayType             int       `gorm:"column:pay_type;default:0;NOT NULL"`                      // 支付类型（1微信，3支付宝）
	PayTime             time.Time `gorm:"column:pay_time;default:0000-00-00 00:00:00;NOT NULL"`    // 支付时间
	Subject             string    `gorm:"column:subject;NOT NULL"`                                 // 订单标题
	ProductType         int       `gorm:"column:product_type;default:0;NOT NULL"`                  // 商品类型，0:普通商品，1:会员商品，2:订阅商品，3:订阅商品续费
	ProductID           int       `gorm:"column:product_id;NOT NULL"`                              // 商品id
	ProductDesc         string    `gorm:"column:product_desc;NOT NULL"`                            // 商品信息描述(例如，使用AB配置的商品，可以将商品信息写在这)
	AppNotifyUrl        string    `gorm:"column:app_notify_url;NOT NULL"`                          // 业务回调通知
	AgreementNo         string    `gorm:"column:agreement_no;NOT NULL"`                            // 支付宝/微信平台订阅协议号
	ExternalAgreementNo string    `gorm:"column:external_agreement_no;NOT NULL"`                   // 内部协议号
	PayAppID            string    `gorm:"column:pay_app_id;NOT NULL"`                              // 第三方支付的appid
	DeviceId            string    `gorm:"column:device_id;NOT NULL"`                               // 用户设备号
	CreatedAt           time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;NOT NULL"`    // 创建时间
	UpdatedAt           time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP;NOT NULL"`    // 修改时间
	DeductTime          time.Time `gorm:"column:deduct_time;default:0000-00-00 00:00:00;NOT NULL"` // 可开始扣款时间(默认是0,不需要关注,只是为了满足产品延迟扣款的需求)
}

func (m *OrderTable) TableName() string {
	return "order"
}

type OrderModel struct {
	DB *gorm.DB
}

func NewOrderModel(dbName string) *OrderModel {
	return &OrderModel{
		DB: db.WithDBContext(dbName),
	}
}

// 创建订单
func (o *OrderModel) Create(info *OrderTable) error {
	err := o.DB.Create(info).Error
	if err != nil {
		logx.Errorf("创建支付订单失败 err:=%v", err)
		createOrderErr.CounterInc()
	}
	return err
}

// 获取订单信息
func (o *OrderModel) GetOneByOutTradeNo(outTradeNo string) (info *OrderTable, err error) {
	var orderInfo OrderTable
	err = o.DB.Where("`out_trade_no` = ? ", outTradeNo).First(&orderInfo).Error
	if err != nil {
		logx.Errorf("获取订单信息失败 err:%v, out_trade_no:%s", err, outTradeNo)
		getOrderErr.CounterInc()
		return nil, err
	}

	return &orderInfo, nil
}

// 根据协议号获取订单信息
func (o *OrderModel) GetOneByExternalAgreementNo(externalAgreementNo string) (info *OrderTable, err error) {
	var orderInfo OrderTable
	err = o.DB.Where("`external_agreement_no` = ? and `product_type` = ?", externalAgreementNo, code.PRODUCT_TYPE_SUBSCRIBE).First(&orderInfo).Error
	if err != nil {
		logx.Errorf("获取订单信息失败 err:%v, external_agreement_no:%s", err, externalAgreementNo)
		getOrderErr.CounterInc()
	}

	return &orderInfo, nil
}

func (o *OrderModel) UpdateNotify(info *OrderTable) error {
	err := o.DB.Save(&info).Error
	if err != nil {
		logx.Errorf("更新回调订单失败，err=%v", err)
		updateOrderNotifyErr.CounterInc()
	}
	return err
}

func (o *OrderModel) UpdatePayAppID(tradeNo string, payAppId string) (err error) {
	err = o.DB.Model(&OrderTable{}).Where("trade_no = ?", tradeNo).Update("app_id", payAppId).Error
	if err != nil {
		err = fmt.Errorf("UpdatePayAppID Err: %v", err)
		util.CheckError(err.Error())
	}
	return
}

func (o *OrderModel) GetFirstUnpaidSubscribeFee() (table *OrderTable, err error) {
	// 订阅状态：0未签约，1签约成功，2失效
	err = o.DB.Where(" product_type = ? and status = 0 ", code.PRODUCT_TYPE_SUBSCRIBE_FEE).Order("id asc").First(&table).Error
	return
}

// 一次批量取数据条数
const VIP_DATA_ONCE_LIMIT = 100

func (o *OrderModel) GetRangeData(id int) (records []*OrderTable, err error) {
	// 只取最近30天的
	err = o.DB.Where("product_type = ? and status = 0 and id > ? and deduct_time < ? and created_at >= ?", code.PRODUCT_TYPE_SUBSCRIBE_FEE, id, time.Now(), time.Now().AddDate(0, 0, -30)).
		Order("id asc").
		Limit(VIP_DATA_ONCE_LIMIT).
		Find(&records).Error
	return
}

func (o *OrderModel) UpdateStatusByOutTradeNo(outTradeNo string, status int) error {
	err := o.DB.Table("order").Where("`out_trade_no` = ? ", outTradeNo).Updates(map[string]interface{}{
		"status": status,
	}).Error
	if err != nil {
		logx.Errorf("UpdateStatusByOutTradeNo err:%v", err)
		updateOrderNotifyErr.CounterInc()
	}
	return err
}

// 根据协议号关闭续费订单
func (o *OrderModel) CloseUnpaidSubscribeFeeOrderByExternalAgreementNo(externalAgreementNo string) (err error) {
	err = o.DB.Table("order").Where("`external_agreement_no` = ? and `product_type` = ? and `status` = ?", externalAgreementNo, code.PRODUCT_TYPE_SUBSCRIBE_FEE, 0).
		Update("`status`", -1).Error
	if err != nil {
		logx.Errorf("更新续费订单信息失败 err:%v, external_agreement_no:%s", err, externalAgreementNo)
		getOrderErr.CounterInc()
	}
	return err
}
