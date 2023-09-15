package logic

import (
	"context"
	"encoding/json"
	"errors"
	alipay2 "gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/clientMgr"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/utils"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
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

type Product struct {
	ProductType     int    `json:"productType"`
	ProductSwitch   bool   `json:"productSwitch"`
	Amount          string `json:"amount"`
	PrepaidAmount   string `json:"prepaidAmount"`
	SubscribePeriod int    `json:"subscribePeriod"`
	VipDays         int    `json:"vipDays"`
	TopText         string `json:"topText"`
	BottomText      string `json:"bottomText"`
}

// 支付宝：支付并签约
func (l *AlipayPagePayAndSignLogic) AlipayPagePayAndSign(in *pb.AlipayPageSignReq) (*pb.AlipayPageSignResp, error) {
	payClient, payAppId, notifyUrl, err := clientMgr.GetAlipayClientByAppPkgWithCache(in.AppPkgName)
	if err != nil {
		return nil, err
	}

	product := Product{}

	err = json.Unmarshal([]byte(in.ProductDesc), &product)
	if err != nil {
		logx.Errorf("%s", err.Error())
		return nil, errors.New("商品信息错误")
	}

	orderInfo := model.OrderTable{
		AppPkg:       in.AppPkgName,
		UserID:       int(in.UserId),
		OutTradeNo:   utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
		PayType:      code.PAY_TYPE_ALI,
		Status:       0,
		PayAppID:     payAppId,
		AppNotifyUrl: in.NotifyURL,
	}

	trade := alipay2.Trade{
		ProductCode:    "CYCLE_PAY_AUTH",
		Subject:        in.Subject,
		OutTradeNo:     orderInfo.OutTradeNo,
		TotalAmount:    product.Amount,
		TimeoutExpress: "30m",
		NotifyURL:      notifyUrl,
	}

	externalAgreementNo := ""

	if product.ProductType == code.PRODUCT_TYPE_SUBSCRIBE {

		accessParam := &alipay2.AccessParams{
			Channel: "ALIPAYAPP",
		}

		rule := &alipay2.PeriodRuleParams{
			PeriodType:   "DAY",
			Period:       strconv.Itoa(product.SubscribePeriod),
			ExecuteTime:  time.Now().Format("2006-01-02"),
			SingleAmount: product.Amount,
		}

		trade.TotalAmount = product.PrepaidAmount

		signParams := &alipay2.SignParams{
			SignScene:           "INDUSTRY|DEFAULT_SCENE", // 固定参数
			ProductCode:         "GENERAL_WITHHOLDING",    // 固定参数
			PersonalProductCode: "CYCLE_PAY_AUTH_P",       // 固定参数
			AccessParams:        accessParam,
			PeriodRuleParams:    rule,
			ExternalAgreementNo: utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
			SignNotifyURL:       utils.GenSignNotifyUrl(notifyUrl, orderInfo.OutTradeNo),
		}

		externalAgreementNo = signParams.ExternalAgreementNo

		trade.AgreementSignParams = signParams
	}

	appPay := alipay2.TradeAppPay{
		Trade: trade,
	}

	result, err := payClient.TradeAppPay(appPay)
	if err != nil {
		logx.Errorf(err.Error())
	}

	l.orderModel.Create(&orderInfo)

	return &pb.AlipayPageSignResp{
		URL:                 result,
		OutTradeNo:          orderInfo.OutTradeNo,
		ExternalAgreementNo: externalAgreementNo,
	}, nil
}
