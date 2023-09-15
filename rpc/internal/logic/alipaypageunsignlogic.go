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
}

func NewAlipayPageUnSignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayPageUnSignLogic {
	return &AlipayPageUnSignLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
	}
}

// 支付宝：解约
func (l *AlipayPageUnSignLogic) AlipayPageUnSign(in *pb.AlipayPageUnSignReq) (*pb.AlipayCommonResp, error) {
	payClient, _, _, err := clientMgr.GetAlipayClientWithCache(in.AppPkgName)
	if err != nil {
		return nil, err
	}

	unSign := alipay2.AgreementUnsign{
		ExternalAgreementNo: in.ExternalAgreementNo,
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
