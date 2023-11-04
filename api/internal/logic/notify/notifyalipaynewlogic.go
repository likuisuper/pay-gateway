package notify

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/clientMgr"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/common/utils"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"gorm.io/gorm"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyAlipayNewLogic struct {
	logx.Logger
	ctx         context.Context
	svcCtx      *svc.ServiceContext
	orderModel  *model.OrderModel
	refundModel *model.RefundModel
}

func NewNotifyAlipayNewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyAlipayNewLogic {
	return &NotifyAlipayNewLogic{
		Logger:      logx.WithContext(ctx),
		ctx:         ctx,
		svcCtx:      svcCtx,
		orderModel:  model.NewOrderModel(define.DbPayGateway),
		refundModel: model.NewRefundModel(define.DbPayGateway),
	}
}

const (
	ALI_NOTIFY_TYPE_TRADE_SYNC = "trade_status_sync"
	ALI_NOTIFY_TYPE_SIGN       = "dut_user_sign"
	ALI_NOTIFY_TYPE_UNSIGN     = "dut_user_unsign"
)

var (
	notifyAlipaySignErrNum   = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "notifyAlipaySignErrNum", nil, "支付宝签约回调失败", nil})}
	notifyAlipayUnSignErrNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "notifyAlipayUnSignErrNum", nil, "支付宝解约回调失败", nil})}
)

