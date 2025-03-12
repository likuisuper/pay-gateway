package notify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	jsoniter "github.com/json-iterator/go"
	"github.com/zeromicro/go-zero/rest/httpx"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// NotifyKspayLogic 快手支付已废弃，暂不使用
type NotifyKspayLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	payOrderModel    *model.PmPayOrderModel
	payConfigKsModel *model.PmPayConfigKsModel
}

var (
	notifyKspayErrNum       = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "notifyKspayErrNum", nil, "快手支付回调失败", nil})}
	notifySettleKsPayErrNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "notifySettleKspayErrNum", nil, "快手结算回调失败", nil})}
	notifyKsErrNum          = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "notifyKsPayErrNum", nil, "快手回调失败", nil})}
)

// 快手订单支付回调 https://open.kuaishou.com/docs/develop/server/epay/open-api-new/prePay-new.html
type ksOrderNotifyData struct {
	Data struct {
		Channel         string `json:"channel"`          //支付渠道。取值：UNKNOWN - 未知｜WECHAT-微信｜ALIPAY-支付宝
		OutOrderNo      string `json:"out_order_no"`     //商户系统内部订单号
		Attach          string `json:"attach"`           //预下单时携带的开发者自定义信息
		Status          string `json:"status"`           //订单支付状态。 取值： PROCESSING-处理中｜SUCCESS-成功｜FAILED-失败
		KsOrderNo       string `json:"ks_order_no"`      //快手小程序平台订单号
		OrderAmount     int    `json:"order_amount"`     //订单金额
		TradeNo         string `json:"trade_no"`         //用户侧支付页交易单号
		ExtraInfo       string `json:"extra_info"`       //订单来源信息，同支付查询接口
		EnablePromotion bool   `json:"enable_promotion"` //是否参与分销，true:分销，false:非分销
		PromotionAmount int    `json:"promotion_amount"` //预计分销金额，单位：分
	} `json:"data"`
	BizType   string `json:"biz_type"`
	MessageId string `json:"message_id"`
	AppId     string `json:"app_id"`
	Timestamp int64  `json:"timestamp"`
}

// 快手订单结算回调 https://open.kuaishou.com/docs/develop/server/epay/open-api-new/settle-new.html
type ksSettleNotifyData struct {
	Data struct {
		KsOrderNo                string `json:"ks_order_no"`                //快手小程序平台订单号
		DeveloperPromotionAmount int64  `json:"developer_promotion_amount"` //服务商分销金额，单位：分（仅通过服务商接入的小程序有）
		AppleChannelFee          int64  `json:"apple_channel_fee"`          //苹果通道费，单位：分（仅苹果支付有此费
		OutSettleNo              string `json:"out_settle_no"`              //外部结算单号，即开发者结算请求的单号。
		Attach                   string `json:"attach"`                     //预下单时携带的开发者自定义信息
		KsSettleNo               string `json:"ks_settle_no"`               //快手小程序平台结算单号。
		PromotionAmount          int    `json:"promotion_amount"`           //预计分销金额，单位：分
		SettleAmount             int64  `json:"settle_amount"`              //结算后给商户的金额，单位：分
		Status                   string `json:"status"`                     //结算状态。 取值：SUCCESS-成功，FAILED-失败
		EnablePromotion          bool   `json:"enable_promotion"`           //是否参与分销，true:分销，false:非分销
	} `json:"data"`
	BizType   string `json:"biz_type"`
	MessageId string `json:"message_id"`
	AppId     string `json:"app_id"`
	Timestamp int64  `json:"timestamp"`
}

// 快手回调
type ksNotifyData struct {
	Data      interface{} `json:"data"`
	BizType   string      `json:"biz_type"`
	MessageId string      `json:"message_id"`
	AppId     string      `json:"app_id"`
	Timestamp int64       `json:"timestamp"`
}

// 回调接口返回
type ksOrderNotifyResp struct {
	Result    int    `json:"result"`
	MessageId string `json:"message_id"`
}

func NewNotifyKspayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyKspayLogic {
	return &NotifyKspayLogic{
		Logger:           logx.WithContext(ctx),
		ctx:              ctx,
		svcCtx:           svcCtx,
		payOrderModel:    model.NewPmPayOrderModel(define.DbPayGateway),
		payConfigKsModel: model.NewPmPayConfigKsModel(define.DbPayGateway),
	}
}

func (l *NotifyKspayLogic) NotifyKspay(r *http.Request, w http.ResponseWriter) (resp *types.EmptyReq, err error) {
	reader := io.LimitReader(r.Body, 8<<20)
	bodyBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		logx.Errorf("NotifyKspay err: %v", err)
		notifyKspayErrNum.CounterInc()
		return
	}

	bodyData := string(bodyBytes)
	logx.Slowf("NotifyKspay form %s", bodyData)
	logx.Slowf("NotifyKspay header %+v", r.Header)

	data := new(ksNotifyData)
	err = jsoniter.UnmarshalFromString(bodyData, data)
	if err != nil {
		logx.Errorf("NotifyKspay err: %v", err)
		notifyKspayErrNum.CounterInc()
		return
	}

	if err = l.validSign(bodyData, r.Header.Get("kwaisign"), data.AppId); err != nil {
		logx.Errorf("NotifyKspay 快手回调失败 err: %v", err)
		notifyKsErrNum.CounterInc()
		return
	}
	switch data.BizType {
	case client.Settle: // 结算回调
		return l.SettleNotify(bodyData, w)
	case client.Payment: // 订单支付回调
		return l.OrderNotify(bodyData, w)
	}

	return
}

