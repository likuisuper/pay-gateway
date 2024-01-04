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

type WechatPayH5OrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	appConfigModel       *model.PmAppConfigModel
	payConfigWechatModel *model.PmPayConfigWechatModel
	orderModel           *model.OrderModel
}

func NewWechatPayH5OrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WechatPayH5OrderLogic {
	return &WechatPayH5OrderLogic{
		ctx:                  ctx,
		svcCtx:               svcCtx,
		Logger:               logx.WithContext(ctx),
		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
		orderModel:           model.NewOrderModel(define.DbPayGateway),
	}
}

// 微信h5支付，对接文档：https://pay.weixin.qq.com/docs/merchant/apis/h5-payment/direct-jsons/h5-prepay.html
func (l *WechatPayH5OrderLogic) WechatPayH5Order(in *pb.AlipayPageSignReq) (*pb.WxH5PayReplay, error) {
	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		//util.CheckError("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		err = fmt.Errorf("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
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
	data, err := l.createWeChatH5Order(orderInfo, in.Ip)
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

	return &pb.WxH5PayReplay{}, nil
}

// 微信统一支付
func (l *WechatPayH5OrderLogic) createWeChatH5Order(orderInfo *model.OrderTable, ip string) (reply *pb.WxH5PayReplay, err error) {
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
	res, err := payClient.WechatPayV3H5(payInfo)
	if err != nil {
		wechatNativePayFailNum.CounterInc()
		util.CheckError("pkgName= %s, wechatUniPay，err:=%v", orderInfo.AppPkg, err)
		return
	}
	reply = &pb.WxH5PayReplay{
		H5Url: *res.H5Url,
	}
	return
}
