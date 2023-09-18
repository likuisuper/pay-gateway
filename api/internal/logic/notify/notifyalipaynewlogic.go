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
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/zeromicro/go-zero/rest/httpx"
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

	var outTradeNo = r.Form.Get("out_trade_no")
	var tradeQuery = alipay.TradeQuery{
		OutTradeNo: outTradeNo,
	}
	res, err := client.TradeQuery(tradeQuery)
	if err != nil {
		err = fmt.Errorf("TradeQuery err=%v", err)
		logx.Error(err)
		notifyAlipayErrNum.CounterInc()
	}
	if res.IsSuccess() == false {
		logx.Errorf("NotifyAlipay success false %s", outTradeNo)
		notifyAlipayErrNum.CounterInc()
		return
	}

	//获取订单信息
	orderInfo, err := l.orderModel.GetOneByOutTradeNo(outTradeNo)
	if err != nil {
		err = fmt.Errorf("获取订单失败！err=%v,order_code = %s", err, outTradeNo)
		util.CheckError(err.Error())
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
		dataMap["notify_type"] = code.NOTIFY_TYPE_PAY
		_, _ = util.HttpPost(orderInfo.AppNotifyUrl, dataMap, 5*time.Second)
	}()

	httpx.OkJson(w, "success")

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
