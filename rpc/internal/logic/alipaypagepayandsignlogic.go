package logic

import (
	"context"
	"encoding/json"
	alipay2 "gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/clientMgr"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	"gitee.com/zhuyunkj/zhuyun-core/util"
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

type SignProduct struct {
	PeriodType   int
	Period       int
	SingleAmount int
	TotalAmount  int
}

// 支付宝：支付并签约
func (l *AlipayPagePayAndSignLogic) AlipayPagePayAndSign(in *pb.AlipayPageSignReq) (*pb.AlipayPageSignResp, error) {
	payClient, payAppId, notifyUrl, err := clientMgr.GetAlipayClientWithCache(in.AppPkgName)
	if err != nil {
		return nil, err
	}

	orderInfo := model.OrderTable{
		AppPkg:       in.AppPkgName,
		UserID:       int(in.UserId),
		OutTradeNo:   util.GetUuid(),
		PayType:      code.PAY_TYPE_ALI,
		Status:       0,
		PayAppID:     payAppId,
		AppNotifyUrl: in.NotifyURL,
	}

	signProduct := SignProduct{}

	err = json.Unmarshal([]byte(in.ProductDesc), &signProduct)
	if err != nil {
		return nil, err
	}

	accessParam := &alipay2.AccessParams{
		Channel: "ALIPAYAPP",
	}

	rule := &alipay2.PeriodRuleParams{
		PeriodType:   strconv.Itoa(signProduct.PeriodType),
		Period:       strconv.Itoa(signProduct.Period),
		ExecuteTime:  time.Now().Format("2006-01-02"),
		SingleAmount: strconv.Itoa(signProduct.SingleAmount),
	}

	signParams := &alipay2.SignParams{
		SignScene:           "INDUSTRY|DEFAULT_SCENE",
		ProductCode:         "GENERAL_WITHHOLDING",
		PersonalProductCode: "CYCLE_PAY_AUTH_P",
		AccessParams:        accessParam,
		PeriodRuleParams:    rule,
		ExternalAgreementNo: util.GetUuid(),
	}

	trade := alipay2.Trade{
		ProductCode:         "CYCLE_PAY_AUTH",
		AgreementSignParams: signParams,
		Subject:             in.Subject,
		OutTradeNo:          orderInfo.OutTradeNo,
		TotalAmount:         strconv.Itoa(signProduct.TotalAmount),
		TimeoutExpress:      "30m",
		NotifyURL:           notifyUrl,
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
		URL: result,
	}, nil
}
