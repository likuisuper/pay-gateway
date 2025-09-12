package model

import (
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db"
	"gorm.io/gorm"
)

// HuaweiOrderTable represents a huawei_order struct data.
type HuaweiOrderTable struct {
	Id                  int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	LogId               int       `gorm:"column:log_id" json:"logId"`                              // notify_huawei_log id
	Version             string    `gorm:"column:version" json:"version"`                           // 通知版本
	EventType           string    `gorm:"column:event_type" json:"eventType"`                      // 通知类型，取值如下：ORDER：订单 SUBSCRIPTION：订阅
	NotifyTime          int       `gorm:"column:notify_time" json:"notifyTime"`                    // 通知时间毫秒
	AppId               string    `gorm:"column:app_id" json:"appId"`                              // 华为应用ID
	AppPkg              string    `gorm:"column:app_pkg" json:"appPkg"`                            // 华为应用包名
	UserId              int       `gorm:"column:user_id" json:"userId"`                            // 应用用户id
	NotificationType    int       `gorm:"column:notification_type" json:"notificationType"`        // 通知事件的类型
	PurchaseToken       string    `gorm:"column:purchase_token" json:"purchaseToken"`              // 购买token
	Environment         string    `gorm:"column:environment" json:"environment"`                   // 订阅购买环境 prod：正式环境, sandbox：沙盒测试
	SubscriptionId      string    `gorm:"column:subscription_id" json:"subscriptionId"`            // 订阅id
	CancellationDate    int       `gorm:"column:cancellation_date" json:"cancellationDate"`        // 撤销订阅时间或退款时间，UTC时间戳，以毫秒为单位，仅在notificationType取值为CANCEL的场景下会传入。
	PayOrderId          string    `gorm:"column:pay_order_id" json:"payOrderId"`                   // 订单ID，唯一标识一笔需要收费的收据，由华为应用内支付服务器在创建订单以及订阅型商品续费时生成。每一笔新的收据都会使用不同的orderId。通知类型为NEW_RENEWAL_PREF时不存在。
	RefundPayOrderId    string    `gorm:"column:refund_pay_order_id" json:"refundPayOrderId"`      // 退款交易号，在notificationType取值为CANCEL时有值
	AutoRenewStatus     int       `gorm:"column:auto_renew_status" json:"autoRenewStatus"`         // 续期状态。取值说明：1：当前周期到期后正常续期 0：用户已终止续期
	ExpirationIntent    int       `gorm:"column:expiration_intent" json:"expirationIntent"`        // 超期原因，仅在notificationType为RENEWAL或INTERACTIVE_RENEWAL时并且续期失败情况下有值
	OutTradeNo          string    `gorm:"column:out_trade_no" json:"outTradeNo"`                   // 内部订单号
	PlatformTradeNo     string    `gorm:"column:platform_trade_no" json:"platformTradeNo"`         // 支付宝、微信、华为等平台的订单号
	Amount              int       `gorm:"column:amount" json:"amount"`                             // 支付金额
	Status              int       `gorm:"column:status" json:"status"`                             // -1:关闭，0:未支付，1:已支付，2:支付失败，3:已退款，4：退款中
	PayType             int       `gorm:"column:pay_type" json:"payType"`                          // 支付方式  1微信支付 2头条小程序支付，3：阿里支付
	PayTime             time.Time `gorm:"column:pay_time" json:"payTime"`                          // 支付时间
	ProductId           string    `gorm:"column:product_id" json:"product_id"`                     // 商品ID
	ProductType         int       `gorm:"column:product_type" json:"productType"`                  // 商品类型，0:普通商品，1:订阅商品，2:会员商品，3:订阅商品续费
	ProductDesc         string    `gorm:"column:product_desc" json:"product_desc"`                 // 商品描述
	AppNotifyUrl        string    `gorm:"column:app_notify_url" json:"appNotifyUrl"`               // 业务回调通知
	AgreementNo         string    `gorm:"column:agreement_no" json:"agreementNo"`                  // 支付宝/微信平台订阅协议号
	ExternalAgreementNo string    `gorm:"column:external_agreement_no" json:"externalAgreementNo"` // 内部协议号
	ExpirationDate      int       `gorm:"column:expiration_date" json:"expirationDate"`            // 订阅商品过期时间
	PayAppId            string    `gorm:"column:pay_app_id" json:"payAppId"`                       // 第三方支付的appid
	DeviceId            string    `gorm:"column:device_id" json:"deviceId"`                        // 用户设备号
	DeductTime          time.Time `gorm:"column:deduct_time" json:"deductTime"`                    // 可开始扣款时间(默认是0,不需要关注,只是为了满足产品延迟扣款的需求)
	// CreatedAt           time.Time `gorm:"column:created_at" json:"createdAt"`                      // 创建时间
	// UpdatedAt           time.Time `gorm:"column:updated_at" json:"updatedAt"`                      // 更新时间
}

