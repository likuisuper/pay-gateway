package logic

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/zeromicro/go-zero/core/logx"
	"gitlab.muchcloud.com/consumer-project/alipay"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/client"
	douyin "gitlab.muchcloud.com/consumer-project/pay-gateway/common/client/douyinGeneralTrade"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/code"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/utils"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/pb/pb"
	kv_m "gitlab.muchcloud.com/consumer-project/zhuyun-core/kv_monitor"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/util"
)

var (
	//getAppConfigFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "getAppConfigFailNum", nil, "根据包名获取配置失败", nil})}
	alipayWapPayFailNum    = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "alipayWapPayFailNum", nil, "支付宝下单失败", nil})}
	wechatUniPayFailNum    = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "wechatUniPayFailNum", nil, "微信支付下单失败", nil})}
	wechatNativePayFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "wechatNativePayFailNum", nil, "微信native支付下单失败", nil})}
	tiktokEcPayFailNum     = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "tiktokEcPayFailNum", nil, "字节支付下单失败", nil})}
	alipayWebPayFailNum    = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "alipayWebPayFailNum", nil, "支付宝下单失败", nil})}
	ksPayFailNum           = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "ksPayFailNum", nil, "快手支付下单失败", nil})}

	orderTableIOFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "orderTableIOFailNum", nil, "订单io失败", nil})}
)

type OrderPayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	payOrderModel         *model.PmPayOrderModel
	payDyPeriodOrderModel *model.PmDyPeriodOrderModel
	appConfigModel        *model.PmAppConfigModel

	payConfigAlipayModel *model.PmPayConfigAlipayModel
	payConfigWechatModel *model.PmPayConfigWechatModel
	payConfigKsModel     *model.PmPayConfigKsModel
}

func NewOrderPayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OrderPayLogic {
	return &OrderPayLogic{
		ctx:                   ctx,
		svcCtx:                svcCtx,
		Logger:                logx.WithContext(ctx),
		payOrderModel:         model.NewPmPayOrderModel(define.DbPayGateway),
		payDyPeriodOrderModel: model.NewPmDyPeriodOrderModel(define.DbPayGateway),
		appConfigModel:        model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigAlipayModel:  model.NewPmPayConfigAlipayModel(define.DbPayGateway),
		payConfigWechatModel:  model.NewPmPayConfigWechatModel(define.DbPayGateway),
		payConfigKsModel:      model.NewPmPayConfigKsModel(define.DbPayGateway),
	}
}

