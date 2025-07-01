package model

import (
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db"
	"gorm.io/gorm"
)

// 转出订单
type PmFundTransOrderTable struct {
	Id         int       `gorm:"column:id;type:int(11);primary_key;AUTO_INCREMENT" json:"id"`
	OrderSn    string    `gorm:"column:order_sn;type:varchar(50);comment:订单唯一标识;NOT NULL" json:"order_sn"`
	AppPkgName string    `gorm:"column:app_pkg_name;type:varchar(30);comment:来源包名;NOT NULL" json:"app_pkg_name"`
	Amount     int       `gorm:"column:amount;type:int(11);default:0;comment:订单金额（分）;NOT NULL" json:"amount"`
	AliName    string    `gorm:"column:ali_name;type:varchar(100);comment:支付宝真实姓名;NOT NULL" json:"ali_name"`
	AliAccount string    `gorm:"column:ali_account;type:varchar(100);comment:支付宝账号;NOT NULL" json:"ali_account"`
	PayAppId   string    `gorm:"column:pay_app_id;type:varchar(50);comment:第三方支付的appid;NOT NULL" json:"pay_app_id"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:datetime" json:"updated_at"`
}

func (m *PmFundTransOrderTable) TableName() string {
	return "pm_fund_trans_order"
}

type PmFundTransOrderModel struct {
	DB *gorm.DB
}

func NewPmFundTransOrderModel(dbName string) *PmFundTransOrderModel {
	return &PmFundTransOrderModel{
		DB: db.WithDBContext(dbName),
	}
}

// 创建订单
func (o *PmFundTransOrderModel) Create(info *PmFundTransOrderTable) (err error) {
	err = o.DB.Create(info).Error
	if err != nil {
		logx.Errorf("创建PmFundTransOrder失败， err=%v", err)
		updateNofityOrderErr.CounterInc()
	}
	return err
}