func (m *HuaweiOrderTable) TableName() string {
	return "huawei_order"
}

type HuaweiOrderModel struct {
	DB *gorm.DB
}

func NewHuaweiOrderModel(dbName string) *HuaweiOrderModel {
	return &HuaweiOrderModel{
		DB: db.WithDBContext(dbName),
	}
}

// 创建记录
func (o *HuaweiOrderModel) CreateHuaweiOrder(info *HuaweiOrderTable) error {
	err := o.DB.Create(info).Error
	if err != nil {
		logx.Errorf("创建失败 err:%v", err)
	}
	return err
}

// 绑定token
func (o *HuaweiOrderModel) BindToken(purchaseToken string, userId int, outTradeNo string) error {
	err := o.DB.Table("huawei_order").Where("user_id", userId).Where("out_trade_no", outTradeNo).Update("purchase_token", purchaseToken).Error
	if err != nil {
		logx.Errorf("更新失败 err:%v", err)
	}
	return err
}

// 根据购买token、订阅id或者原订阅id获取记录
func (o *HuaweiOrderModel) GetOneByTokenAndSubId(purchaseToken, subscriptionId, oriSubscriptionId, orderId string) (*HuaweiOrderTable, error) {
	tbl := new(HuaweiOrderTable)
	err := o.DB.Table("huawei_order").Where("`purchase_token`", purchaseToken).First(tbl).Error
	if err != nil {
		if subscriptionId != "" || oriSubscriptionId != "" {
			// 给个不存在的值占位
			if subscriptionId == "" {
				subscriptionId = "-100"
			}
			if oriSubscriptionId == "" {
				oriSubscriptionId = "-100"
			}

			// 再根据订阅id获取
			err = o.DB.Table("huawei_order").Where("`subscription_id` = ? or `subscription_id` = ? or `pay_order_id` = ?", subscriptionId, oriSubscriptionId, orderId).First(tbl).Error
			if err != nil {
				logx.Errorf("GetOneByTokenAndSubId err: %v, token: %s, subscriptionId: %s, oriSubscriptionId: %s", err, purchaseToken, subscriptionId, oriSubscriptionId)
			}
		} else {
			logx.Errorf("GetOneByTokenAndSubId err: %v, token: %s,", err, purchaseToken)
		}
	}

	return tbl, err
}

func (o *HuaweiOrderModel) GetOneByOutTradeNo(outTradeNo string) (*HuaweiOrderTable, error) {
	if outTradeNo == "" {
		return nil, errors.New("订单号为空")
	}

	tbl := new(HuaweiOrderTable)
	err := o.DB.Table("huawei_order").Where("`out_trade_no`", outTradeNo).First(tbl).Error
	if err != nil {
		logx.Errorf("GetOneByOutTradeNo err: %v, outTradeNo: %s,", err, outTradeNo)
	}
	return tbl, err
}

func (o *HuaweiOrderModel) UpdateData(id int, data map[string]interface{}) error {
	err := o.DB.Table("huawei_order").Where("id", id).Updates(data).Error
	if err != nil {
		logx.Errorf("更新失败 err:%v", err)
	}
	return err
}
