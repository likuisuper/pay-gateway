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

type WechatMiniXPayRefundLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigWechatModel *model.PmPayConfigWechatModel
	refundOrderModel     *model.PmRefundOrderModel
}

func NewWechatMiniXPayRefundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WechatMiniXPayRefundLogic {
	return &WechatMiniXPayRefundLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
		refundOrderModel:     model.NewPmRefundOrderModel(define.DbPayGateway),
	}
}

// 微信虚拟支付-退款申请
func (l *WechatMiniXPayRefundLogic) WechatMiniXPayRefund(in *pb.WechatMiniXPayRefundReq) (out *pb.WechatMiniXPayRefundResp, err error) {
	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		//util.CheckError("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		err = fmt.Errorf("WechatMiniXPayRefund pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		util.CheckError(err.Error())
		return
	}

	//---微信虚拟支付，暂时未经过pay-gateway，此处不对支付单校验---

	payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(pkgCfg.WechatPayAppID)
	if cfgErr != nil || payCfg.XPayAppKey == "" {
		err = fmt.Errorf("WechatMiniXPayRefund pkgName= %s, 读取微信xpay支付配置失败，err:=%v", in.AppPkgName, cfgErr)
		util.CheckError(err.Error())
		return
	}
	token, err := l.svcCtx.BaseAppConfigServerApi.GetWxAccessToken(l.ctx, pkgCfg.WechatPayAppID)
	if err != nil {
		l.Errorf("WechatMiniXPayQueryOrder accessToken fail，err= %v, appid:%s, pkgName:%s", err, pkgCfg.WechatPayAppID, in.AppPkgName)
		return
	}
	queryParam := &thirdApis.XPayQueryOrderParam{
		OpenId:  in.OpenId,
		OrderId: in.OutOrderNo,
	}
	wechatOrderRes, err := thirdApis.WechatXPayApi.QueryOrder(queryParam, payCfg.XPayAppKey, token)
	if err != nil && wechatOrderRes.ErrCode != 0 {
		l.Errorf("WechatXPayApi QueryOrder fail，err= %v, appid:%s, pkgName:%s", err, pkgCfg.WechatPayAppID, in.AppPkgName)
		return
	}

	param := &thirdApis.XPayRefundOrderParam{
		OpenId:        in.OpenId,
		OrderId:       in.OutOrderNo,
		RefundOrderId: in.OutRefundNo,
		LeftFee:       wechatOrderRes.Order.LeftFee,
		RefundFee:     int(in.RefundAmount),
		BizMeta:       "",
		//refund_reason错误，当前只支持"0"-暂无描述 "1"-产品问题，影响使用或效果不佳
		//"2"-售后问题，无法满足需求 "3"-意愿问题，用户主动退款 "4"-价格问题 "5"-其他原因
		RefundReason: "0",
		//req_from错误，当前只支持"1"-人工客服退款，即用户电话给客服，由客服发起退款流程 "2"-用户自己发起退款流程 "3"-其他
		ReqFrom: "3",
	}
	refundOrderInfo, err := thirdApis.WechatXPayApi.RefundOrder(param, payCfg.XPayAppKey, token)
	if err != nil {
		l.Errorf("WechatXPayApi QueryOrder fail，err= %v, appid:%s, pkgName:%s", err, pkgCfg.WechatPayAppID, in.AppPkgName)
		return
	}

	//写入数据库
	refundObj := &model.PmRefundOrderTable{
		AppID:        pkgCfg.WechatPayAppID,
		OutOrderNo:   in.OutOrderNo,
		OutRefundNo:  in.OutRefundNo,
		Reason:       in.RefundReason,
		RefundAmount: int(in.RefundAmount),
		//NotifyUrl:    in.RefundNotifyUrl,  //微信虚拟支付 微信开放平台暂无回调功能
		RefundNo:     refundOrderInfo.RefundWxOrderId,
		RefundStatus: 0,
	}
	_ = l.refundOrderModel.Create(refundObj)

	out = &pb.WechatMiniXPayRefundResp{
		RefundId: refundOrderInfo.RefundWxOrderId,
	}

	return
}
