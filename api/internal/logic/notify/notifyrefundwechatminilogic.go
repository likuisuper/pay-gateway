package notify

import (
	"context"
	"strings"
	"time"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/code"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/exception"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/util"
	"k8s.io/apimachinery/pkg/util/json"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyRefundWechatMiniLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	appConfigModel       *model.PmAppConfigModel
	payConfigWechatModel *model.PmPayConfigWechatModel
	refundOrderModel     *model.PmRefundOrderModel
	orderModel           *model.PmPayOrderModel
}

// 小程序业务-微信商户退款回调通知
func NewNotifyRefundWechatMiniLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyRefundWechatMiniLogic {
	return &NotifyRefundWechatMiniLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
		refundOrderModel:     model.NewPmRefundOrderModel(define.DbPayGateway),
		orderModel:           model.NewPmPayOrderModel(define.DbPayGateway),
	}
}

// https://pay.weixin.qq.com/doc/v3/merchant/4012791906
func (l *NotifyRefundWechatMiniLogic) NotifyRefundWechatMini(req *types.WechatMiniRefundReq) (resp *types.WeChatResp, err error) {

	defer func() {
		l.Slowf("NotifyRefundWechatMini req[%+v], respData[%+v],err[%v]", req, resp, err)
	}()

	//退款订单信息
	refundOrderInfo, err := l.refundOrderModel.GetInfo(req.OutRefundNo)

	if err != nil {
		//db此时异常，让继续回调通知，待db恢复可继续处理
		logx.Errorf("NotifyRefundWechatMini refundOrderModel err[%s] OutRefundNo[%s]", err.Error(), req.OutRefundNo)
		return
	}

	resp = &types.WeChatResp{
		Code:    "200",
		Message: "OK",
	}
	if refundOrderInfo == nil || refundOrderInfo.ID == 0 {
		logx.Errorf("NotifyRefundWechatMini 无系统退款单记录 OutRefundNo[%s]", req.OutRefundNo)
		//无记录
		return
	}

	//保存处理该条记录时，第三方的参数
	reqJsBt, _ := json.Marshal(req)
	refundOrderInfo.NotifyData = string(reqJsBt)

	switch req.EventType {
	case "REFUND.SUCCESS": //退款成功

		refundOrderInfo.RefundStatus = code.ORDER_SUCCESS

	case "REFUND.ABNORMAL": //退款异常通知

		refundOrderInfo.RefundStatus = model.PmRefundOrderTableRefundStatusFail

	case "REFUND.CLOSED": //退款关闭通知

	}

	//修改订单退款状态
	l.refundOrderModel.Update(req.OutRefundNo, refundOrderInfo)

	//回调业务方接口
	go func() {
		defer exception.Recover()
		if refundOrderInfo.RefundStatus != code.ORDER_SUCCESS && refundOrderInfo.RefundStatus != model.PmRefundOrderTableRefundStatusFail {
			return
		}

		order, _ := l.orderModel.GetOneByOrderSnAndAppId(refundOrderInfo.OutOrderNo, refundOrderInfo.AppID)
		if order == nil {
			return
		}

		headMap := map[string]string{
			"App-Origin": order.AppPkgName,
			"From-App":   order.AppPkgName,
		}

		signBeforeList := []string{
			order.AppPkgName,
			req.OutRefundNo,
			refundOrderInfo.RefundNo,
			req.EventType,
		}

		originData := map[string]string{
			"refund_id":     req.OutRefundNo,          //退款单号
			"ext_refund_id": refundOrderInfo.RefundNo, //三方平台退款单号
			"event_type":    req.EventType,
			"summary":       req.Summary,                                 //eg: 退款成功
			"sign":          util.Md5(strings.Join(signBeforeList, ",")), //签名完整性校验
		}
		respData, requestErr := util.HttpPostWithHeader(refundOrderInfo.NotifyUrl, originData, headMap, 5*time.Second)
		if requestErr != nil {
			CallbackRefundFailNum.CounterInc()
			CallbackBizFailNum.CounterInc()
			util.CheckError("NotifyRefundWechatMini NotifyRefund-post, req:%+v, err:%v", originData, requestErr)
			l.Errorf("NotifyRefundWechatMini req:%+v, err:%v, url:%v", originData, requestErr, refundOrderInfo.NotifyUrl)
			return
		}

		l.Slowf("NotifyRefundWechatMini req:%+v, respData:%s", originData, respData)
	}()

	return
}