// OrderNotify 快手支付回调
func (l *NotifyKspayLogic) OrderNotify(bodyData string, w http.ResponseWriter) (resp *types.EmptyReq, err error) {
	notifyData := new(ksOrderNotifyData)
	err = jsoniter.UnmarshalFromString(bodyData, notifyData)
	if err != nil {
		logx.Errorf("NotifyKspay 快手支付回调失败 err: %v", err)
		notifyKspayErrNum.CounterInc()
		return
	}

	//升级为根据订单号和Appid查询
	orderInfo, err := l.payOrderModel.GetOneByOrderSnAndAppId(notifyData.Data.OutOrderNo, notifyData.AppId)
	if err != nil || orderInfo == nil || orderInfo.ID < 1 {
		err = fmt.Errorf("获取订单失败 err:%v, order_code:%s, appId:%s", err, notifyData.Data.OutOrderNo, notifyData.AppId)
		util.CheckError(err.Error())
		return
	}

	if orderInfo.PayStatus != model.PmPayOrderTablePayStatusNo {
		notifyOrderHasDispose.CounterInc()
		err = fmt.Errorf("订单已处理")
		return
	}

	//修改数据库
	orderInfo.NotifyAmount = notifyData.Data.OrderAmount
	orderInfo.PayStatus = model.PmPayOrderTablePayStatusPaid
	orderInfo.ThirdOrderNo = notifyData.Data.KsOrderNo
	//orderInfo.PayType = model.PmPayOrderTablePayTypeKs //改为创建订单时指定支付类型，用于补偿机制建设
	err = l.payOrderModel.UpdateNotify(orderInfo)
	if err != nil {
		err = fmt.Errorf("orderSn=%s, UpdateNotify err:=%v", orderInfo.OrderSn, err)
		util.CheckError(err.Error())
		return
	}

	//回调业务方接口
	go func() {
		defer exception.Recover()
		headerMap := make(map[string]string, 2)
		headerMap["App-Origin"] = orderInfo.AppPkgName
		headerMap["From-App"] = orderInfo.AppPkgName
		result, err := util.HttpPostWithHeader(orderInfo.NotifyUrl, notifyData, headerMap, 5*time.Second)
		l.Sloww("ks notify callback", logx.Field("NotifyUrl", orderInfo.NotifyUrl), logx.Field("result", result), logx.Field("err", err), logx.Field("notifyData", notifyData), logx.Field("AppPkgName", orderInfo.AppPkgName))
	}()

	resData := &ksOrderNotifyResp{
		Result:    1,
		MessageId: notifyData.MessageId,
	}
	httpx.OkJson(w, resData)
	return
}

func (l *NotifyKspayLogic) validSign(bodyData, kwaiSign, appId string) error {
	// 验签
	config, err := l.payConfigKsModel.GetOneByAppID(appId)
	if err != nil {
		logx.Errorf("NotifyKspay 快手回调验签失败 err: %v", err)
		notifyKsErrNum.CounterInc()
		return err
	}

	cliConfig := config.TransClientConfig()
	ksPayCli := client.NewKsPay(*cliConfig)
	calSign := ksPayCli.NotifySign(bodyData)
	if kwaiSign != calSign {
		logx.Errorf("NotifyKspay 快手回调验签失败 signErr: %v,%v", kwaiSign, calSign)
		notifyKsErrNum.CounterInc()
		return errors.New("NotifyKspay signErr")
	}
	return nil
}

func (l *NotifyKspayLogic) SettleNotify(bodyData string, w http.ResponseWriter) (resp *types.EmptyReq, err error) {
	notifyData := new(ksSettleNotifyData)
	err = jsoniter.UnmarshalFromString(bodyData, notifyData)
	if err != nil {
		logx.Errorf("结算订单 SettleNotify  NotifyKspay err: %v", err)
		notifySettleKsPayErrNum.CounterInc()
		return
	}

	//升级为根据订单号和Appid查询
	orderInfo, err := l.payOrderModel.GetOneByOrderSnAndAppId(notifyData.Data.OutSettleNo, notifyData.AppId)
	if err != nil || orderInfo == nil || orderInfo.ID < 1 {
		err = fmt.Errorf("结算订单处理 SettleNotify 获取订单失败 err:%v, order_code:%s, appId:%s", err, notifyData.Data.OutSettleNo, notifyData.AppId)
		util.CheckError(err.Error())
		return
	}

	if orderInfo.PayStatus != model.PmPayOrderTablePayStatusPaid {
		notifyOrderHasDispose.CounterInc()
		err = fmt.Errorf("订单未支付成功")
		return
	}

	resData := &ksOrderNotifyResp{
		Result:    1,
		MessageId: notifyData.MessageId,
	}
	httpx.OkJson(w, resData)
	return
}
