package logic

import (
	"context"
	alipay2 "gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/clientMgr"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AlipayRefundLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigAlipayModel *model.PmPayConfigAlipayModel
}

func NewAlipayRefundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayRefundLogic {
	return &AlipayRefundLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
	}
}

// 支付宝：退款
func (l *AlipayRefundLogic) AlipayRefund(in *pb.AlipayRefundReq) (*pb.AlipayCommonResp, error) {
	payClient, _, _, err := clientMgr.GetAlipayClientByAppPkgWithCache(in.AppPkgName)
	if err != nil {
		return nil, err
	}

	tradeRefund := alipay2.TradeRefund{
		TradeNo:      in.TradeNo,
		RefundAmount: in.RefundAmount,
		RefundReason: in.RefundReason,
	}

	result, err := payClient.TradeRefund(tradeRefund)
	if err != nil {
		logx.Errorf(err.Error())
	}

	if result.Content.Code == alipay2.CodeSuccess {
		return &pb.AlipayCommonResp{
			Status: code.ALI_PAY_SUCCESS,
		}, nil
	} else {
		return &pb.AlipayCommonResp{
			Status: code.ALI_PAY_FAIL,
			Desc:   "Msg: " + result.Content.Msg + " SubMsg: " + result.Content.SubMsg,
		}, err
	}
}
