package inter

import (
	"context"
	alipay2 "gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/api/common/response"
	"gitee.com/zhuyunkj/pay-gateway/common/clientMgr"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"strconv"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type HandleRefundLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	refundModel *model.RefundModel
	orderModel  *model.OrderModel
}

func NewHandleRefundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HandleRefundLogic {
	return &HandleRefundLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,

		refundModel: model.NewRefundModel(define.DbPayGateway),
		orderModel:  model.NewOrderModel(define.DbPayGateway),
	}
}

func (l *HandleRefundLogic) HandleRefund(req *types.RefundReq) (resp *types.ResultResp, err error) {
	// todo: add your logic here and delete this line
	if req.Reviewer == "" {
		res := response.MakeResult(code.CODE_ERROR, "审核人员必填", nil)
		return &res, nil
	}

	table, err := l.refundModel.GetOneByOutTradeRefundNo(req.OutTradeRefundNo)
	if err != nil {
		res := response.MakeResult(code.CODE_ERROR, "退款单号不存在", nil)
		return &res, nil
	}

	table.RefundStatus = req.Status
	if req.Status == model.REFUND_STATUS_SUCCESS {

		payClient, _, _, err := clientMgr.GetAlipayClientByAppPkgWithCache(table.AppPkg)
		if err != nil {
			return nil, err
		}

		a := strconv.Itoa(table.RefundAmount / 100)
		b := strconv.Itoa(table.RefundAmount % 100)

		tradeRefund := alipay2.TradeRefund{
			TradeNo:      table.OutTradeNo,
			RefundAmount: a + "." + b,
			RefundReason: table.Reason,
		}

		result, err := payClient.TradeRefund(tradeRefund)
		if err != nil {
			logx.Errorf(err.Error())
			res := response.MakeResult(code.CODE_ERROR, "退款异常", nil)
			return &res, nil
		}

		if result.Content.Code == alipay2.CodeSuccess {
			table.RefundStatus = model.REFUND_STATUS_SUCCESS
			err = l.refundModel.Update(table.OutTradeRefundNo, table)
			if err != nil {
				res := response.MakeResult(code.CODE_ERROR, "更新退款状态异常", nil) // TODO: 更新时异常的处理
				return &res, nil
			}
		}
	} else {
		table.RefundStatus = model.REFUND_STATUS_FAILD
		table.ReviewerComment = req.ReviewerComment
		err = l.refundModel.Update(table.OutTradeRefundNo, table)
		if err != nil {
			res := response.MakeResult(code.CODE_ERROR, "更新退款状态异常", nil) // TODO: 更新时异常的处理
			return &res, nil
		}
	}

	res := response.MakeResult(code.CODE_OK, "操作成功", nil)
	return &res, nil

}
