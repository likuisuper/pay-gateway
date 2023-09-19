package notify

import (
	"context"
	"fmt"
	"gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/clientMgr"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"net/http"
	"net/url"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyAlipayNewLogic struct {
	logx.Logger
	ctx        context.Context
	svcCtx     *svc.ServiceContext
	orderModel *model.OrderModel
}

func NewNotifyAlipayNewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyAlipayNewLogic {
	return &NotifyAlipayNewLogic{
		Logger:     logx.WithContext(ctx),
		ctx:        ctx,
		svcCtx:     svcCtx,
		orderModel: model.NewOrderModel(define.DbPayGateway),
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
	if ALI_NOTIFY_TYPE_TRADE_SYNC == notifyType {

		var outTradeNo = r.Form.Get("out_trade_no")
		var tradeNo = r.Form.Get("trade_no")
		var tradeQuery = alipay.TradeQuery{
			OutTradeNo: outTradeNo,
		}
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
			_, _ = util.HttpPost(orderInfo.AppNotifyUrl, dataMap, 5*time.Second)
		}()

	} else if ALI_NOTIFY_TYPE_SIGN == notifyType {

		agreementNo := r.Form.Get("agreement_no")
		externalAgreementNo := r.Form.Get("external_agreement_no")
		outTradeNo := r.Form.Get("out_trade_no")

		if agreementNo == "" || externalAgreementNo == "" || outTradeNo == "" {
			logx.Errorf("签约回调参数异常, %s", bodyData)
			return
		}

		if err != nil {
			logx.Errorf(err.Error())
			notifyAlipaySignErrNum.CounterInc()
			return
		}

		order, dbErr := l.orderModel.GetOneByOutTradeNo(outTradeNo)
		if dbErr != nil {
			logx.Errorf("获取订单详情失败: %v", dbErr.Error())
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
			dataMap["notify_type"] = code.APP_NOTIFY_TYPE_SIGN
			_, _ = util.HttpPost(order.AppNotifyUrl, dataMap, 5*time.Second)
		}()

	} else if ALI_NOTIFY_TYPE_UNSIGN == notifyType {
		externalAgreement := r.Form.Get("external_agreement_no")
		order, dbErr := l.orderModel.GetOneByOutTradeNo(externalAgreement)
		if dbErr != nil {
			logx.Errorf("根据external_agreement_no获取订单失败: %v", dbErr.Error())
			notifyAlipayUnSignErrNum.CounterInc()
			return
		}
		go func() {
			defer exception.Recover()
			dataMap := l.transFormDataToMap(bodyData)
			dataMap["notify_type"] = code.APP_NOTIFY_TYPE_UNSIGN
			_, _ = util.HttpPost(order.AppNotifyUrl, dataMap, 5*time.Second)
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
