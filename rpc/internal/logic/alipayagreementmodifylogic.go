package logic

import (
	"context"
	"errors"
	"time"

	"gitlab.muchcloud.com/consumer-project/alipay"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/clientMgr"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/code"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AlipayAgreementModifyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	orderModel *model.OrderModel
}

func NewAlipayAgreementModifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayAgreementModifyLogic {
	return &AlipayAgreementModifyLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		orderModel: model.NewOrderModel(define.DbPayGateway),
	}
}

// 支付宝：签约延期
func (l *AlipayAgreementModifyLogic) AlipayAgreementModify(in *pb.AlipayAgreementModifyReq) (*pb.AlipayCommonResp, error) {
	client, _, _, err := clientMgr.GetAlipayClientByAppPkgWithCache(in.AppPkgName)
	if err != nil {
		logx.Errorf("延期扣款：获取支付宝客户端失败 agreementNo=%s err=%s", in.OutTradeNo, err.Error())
		return nil, errors.New("扣款失败")
	}

	tb, err := l.orderModel.GetOneByOutTradeNo(in.OutTradeNo)
	if err != nil || tb == nil || tb.ID < 1 {
		logx.Error(err.Error())
		return nil, err
	}

	if tb.AgreementNo == "" || tb.ProductType != code.PRODUCT_TYPE_SUBSCRIBE {
		return nil, errors.New("订单号异常，不是签约订单")
	}

	params := alipay.AgreementExecutionPlanModify{
		AgreementNo: tb.AgreementNo,
		DeductTime:  in.DeductTime,
	}

	deductOrder, err := l.orderModel.GetOneByOutTradeNo(in.DeductOutTradeNo)
	if err != nil {
		logx.Error(err.Error())
		return nil, err
	}

	if deductOrder == nil || deductOrder.ID < 1 {
		return nil, errors.New("扣款订单不存在， out_trade_no: " + in.DeductOutTradeNo)
	}

	result, err := client.AgreementExecutionPlanModify(params)
	if err != nil {
		logx.Error(err.Error())
		return nil, err
	}

	if result.Content.Code == alipay.CodeSuccess {
		deductTime, _ := time.Parse("2006-01-02", in.DeductTime)

		deductOrder.DeductTime = deductTime.AddDate(0, 0, -5) // 新的截止日期的前5天可以开始扣款

		err = l.orderModel.UpdateNotify(deductOrder)
		if err != nil {
			logx.Errorf("延期扣款：更新扣款订单失败 agreementNo=%s err=%s", tb.AgreementNo, err.Error())
			return nil, err
		}

		return &pb.AlipayCommonResp{
			Status: code.ALI_PAY_SUCCESS,
		}, nil
	} else {

		deductOrder.Status = model.PmPayOrderTablePayStatusFailed

		err = l.orderModel.UpdateNotify(deductOrder)
		if err != nil {
			logx.Errorf("延期扣款：更新扣款订单失败 agreementNo=%s err=%s", tb.AgreementNo, err.Error())
			return nil, err
		}

		return &pb.AlipayCommonResp{
			Status: code.ALI_PAY_FAIL,
			Desc:   "Msg: " + result.Content.Msg + " SubMsg: " + result.Content.SubMsg,
		}, err
	}

}
