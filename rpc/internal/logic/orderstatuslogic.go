package logic

import (
	"context"
	"fmt"

	"gitee.com/zhuyunkj/pay-gateway/common/client"
	douyin "gitee.com/zhuyunkj/pay-gateway/common/client/douyinGeneralTrade"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	jsoniter "github.com/json-iterator/go"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"

	"github.com/zeromicro/go-zero/core/logx"
)

type OrderStatusLogic struct {
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

func NewOrderStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OrderStatusLogic {
	return &OrderStatusLogic{
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

// OrderStatus 查询订单
func (l *OrderStatusLogic) OrderStatus(in *pb.OrderStatusReq) (resp *pb.OrderStatusResp, err error) {
	resp = new(pb.OrderStatusResp)
	resp.OrderSn = in.OrderSn

	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		err = fmt.Errorf("读取应用配置失败 pkgName= %s, err:=%v", in.AppPkgName, err)
		util.CheckError(err.Error())
		return
	}

	switch in.PayType {
	case pb.PayType_WxUniApp, pb.PayType_WxWeb:
		payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(pkgCfg.WechatPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("读取微信支付配置失败 pkgName= %s, err:=%v", in.AppPkgName, cfgErr)
			util.CheckError(err.Error())
			return
		}
		transaction, err := l.wxOrderStatus(in, payCfg.TransClientConfig())
		if err != nil {
			err = fmt.Errorf("查询微信订单失败, orderSn=%s, err=%v", in.OrderSn, err)
			util.CheckError(err.Error())
			return nil, err
		}
		jsonStr, _ := jsoniter.MarshalToString(transaction)
		resp.ThirdRespJson = jsonStr
		if *transaction.TradeState == "SUCCESS" {
			resp.Status = 1
			resp.PayAmount = *transaction.Amount.PayerTotal
		}
	case pb.PayType_TiktokEc:
		payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(pkgCfg.TiktokPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取字节支付配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.CheckError(err.Error())
			return
		}
		orderInfo, err := l.tiktokOrderStatus(in, payCfg.TransClientConfig())
		if err != nil {
			err = fmt.Errorf("查询字节订单失败, orderSn=%s, err=%v", in.OrderSn, err)
			util.CheckError(err.Error())
			return nil, err
		}
		jsonStr, _ := jsoniter.MarshalToString(orderInfo)
		resp.ThirdRespJson = jsonStr
		if orderInfo.OrderStatus == "SUCCESS" {
			resp.Status = 1
			resp.PayAmount = int64(orderInfo.TotalFee)
		}
	case pb.PayType_KsUniApp:
		// 快手
		payCfg, cfgErr := l.payConfigKsModel.GetOneByAppID(pkgCfg.KsPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName=%s, 读取快手支付配置失败 err=%v", in.AppPkgName, cfgErr)
			util.CheckError(err.Error())
			return
		}

		payClient := client.NewKsPay(*payCfg.TransClientConfig())
		orderInfo, err := payClient.QueryOrder(in.OrderSn)
		if err != nil {
			err = fmt.Errorf("查询快手订单失败, orderSn=%s, err=%v", in.OrderSn, err)
			util.CheckError(err.Error())
			return nil, err
		}

		if orderInfo.PayStatus == "SUCCESS" {
			resp.Status = 1
			resp.PayAmount = int64(orderInfo.TotalAmount)
		}
	case pb.PayType_DouyinGeneralTrade:
		payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(pkgCfg.TiktokPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取抖音支付配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.Error(l.ctx, err.Error())
			return
		}

		douyinPayConfig := payCfg.GetGeneralTradeConfig()
		payClient := douyin.NewDouyinPay(douyinPayConfig)

		clientToken, err := l.svcCtx.BaseAppConfigServerApi.GetDyClientToken(l.ctx, douyinPayConfig.AppId)
		if err != nil {
			l.Errorw("get douyin client token fail", logx.Field("err", err), logx.Field("appId", douyinPayConfig.AppId))
			return nil, err
		}

		orderInfo, err := payClient.QueryOrder("", in.OrderSn, clientToken)
		if err != nil {
			err = fmt.Errorf("查询抖音订单失败, orderSn=%s, err=%v", in.OrderSn, err)
			util.Error(l.ctx, err.Error())
			return nil, err
		}

		jsonStr, _ := jsoniter.MarshalToString(orderInfo)
		resp.ThirdRespJson = jsonStr
		if orderInfo.Data != nil && orderInfo.Data.PayStatus == "SUCCESS" {
			resp.Status = 1
			resp.PayAmount = orderInfo.Data.TotalAmount
		}
	}

	return
}

func (l *OrderStatusLogic) wxOrderStatus(in *pb.OrderStatusReq, payConf *client.WechatPayConfig) (transaction *payments.Transaction, err error) {
	payClient := client.NewWeChatCommPay(*payConf)
	transaction, err = payClient.GetOrderStatus(in.OrderSn)
	return
}

func (l *OrderStatusLogic) tiktokOrderStatus(in *pb.OrderStatusReq, payConf *client.TikTokPayConfig) (orderInfo *client.TikTokPaymentInfo, err error) {
	payCli := client.NewTikTokPay(*payConf)
	orderInfo, err = payCli.GetOrderStatus(in.OrderSn)
	return
}
