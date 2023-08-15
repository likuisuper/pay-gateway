package logic

import (
	"context"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"

	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ClosePayOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	payOrderModel  *model.PmPayOrderModel
	appConfigModel *model.PmAppConfigModel

	payConfigAlipayModel *model.PmPayConfigAlipayModel
	payConfigTiktokModel *model.PmPayConfigTiktokModel
	payConfigWechatModel *model.PmPayConfigWechatModel
	payConfigKsModel     *model.PmPayConfigKsModel
}

func NewClosePayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClosePayOrderLogic {
	return &ClosePayOrderLogic{
		ctx:                  ctx,
		svcCtx:               svcCtx,
		Logger:               logx.WithContext(ctx),
		payOrderModel:        model.NewPmPayOrderModel(define.DbPayGateway),
		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
		payConfigTiktokModel: model.NewPmPayConfigTiktokModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
		payConfigKsModel:     model.NewPmPayConfigKsModel(define.DbPayGateway),
	}
}

// 关闭订单
func (l *ClosePayOrderLogic) ClosePayOrder(in *pb.ClosePayOrderReq) (resp *pb.Empty, err error) {
	resp = &pb.Empty{}

	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		err = fmt.Errorf("读取应用配置失败 pkgName= %s, err:=%v", in.AppPkgName, err)
		util.CheckError(err.Error())
		return
	}

	switch in.PayType {
	case pb.PayType_WxUniApp:
		payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(pkgCfg.WechatPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("读取微信支付配置失败 pkgName= %s, err:=%v", in.AppPkgName, cfgErr)
			util.CheckError(err.Error())
			return
		}
		err = l.wxClosePayOrder(in, payCfg.TransClientConfig())
		if err != nil {
			err = fmt.Errorf("关闭微信订单失败, orderSn=%s, err=%v", in.OrderSn, err)
			util.CheckError(err.Error())
			return
		}
	case pb.PayType_KsUniApp:
		payCfg, cfgErr := l.payConfigKsModel.GetOneByAppID(pkgCfg.KsPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("读取快手支付配置失败 pkgName= %s, err:=%v", in.AppPkgName, cfgErr)
			util.CheckError(err.Error())
			return
		}
		payClient := client.NewKsPay(*payCfg.TransClientConfig())
		err = payClient.CancelChannel(in.OrderSn)
		if err != nil {
			err = fmt.Errorf("关闭微信订单失败, orderSn=%s, err=%v", in.OrderSn, err)
			util.CheckError(err.Error())
			return
		}
	}

	return
}

func (l *ClosePayOrderLogic) wxClosePayOrder(in *pb.ClosePayOrderReq, payConf *client.WechatPayConfig) (err error) {
	payClient := client.NewWeChatCommPay(*payConf)
	err = payClient.CloseOrder(in.OrderSn)
	return
}
