package model

import (
	"fmt"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/db"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// 抖音周期签约订单表
type PmDyPeriodOrderTable struct {
	ID                int       `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	OrderSn           string    `gorm:"column:order_sn;NOT NULL" json:"order_sn"`                            // 内部 订单唯一标识 (开发者侧代扣单的单号)
	SignNo            string    `gorm:"column:sign_no;NOT NULL" json:"sign_no"`                              // 内部 签约单号 (开发者侧签约单号)
	AppPkgName        string    `gorm:"column:app_pkg_name;NOT NULL" json:"app_pkg_name"`                    // 来源包名
	UserId            int       `gorm:"column:user_id;NOT NULL" json:"user_id"`                              // 内部用户id
	Amount            int       `gorm:"column:amount;default:0;NOT NULL" json:"amount"`                      // 订单金额（分）
	NotifyAmount      int       `gorm:"column:notify_amount;default:0;NOT NULL" json:"notify_amount"`        // 回调金额（分）
	Subject           string    `gorm:"column:subject;NOT NULL" json:"subject"`                              // 订单标题
	PayType           int       `gorm:"column:pay_type;default:0;NOT NULL" json:"pay_type"`                  // 支付方式  1微信小程序支付 2头条小程序支付
	NotifyUrl         string    `gorm:"column:notify_url;NOT NULL" json:"notify_url"`                        // 回调通知地址
	PayStatus         int       `gorm:"column:pay_status;NOT NULL" json:"pay_status"`                        // 支付状态 0未支付 1已支付
	PayChannel        int       `gorm:"column:pay_channel;NOT NULL" json:"pay_channel"`                      // 支付渠道 扣款成功时才有
	SignStatus        int       `gorm:"column:sign_status;NOT NULL" json:"sign_status"`                      // 签约状态, 0 待签约 , 1已签约 , 2取消签约 , 3 签约到期(服务已完成)
	PayAppId          string    `gorm:"column:pay_app_id;NOT NULL" json:"pay_app_id"`                        // 第三方支付的appid
	ThirdOrderSn      string    `gorm:"column:third_order_sn;NULL" json:"third_order_sn"`                    // 抖音平台返回的渠道支付单号
	ThirdOrderNo      string    `gorm:"column:third_order_no;NULL" json:"third_order_no"`                    // 抖音平台返回的代扣单的单号
	ThirdSignOrderNo  string    `gorm:"column:third_sign_order_no;NULL" json:"third_sign_order_no"`          // 抖音平台返回的签约单号
	Currency          string    `gorm:"column:currency;type:varchar(16);NOT NULL"`                           // 支付币种
	UserBillPayId     string    `gorm:"column:user_bill_pay_id;NULL" json:"user_bill_pay_id"`                // 用户抖音交易单号（账单号），和用户抖音钱包-账单中所展示的交易单号相同
	SignDate          time.Time `gorm:"column:sign_date;type:datetime" json:"sign_date"`                     // 签约时间 默认值2000-01-01 00:00:01
	UnsignDate        time.Time `gorm:"column:unsign_date;type:datetime" json:"unsign_date"`                 // 解约时间 默认值2000-01-01 00:00:01
	ExpireDate        time.Time `gorm:"column:expire_date;type:datetime" json:"expire_date"`                 // 签约到期时间 默认值2000-01-01 00:00:01
	NextDecuctionTime time.Time `gorm:"column:next_decuction_time;type:datetime" json:"next_decuction_time"` // 下次扣款时间 默认值2000-01-01 00:00:01
	DyProductId       string    `gorm:"column:dy_product_id;NOT NULL" json:"dy_product_id"`                  // 抖音商品id
	NthNum            int       `gorm:"column:nth_num;default:0;NOT NULL" json:"nth_num"`                    // 第几期代扣单
	// CreatedAt    time.Time `gorm:"column:created_at;type:datetime" json:"created_at"`
	// UpdatedAt    time.Time `gorm:"column:updated_at;type:datetime" json:"updated_at"`
}

// 签约状态
const (
	Sign_Status_Wait    = 0 // 等待签约
	Sign_Status_Success = 1 // 签约成功
	Sign_Status_Cancel  = 2 // 取消签约
	Sign_Status_Done    = 3 // 签约到期(服务已完成)
)

// 2000-01-01 00:00:01 对应的时间戳就是946656001
var Default2000Date = time.Unix(946656001, 0)

// 表名
const PmDyPeriodOrderTableName = "pm_dy_period_order"

func (m *PmDyPeriodOrderTable) TableName() string {
	return PmDyPeriodOrderTableName
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

// 根据内部订单号和包名获取订单信息
func (o *PmDyPeriodOrderModel) GetOneByOrderSnAndPkg(orderSn, pkg string) (*PmDyPeriodOrderTable, error) {
	orderInfo := new(PmDyPeriodOrderTable)
	err := o.DB.Table(PmDyPeriodOrderTableName).Where("`order_sn` = ? and `app_pkg_name` = ?", orderSn, pkg).First(orderInfo).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		logx.Errorf("GetOneByOrderSnAndPkg 获取订单信息失败 err:%v, pkg:%s, orderSn:%s", err, pkg, orderSn)
	}

	return orderInfo, err
}

// 根据内部订单号和appid获取订单信息
func (o *PmDyPeriodOrderModel) GetOneByOrderSnAndAppId(orderSn, appId string) (*PmDyPeriodOrderTable, error) {
	orderInfo := new(PmDyPeriodOrderTable)
	err := o.DB.Table(PmDyPeriodOrderTableName).Where("`order_sn` = ? and `pay_app_id` = ?", orderSn, appId).First(orderInfo).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		logx.Errorf("GetOneByOrderSnAndAppId 获取订单信息失败 err:%v, appId:%s, orderSn:%s", err, appId, orderSn)
	}

	return orderInfo, err
}

// 根据内部签约单号和appid获取订单信息
func (o *PmDyPeriodOrderModel) GetOneBySignNoAndAppId(signNo, appId string) (*PmDyPeriodOrderTable, error) {
	orderInfo := new(PmDyPeriodOrderTable)
	err := o.DB.Table(PmDyPeriodOrderTableName).Where("`sign_no` = ? and `pay_app_id` = ?", signNo, appId).First(orderInfo).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		logx.Errorf("GetOneBySignNoAndAppId 获取订单信息失败 err:%v, appId:%s, signNo:%s", err, appId, signNo)
	}

	return orderInfo, err
}

// 根据用户id和包名获取已签约订单信息
func (o *PmDyPeriodOrderModel) GetSignedByUserIdAndPkg(userId int, pkg string, signStatus int) (*PmDyPeriodOrderTable, error) {
	orderInfo := new(PmDyPeriodOrderTable)
	err := o.DB.Table(PmDyPeriodOrderTableName).Where("`user_id` = ? and `app_pkg_name` = ? and sign_status = ?", userId, pkg, signStatus).Last(orderInfo).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		logx.Errorf("GetOneByUserIdAndPkg 获取订单信息失败 err:%v, pkg:%s, userId:%d", err, pkg, userId)
	}

	return orderInfo, err
}

func (o *PmDyPeriodOrderModel) UpdatePayAppID(orderSn string, payAppId string) (err error) {
	err = o.DB.Table(PmDyPeriodOrderTableName).Where("order_sn = ?", orderSn).Update("pay_app_id", payAppId).Error
	if err != nil {
		err = fmt.Errorf("UpdatePayAppID Err: %v", err)
		util.CheckError(err.Error())
	}
	return
}

func (o *PmDyPeriodOrderModel) GetOneByThirdOrderNoAndAppId(orderSn, appId string) (*PmDyPeriodOrderTable, error) {
	orderInfo := new(PmDyPeriodOrderTable)
	err := o.DB.Table(PmDyPeriodOrderTableName).Where("`third_order_no` = ? and pay_app_id = ?", orderSn, appId).First(orderInfo).Error
	if err != nil {
		logx.Errorf("GetOneByThirdOrderNoAndAppId 获取订单信息失败 err:%v, order_sn:%s", err, orderSn)
	}

	return orderInfo, err
}

// 更新数据
func (o *PmDyPeriodOrderModel) UpdateSomeData(id int, updateData map[string]interface{}) error {
	err := o.DB.Table(PmDyPeriodOrderTableName).Where("`id` = ?", id).Updates(updateData).Error
	if err != nil {
		err = fmt.Errorf("UpdateSomeData Err: %v", err)
		util.CheckError(err.Error())
	}

	return err
}

// 获取今日可以扣款的签约订单信息
func (o *PmDyPeriodOrderModel) GetSignedPayList() ([]PmDyPeriodOrderTable, error) {
	var info []PmDyPeriodOrderTable
	startAt := time.Now().Format("2006-01-02")
	endAt := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	err := o.DB.Table(PmDyPeriodOrderTableName).Where("`sign_status` = 1 and `next_decuction_time` >= ? and `next_decuction_time` < ?", startAt, endAt).Find(&info).Error
	return info, err
}
