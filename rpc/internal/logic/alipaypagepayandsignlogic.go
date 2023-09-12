package logic

import (
	"context"
	alipay2 "gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/clientMgr"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	"github.com/zeromicro/go-zero/core/logx"
)

type AlipayPagePayAndSignLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigAlipayModel *model.PmPayConfigAlipayModel
}

func NewAlipayPagePayAndSignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayPagePayAndSignLogic {
	return &AlipayPagePayAndSignLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
	}
}

// 支付宝：支付并签约
func (l *AlipayPagePayAndSignLogic) AlipayPagePayAndSign(in *pb.AlipayPageSignReq) (*pb.AlipayPageSignResp, error) {
	payClient, err := clientMgr.GetAlipayClientWithCache(in.AppPkgName)
	if err != nil {
		return nil, err
	}

	accessParam := &alipay2.AccessParams{
		Channel: "ALIPAYAPP",
	}

	rule := &alipay2.PeriodRuleParams{
		PeriodType:   "DAY",
		Period:       "7",
		ExecuteTime:  in.ExecuteTime,
		SingleAmount: in.SingleAmount,
	}

	signParams := &alipay2.SignParams{
		SignScene:           "INDUSTRY|DEFAULT_SCENE",
		ProductCode:         "GENERAL_WITHHOLDING",
		PersonalProductCode: "CYCLE_PAY_AUTH_P",
		AccessParams:        accessParam,
		PeriodRuleParams:    rule,
		ExternalAgreementNo: in.ExternalAgreementNo,
	}

	trade := alipay2.Trade{
		ProductCode:         "CYCLE_PAY_AUTH",
		AgreementSignParams: signParams,
		Subject:             in.Subject,
		OutTradeNo:          in.OutTradeNo,
		TotalAmount:         in.TotalAmount,
		TimeoutExpress:      "30m",
	}

	appPay := alipay2.TradeAppPay{
		Trade: trade,
	}

	result, err := payClient.TradeAppPay(appPay)
	if err != nil {
		logx.Errorf(err.Error())
	}

	return &pb.AlipayPageSignResp{
		URL: result,
	}, nil
}
