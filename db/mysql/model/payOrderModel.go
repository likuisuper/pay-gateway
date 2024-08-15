package model

import (
	"errors"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/db"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"time"
)

var (
	createPayOrderErr    = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "createPayOrderErr", nil, "创建支付订单失败", nil})}
	updateNofityOrderErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "updateNofityOrderErr", nil, "更新回调订单失败", nil})}
	getPayOrderErr       = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "getPayOrderErr", nil, "获取支付订单失败", nil})}
)

var NoNeedSupplementaryError = errors.New("order has been handled")

const (
	// 支付状态
	PmPayOrderTablePayStatusNo     = 0 // 未支付
	PmPayOrderTablePayStatusPaid   = 1 // 支付成功
	PmPayOrderTablePayStatusFailed = 2 // 支付失败
	PmPayOrderTablePayStatusRefund = 3 // 退款
	// 支付方式
	PmPayOrderTablePayTypeWechatPayUni       = 1 // 微信JSAPI支付
	PmPayOrderTablePayTypeTiktokPayEc        = 2
	PmPayOrderTablePayTypeAlipay             = 3
	PmPayOrderTablePayTypeKs                 = 4 //已废弃，pb入参，4为 PayType_WxWeb
	PmPayOrderTablePayTypeWechatPayH5        = 5 // 暂时没用，微信H5支付，pb入参 5为 PayType_KsUniApp
	PmPayOrderTablePayWxUnified              = 6 //微信统一下单接口 ,暂未用到回调接口被误用为8
	PmPayOrderTablePayWxV3H5                 = 7 // 微信h5支付
	PmPayOrderTablePayTypeDouyinGeneralTrade = 8 //抖音小程序支付-通用交易系统,由6调整为8和Pb入参一致

)

// 支付订单
type PmPayOrderTable struct {
	ID           uint      `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	OrderSn      string    `gorm:"column:order_sn;NOT NULL" json:"order_sn"`                     // 订单唯一标识
	AppPkgName   string    `gorm:"column:app_pkg_name;NOT NULL" json:"app_pkg_name"`             // 来源包名
	Amount       int       `gorm:"column:amount;default:0;NOT NULL" json:"amount"`               // 订单金额（分）
	NotifyAmount int       `gorm:"column:notify_amount;default:0;NOT NULL" json:"notify_amount"` // 回调金额（分）
	Subject      string    `gorm:"column:subject;NOT NULL" json:"subject"`                       // 订单标题
	PayType      int       `gorm:"column:pay_type;default:0;NOT NULL" json:"pay_type"`           // 支付方式  1微信小程序支付 2头条小程序支付
	NotifyUrl    string    `gorm:"column:notify_url;NOT NULL" json:"notify_url"`                 // 回调通知地址
	PayStatus    int       `gorm:"column:pay_status;NOT NULL" json:"pay_status"`                 // 支付状态 0未支付  1已支付
	PayAppId     string    `gorm:"column:pay_app_id;NOT NULL" json:"pay_app_id"`                 //第三方支付的appid
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:datetime" json:"updated_at"`
	ThirdOrderNo string    `gorm:"column:third_order_no;NULL" json:"third_order_no"` //三方订单号
	Currency     string    `gorm:"column:currency;type:varchar(16);NOT NULL"`        // 支付币种
}

func (m *PmPayOrderTable) TableName() string {
	return "pm_pay_order"
}

type PmPayOrderModel struct {
	DB *gorm.DB
}

func NewPmPayOrderModel(dbName string) *PmPayOrderModel {
	return &PmPayOrderModel{
		DB: db.WithDBContext(dbName),
	}
}

// 创建订单
func (o *PmPayOrderModel) Create(info *PmPayOrderTable) error {
	err := o.DB.Create(info).Error
	if err != nil {
		logx.Errorf("创建支付订单失败，err:=%v", err)
		createPayOrderErr.CounterInc()
	}
	return err
}

// GetOneByCode 获取订单信息
func (o *PmPayOrderModel) GetOneByCode(orderSn string) (info *PmPayOrderTable, err error) {
	var orderInfo PmPayOrderTable
	err = o.DB.Where("`order_sn` = ? ", orderSn).First(&orderInfo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		logx.Errorf("获取订单信息失败，err:=%v,order_sn=%s", err, orderSn)
		getPayOrderErr.CounterInc()
		return nil, err
	}
	return &orderInfo, nil
}

