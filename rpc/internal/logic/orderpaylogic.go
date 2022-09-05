package logic

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/skip2/go-qrcode"
	"github.com/smartwalle/alipay/v3"
	"github.com/zeromicro/go-zero/core/logx"
	"strconv"
)

var (
	//getAppConfigFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "getAppConfigFailNum", nil, "根据包名获取配置失败", nil})}
	alipayWapPayFailNum    = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "alipayWapPayFailNum", nil, "支付宝下单失败", nil})}
	wechatUniPayFailNum    = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "wechatUniPayFailNum", nil, "微信支付下单失败", nil})}
	wechatNativePayFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "wechatNativePayFailNum", nil, "微信native支付下单失败", nil})}
	tiktokEcPayFailNum     = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "tiktokEcPayFailNum", nil, "字节支付下单失败", nil})}
	alipayWebPayFailNum    = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "alipayWebPayFailNum", nil, "支付宝下单失败", nil})}

	orderTableIOFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "orderTableIOFailNum", nil, "订单io失败", nil})}
)

type OrderPayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	payOrderModel  *model.PmPayOrderModel
	appConfigModel *model.PmAppConfigModel

	payConfigAlipayModel *model.PmPayConfigAlipayModel
	payConfigTiktokModel *model.PmPayConfigTiktokModel
	payConfigWechatModel *model.PmPayConfigWechatModel
}

func NewOrderPayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OrderPayLogic {
	return &OrderPayLogic{
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

// 创建支付订单
func (l *OrderPayLogic) OrderPay(in *pb.OrderPayReq) (out *pb.OrderPayResp, err error) {
	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		//util.CheckError("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		err = fmt.Errorf("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		util.CheckError(err.Error())
		return
	}

	//获取订单信息
	orderInfo, err := l.payOrderModel.GetOneByCode(in.OrderSn)
	if err != nil {
		err = fmt.Errorf("获取订单信息错误 %w", err)
		logx.Error(err)
		orderTableIOFailNum.CounterInc()
		return
	}

	if orderInfo == nil {
		orderInfo = &model.PmPayOrderTable{
			OrderSn:    in.OrderSn,
			AppPkgName: in.AppPkgName,
			Amount:     int(in.Amount),
			Subject:    in.Subject,
			NotifyUrl:  in.NotifyURL,
			PayStatus:  model.PmPayOrderTablePayStatusNo,
		}
		err = l.payOrderModel.Create(orderInfo)
		if err != nil {
			err = fmt.Errorf("创建支付订单失败 %w", err)
			logx.Error(err)
			orderTableIOFailNum.CounterInc()
			return
		}
	} else {
		if orderInfo.PayStatus != model.PmPayOrderTablePayStatusNo {
			err = errors.New("订单不是未支付状态")
			util.CheckError(err.Error())
			return
		}
	}

	out = new(pb.OrderPayResp)
	out.PayType = in.PayType

	payOrder := &client.PayOrder{
		OrderSn: orderInfo.OrderSn,
		Amount:  orderInfo.Amount,
		Subject: orderInfo.Subject,
	}

	var payAppId string
	switch in.PayType {
	case pb.PayType_AlipayWap:
		payAppId = pkgCfg.AlipayAppID
	case pb.PayType_AlipayWeb:
		payAppId = pkgCfg.AlipayAppID
	case pb.PayType_WxUniApp:
		payAppId = pkgCfg.WechatPayAppID
	case pb.PayType_WxWeb:
		payAppId = pkgCfg.WechatPayAppID
	case pb.PayType_TiktokEc:
		payAppId = pkgCfg.TiktokPayAppID
	}
	err = l.payOrderModel.UpdatePayAppID(orderInfo.OrderSn, payAppId)
	if err != nil {
		return
	}

	switch out.PayType {
	case pb.PayType_AlipayWap:
		payCfg, cfgErr := l.payConfigAlipayModel.GetOneByAppID(pkgCfg.AlipayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取支付宝配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.CheckError(err.Error())
			return
		}
		out.AlipayWap, err = l.createAlipayWapOrder(in, payCfg.TransClientConfig())
	case pb.PayType_AlipayWeb:
		payCfg, cfgErr := l.payConfigAlipayModel.GetOneByAppID(pkgCfg.AlipayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取支付宝配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.CheckError(err.Error())
			return
		}
		out.AlipayWeb, err = l.createAlipayWebOrder(in, payCfg.TransClientConfig())
	case pb.PayType_WxUniApp:
		payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(pkgCfg.WechatPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取微信支付配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.CheckError(err.Error())
			return
		}
		out.WxUniApp, err = l.createWeChatUniOrder(in, payOrder, payCfg.TransClientConfig())
	case pb.PayType_WxWeb:
		payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(pkgCfg.WechatPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取微信支付配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.CheckError(err.Error())
			return
		}
		out.WxNative, err = l.createWeChatNativeOrder(in, payOrder, payCfg.TransClientConfig())
	case pb.PayType_TiktokEc:
		payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(pkgCfg.TiktokPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取字节支付配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.CheckError(err.Error())
			return
		}
		out.TikTokEc, err = l.createTikTokEcOrder(in, payOrder, payCfg.TransClientConfig())
	}

	return
}

//支付宝wap支付
func (l *OrderPayLogic) createAlipayWapOrder(in *pb.OrderPayReq, payConf *client.AliPayConfig) (payUrl string, err error) {
	// 将 key 的验证调整到初始化阶段
	payClient, err := client.GetAlipayClient(*payConf)
	if err != nil {
		util.CheckError("pkgName= %s, 初使化支付错误，err:=%v", in.AppPkgName, err)
		return
	}
	//发起支付请求
	var amount float64 = float64(in.Amount) / 100
	sendAmount := strconv.FormatFloat(amount, 'f', 2, 32)
	var p = alipay.TradeWapPay{}
	p.NotifyURL = payConf.NotifyUrl
	p.ReturnURL = in.ReturnURL
	p.Subject = in.Subject
	p.OutTradeNo = in.OrderSn
	p.TotalAmount = sendAmount
	p.ProductCode = "QUICK_WAP_WAY"

	res, err := payClient.TradeWapPay(p)
	if err != nil {
		alipayWapPayFailNum.CounterInc()
		util.CheckError("pkgName= %s, alipayWapPay，err:=%v", in.AppPkgName, err)
		return
	}
	payUrl = res.String()

	return
}

//支付宝web支付
func (l *OrderPayLogic) createAlipayWebOrder(in *pb.OrderPayReq, payConf *client.AliPayConfig) (payUrl string, err error) {
	// 将 key 的验证调整到初始化阶段
	payClient, err := client.GetAlipayClient(*payConf)
	if err != nil {
		util.CheckError("pkgName= %s, 初使化支付错误，err:=%v", in.AppPkgName, err)
		return
	}
	//发起支付请求
	var amount float64 = float64(in.Amount) / 100
	sendAmount := strconv.FormatFloat(amount, 'f', 2, 32)
	var p = alipay.TradePagePay{}
	p.NotifyURL = payConf.NotifyUrl
	p.ReturnURL = in.ReturnURL
	p.Subject = in.Subject
	p.OutTradeNo = in.OrderSn
	p.TotalAmount = sendAmount
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"

	res, err := payClient.TradePagePay(p)
	if err != nil {
		alipayWebPayFailNum.CounterInc()
		util.CheckError("pkgName= %s, alipayWapPay，err:=%v", in.AppPkgName, err)
		return
	}
	payUrl = res.String()

	return
}

//微信小程序支付
func (l *OrderPayLogic) createWeChatUniOrder(in *pb.OrderPayReq, info *client.PayOrder, payConf *client.WechatPayConfig) (reply *pb.WxUniAppPayReply, err error) {
	payClient := client.NewWeChatCommPay(*payConf)
	res, err := payClient.WechatPayV3(info, in.WxOpenID)
	if err != nil {
		wechatUniPayFailNum.CounterInc()
		util.CheckError("pkgName= %s, wechatUniPay，err:=%v", in.AppPkgName, err)
		return
	}
	reply = &pb.WxUniAppPayReply{
		OrderInfo: res.OrderInfo,
		TimeStamp: res.TimeStamp,
		NonceStr:  res.NonceStr,
		Package:   res.Package,
		SignType:  res.SignType,
		PaySign:   res.PaySign,
		OrderSn:   res.OrderCode,
	}
	return
}

//微信web支付
func (l *OrderPayLogic) createWeChatNativeOrder(in *pb.OrderPayReq, info *client.PayOrder, payConf *client.WechatPayConfig) (reply *pb.WxNativePayReply, err error) {
	payClient := client.NewWeChatCommPay(*payConf)
	res, err := payClient.WechatPayV3Native(info)
	if err != nil {
		wechatNativePayFailNum.CounterInc()
		util.CheckError("pkgName= %s, wechatUniPay，err:=%v", in.AppPkgName, err)
		return
	}

	var png []byte
	png, err = qrcode.Encode(*res.CodeUrl, qrcode.Medium, 256)
	if err != nil {
		wechatNativePayFailNum.CounterInc()
		util.CheckError("pkgName= %s, wechatUniPay，err:=%v", in.AppPkgName, err)
		return
	}
	baseEncode := base64.StdEncoding.EncodeToString(png)

	reply = &pb.WxNativePayReply{
		CodeUrl:    *res.CodeUrl,
		CodeBase64: baseEncode,
	}
	return
}

//抖音小程序支付
func (l *OrderPayLogic) createTikTokEcOrder(in *pb.OrderPayReq, info *client.PayOrder, payConf *client.TikTokPayConfig) (reply *pb.TiktokEcPayReply, err error) {
	payClient := client.NewTikTokPay(*payConf)
	res, err := payClient.CreateEcPayOrder(info)
	if err != nil {
		tiktokEcPayFailNum.CounterInc()
		util.CheckError("pkgName= %s, tiktokEcPay，err:=%v", in.AppPkgName, err)
		return
	}
	reply = &pb.TiktokEcPayReply{
		OrderId:    res.Data.OrderId,
		OrderToken: res.Data.OrderToken,
	}
	return
}
