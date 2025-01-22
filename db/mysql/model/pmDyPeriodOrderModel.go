package model

import (
	"errors"
	"fmt"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/db"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// 抖音周期签约订单表
type PmDyPeriodOrderTable struct {
	ID                uint      `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	OrderSn           string    `gorm:"column:order_sn;NOT NULL" json:"order_sn"`                            // 订单唯一标识
	AppPkgName        string    `gorm:"column:app_pkg_name;NOT NULL" json:"app_pkg_name"`                    // 来源包名
	UserId            int       `gorm:"column:user_id;NOT NULL" json:"user_id"`                              // 内部用户id
	Amount            int       `gorm:"column:amount;default:0;NOT NULL" json:"amount"`                      // 订单金额（分）
	NotifyAmount      int       `gorm:"column:notify_amount;default:0;NOT NULL" json:"notify_amount"`        // 回调金额（分）
	Subject           string    `gorm:"column:subject;NOT NULL" json:"subject"`                              // 订单标题
	PayType           int       `gorm:"column:pay_type;default:0;NOT NULL" json:"pay_type"`                  // 支付方式  1微信小程序支付 2头条小程序支付
	NotifyUrl         string    `gorm:"column:notify_url;NOT NULL" json:"notify_url"`                        // 回调通知地址
	PayStatus         int       `gorm:"column:pay_status;NOT NULL" json:"pay_status"`                        // 支付状态 0未支付  1已支付
	PayAppId          string    `gorm:"column:pay_app_id;NOT NULL" json:"pay_app_id"`                        //第三方支付的appid
	ThirdOrderNo      string    `gorm:"column:third_order_no;NULL" json:"third_order_no"`                    //三方订单号
	Currency          string    `gorm:"column:currency;type:varchar(16);NOT NULL"`                           // 支付币种
	IsSign            int       `gorm:"column:is_sign;NOT NULL" json:"is_sign"`                              // 是否签约周期代扣 0否 1是
	ExpireDate        time.Time `gorm:"column:expire_date;type:datetime" json:"expire_date"`                 // 签约到期时间 默认值2000-01-01 00:00:01
	NextDecuctionTime time.Time `gorm:"column:next_decuction_time;type:datetime" json:"next_decuction_time"` // 下次扣款时间 默认值2000-01-01 00:00:01
	// CreatedAt    time.Time `gorm:"column:created_at;type:datetime" json:"created_at"`
	// UpdatedAt    time.Time `gorm:"column:updated_at;type:datetime" json:"updated_at"`
}

func (m *PmDyPeriodOrderTable) TableName() string {
	return "pm_dy_period_order"
}

type PmDyPeriodOrderModel struct {
	DB *gorm.DB
}

func NewPmDyPeriodOrderModel(dbname string) *PmDyPeriodOrderModel {
	return &PmDyPeriodOrderModel{
		DB: db.WithDBContext(dbname),
	}
}

// 创建订单
func (o *PmDyPeriodOrderModel) Create(info *PmDyPeriodOrderTable) error {
	err := o.DB.Create(info).Error
	if err != nil {
		logx.Errorf("创建支付订单失败 err: %v", err)
	}
	return err
}

// 根据订单号和包名获取订单信息
func (o *PmDyPeriodOrderModel) GetOneByOrderSnAndPkg(orderSn, pkg string) (*PmDyPeriodOrderTable, error) {
	orderInfo := new(PmDyPeriodOrderTable)
	err := o.DB.Table("pm_dy_period_order").Where("`order_sn` = ? and `app_pkg_name` = ?", orderSn, pkg).First(orderInfo).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		logx.Errorf("GetOneByOrderSnAndPkg 获取订单信息失败 err:%v, pkg:%s, orderSn:%s", err, pkg, orderSn)
	}

	return orderInfo, err
}

// 根据用户id和包名获取已签约订单信息
func (o *PmDyPeriodOrderModel) GetSignedByUserIdAndPkg(userId int, pkg string) (*PmDyPeriodOrderTable, error) {
	orderInfo := new(PmDyPeriodOrderTable)
	err := o.DB.Table("pm_dy_period_order").Where("`user_id` = ? and `app_pkg_name` = ? and is_sign = 1", userId, pkg).First(orderInfo).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		logx.Errorf("GetOneByUserIdAndPkg 获取订单信息失败 err:%v, pkg:%s, userId:%d", err, pkg, userId)
	}

	return orderInfo, err
}

// QueryAfterUpdate 查询后修改订单状态
func (o *PmDyPeriodOrderModel) QueryAfterUpdate(orderSn, appId, thirdOrderNo string, totalAmount int) (bool, error) {
	var orderInfo PmDyPeriodOrderTable
	tx := o.DB.Begin()
	err := o.DB.Table("pm_dy_period_order").Where("`order_sn` = ? and pay_app_id = ? ", orderSn, appId).First(&orderInfo).Error
	if err != nil {
		tx.Rollback()
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logx.Errorf("QueryAfterUpdate:获取订单信息失败 err:=%v,order_sn=%s", err, orderSn)
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
		logx.Errorf("QueryAfterUpdate:更新回调订单失败 err=%v", err)
		updateNofityOrderErr.CounterInc()
		return false, err
	}

	//正常逻辑
	tx.Commit()
	return true, nil
}

func (o *PmDyPeriodOrderModel) UpdateNotify(info *PmDyPeriodOrderTable) error {
	err := o.DB.Save(&info).Error
	if err != nil {
		logx.Errorf("更新回调订单失败 err: %v ", err)
	}
	return err
}

func (o *PmDyPeriodOrderModel) UpdatePayAppID(orderSn string, payAppId string) (err error) {
	err = o.DB.Table("pm_dy_period_order").Where("order_sn = ?", orderSn).Update("pay_app_id", payAppId).Error
	if err != nil {
		err = fmt.Errorf("UpdatePayAppID Err: %v", err)
		util.CheckError(err.Error())
	}
	return
}

func (o *PmDyPeriodOrderModel) GetOneByThirdOrderNoAndAppId(orderSn, appId string) (*PmDyPeriodOrderTable, error) {
	orderInfo := new(PmDyPeriodOrderTable)
	err := o.DB.Table("pm_dy_period_order").Where("`third_order_no` = ? and pay_app_id = ?", orderSn, appId).First(orderInfo).Error
	if err != nil {
		logx.Errorf("GetOneByThirdOrderNoAndAppId 获取订单信息失败 err:%v, order_sn:%s", err, orderSn)
		return nil, err
	}

	return orderInfo, nil
}
