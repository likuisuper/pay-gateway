package logic

import (
	"context"
	"fmt"
	alipay2 "gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"

	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"

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

	trade := alipay2.Trade{
		ProductCode:    "QUICK_MSECURITY_PAY",
		Subject:        in.Subject,
		OutTradeNo:     in.OutTradeNo,
		TotalAmount:    in.TotalAmount,
		TimeoutExpress: "30m",
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
