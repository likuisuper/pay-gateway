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

type AlipayPageUnSignLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigAlipayModel *model.PmPayConfigAlipayModel
	orderModel           *model.OrderModel
}

func NewAlipayPageUnSignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayPageUnSignLogic {
	return &AlipayPageUnSignLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
		orderModel:           model.NewOrderModel(define.DbPayGateway),
	}
}

// 支付宝：解约
func (l *AlipayPageUnSignLogic) AlipayPageUnSign(in *pb.AlipayPageUnSignReq) (*pb.AlipayCommonResp, error) {
	payClient, _, _, err := clientMgr.GetAlipayClientByAppPkgWithCache(in.AppPkgName)
	if err != nil {
		return nil, err
	}

	table, err := l.orderModel.GetOneByOutTradeNo(in.OutTradeNo)
	if err != nil {
		logx.Errorf("根据out_trade_no获取订单失败, err = %v", err.Error())
	}

	unSign := alipay2.AgreementUnsign{
		AgreementNo: table.AgreementNo,
	}

	result, err := payClient.AgreementUnsign(unSign)
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
