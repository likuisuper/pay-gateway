package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/types"
	"gitee.com/zhuyunkj/pay-gateway/common/utils"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"

	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type WechatUnifiedOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext

	logx.Logger
	appConfigModel       *model.PmAppConfigModel
	payConfigWechatModel *model.PmPayConfigWechatModel
	orderModel           *model.OrderModel
	huaweiOrderModel     *model.HuaweiOrderModel
}

func NewWechatUnifiedOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WechatUnifiedOrderLogic {
	return &WechatUnifiedOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,

		Logger:               logx.WithContext(ctx),
		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
		orderModel:           model.NewOrderModel(define.DbPayGateway),
		huaweiOrderModel:     model.NewHuaweiOrderModel(define.DbPayGateway),
	}
}

// 微信统一下单接口
func (l *WechatUnifiedOrderLogic) WechatUnifiedOrder(in *pb.AlipayPageSignReq) (*pb.WxUnifiedPayReply, error) {
	l.Sloww("WechatUnifiedOrder param", logx.Field("in", in))

	if in.GetIsHwPayProduct() {
		// 华为订阅商品 华为应用内购买 只创建订单
		return l.pureCreateHuaweiOrder(in)
	}

	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		err = fmt.Errorf("pkgName= %s, 读取应用配置失败 err: %v ", in.AppPkgName, err)
		util.CheckError(err.Error())
		return nil, err
	}

	product := types.Product{}
	var productType, intAmount int

	err = json.Unmarshal([]byte(in.ProductDesc), &product)
	if err != nil {
		parseProductDescErr.CounterInc()
		logx.Errorf("创建订单异常：商品信息错误 err = %s product = %s", err.Error(), in.ProductDesc)
		return nil, errors.New("商品信息错误")
	}
	intAmount = int(product.Amount * 100)
	productType = product.ProductType
	if intAmount <= 0 {
		parseProductDescErr.CounterInc()
		logx.Errorf("创建订单异常：商品金额异常 product = %s", in.ProductDesc)
		return nil, errors.New("商品信息错误")
	}
	orderInfo := &model.OrderTable{
		AppPkg:       in.AppPkgName,
		UserID:       int(in.UserId),
		OutTradeNo:   utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
		PayType:      model.PmPayOrderTablePayTypeWechatPayUni,
		Status:       0,
		PayAppID:     pkgCfg.WechatPayAppID,
		AppNotifyUrl: in.NotifyURL,
		Amount:       intAmount,
		ProductDesc:  in.ProductDesc,
		ProductType:  productType,
		ProductID:    int(in.ProductId),
		Subject:      in.Subject,
	}
	data, err := l.createWeChatUnifiedOrder(orderInfo, in.Ip)
	if err != nil {
		return nil, err
	}

	err = l.orderModel.Create(orderInfo)
	if err != nil {
		payAndSignCreateOrderErr.CounterInc()
		logx.Errorf("创建订单异常：创建订单表失败， err = %s", err.Error())
		return nil, errors.New("创建订单异常")
	}

	return data, nil
}

// 华为订阅商品 华为应用内购买 只创建订单
func (l *WechatUnifiedOrderLogic) pureCreateHuaweiOrder(in *pb.AlipayPageSignReq) (*pb.WxUnifiedPayReply, error) {
	productType := int(in.ProductType)
	orderInfo := model.HuaweiOrderTable{
		AppPkg:              in.GetAppPkgName(),
		AppId:               in.GetAppId(),
		UserId:              int(in.GetUserId()),
		Status:              0,
		Environment:         in.GetPurchaseEnv(),
		OutTradeNo:          utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
		PayType:             model.PmPayOrderTablePayTypeAlipay,
		Amount:              int(in.GetAmount()),
		ProductId:           in.GetProductIdStr(),
		ProductType:         productType,
		AppNotifyUrl:        in.GetNotifyURL(),
		PayAppId:            "",
		ProductDesc:         in.GetProductDesc(),
		DeviceId:            in.GetDeviceId(), // 在回调的时候 需要带到app_alipay_order表
		ExternalAgreementNo: utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
	}

	err := l.huaweiOrderModel.Create(&orderInfo)
	if err != nil {
		payAndSignCreateOrderErr.CounterInc()
		logx.Errorf("创建订单异常,创建订单表失败 err:%s, orderInfo: %+v", err.Error(), orderInfo)
		return nil, errors.New("创建订单异常,创建订单表失败")
	}

	return &pb.WxUnifiedPayReply{
		Prepayid:            "",
		MwebUrl:             "",
		OutTradeNo:          orderInfo.OutTradeNo,
		ExternalAgreementNo: orderInfo.ExternalAgreementNo,
	}, nil
}

// 微信统一支付
func (l *WechatUnifiedOrderLogic) createWeChatUnifiedOrder(orderInfo *model.OrderTable, ip string) (reply *pb.WxUnifiedPayReply, err error) {
	payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(orderInfo.PayAppID)
	if cfgErr != nil {
		err = fmt.Errorf("pkgName= %s, 读取微信支付配置失败，err:=%v", orderInfo.AppPkg, cfgErr)
		util.CheckError(err.Error())
		return nil, cfgErr
	}

	payClient := client.NewWeChatCommPay(*payCfg.TransClientConfig())
	payInfo := &client.PayOrder{
		OrderSn: orderInfo.OutTradeNo,
		Amount:  orderInfo.Amount,
		Subject: orderInfo.Subject,
		IP:      ip,
	}

	wechatPayConfig := payCfg.TransClientConfig()
	res, err := payClient.WechatPayUnified(payInfo, wechatPayConfig)
	if err != nil || res == nil {
		wechatNativePayFailNum.CounterInc()
		util.CheckError("WechatPayUnified pkgName: %s, payInfo:%v , err: %v", orderInfo.AppPkg, payInfo, err)
		return
	}

	reply = &pb.WxUnifiedPayReply{
		Prepayid:   res.PrepayID,
		MwebUrl:    res.MwebURL,
		OutTradeNo: orderInfo.OutTradeNo,
	}

	if wechatPayConfig.WapName != "" && wechatPayConfig.WapUrl != "" {
		reply.WapName = wechatPayConfig.WapName
		reply.WapUrl = wechatPayConfig.WapUrl
	}

	return
}