// OrderPay 创建支付订单
func (l *OrderPayLogic) OrderPay(in *pb.OrderPayReq) (out *pb.OrderPayResp, err error) {
	if in.GetIsBaseExistSignedOrder() {
		// 基于已存在周期代扣签约订单创建代扣单
		return l.createOrderBaseSignedOrder(in)
	}

	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		err = fmt.Errorf("GetOneByPkgName failed pkgName: %s, err: %v ", in.AppPkgName, err)
		util.Error(l.ctx, err.Error())
		return
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
	case pb.PayType_TiktokEc, pb.PayType_DouyinGeneralTrade:
		payAppId = pkgCfg.TiktokPayAppID
	case pb.PayType_KsUniApp:
		payAppId = pkgCfg.KsPayAppID
	case pb.PayType_WxUnified:
		payAppId = pkgCfg.WechatPayAppID
	}

	tmpOrderSn := ""
	tmpOrderAmount := 0
	tmpOrderSubject := ""

	if in.IsPeriodProduct {
		// 周期签约订单 (目前只有抖音有)
		orderInfo := &model.PmDyPeriodOrderTable{
			OrderSn:           in.GetOrderSn(),                        // 内部 订单唯一标识
			SignNo:            in.GetOrderSn() + dy_sign_order_suffix, // 内部 签约单号
			UserId:            int(in.InnerUserId),
			AppPkgName:        in.AppPkgName,
			Amount:            int(in.Amount),
			Subject:           in.Subject,
			NotifyUrl:         in.NotifyURL,
			PayAppId:          payAppId,        //创建订单时，直接指定PayAppid，减少一次DB操作
			PayType:           int(in.PayType), // 创建订单时，传入支付类型，补偿机制依赖
			PayStatus:         model.PmPayOrderTablePayStatusNo,
			Currency:          in.Currency.String(),
			SignDate:          model.Default2000Date, // 默认时间
			UnsignDate:        model.Default2000Date,
			ExpireDate:        model.Default2000Date,
			NextDecuctionTime: model.Default2000Date,
			DyProductId:       in.DouyinGeneralTradeReq.GetSkuId(), // 抖音商品id
		}
		// 数据库有唯一约束 如果重复 创建的时候会报错
		err = l.payDyPeriodOrderModel.Create(orderInfo)
		if err != nil {
			err = fmt.Errorf("创建周期签约订单失败 %w", err)
			l.Errorw("创建周期签约订单失败", logx.Field("err", err), logx.Field("orderInfo", orderInfo))
			orderTableIOFailNum.CounterInc()
			return
		}

		tmpOrderSn = orderInfo.OrderSn
		tmpOrderAmount = orderInfo.Amount
		tmpOrderSubject = orderInfo.Subject
	} else {
		//创建订单时订单号对包隔离 规避业务方订单号重复case
		var orderInfo *model.PmPayOrderTable
		orderInfo, err = l.payOrderModel.GetOneByOrderSnAndAppId(in.OrderSn, payAppId)
		if err != nil {
			err = fmt.Errorf("GetOneByOrderSnAndAppId err:%v, orderSn:%s, appId:%s ", err, in.OrderSn, payAppId)
			l.Error(err)
			orderTableIOFailNum.CounterInc()
			return
		}

		if orderInfo != nil && orderInfo.ID > 0 {
			// 其实到这里 应该是出错了 订单号不能重复
			err = fmt.Errorf("订单号不能重复 orderSn:%s, appId:%s ", in.OrderSn, payAppId)
			l.Error(err.Error())
			orderTableIOFailNum.CounterInc()
			return
		}

		orderInfo = &model.PmPayOrderTable{
			OrderSn:    in.OrderSn,
			AppPkgName: in.AppPkgName,
			Amount:     int(in.Amount),
			Subject:    in.Subject,
			NotifyUrl:  in.NotifyURL,
			PayAppId:   payAppId,        //创建订单时，直接指定PayAppid，减少一次DB操作
			PayType:    int(in.PayType), // 创建订单时，传入支付类型，补偿机制依赖
			PayStatus:  model.PmPayOrderTablePayStatusNo,
			Currency:   in.Currency.String(),
		}
		err = l.payOrderModel.Create(orderInfo)
		if err != nil {
			err = fmt.Errorf("创建支付订单失败 %w", err)
			l.Errorw("创建支付订单失败", logx.Field("err", err), logx.Field("orderInfo", orderInfo))
			orderTableIOFailNum.CounterInc()
			return
		}

		tmpOrderSn = orderInfo.OrderSn
		tmpOrderAmount = orderInfo.Amount
		tmpOrderSubject = orderInfo.Subject
	}

	out = new(pb.OrderPayResp)
	out.PayType = in.PayType

	payOrder := &client.PayOrder{
		OrderSn: tmpOrderSn,
		Amount:  tmpOrderAmount,
		Subject: tmpOrderSubject,
	}

	switch out.PayType {
	case pb.PayType_AlipayWap:
		//小程序未用到
		payCfg, cfgErr := l.payConfigAlipayModel.GetOneByAppID(pkgCfg.AlipayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取支付宝配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.Error(l.ctx, err.Error())
			return
		}
		out.AlipayWap, err = l.createAlipayWapOrder(in, payCfg.TransClientConfig())
	case pb.PayType_AlipayWeb:
		//小程序未用到
		payCfg, cfgErr := l.payConfigAlipayModel.GetOneByAppID(pkgCfg.AlipayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取支付宝配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.Error(l.ctx, err.Error())
			return
		}
		out.AlipayWeb, err = l.createAlipayWebOrder(in, payCfg.TransClientConfig())
	case pb.PayType_WxUniApp:
		payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(pkgCfg.WechatPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取微信支付配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.Error(l.ctx, err.Error())
			return
		}

		l.Sloww("payCfg", logx.Field("payCfg", payCfg))

		out.WxUniApp, err = l.createWeChatUniOrder(in, payOrder, payCfg.TransClientConfig())
	case pb.PayType_WxWeb:
		//未用
		payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(pkgCfg.WechatPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取微信支付配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.Error(l.ctx, err.Error())
			return
		}
		out.WxNative, err = l.createWeChatNativeOrder(in, payOrder, payCfg.TransClientConfig())
	case pb.PayType_TiktokEc:
		//未用到
		payCfg, cfgErr := model.NewPmPayConfigTiktokModel(define.DbPayGateway).GetOneByAppID(pkgCfg.TiktokPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取字节支付配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.Error(l.ctx, err.Error())
			return
		}
		out.TikTokEc, err = l.createTikTokEcOrder(in, payOrder, payCfg.TransClientConfig())
	case pb.PayType_KsUniApp:
		// 快手小程序
		payCfg, cfgErr := l.payConfigKsModel.GetOneByAppID(pkgCfg.KsPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取快手支付配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.Error(l.ctx, err.Error())
			return
		}

		// 经营类目 虚拟/服务 虚拟卡/会员/游戏 在线影视/音乐/阅读/社交软件会员
		// 经营类目编号 1273
		// 状态 已通过
		// 快手的微信和支付宝账号 已经在快手开发平台绑定好了, 路径: 交易管理-支付管理
		payOrder.KsTypeId = 1273 // 固定值
		out.KsUniApp, err = l.createKsOrder(in, payOrder, payCfg.TransClientConfig())
	case pb.PayType_WxUnified: //未用到
		payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(pkgCfg.WechatPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("pkgName= %s, 读取微信支付配置失败，err:=%v", in.AppPkgName, cfgErr)
			util.Error(l.ctx, err.Error())
			return
		}
		out.WxUnified, err = l.createWeChatUnifiedOrder(in, payOrder, payCfg.TransClientConfig())
	case pb.PayType_DouyinGeneralTrade:
		if !in.IsPeriodProduct {
			// 周期代扣没有商品信息 不需要检查
			// 非周期代扣才需要检查
			checkParamErr := l.checkDouyinGeneralTradeParam(in)
			if checkParamErr != nil {
				err = checkParamErr
				util.CheckError("checkDouyinGeneralTradeParam fail pkgName: %s, err: %v ", in.AppPkgName, checkParamErr)
				return
			}
		}

		payCfg, cfgErr := model.NewPmPayConfigTiktokModel(define.DbPayGateway).GetOneByAppID(pkgCfg.TiktokPayAppID)
		if cfgErr != nil {
			err = fmt.Errorf("GetOneByAppID pkgName: %s, err: %v ", in.AppPkgName, cfgErr)
			util.Error(l.ctx, err.Error())
			return
		}

		out.DouyinGeneralTrade, err = l.createDouyinGeneralTradeOrder(in, payCfg.GetGeneralTradeConfig())
	}

	return
}

