package logic

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/comm/client"
	"gitee.com/zhuyunkj/pay-gateway/comm/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/pb/pb"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/smartwalle/alipay/v3"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	getPkgConfigFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "getPkgConfigFailNum", nil, "根据包名获取配置失败数量", nil})}
	alipayWapPayFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "alipayWapPayFailNum", nil, "根据包名获取配置失败数量", nil})}
)

type OrderPayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	payOrderModel *model.PmPayOrderModel
}

func NewOrderPayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OrderPayLogic {
	return &OrderPayLogic{
		ctx:           ctx,
		svcCtx:        svcCtx,
		Logger:        logx.WithContext(ctx),
		payOrderModel: model.NewPmPayOrderModel(define.DbPayGatGateway),
	}
}

// 创建支付订单
func (l *OrderPayLogic) OrderPay(in *pb.OrderPayReq) (out *pb.OrderPayResp, err error) {
	//读取应用配置
	pkgCfg := l.svcCtx.AppConfigMap[in.AppPkgName]
	if pkgCfg == nil {
		getPkgConfigFailNum.CounterInc()
		err = errors.New("读取应用配置appRel错误")
		return
	}

	//获取订单信息
	orderInfo, err := l.payOrderModel.GetOneByCode(in.OrderSn)
	if err != nil {
		err = fmt.Errorf("获取订单信息错误 %w", err)
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
			return
		}
	} else {
		if orderInfo.PayStatus != model.PmPayOrderTablePayStatusNo {
			err = errors.New("订单不是未支付状态")
			return
		}
	}

	out = new(pb.OrderPayResp)
	out.PayType = in.PayType

	switch out.PayType {
	case pb.PayType_AlipayWap:
		out.AlipayWap, err = l.createAlipayWapOrder(in, pkgCfg)
	case pb.PayType_WxUniApp:
		out.WxUniApp, err = l.createWeChatUniOrder(in, orderInfo, pkgCfg)
	case pb.PayType_TiktokEc:
		out.TikTokEc, err = l.createTikTokEcOrder(orderInfo, pkgCfg)
	}

	return
}

//支付宝wap支付
func (l *OrderPayLogic) createAlipayWapOrder(in *pb.OrderPayReq, pkgCfg *svc.AppPkgConfig) (payUrl string, err error) {
	// 将 key 的验证调整到初始化阶段
	payClient, err := client.GetAlipayClient(pkgCfg.Alipay)
	if err != nil {
		util.CheckError("pkgName= %s, 初使化支付错误，err:=%v", pkgCfg.AppRel.AppPkgName, err)
		return
	}
	//发起支付请求
	var amount float64 = float64(in.Amount) / 100
	sendAmount := strconv.FormatFloat(amount, 'f', 2, 32)
	var p = alipay.TradeWapPay{}
	p.NotifyURL = in.NotifyURL
	p.ReturnURL = in.ReturnURL
	p.Subject = in.Subject
	p.OutTradeNo = in.OrderSn
	p.TotalAmount = sendAmount
	p.ProductCode = "QUICK_WAP_WAY"

	res, err := payClient.TradeWapPay(p)
	if err != nil {
		alipayWapPayFailNum.CounterInc()
		util.CheckError("pkgName= %s, alipayWapPay，err:=%v", pkgCfg.AppRel.AppPkgName, err)
		return
	}
	payUrl = res.String()

	return
}

//微信小程序支付
func (l *OrderPayLogic) createWeChatUniOrder(in *pb.OrderPayReq, info *model.PmPayOrderTable, pkgCfg *svc.AppPkgConfig) (reply *pb.WxUniAppPayReply, err error) {
	payClient := client.NewWeChatCommPay(pkgCfg.WechatPay)
	res, err := payClient.WechatPayV3(info, in.WxOpenID)
	if err != nil {
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

//抖音小程序支付
func (l *OrderPayLogic) createTikTokEcOrder(info *model.PmPayOrderTable, pkgCfg *svc.AppPkgConfig) (reply *pb.TiktokEcPayReply, err error) {
	payClient := client.NewTikTokPay(pkgCfg.TikTokPay)
	res, err := payClient.CreateEcPayOrder(info)
	if err != nil {
		err = fmt.Errorf("创建订单失败 %w", err)
		return
	}
	reply = &pb.TiktokEcPayReply{
		OrderId:    res.Data.OrderId,
		OrderToken: res.Data.OrderToken,
	}
	return
}
