package logic

import (
	"context"
	"fmt"

	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/thirdApis"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"

	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type WechatMiniXPayQueryOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigWechatModel *model.PmPayConfigWechatModel
	refundOrderModel     *model.PmRefundOrderModel
}

func NewWechatMiniXPayQueryOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WechatMiniXPayQueryOrderLogic {
	return &WechatMiniXPayQueryOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
		refundOrderModel:     model.NewPmRefundOrderModel(define.DbPayGateway),
	}
}

// 微信虚拟支付-退款/订单详情
func (l *WechatMiniXPayQueryOrderLogic) WechatMiniXPayQueryOrder(in *pb.WechatMiniXPayQueryOrderReq) (out *pb.WechatMiniXPayQueryOrderResp, err error) {
	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		//util.CheckError("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		err = fmt.Errorf("WechatMiniXPayQueryOrder pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		util.CheckError(err.Error())
		return
	}

	payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(pkgCfg.WechatPayAppID)
	if cfgErr != nil || payCfg.XPayAppKey == "" {
		err = fmt.Errorf("WechatMiniXPayQueryOrder pkgName= %s, 读取微信xpay支付配置失败 err:=%v", in.AppPkgName, cfgErr)
		util.CheckError(err.Error())
		return
	}

	token, err := l.svcCtx.BaseAppConfigServerApi.GetWxAccessToken(l.ctx, pkgCfg.WechatPayAppID)
	if err != nil {
		l.Errorf("WechatMiniXPayQueryOrder accessToken fail err= %v, appid:%s, pkgName:%s", err, pkgCfg.WechatPayAppID, in.AppPkgName)
		return
	}

	param := &thirdApis.XPayQueryOrderParam{
		OpenId:    in.OpenId,
		OrderId:   in.OutOrderNo,
		WxOrderId: in.WxOrderId,
	}
	wechatOrderRes, err := thirdApis.WechatXPayApi.QueryOrder(param, payCfg.XPayAppKey, token)
	if err != nil {
		l.Errorf("WechatXPayApi QueryOrder fail err= %v, appid:%s, pkgName:%s", err, pkgCfg.WechatPayAppID, in.AppPkgName)
		return
	}

	if wechatOrderRes.ErrCode != 0 {
		err = fmt.Errorf("WechatXPayApi.QueryOrder errcode[%d],errmsg[%s]", wechatOrderRes.ErrCode, wechatOrderRes.ErrMsg)
		return
	}

	out = &pb.WechatMiniXPayQueryOrderResp{
		OutOrderNo:   wechatOrderRes.Order.OrderId,
		WxOrderId:    wechatOrderRes.Order.WxOrderId,
		CreateTime:   wechatOrderRes.Order.CreateTime,
		UpdateTime:   wechatOrderRes.Order.UpdateTime,
		Status:       int64(wechatOrderRes.Order.Status),
		BizType:      int64(wechatOrderRes.Order.BizType),
		OrderFee:     int64(wechatOrderRes.Order.OrderFee),
		CouponFee:    int64(wechatOrderRes.Order.CouponFee),
		PaidFee:      int64(wechatOrderRes.Order.PaidFee),
		OrderType:    int64(wechatOrderRes.Order.OrderType),
		RefundFee:    int64(wechatOrderRes.Order.RefundFee),
		PaidTime:     int64(wechatOrderRes.Order.PaidTime),
		LeftFee:      int64(wechatOrderRes.Order.LeftFee),
		WxPayOrderId: wechatOrderRes.Order.WxPayOrderId,
	}

	refundInfo, err := l.refundOrderModel.GetInfo(in.OutOrderNo)
	if err == nil && refundInfo.OutRefundNo != "" && refundInfo.AppID == pkgCfg.WechatPayAppID {
		//当前状态 0-订单初始化（未创建成功，不可用于支付）1-订单创建成功 2-订单已经支付，待发货 3-订单发货中 4-订单已发货 5-订单已经退款 6-订单已经关闭（不可再使用） 7-订单退款失败 8-用户退款完成 9-回收广告金完成 10-分账回退完成
		//
		if refundInfo.RefundStatus != model.PmRefundOrderTableRefundStatusSuccess && wechatOrderRes.Order.Status == 8 {
			refundInfo.RefundStatus = model.PmRefundOrderTableRefundStatusSuccess
			l.refundOrderModel.Update(in.OutOrderNo, refundInfo)
		}

		if refundInfo.RefundStatus != model.PmRefundOrderTableRefundStatusFail && wechatOrderRes.Order.Status == 7 {
			refundInfo.RefundStatus = model.PmRefundOrderTableRefundStatusFail
			l.refundOrderModel.Update(in.OutOrderNo, refundInfo)
		}

	}

	return
}