// 基于已存在周期代扣签约订单创建代扣单
func (l *OrderPayLogic) createOrderBaseSignedOrder(in *pb.OrderPayReq) (*pb.OrderPayResp, error) {
	l.Sloww("createOrderBaseSignedOrder", logx.Field("params", in))

	out := new(pb.OrderPayResp)
	orderNo := in.GetExistSignedOrderNo()
	if orderNo == "" {
		return out, errors.New("参数ExistSignedOrderNo为空")
	}

	tbl, err := l.payDyPeriodOrderModel.GetOneByOrderSnAndPkg(orderNo, in.GetAppPkgName())
	if err != nil || tbl == nil {
		return out, errors.New("GetOneByOrderSnAndPkg失败: " + err.Error())
	}

	if tbl.UserId != int(in.GetInnerUserId()) {
		return out, errors.New("用户ID不匹配")
	}

	newAmount := int(in.GetAmount())
	newOrderSn := utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo)

	orderInfo := &model.PmDyPeriodOrderTable{
		OrderSn:           newOrderSn,                        // 内部 订单唯一标识
		SignNo:            newOrderSn + dy_sign_order_suffix, // 内部 签约单号
		UserId:            tbl.UserId,
		AppPkgName:        tbl.AppPkgName,
		Amount:            newAmount,
		Subject:           "签约代扣单",
		NotifyUrl:         tbl.NotifyUrl,
		PayAppId:          tbl.PayAppId,
		PayType:           tbl.PayType,
		PayStatus:         model.PmPayOrderTablePayStatusNo,
		Currency:          tbl.Currency,
		SignDate:          tbl.SignDate, // 默认时间
		UnsignDate:        tbl.UnsignDate,
		ExpireDate:        tbl.ExpireDate,
		NextDecuctionTime: tbl.NextDecuctionTime.AddDate(0, 1, 0),
		DyProductId:       tbl.DyProductId,     // 抖音商品id
		NthNum:            int(in.GetNthNum()), // 第几期代扣单
	}

	// 数据库有唯一约束 如果重复 创建的时候会报错
	err = l.payDyPeriodOrderModel.Create(orderInfo)
	if err != nil {
		err = fmt.Errorf("创建周期签约订单失败 %w", err)
		l.Errorf("payDyPeriodOrderModel.Create failed", logx.Field("err", err), logx.Field("orderInfo", orderInfo))
		return out, err
	}

	out.OrderSn = newOrderSn

	return out, nil
}

