package notify

import (
	"context"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	jsoniter "github.com/json-iterator/go"
	"net/http"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyKspayLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

var (
	notifyKspayErrNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "notifyKspayErrNum", nil, "快手支付回调失败", nil})}
)

//快手回调
type ksOrderNotifyData struct {
	Data struct {
		Channel         string `json:"channel"`
		OutOrderNo      string `json:"out_order_no"`
		Attach          string `json:"attach"`
		Status          string `json:"status"`
		KsOrderNo       string `json:"ks_order_no"`
		OrderAmount     int    `json:"order_amount"`
		TradeNo         string `json:"trade_no"`
		ExtraInfo       string `json:"extra_info"`
		EnablePromotion bool   `json:"enable_promotion"`
		PromotionAmount int    `json:"promotion_amount"`
	} `json:"data"`
	BizType   string `json:"biz_type"`
	MessageId string `json:"message_id"`
	AppId     string `json:"app_id"`
	Timestamp int64  `json:"timestamp"`
}

func NewNotifyKspayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyKspayLogic {
	return &NotifyKspayLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NotifyKspayLogic) NotifyKspay(r *http.Request, w http.ResponseWriter) (resp *types.EmptyReq, err error) {
	err = r.ParseForm()
	if err != nil {
		logx.Errorf("NotifyKspay err: %v", err)
		notifyKspayErrNum.CounterInc()
		return
	}
	bodyData := r.Form.Encode()
	logx.Slowf("NotifyKspay form %s", bodyData)

	notifyData := new(ksOrderNotifyData)
	err = jsoniter.UnmarshalFromString(bodyData, notifyData)
	if err != nil {
		logx.Errorf("NotifyKspay err: %v", err)
		notifyKspayErrNum.CounterInc()
		return
	}

	return
}