func (l *NotifyAlipayNewLogic) NotifyAlipayNew(r *http.Request, w http.ResponseWriter) (resp *types.EmptyReq, err error) {
	// todo: add your logic here and delete this line

	err = r.ParseForm()
	if err != nil {
		logx.Errorf("NotifyAlipay err: %v", err)
		notifyAlipayErrNum.CounterInc()
		return
	}
	bodyData := r.Form.Encode()
	logx.Slowf("NotifyAlipay form %s", bodyData)

	appId := r.Form.Get("app_id")

	client, _, _, err := clientMgr.GetAlipayClientByAppIdWithCache(appId)
	if err != nil {
		logx.Errorf(err.Error())
		notifyAlipayErrNum.CounterInc()
		return
	}

	notifyType := r.Form.Get("notify_type")

	if ok, signErr := client.VerifySign(r.Form); !ok || signErr != nil {
		desc := "支付宝回调验签失败!!"
		if signErr != nil {
			desc += signErr.Error()
		}
		logx.Errorf(desc)
		notifyAlipayErrNum.CounterInc()
		return
	}

	if ALI_NOTIFY_TYPE_TRADE_SYNC == notifyType {

		var outTradeNo = r.Form.Get("out_trade_no")
		var tradeNo = r.Form.Get("trade_no")
		var tradeQuery = alipay.TradeQuery{
			OutTradeNo: outTradeNo,
		}
		refundFee := r.Form.Get("refund_fee")
		if refundFee == "" { // 支付成功
			res, aliErr := client.TradeQuery(tradeQuery)
			if aliErr != nil {
				aliErr = fmt.Errorf("TradeQuery err=%v", aliErr)
				logx.Error(aliErr)
				notifyAlipayErrNum.CounterInc()
			}
			if res.IsSuccess() == false {
				logx.Errorf("NotifyAlipay success false %s", outTradeNo)
				notifyAlipayErrNum.CounterInc()
				return
			}

			//获取订单信息
			orderInfo, dbErr := l.orderModel.GetOneByOutTradeNo(outTradeNo)
			if dbErr != nil {
				dbErr = fmt.Errorf("获取订单失败！err=%v,order_code = %s", dbErr, outTradeNo)
				util.CheckError(dbErr.Error())
				return
			}
			if orderInfo.Status != model.PmPayOrderTablePayStatusNo {
				notifyOrderHasDispose.CounterInc()
				err = fmt.Errorf("订单已处理")
				return
			}
			//修改数据库
			orderInfo.Status = model.PmPayOrderTablePayStatusPaid
			orderInfo.PayType = model.PmPayOrderTablePayTypeAlipay
			orderInfo.PlatformTradeNo = tradeNo
			err = l.orderModel.UpdateNotify(orderInfo)
			if err != nil {
				err = fmt.Errorf("trade_no = %s, UpdateNotify，err:=%v", orderInfo.PlatformTradeNo, err)
				util.CheckError(err.Error())
				return
			}

			//回调业务方接口
			go func() {
				defer exception.Recover()
				dataMap := l.transFormDataToMap(bodyData)
				dataMap["notify_type"] = code.APP_NOTIFY_TYPE_PAY
				if orderInfo.ProductType == code.PRODUCT_TYPE_SUBSCRIBE_FEE {
					dataMap["external_agreement_no"] = orderInfo.ExternalAgreementNo
				}
				utils.CallbackWithRetry(orderInfo.AppNotifyUrl, dataMap, 5*time.Second)
			}()
		} else { // 退款

			amountFloat, parseErr := strconv.ParseFloat(refundFee, 64)
			if parseErr != nil || amountFloat <= 0 {
				err = fmt.Errorf("退款参数异常， out_trade_no = %s, err:=%v", outTradeNo, err)
				util.CheckError(parseErr.Error())
				return
			}

			//获取订单信息
			orderInfo, dbErr := l.orderModel.GetOneByOutTradeNo(outTradeNo)
			if dbErr != nil {
				dbErr = fmt.Errorf("获取订单失败！err=%v,order_code = %s", dbErr, outTradeNo)
				util.CheckError(dbErr.Error())
				return
			}
			if orderInfo.Status != model.PmPayOrderTablePayStatusPaid {
				notifyOrderHasDispose.CounterInc()
				err = fmt.Errorf("订单状态异常")
				return
			}

			table, dbErr := l.refundModel.GetOneByOutTradeNo(outTradeNo)
			if dbErr != nil && !errors.Is(dbErr, gorm.ErrRecordNotFound) {
				err = fmt.Errorf("退款回调db服务异常， out_trade_no = %s, err:=%v", outTradeNo, dbErr)
				util.CheckError(err.Error())
			}
			if table != nil { // 已经有退款单，是用户主动退款，不在这处理
				return
			}

			refundOutSideApp := false
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				err = fmt.Errorf("退款回调db服务异常， out_trade_no = %s, err:=%v", outTradeNo, err)
				util.CheckError(err.Error())
			}

			refundAmount := int(amountFloat * 100)

			//if table == nil { // 支付网关中没有，可能是用户自己通过申诉退款，不走我们的退款途径，创建一个退款单
			//	refundOutSideApp = true
			//	table = &model.RefundTable{
			//		PayType:          orderInfo.PayType,
			//		OutTradeNo:       orderInfo.OutTradeNo,
			//		OutTradeRefundNo: utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
			//		Reason:           "用户通过支付宝退款",
			//		RefundAmount:     refundAmount,
			//		NotifyUrl:        orderInfo.AppNotifyUrl,
			//		Operator:         "user",
			//		AppPkg:           orderInfo.AppPkg,
			//		RefundedAt:       time.Now(),
			//		RefundNo:         orderInfo.PlatformTradeNo, // 支付宝没有退款单号，先用支付单号
			//	}
			//	err = l.refundModel.Create(table)
			//	if err != nil {
			//		err = fmt.Errorf("退款回调：创建退款单失败， out_trade_no = %s, err:=%v", outTradeNo, err)
			//		util.CheckError(err.Error())
			//	}
			//}

			// 回调通知退款成功
			go func() {
				defer exception.Recover()
				dataMap := make(map[string]interface{})
				dataMap["notify_type"] = code.APP_NOTIFY_TYPE_REFUND
				dataMap["out_trade_refund_no"] = table.OutTradeRefundNo
				dataMap["out_trade_no"] = outTradeNo
				dataMap["refund_out_side_app"] = refundOutSideApp
				dataMap["refund_status"] = model.REFUND_STATUS_SUCCESS
				dataMap["refund_fee"] = refundAmount
				utils.CallbackWithRetry(table.NotifyUrl, dataMap, 5*time.Second)
			}()
		}

	} else if ALI_NOTIFY_TYPE_SIGN == notifyType {

		agreementNo := r.Form.Get("agreement_no")
		externalAgreementNo := r.Form.Get("external_agreement_no")

		if agreementNo == "" || externalAgreementNo == "" {
			logx.Errorf("签约回调参数异常, %s", bodyData)
			return
		}

		if err != nil {
			logx.Errorf(err.Error())
			notifyAlipaySignErrNum.CounterInc()
			return
		}

		order, dbErr := l.orderModel.GetOneByExternalAgreementNo(externalAgreementNo)
		if dbErr != nil {
			logx.Errorf("获取订单详情失败: %v", dbErr.Error())
			notifyAlipaySignErrNum.CounterInc()
			return
		}

		if order.AgreementNo != "" {
			logx.Errorf("已经签约成功: agreementNo=%v, externalAgreementNo=%v", agreementNo, externalAgreementNo)
			notifyAlipaySignErrNum.CounterInc()
			return
		}

		order.AgreementNo = agreementNo
		order.ExternalAgreementNo = externalAgreementNo
		err = l.orderModel.UpdateNotify(order)
		if err != nil {
			logx.Errorf("更新订单详情失败: %v", err.Error())
			notifyAlipaySignErrNum.CounterInc()
			return
		}

		go func() {
			defer exception.Recover()
			dataMap := l.transFormDataToMap(bodyData)
			dataMap["out_trade_no"] = order.OutTradeNo
			dataMap["notify_type"] = code.APP_NOTIFY_TYPE_SIGN
			utils.CallbackWithRetry(order.AppNotifyUrl, dataMap, 5*time.Second)
		}()

	} else if ALI_NOTIFY_TYPE_UNSIGN == notifyType {
		externalAgreement := r.Form.Get("external_agreement_no")
		order, dbErr := l.orderModel.GetOneByExternalAgreementNo(externalAgreement)
		if dbErr != nil {
			logx.Errorf("根据external_agreement_no获取订单失败: %v", dbErr.Error())
			notifyAlipayUnSignErrNum.CounterInc()
			return
		}
		go func() {
			l.orderModel.CloseUnpaidSubscribeFeeOrderByExternalAgreementNo(externalAgreement)
		}()
		go func() {
			defer exception.Recover()
			dataMap := l.transFormDataToMap(bodyData)
			dataMap["notify_type"] = code.APP_NOTIFY_TYPE_UNSIGN
			utils.CallbackWithRetry(order.AppNotifyUrl, dataMap, 5*time.Second)
		}()

	}

	bytes := []byte("success")
	w.Write(bytes)

	return
}

//formdata数据转成map
func (l *NotifyAlipayNewLogic) transFormDataToMap(formData string) (dataMap map[string]interface{}) {
	dataMap = make(map[string]interface{}, 0)
	values, _ := url.ParseQuery(formData)
	for key, datas := range values {
		if len(datas) > 0 {
			dataMap[key] = datas[0]
		}
	}
	return
}