// 支付宝wap支付
func (l *OrderPayLogic) createAlipayWapOrder(in *pb.OrderPayReq, payConf *client.AliPayConfig) (payUrl string, err error) {
	// 将 key 的验证调整到初始化阶段
	payClient, err := client.GetAlipayClient(*payConf)
	if err != nil {
		util.Error(l.ctx, "pkgName= %s, 初使化支付错误，err:=%v", in.AppPkgName, err)
		return
	}
	//发起支付请求
	var amount = float64(in.Amount) / 100
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
		util.Error(l.ctx, "pkgName= %s, alipayWapPay，err:=%v", in.AppPkgName, err)
		return
	}
	payUrl = res.String()

	return
}

// 支付宝web支付
func (l *OrderPayLogic) createAlipayWebOrder(in *pb.OrderPayReq, payConf *client.AliPayConfig) (payUrl string, err error) {
	// 将 key 的验证调整到初始化阶段
	payClient, err := client.GetAlipayClient(*payConf)
	if err != nil {
		util.Error(l.ctx, "pkgName= %s, 初使化支付错误，err:=%v", in.AppPkgName, err)
		return
	}
	//发起支付请求
	var amount = float64(in.Amount) / 100
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
		util.Error(l.ctx, "pkgName= %s, alipayWapPay，err:=%v", in.AppPkgName, err)
		return
	}
	payUrl = res.String()

	return
}

