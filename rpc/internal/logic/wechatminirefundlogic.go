package logic

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"net/url"

	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type WechatMiniRefundLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigWechatModel *model.PmPayConfigWechatModel
	refundOrderModel     *model.PmRefundOrderModel
	orderModel           *model.PmPayOrderModel
}

func NewWechatMiniRefundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WechatMiniRefundLogic {
	return &WechatMiniRefundLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
		refundOrderModel:     model.NewPmRefundOrderModel(define.DbPayGateway),
		orderModel:           model.NewPmPayOrderModel(define.DbPayGateway),
	}
}

// 小程序-微信的退款申请
func (l *WechatMiniRefundLogic) WechatMiniRefund(in *pb.WechatMiniRefundReq) (out *pb.WechatMiniRefundResp, err error) {

	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		//util.CheckError("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		err = fmt.Errorf("WechatMiniRefund pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		util.CheckError(err.Error())
		return
	}

	orderInfo, err := l.orderModel.GetOneByOrderSnAndAppId(in.OutOrderNo, pkgCfg.WechatPayAppID)
	if err != nil || orderInfo == nil || orderInfo.ID < 1 {
		util.CheckError("WechatMiniRefund 获取订单失败订单号OrderSn:[%s] err:%v", in.OrderSn, err)
		err = errors.New("读取应用配置失败")
		return
	}

	payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(pkgCfg.WechatPayAppID)
	if cfgErr != nil {
		err = fmt.Errorf("WechatMiniRefund pkgName= %s, 读取微信支付配置失败，err:=%v", in.AppPkgName, cfgErr)
		util.CheckError(err.Error())
		return
	}

	data := &client.MiniRefundOrder{
		OutTradeNo:  in.OutOrderNo,
		OutRefundNo: in.OutRefundNo,
		TotalFee:    in.TotalAmount,
		RefundFee:   in.RefundAmount,
		Reason:      in.RefundReason,
	}

	clientConfig := *payCfg.TransClientConfig()
	payClient := client.NewWeChatCommPay(clientConfig)
	resp, err := payClient.MiniRefundOrder(data)
	if err != nil {
		err = fmt.Errorf("WechatMiniRefund 发起退款失败:OutRefundNo = %s .err =%v ", data.OutRefundNo, err)
		util.CheckError(err.Error())
		return nil, err
	}

	//回调地址
	if in.RefundNotifyUrl == "" && orderInfo.NotifyUrl != "" {
		urlInfo, _ := url.Parse(orderInfo.NotifyUrl)
		if urlInfo != nil && urlInfo.Host != "" {
			urlInfo.Path = "/notify/wechat/refund"
			in.RefundNotifyUrl = urlInfo.String()
		}
	}

	//写入数据库
	refundObj := &model.PmRefundOrderTable{
		AppID:        clientConfig.AppId,
		OutOrderNo:   in.OutOrderNo,
		OutRefundNo:  in.OutRefundNo,
		Reason:       in.RefundReason,
		RefundAmount: int(in.RefundAmount),
		NotifyUrl:    in.RefundNotifyUrl,
		RefundNo:     *resp.RefundId,
		RefundStatus: model.PmRefundOrderTableRefundStatusApply,
	}
	_ = l.refundOrderModel.Create(refundObj)

	out = &pb.WechatMiniRefundResp{
		RefundId: *resp.RefundId,
	}

	return
}
