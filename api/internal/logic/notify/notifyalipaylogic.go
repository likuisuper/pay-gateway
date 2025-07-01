package notify

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gitlab.muchcloud.com/consumer-project/alipay"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/types"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/client"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/exception"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	kv_m "gitlab.muchcloud.com/consumer-project/zhuyun-core/kv_monitor"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/util"
)

//短剧表-暂未使用

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

// NotifyAlipay 支付宝回调，暂未使用
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
	logx.Slowf("appId:%v", appId)
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

	if !res.IsSuccess() {
		logx.Errorf("NotifyAlipay success false %s", outTradeNo)
		notifyAlipayErrNum.CounterInc()
		return
	}

	//升级为根据订单号和appid查询
	orderInfo, err := l.payOrderModel.GetOneByOrderSnAndAppId(outTradeNo, appId)
	if err != nil || orderInfo == nil || orderInfo.ID < 1 {
		err = fmt.Errorf("获取订单失败 err:%v, order_code:%s, appId:%v", err, outTradeNo, appId)
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
	//orderInfo.PayType = model.PmPayOrderTablePayTypeAlipay //改为创建订单时指定支付类型，用于补偿机制建设
	err = l.payOrderModel.UpdateNotify(orderInfo)
	if err != nil {
		err = fmt.Errorf("orderSn = %s, UpdateNotify err:=%v", orderInfo.OrderSn, err)
		util.CheckError(err.Error())
		return
	}

	//回调业务方接口
	go func() {
		defer exception.Recover()
		dataMap := l.transFormDataToMap(bodyData)
		headerMap := make(map[string]string, 1)
		headerMap["App-Origin"] = orderInfo.AppPkgName
		_, _ = util.HttpPostWithHeader(orderInfo.NotifyUrl, dataMap, headerMap, 5*time.Second)
	}()

	bytes := []byte("success")
	w.Write(bytes)

	return
}

// formdata数据转成map
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
