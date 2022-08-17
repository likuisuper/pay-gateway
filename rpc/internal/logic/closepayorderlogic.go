package logic

import (
	"context"
	"errors"
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
	}
}

// 关闭订单
func (l *ClosePayOrderLogic) ClosePayOrder(in *pb.ClosePayOrderReq) (resp *pb.Empty, err error) {
	resp = &pb.Empty{}

	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		util.CheckError("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		err = errors.New("读取应用配置失败")
		return
	}

	switch in.PayType {
	case pb.PayType_WxUniApp:
		payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(pkgCfg.WechatPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取微信支付配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.CheckError(err.Error())
			return
		}
		err = l.wxClosePayOrder(in, payCfg.TransClientConfig())
	}

	return
}

func (l *ClosePayOrderLogic) wxClosePayOrder(in *pb.ClosePayOrderReq, payConf *client.WechatPayConfig) (err error) {
	payClient := client.NewWeChatCommPay(*payConf)
	err = payClient.CloseOrder(in.OrderSn)
	return
}
