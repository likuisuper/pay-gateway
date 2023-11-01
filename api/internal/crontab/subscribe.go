package crontab

import (
	"encoding/json"
	"errors"
	"fmt"
	alipay2 "gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/clientMgr"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/common/types"
	"strconv"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/config"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	dbmodel "gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/nacos"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/robfig/cron"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

var (
	GetFirstUnpaidSubscribeFeeErrNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "GetFirstUnpaidSubscribeFeeErrNum", nil, "获取未支付的续费订单失败", nil})}
	PaySubscribeFeeErrNum            = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "PaySubscribeFeeErrNum", nil, "续费失败", nil})}
)

type CrontabOrder struct {
	Nacos   *nacos.Instance
	SvcName string
	Conf    *config.Config
	SvcCtx  *svc.ServiceContext
}

const (
	payOrderTime = "0 30 23 * * ?"
)

var crontabOrder *CrontabOrder

func InitCrontabOrder(namingClient *nacos.Instance, svcName string, c *config.Config, s *svc.ServiceContext) {
	crontabOrder = &CrontabOrder{
		Nacos:   namingClient,
		SvcName: svcName,
		Conf:    c,
		SvcCtx:  s,
	}

	// 定时任务
	cronTask := cron.New()

	err := cronTask.AddFunc(payOrderTime, func() {
		crontabOrder.PayOrder()
	})
	if err != nil {
		logx.Errorf("创建支付订单任务定时任务失败，err= %v", err)
	}

	cronTask.Start()
	logx.Info("InitCrontabOrder success")
}

var orderModel *dbmodel.OrderModel

func (c *CrontabOrder) PayOrder() {
	instances, err := c.Nacos.SelectAllInstances(&c.SvcName)
	if err != nil {
		logx.Errorf("获取dsp服务 %s 的注册实例失败, err= %v", c.SvcName, err)
		return
	}

	// 判断是否在此服务执行定时任务
	localDo := c.CheckLocalMachineDo(&instances)
	if !localDo {
		return
	}

	orderModel = dbmodel.NewOrderModel(define.DbPayGateway)

	firstModel, err := orderModel.GetFirstUnpaidSubscribeFee()
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			// 记录获取失败数
			GetFirstUnpaidSubscribeFeeErrNum.CounterInc()
		}

		logx.Errorf("CrontabOrder::CreateOrder error: ", err)
		return
	}

	if firstModel.ID == 0 {
		logx.Info("暂时没有需要扣款的VIP订阅")
		return
	}

	lastId := firstModel.ID - 1
	for {
		models, err := orderModel.GetRangeData(lastId)
		if err != nil {
			logx.Errorf("orderModel::GetRangeData error: ", err)
			break
		}

		if len(models) == 0 {
			break
		}

		for _, tmpOrderModel := range models {
			lastId = tmpOrderModel.ID
			err = c.PaySubscribeFee(tmpOrderModel)
			if err != nil {
				PaySubscribeFeeErrNum.CounterInc()
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// 创建续费订单
func (c *CrontabOrder) PaySubscribeFee(tb *dbmodel.OrderTable) error {

	agreementSignParams := &alipay2.AgreementParams{
		AgreementNo: tb.AgreementNo,
	}

	product := types.Product{}
	err := json.Unmarshal([]byte(tb.ProductDesc), &product)
	if err != nil {
		errDesc := fmt.Sprintf("订阅扣款： 解析订单商品详情 outTradeNo=%s err=%s", tb.OutTradeNo, err.Error())
		logx.Errorf(errDesc)
		return errors.New(errDesc)
	}

	client, _, notifyUrl, err := clientMgr.GetAlipayClientByAppIdWithCache(tb.PayAppID)
	if err != nil {
		errDesc := fmt.Sprintf("订阅扣款：获取支付宝客户端失败 outTradeNo=%s err=%s", tb.OutTradeNo, err.Error())
		logx.Errorf(errDesc)
		return errors.New(errDesc)
	}

	trade := alipay2.Trade{
		OutTradeNo:     tb.OutTradeNo,
		TotalAmount:    fmt.Sprintf("%.2f", product.Amount),
		Subject:        product.TopText,
		ProductCode:    "GENERAL_WITHHOLDING",
		TimeoutExpress: "30m",
		NotifyURL:      notifyUrl,
	}
	tradePayApp := alipay2.TradePay{
		Trade:           trade,
		AgreementParams: agreementSignParams,
	}

	result, err := client.TradePay(tradePayApp)
	if err != nil {
		errDesc := fmt.Sprintf("订阅扣款：扣款失败 outTradeNo=%s, err=%s", result, err.Error())
		logx.Errorf(errDesc)
		go func() {
			defer exception.Recover()
			dataMap := make(map[string]interface{})
			dataMap["notify_type"] = code.APP_NOTIFY_TYPE_SIGN_FEE_FAILED
			dataMap["external_agreement_no"] = tb.ExternalAgreementNo
			_, _ = util.HttpPost(tb.AppNotifyUrl, dataMap, 5*time.Second)
		}()
		return errors.New(errDesc)
	} else {
		if result.Content.Code == alipay2.CodeSuccess {
			infoDesc := fmt.Sprintf("续费成功: appPkg=%v, userid=%v, outTradeNo=%v", tb.AppPkg, tb.UserID, tb.OutTradeNo)
			logx.Info(infoDesc)
		} else {
			logx.Errorf("续费失败：out_trade_no = %v, msg = %v, subMsg = %v", tb.OutTradeNo, result.Content.Msg, result.Content.SubMsg)
		}
		return nil
	}

}

// 检查是否在本机执行
func (c *CrontabOrder) CheckLocalMachineDo(instances *[]model.Instance) bool {
	// 本机 ip
	ip, err := util.ExternalIP()
	if err != nil {
		logx.Errorf("获取本机 ip 失败, err= %v", err)
		return false
	}
	localIp := ip.String()
	var minTime int64 = 0
	var hitIp string
	for _, instance := range *instances {
		if metadate, ok := instance.Metadata["startTime"]; ok && instance.Enable {
			startTime, err := strconv.ParseInt(metadate, 10, 64)
			if err != nil {
				logx.Errorf("获取生成续费订单定时任务 metadata 失败, err= %v, metadata= %v", err, metadate)
				return false
			}
			if minTime == 0 || startTime < minTime {
				minTime = startTime
				hitIp = instance.Ip
			}
		}
	}

	return localIp == hitIp
}
