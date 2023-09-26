package logic

import (
	"context"
	"encoding/json"
	"errors"
	alipay2 "gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/clientMgr"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/types"
	"gitee.com/zhuyunkj/pay-gateway/common/utils"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"github.com/zeromicro/go-zero/core/logx"
	"strconv"
	"time"
)

type AlipayPagePayAndSignLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigAlipayModel *model.PmPayConfigAlipayModel
	orderModel           *model.OrderModel
}

var (
	parseProductDescErr      = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "parseProductDescErr", nil, "解析商品详情失败", nil})}
	payAndSignCreateOrderErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "payAndSignCreateOrderErr", nil, "创建订单失败", nil})}
)

func NewAlipayPagePayAndSignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayPagePayAndSignLogic {
	return &AlipayPagePayAndSignLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
		orderModel:           model.NewOrderModel(define.DbPayGateway),
	}
}

// 支付宝：支付并签约
func (l *AlipayPagePayAndSignLogic) AlipayPagePayAndSign(in *pb.AlipayPageSignReq) (*pb.AlipayPageSignResp, error) {
	payClient, payAppId, notifyUrl, err := clientMgr.GetAlipayClientByAppPkgWithCache(in.AppPkgName)
	if err != nil {
		return nil, err
	}

	var amount, prepaidAmount string
	var productType, intAmount, period int

	productType = int(in.ProductType)

	if in.ProductId == 0 { // 目前没有商品的配置，通过解析商品详情来获取商品的内容
		product := types.Product{}

		err = json.Unmarshal([]byte(in.ProductDesc), &product)
		if err != nil {
			parseProductDescErr.CounterInc()
			logx.Errorf("创建订单异常：商品信息错误 err = %s product = %s", err.Error(), in.ProductDesc)
			return nil, errors.New("商品信息错误")
		}

		if !product.ProductSwitch {
			parseProductDescErr.CounterInc()
			logx.Errorf("创建订单异常：商品不允许购买 product = %s", in.ProductDesc)
			return nil, errors.New("商品信息错误")
		}

		floatAmount, parseErr := strconv.ParseFloat(product.Amount, 64)
		if parseErr != nil {
			parseProductDescErr.CounterInc()
			logx.Errorf("创建订单异常：商品金额异常 product = %s", in.ProductDesc)
			return nil, errors.New("商品信息错误")
		}

		intAmount = int(floatAmount * 100)
		prepaidAmount = product.PrepaidAmount
		period = product.SubscribePeriod
		productType = product.ProductType
		amount = product.Amount
	}

	if intAmount <= 0 {
		parseProductDescErr.CounterInc()
		logx.Errorf("创建订单异常：商品金额异常 product = %s", in.ProductDesc)
		return nil, errors.New("商品信息错误")
	}

	orderInfo := model.OrderTable{
		AppPkg:       in.AppPkgName,
		UserID:       int(in.UserId),
		OutTradeNo:   utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
		PayType:      model.PmPayOrderTablePayTypeAlipay,
		Status:       0,
		PayAppID:     payAppId,
		AppNotifyUrl: in.NotifyURL,
		Amount:       intAmount,
		ProductDesc:  in.ProductDesc,
		ProductType:  productType,
		ProductID:    int(in.ProductId),
	}

	trade := alipay2.Trade{
		ProductCode:    "CYCLE_PAY_AUTH", // 固定参数
		Subject:        in.Subject,
		OutTradeNo:     orderInfo.OutTradeNo,
		TotalAmount:    amount,
		TimeoutExpress: "30m",
		NotifyURL:      notifyUrl,
	}

	externalAgreementNo := ""

	if productType == code.PRODUCT_TYPE_SUBSCRIBE {

		accessParam := &alipay2.AccessParams{
			Channel: "ALIPAYAPP",
		}

		rule := &alipay2.PeriodRuleParams{
			PeriodType:   "DAY",
			Period:       strconv.Itoa(period),
			ExecuteTime:  time.Now().Format("2006-01-02"),
			SingleAmount: amount,
		}

		trade.TotalAmount = prepaidAmount // 订阅商品，首次付款的金额是预付金额

		signParams := &alipay2.SignParams{
			SignScene:           "INDUSTRY|DEFAULT_SCENE", // 固定参数
			ProductCode:         "GENERAL_WITHHOLDING",    // 固定参数
			PersonalProductCode: "CYCLE_PAY_AUTH_P",       // 固定参数
			AccessParams:        accessParam,
			PeriodRuleParams:    rule,
			ExternalAgreementNo: utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
			SignNotifyURL:       notifyUrl,
		}

		externalAgreementNo = signParams.ExternalAgreementNo

		orderInfo.ExternalAgreementNo = externalAgreementNo

		trade.AgreementSignParams = signParams
	}

	appPay := alipay2.TradeAppPay{
		Trade: trade,
	}

	bytes, err := json.Marshal(appPay)
	logx.Slowf("请求参数: %v", string(bytes))

	result, err := payClient.TradeAppPay(appPay)
	if err != nil {
		logx.Errorf("创建订单异常：生成支付宝加签串失败， err = %s", err.Error())
		return nil, errors.New("创建订单异常")
	}

	if in.ProductType == code.PRODUCT_TYPE_SUBSCRIBE_FEE && in.BelongSignOrder != "" {
		tb, err := l.orderModel.GetOneByOutTradeNo(in.BelongSignOrder)
		if err != nil {
			logx.Errorf("创建续费订单异常：获取所属的签约订单失败， err = %s", err.Error())
			return nil, errors.New("创建订单异常")
		}
		orderInfo.AgreementNo = tb.AgreementNo
		orderInfo.ExternalAgreementNo = tb.ExternalAgreementNo
		orderInfo.ProductType = int(in.ProductType)
	}

	err = l.orderModel.Create(&orderInfo)
	if err != nil {
		payAndSignCreateOrderErr.CounterInc()
		logx.Errorf("创建订单异常：创建订单表失败， err = %s", err.Error())
		return nil, errors.New("创建订单异常")
	}

	return &pb.AlipayPageSignResp{
		URL:                 result,
		OutTradeNo:          orderInfo.OutTradeNo,
		ExternalAgreementNo: externalAgreementNo,
	}, nil
}
