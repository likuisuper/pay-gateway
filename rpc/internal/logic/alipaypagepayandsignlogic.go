package logic

import (
	"context"
	"fmt"
	alipay2 "gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	"gitee.com/zhuyunkj/zhuyun-core/util"

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
	// todo: add your logic here and delete this line

	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		util.CheckError("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		return nil, err
	}

	payCfg, cfgErr := l.payConfigAlipayModel.GetOneByAppID(pkgCfg.AlipayAppID)
	if cfgErr != nil {
		err = fmt.Errorf("pkgName= %s, 读取支付宝配置失败，err:=%v", in.AppPkgName, cfgErr)
		util.CheckError(err.Error())
		return nil, cfgErr
	}

	// 将 key 的验证调整到初始化阶段
	payClient, err := client.GetAlipayClient(*payCfg.TransClientConfig())
	if err != nil {
		err = fmt.Errorf("pkgName= %s, 初使化支付错误，err:=%v", in.AppPkgName, err)
		util.CheckError(err.Error())
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
