package logic

import (
	"context"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AlipayTradeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigAlipayModel *model.PmPayConfigAlipayModel
}

func NewAlipayTradeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayTradeLogic {
	return &AlipayTradeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
	}
}

// 支付宝：创建支付
func (l *AlipayTradeLogic) AlipayTrade(in *pb.AlipayTradeReq) (*pb.AlipayPageSignResp, error) {
	//payClient, _, _, err := clientMgr.GetAlipayClientWithCache(in.AppPkgName)
	//if err != nil {
	//	return nil, err
	//}
	//
	//trade := alipay2.Trade{
	//	ProductCode:    "QUICK_MSECURITY_PAY",
	//	Subject:        in.Subject,
	//	OutTradeNo:     in.OutTradeNo,
	//	TotalAmount:    in.TotalAmount,
	//	TimeoutExpress: "30m",
	//}
	//
	//appPay := alipay2.TradeAppPay{
	//	Trade: trade,
	//}
	//
	//result, err := payClient.TradeAppPay(appPay)
	//if err != nil {
	//	logx.Errorf(err.Error())
	//}
	//
	//return &pb.AlipayPageSignResp{
	//	URL: result,
	//}, nil
	return nil, nil
}