// GetOneByOrderSnAndAppId 根据订单号和包名获取订单信息
func (o *PmPayOrderModel) GetOneByOrderSnAndAppId(orderSn, appId string) (info *PmPayOrderTable, err error) {
	var orderInfo PmPayOrderTable
	err = o.DB.Where("`order_sn` = ? and pay_app_id = ?", orderSn, appId).First(&orderInfo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		logx.Errorf("GetOneByOrderSnAndPkgName 获取订单信息失败，err:=%v,order_sn=%s", err, orderSn)
		getPayOrderErr.CounterInc()
		return nil, err
	}
	return &orderInfo, nil
}

// QueryAfterUpdate 查询后修改订单状态
func (o *PmPayOrderModel) QueryAfterUpdate(orderSn, appId, thirdOrderNo string, totalAmount int) (bool, error) {
	var orderInfo PmPayOrderTable
	tx := o.DB.Begin()
	err := o.DB.Where("`order_sn` = ? and  pay_app_id = ? ", orderSn, appId).First(&orderInfo).Error
	if err != nil {
		tx.Rollback()
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logx.Errorf("QueryAfterUpdate:获取订单信息失败，err:=%v,order_sn=%s", err, orderSn)
			getPayOrderErr.CounterInc()
		}
		return false, err
	}

	if orderInfo.PayStatus != PmPayOrderTablePayStatusNo { //订单已被处理
		tx.Rollback()
		return false, NoNeedSupplementaryError
	}

	orderInfo.NotifyAmount = totalAmount
	orderInfo.PayStatus = PmPayOrderTablePayStatusPaid
	orderInfo.ThirdOrderNo = thirdOrderNo
	err = o.DB.Save(&orderInfo).Error
	if err != nil {
		tx.Rollback()
		logx.Errorf("QueryAfterUpdate:更新回调订单失败，err=%v", err)
		updateNofityOrderErr.CounterInc()
		return false, err
	}

	//正常逻辑
	tx.Commit()
	return true, nil
}

func (o *PmPayOrderModel) UpdateNotify(info *PmPayOrderTable) error {
	err := o.DB.Save(&info).Error
	if err != nil {
		logx.Errorf("更新回调订单失败，err=%v", err)
		updateNofityOrderErr.CounterInc()
	}
	return err
}

func (o *PmPayOrderModel) UpdatePayAppID(orderSn string, payAppId string) (err error) {
	err = o.DB.Model(&PmPayOrderTable{}).Where("order_sn = ?", orderSn).Update("pay_app_id", payAppId).Error
	if err != nil {
		err = fmt.Errorf("UpdatePayAppID Err: %v", err)
		util.CheckError(err.Error())
	}
	return

}

// GetListByCreateTimeRange 获取指定时间区间的数据,批量获取，每次获取3000条
func (o *PmPayOrderModel) GetListByCreateTimeRange(startTime, endTime time.Time) (pmPayList []*PmPayOrderTable, err error) {
	pmPayList = make([]*PmPayOrderTable, 0)
	batch := make([]*PmPayOrderTable, 0)

	fromIndex := uint(0)
	for {
		err = o.DB.Where("created_at >= ?", startTime).
			Where("created_at <= ?", endTime).
			Where("pay_status = 0").
			Where("id > ?", fromIndex).
			Order("id asc").
			Limit(3000).
			Find(&batch).Error
		if len(batch) == 0 {
			break
		}

		lastOne := batch[len(batch)-1]
		fromIndex = lastOne.ID
		pmPayList = append(pmPayList, batch...)
	}

	if err != nil {
		logx.Errorf("GetListByCreateTimeRange，err:%v, params:%v, %v", err, startTime, endTime)
		return
	}

	return pmPayList, nil
}

//
func (o *PmPayOrderModel) GetOneByThirdOrderNoAndAppId(orderSn, appId string) (info *PmPayOrderTable, err error) {
	var orderInfo PmPayOrderTable
	err = o.DB.Where("`third_order_no` = ? and pay_app_id = ?", orderSn, appId).First(&orderInfo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		logx.Errorf("GetOneByThirdOrderNoAndAppId 获取订单信息失败，err:=%v,order_sn=%s", err, orderSn)
		getPayOrderErr.CounterInc()
		return nil, err
	}
	return &orderInfo, nil
}
