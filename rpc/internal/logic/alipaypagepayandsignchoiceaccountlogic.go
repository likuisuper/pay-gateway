package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	alipay2 "gitee.com/zhuyunkj/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/clientMgr"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/types"
	"gitee.com/zhuyunkj/pay-gateway/common/utils"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	"github.com/zeromicro/go-zero/core/logx"
)

type AlipayPagePayAndSignChoiceAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigAlipayModel *model.PmPayConfigAlipayModel
	orderModel           *model.OrderModel
}

func NewAlipayPagePayAndSignChoiceAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayPagePayAndSignChoiceAccountLogic {
	return &AlipayPagePayAndSignChoiceAccountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
		orderModel:           model.NewOrderModel(define.DbPayGateway),
	}
}

// 支付宝新的充值订阅 可以选择不同的支付宝账号：支付并签约
func (l *AlipayPagePayAndSignChoiceAccountLogic) AlipayPagePayAndSignChoiceAccount(in *pb.AlipayPageSignReq) (*pb.AlipayPageSignResp, error) {
	// 选择不同的支付宝号
	payClient, payAppId, notifyUrl, merchantNo, _, err := clientMgr.GetAlipayClienMerchantInfo(in.AppPkgName)
	if err != nil {
		return nil, err
	}

	if merchantNo == "" {
		return nil, errors.New("获取商户号信息失败")
	}

	var amount, prepaidAmount string
	var productType, intAmount, period int

	productType = int(in.ProductType)

	if in.ProductId == 0 {
		// 目前没有商品的配置，通过解析商品详情来获取商品的内容
		product := types.Product{}
		err = json.Unmarshal([]byte(in.ProductDesc), &product)
		if err != nil {
			parseProductDescErr.CounterInc()
			logx.Errorf("创建订单异常：商品信息错误 err = %s product = %s", err.Error(), in.ProductDesc)
			return nil, errors.New("商品信息错误")
		}

		productType = product.ProductType
		if productType == code.PRODUCT_TYPE_SUBSCRIBE {
			intAmount = int(product.PrepaidAmount * 100)
		} else {
			intAmount = int(product.Amount * 100)
		}
		period = product.SubscribePeriod
		prepaidAmount = fmt.Sprintf("%.2f", product.PrepaidAmount)
		amount = fmt.Sprintf("%.2f", product.Amount)
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
		DeviceId:     in.DeviceId, // 在回调的时候 需要带到app_alipay_order表
	}

	trade := alipay2.Trade{
		ProductCode: "QUICK_MSECURITY_PAY", // 固定参数
		// Subject:        in.Subject,
		Subject:        "【客服电话：18150156227】如您需办理退款，请拨打VIP售后电话，将极速为您办理，感谢您的订阅！",
		OutTradeNo:     orderInfo.OutTradeNo,
		TotalAmount:    amount,
		TimeoutExpress: "30m",
		NotifyURL:      notifyUrl,
	}

	externalAgreementNo := ""

	if productType == code.PRODUCT_TYPE_SUBSCRIBE {
		// 订阅商品
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
		trade.ProductCode = "CYCLE_PAY_AUTH"
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

		// 指定商户信息
		//signParams.SubMerchant = &alipay2.SubMerchantParams{
		//	SubMerchantId:          merchantNo,
		//	SubMerchantName:        merchantName,
		//	SubMerchantServiceName: "【客服电话：18150156227】如您需办理退款，请拨打VIP售后电话，将极速为您办理，感谢您的订阅！",
		//}

		trade.AgreementSignParams = signParams
	}

	appPay := alipay2.TradeAppPay{
		Trade: trade,
	}

	bytes, err := json.Marshal(appPay)
	logx.Slowf("请求参数: %v, err:%v", string(bytes), err)

	result, err := payClient.TradeAppPay(appPay)
	if err != nil {
		logx.Errorf("创建订单异常,生成支付宝加签串失败,err=%s", err.Error())
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
		periodAmount, _ := strconv.ParseFloat(amount, 64)
		orderInfo.Amount = int(periodAmount * 100)
	}

	err = l.orderModel.Create(&orderInfo)
	if err != nil {
		payAndSignCreateOrderErr.CounterInc()
		logx.Errorf("创建订单异常,创建订单表失败 err:%s, orderInfo: %+v", err.Error(), orderInfo)
		return nil, errors.New("创建订单异常")
	}

	return &pb.AlipayPageSignResp{
		URL:                 result,
		OutTradeNo:          orderInfo.OutTradeNo,
		ExternalAgreementNo: externalAgreementNo,
	}, nil
}
