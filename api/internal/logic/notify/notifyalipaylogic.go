package notify

import (
	"context"
	"fmt"
	"gitee.com/yan-yixin0612/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"net/url"
	"time"
)

var (
	notifyAlipayErrNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "notifyAlipayErrNum", nil, "支付宝回调失败", nil})}
)

type NotifyAlipayLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	payOrderModel        *model.PmPayOrderModel
	payConfigAlipayModel *model.PmPayConfigAlipayModel
}

func NewNotifyAlipayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyAlipayLogic {
	return &NotifyAlipayLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,

		payOrderModel:        model.NewPmPayOrderModel(define.DbPayGateway),
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
	}
}

func (l *NotifyAlipayLogic) NotifyAlipay(r *http.Request, w http.ResponseWriter) (resp *types.EmptyReq, err error) {

	err = r.ParseForm()
	if err != nil {
		logx.Errorf("NotifyAlipay err: %v", err)
		notifyAlipayErrNum.CounterInc()
		return
	}
	bodyData := r.Form.Encode()
	logx.Slowf("NotifyAlipay form %s", bodyData)
	appId := r.Form.Get("app_id")
	logx.Slowf(appId)
	payCfg, err := l.payConfigAlipayModel.GetOneByAppID(appId)
	if err != nil {
		err = fmt.Errorf("pkgName= %s, 读取支付配置失败，err:=%v", "all", err)
		util.CheckError(err.Error())
		return
	}
	payClient, err := client.GetAlipayClient(*payCfg.TransClientConfig())
	if err != nil {
		util.CheckError("pkgName= %s, 初使化支付错误，err:=%v", "all", err)
		return
	}
	//ok, err := payClient.VerifySign(r.Form)
	//if err != nil {
	//	logx.Errorf("NotifyAlipay err: %v", err)
	//	notifyAlipayErrNum.CounterInc()
	//	return
	//}
	//if !ok {
	//	err = errors.New("verify sign err")
	//	logx.Error(err)
	//	notifyAlipayErrNum.CounterInc()
	//	return
	//}
	var outTradeNo = r.Form.Get("out_trade_no")
	var tradeQuery = alipay.TradeQuery{
		OutTradeNo: outTradeNo,
	}
	res, err := payClient.TradeQuery(tradeQuery)
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
	orderInfo, err := l.payOrderModel.GetOneByCode(outTradeNo)
	if err != nil {
		err = fmt.Errorf("获取订单失败！err=%v,order_code = %s", err, outTradeNo)
		util.CheckError(err.Error())
		return
	}
	if orderInfo.PayStatus != model.PmPayOrderTablePayStatusNo {
		notifyOrderHasDispose.CounterInc()
		err = fmt.Errorf("订单已处理")
		return
	}
	//修改数据库
	amount := util.String2Float64(res.Content.TotalAmount) * 100
	orderInfo.NotifyAmount = int(amount)
	orderInfo.PayStatus = model.PmPayOrderTablePayStatusPaid
	orderInfo.PayType = model.PmPayOrderTablePayTypeAlipay
	err = l.payOrderModel.UpdateNotify(orderInfo)
	if err != nil {
		err = fmt.Errorf("orderSn = %s, UpdateNotify，err:=%v", orderInfo.OrderSn, err)
		util.CheckError(err.Error())
		return
	}

	//回调业务方接口
	go func() {
		defer exception.Recover()
		dataMap := l.transFormDataToMap(bodyData)
		_, _ = util.HttpPost(orderInfo.NotifyUrl, dataMap, 5*time.Second)
	}()

	httpx.OkJson(w, "success")
	return
}

//formdata数据转成map
func (l *NotifyAlipayLogic) transFormDataToMap(formData string) (dataMap map[string]interface{}) {
	dataMap = make(map[string]interface{}, 0)
	values, _ := url.ParseQuery(formData)
	for key, datas := range values {
		if len(datas) > 0 {
			dataMap[key] = datas[0]
		}
	}
	return
}