// 微信小程序支付 JSAPI
func (l *OrderPayLogic) createWeChatUniOrder(in *pb.OrderPayReq, info *client.PayOrder, payConf *client.WechatPayConfig) (reply *pb.WxUniAppPayReply, err error) {
	payClient := client.NewWeChatCommPay(*payConf)
	res, err := payClient.WechatPayV3(info, in.WxOpenID)
	if err != nil {
		wechatUniPayFailNum.CounterInc()
		util.Error(l.ctx, "pkgName= %s, wechatUniPay，err:=%v", in.AppPkgName, err)
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

// 微信web支付
func (l *OrderPayLogic) createWeChatNativeOrder(in *pb.OrderPayReq, info *client.PayOrder, payConf *client.WechatPayConfig) (reply *pb.WxNativePayReply, err error) {
	payClient := client.NewWeChatCommPay(*payConf)
	res, err := payClient.WechatPayV3Native(info)
	if err != nil {
		wechatNativePayFailNum.CounterInc()
		util.Error(l.ctx, "pkgName= %s, wechatUniPay，err:=%v", in.AppPkgName, err)
		return
	}

	var png []byte
	png, err = qrcode.Encode(*res.CodeUrl, qrcode.Medium, 256)
	if err != nil {
		wechatNativePayFailNum.CounterInc()
		util.Error(l.ctx, "pkgName= %s, wechatUniPay，err:=%v", in.AppPkgName, err)
		return
	}
	baseEncode := base64.StdEncoding.EncodeToString(png)

	reply = &pb.WxNativePayReply{
		CodeUrl:    *res.CodeUrl,
		CodeBase64: baseEncode,
	}
	return
}

// 微信统一下单
func (l *OrderPayLogic) createWeChatUnifiedOrder(in *pb.OrderPayReq, info *client.PayOrder, payConf *client.WechatPayConfig) (reply *pb.WxUnifiedPayReply, err error) {
	payClient := client.NewWeChatCommPay(*payConf)
	res, err := payClient.WechatPayUnified(info, payConf)
	if err != nil || res == nil {
		wechatNativePayFailNum.CounterInc()
		util.Error(l.ctx, "WechatPayUnified pkgName: %s, err: %v", in.AppPkgName, err)
		return
	}

	reply = &pb.WxUnifiedPayReply{
		Prepayid: res.PrepayID,
		MwebUrl:  res.MwebURL,
	}
	if payConf.WapName != "" && payConf.WapUrl != "" {
		reply.WapName = payConf.WapName
		reply.WapUrl = payConf.WapUrl
	}
	return
}

// 抖音小程序支付
func (l *OrderPayLogic) createTikTokEcOrder(in *pb.OrderPayReq, info *client.PayOrder, payConf *client.TikTokPayConfig) (reply *pb.TiktokEcPayReply, err error) {
	payClient := client.NewTikTokPay(*payConf)
	res, err := payClient.CreateEcPayOrder(info)
	if err != nil {
		tiktokEcPayFailNum.CounterInc()
		util.Error(l.ctx, "CreateEcPayOrder pkgName: %s, err: %v ", in.AppPkgName, err)
		return
	}

	reply = &pb.TiktokEcPayReply{
		OrderId:    res.Data.OrderId,
		OrderToken: res.Data.OrderToken,
	}
	return
}

// 快手小程序支付
func (l *OrderPayLogic) createKsOrder(in *pb.OrderPayReq, info *client.PayOrder, payConf *client.KsPayConfig) (reply *pb.KsUniAppReply, err error) {
	ksAccessToken, err := l.svcCtx.BaseAppConfigServerApi.GetKsAppidToken(l.ctx, payConf.AppId)
	if err != nil {
		util.Error(l.ctx, "快手获取access token失败 pkgName:%s, appId:%v, err:%v", in.AppPkgName, payConf.AppId, err)
		return
	}

	payClient := client.NewKsPay(*payConf)

	var res *client.KsCreateOrderWithChannelResp

	// in.WxOpenID 实际上是快手open id, 名称相同而已
	if strings.ToLower(in.Os) == "ios" {
		// ios订单
		res, err = payClient.CreateOrderIos(info, in.WxOpenID, ksAccessToken)
		if err != nil || res == nil {
			ksPayFailNum.CounterInc()
			util.Error(l.ctx, "CreateOrderIos failed pkgName: %s, err: %v", in.AppPkgName, err)
			err = errors.New("创建IOS订单失败: " + err.Error())
			return
		}
	} else {
		// 默认安卓订单
		res, err = payClient.CreateOrder(info, in.WxOpenID, ksAccessToken)
		if err != nil || res == nil {
			ksPayFailNum.CounterInc()
			util.Error(l.ctx, "CreateOrder failed pkgName: %s, err: %v", in.AppPkgName, err)
			err = errors.New("创建安卓订单失败: " + err.Error())
			return
		}
	}

	reply = &pb.KsUniAppReply{
		OrderNo:        res.OrderNo,
		OrderInfoToken: res.OrderInfoToken,
	}
	return
}

func (l *OrderPayLogic) checkDouyinGeneralTradeParam(in *pb.OrderPayReq) error {
	if in.DouyinGeneralTradeReq == nil {
		return errors.New("invalid DouyinGeneralTradeReq")
	}

	req := in.DouyinGeneralTradeReq
	if req == nil {
		return errors.New("invalid DouyinGeneralTradeReq")
	}

	if req.Type == pb.DouyinGeneralTradeReq_Unknown || pb.DouyinGeneralTradeReq_SkuType_name[int32(req.Type)] == "" {
		return errors.New("invalid sku type")
	}

	return nil
}

// 抖音小程序通用交易系统
func (l *OrderPayLogic) createDouyinGeneralTradeOrder(in *pb.OrderPayReq, payConf *douyin.PayConfig) (reply *pb.DouyinGeneralTradeReply, err error) {
	if in.IsPeriodProduct {
		// 抖音周期代扣
		return l.createDouyinPeriodOrder(in, payConf)
	}

	payClient := douyin.NewDouyinPay(payConf)
	douyinReq := in.DouyinGeneralTradeReq
	sku := &douyin.Sku{
		SkuId:       douyinReq.SkuId,
		Price:       douyinReq.Price,
		Quantity:    douyinReq.Quantity,
		Title:       douyinReq.Title,
		ImageList:   douyinReq.ImageList,
		Type:        douyin.SkuType(douyinReq.Type),
		TagGroupId:  douyin.SkuTagGroupId(douyinReq.TagGroupId),
		EntrySchema: nil,
		SkuAttr:     douyinReq.SkuAttr,
	}
	if douyinReq.GetEntrySchema() != nil {
		sku.EntrySchema = &douyin.Schema{
			Path:   douyinReq.GetEntrySchema().GetPath(),
			Params: douyinReq.GetEntrySchema().GetParams(),
		}
	}
	if douyinReq.Type == pb.DouyinGeneralTradeReq_ContentRecharge {
		sku.Type = douyin.SkuContentRecharge
		sku.TagGroupId = douyin.SKuTagGroupIdContentRecharge
	}
	data := &douyin.RequestOrderData{
		SkuList: []*douyin.Sku{
			sku,
		},
		OutOrderNo:       in.OrderSn,
		TotalAmount:      int32(in.Amount),
		PayExpireSeconds: code.DouyinPayExpireSeconds, // 默认是半个小时
		PayNotifyUrl:     payConf.NotifyUrl,
		MerchantUid:      payConf.MerchantUid,
		OrderEntrySchema: &douyin.Schema{
			Path:   douyinReq.GetOrderEntrySchema().GetPath(),
			Params: douyinReq.GetOrderEntrySchema().GetParams(),
		},
		LimitPayWayList: douyinReq.LimitPayWayList,
	}

	if in.Os == code.OsIos {
		switch in.DouyinGeneralTradeReq.IosPayType {
		case pb.DouyinGeneralTradeReq_IosPayTypeIm:
			data.PayScene = douyin.PaySceneIM
		case pb.DouyinGeneralTradeReq_IosPayTypeDiamond:
			data.Currency = douyin.CurrencyDiamond
			// 钻石支付暂不使用自定义商户号：钻石支付的商户号是新生成的，和普通支付不同
			data.MerchantUid = ""
		default:
			data.PayScene = douyin.PaySceneIM // 版本兼容
		}
	}

	// 生成签名
	dataStr, byteAuthorization, err := payClient.RequestOrder(data)
	if err != nil {
		tiktokEcPayFailNum.CounterInc()
		msg := fmt.Sprintf("douyin RequestOrder failed pkgName=%s, err:=%v", in.AppPkgName, err)
		l.Error(msg)
		return
	}

	reply = &pb.DouyinGeneralTradeReply{
		Data:              dataStr,
		ByteAuthorization: byteAuthorization,
		CustomerImId:      payConf.CustomerImId,
	}
	return
}

// 抖音签约单后缀
const dy_sign_order_suffix = "0"

// 抖音小程序生成周期代扣签名等
// https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/api/industry/credit-products/createSignOrder
func (l *OrderPayLogic) createDouyinPeriodOrder(in *pb.OrderPayReq, payConf *douyin.PayConfig) (reply *pb.DouyinGeneralTradeReply, err error) {
	douyinReq := in.DouyinGeneralTradeReq

	authPayOrder := &douyin.AuthPayOrderObj{
		OutPayOrderNo: in.GetOrderSn(), // 代扣单单号
		MerchantUid:   in.GetMerchantUid(),
		NotifyUrl:     payConf.NotifyUrl,
	}

	// 首期代扣金额
	initialAmount := in.GetFirstDeductionAmount()
	if initialAmount > 0 {
		authPayOrder.InitialAmount = &initialAmount
	}

	// 首次扣款日期
	firstDeductionDate := time.Now().Format("2006-01-02")
	data := &douyin.RequestPeriodOrderData{
		OutAuthOrderNo:     in.GetOrderSn() + dy_sign_order_suffix, // 周期签约订单后面固定添加后缀
		ServiceId:          douyinReq.GetSkuId(),
		OpenId:             in.GetInnerUserOpen(),
		ExpireSeconds:      code.DouyinPayExpireSeconds, // 默认是半个小时
		NotifyUrl:          payConf.NotifyUrl,
		OnBehalfUid:        strconv.Itoa(int(in.GetInnerUserId())),
		AuthPayOrder:       authPayOrder,
		FirstDeductionDate: &firstDeductionDate, // 首次扣款日期
	}

	l.Sloww("createDouyinPeriodOrder", logx.Field("data", data))

	// 生成签名
	dataStr, byteAuthorization, err := douyin.NewDouyinPay(payConf).CreateSignOrder(data)
	if err != nil {
		tiktokEcPayFailNum.CounterInc()
		msg := fmt.Sprintf("douyin CreateSignOrder failed pkgName: %s, err: %v ", in.AppPkgName, err)
		l.Error(msg)
		return
	}

	reply = &pb.DouyinGeneralTradeReply{
		Data:              dataStr,
		ByteAuthorization: byteAuthorization,
	}
	return
}
