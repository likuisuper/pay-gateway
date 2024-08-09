package notify

import (
	"context"
	"fmt"
	douyin "gitee.com/zhuyunkj/pay-gateway/common/client/douyinGeneralTrade"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/db"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/cache"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"io"
	"net/http"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

var CallbackBizFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "callbackBizFailNum", nil, "网关回调业务异常", nil})}

type NotifyDouyinLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	payOrderModel        *model.PmPayOrderModel
	payConfigTiktokModel *model.PmPayConfigTiktokModel
	refundOrderModel     *model.PmRefundOrderModel

	Rdb *cache.RedisInstance
}

func NewNotifyDouyinLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyDouyinLogic {
	return &NotifyDouyinLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,

		payOrderModel:        model.NewPmPayOrderModel(define.DbPayGateway),
		payConfigTiktokModel: model.NewPmPayConfigTiktokModel(define.DbPayGateway),
		refundOrderModel:     model.NewPmRefundOrderModel(define.DbPayGateway),
		Rdb:                  db.WithRedisDBContext(define.DbPayGateway),
	}
}

/*
{
    "@timestamp":"2024-02-22T15:26:18.155+08:00",
    "caller":"notify/notifydouyinlogic.go:94",
    "content":"notifyPayment, reqHeader:map[Accept-Charset:[utf-8] Accept-Encoding:[gzip] Byte-Env:[prod] Byte-Eventid:[basic_industry_/msg/basic/payment/notify_tt1683603e89bd1ac801_PaySucDeveloperNotify_motb73383242802221734767173] Byte-Identifyname:[/msg/basic/payment/notify] Byte-Logid:[20240222152617458BC905C044040CE7DB] Byte-Nonce-Str:[m7cgbhrKIYu9V94xgCOEekbG9Vh7y50j] Byte-Signature:[d42bUx5HzUi+swu/4DhDtr/vNN00EFgzKnBIH9AyWZTh21WNEbIrVgKMhJ4rr+fXftcyUc1xG0wRMVVWIqHA7z+DnFlPgipr2klx8aACMPS7uYVzh6C+7Z1v9d4GVQaOj1wzJ6tf9Izg6VHnESewz8X39FYYuxu+xwpl/EwioucD3bHiUhsJSGOBIxb6zvDk+khTaNDg2WwN+VnVV8dc6ynWjTmPMFUnmAAzAvDXE3HgznQj8p6KL+06gZ2fx832z/wYvnvRTN5FAJVTJ73qvBFlxicSZDw/P9CG+gVPRsNQFpDMcnQM4kbwuzCD9hlnQaa2AV/c+sSWb4ebXt1gmw==] Byte-Timestamp:[1708586777] Connection:[close] Content-Length:[432] Content-Type:[application/json] Signature:[37ad4ca1b7e17cfa709b612c68fd553415d46781] User-Agent:[Go-http-client/2.0] X-Forwarded-For:[36.110.131.76]], msgJson:{\"app_id\":\"tt1683603e89bd1ac801\",\"out_order_no\":\"1210246155355688960\",\"order_id\":\"motb73383242802221734767173\",\"status\":\"SUCCESS\",\"total_amount\":1,\"discount_amount\":0,\"pay_channel\":10,\"channel_pay_id\":\"TP2024022215260600681574525640\",\"user_bill_pay_id\":\"DTPS2402221526119010630764156176\",\"merchant_uid\":\"71223625266663938860\",\"event_time\":1708586777000}",
    "level":"slow",
    "span":"964cec205a7f3661",
    "trace":"5897cac703dbb95bb4dbdaaf8cb93c66"
}
*/

func (l *NotifyDouyinLogic) NotifyDouyin(req *http.Request) (resp *types.DouyinResp, err error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		l.Errorf("NotifyDouyin, read from body fail, err:%s", err.Error())
		return &types.DouyinResp{
			ErrNo:   500,
			ErrTips: "read from body fail",
		}, nil
	}
	_ = req.Body.Close()

	data := new(douyin.GeneralTradeCallbackData)
	err = sonic.Unmarshal(body, data)
	if err != nil {
		l.Errorf("NotifyDouyin, unmarshal body fail, err:%s, body:%s", err.Error(), body)
		return &types.DouyinResp{
			ErrNo:   500,
			ErrTips: "unmarshal body fail",
		}, nil
	}

	switch data.Type {
	case douyin.EventPayment:
		return l.notifyPayment(req, body, data.Msg, data)
	case douyin.EventRefund:
		return l.notifyRefund(req, body, data.Msg, data)
	case douyin.EventSettle: //该类型线上未接入，后续需要再实现对应逻辑
		return &types.DouyinResp{
			ErrNo:   0,
			ErrTips: "success",
		}, nil
	case douyin.EventPreCreateRefund:

	}

	l.Errorf("NotifyDouyin, invalid msg type:%s, data:%v", data.Type, data)
	return &types.DouyinResp{
		ErrNo:   500,
		ErrTips: "invalid payment",
	}, nil
}

// notifyPayment 抖音回调
func (l *NotifyDouyinLogic) notifyPayment(req *http.Request, body []byte, msgJson string, originData interface{}) (*types.DouyinResp, error) {
	msg := new(douyin.GeneralTradeMsg)
	err := sonic.UnmarshalString(msgJson, msg)
	if err != nil {
		err = fmt.Errorf("unmarshalString fial, msgJson:%v, err:%v", msgJson, err)
		util.CheckError(err.Error())
		return nil, err
	}

	l.Slowf("notifyPayment, reqHeader:%v, msgJson:%s", req.Header, msgJson)

	payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(msg.AppId)
	if cfgErr != nil {
		err = fmt.Errorf("appid = %s, 读取抖音支付配置失败，err:=%v", msg.AppId, cfgErr)
		util.CheckError(err.Error())
		return nil, cfgErr
	}

	// 验签
	client := douyin.NewDouyinPay(payCfg.GetGeneralTradeConfig())
	err = client.VerifyNotify(req, body)
	if err != nil {
		l.Errorf("验签未通过，或者解密失败！err=%v", err)
		return &types.DouyinResp{
			ErrNo:   400,
			ErrTips: "验签未通过，或者解密失败",
		}, nil
	}

	if msg.Status != "SUCCESS" { // todo: 处理非成功的状态
		l.Slowf("douyin支付回调异常: %s", msgJson)
		return nil, nil
	}

	// redis 并发控制
	concurrentKey, value := fmt.Sprintf("payGateway:paymentNotify:douyin:%s", msg.OutOrderNo), uuid.New().String()
	isLock, err := l.Rdb.TryLockWithTimeout(context.Background(), concurrentKey, value, 1000)
	if err != nil || !isLock {
		l.Slowf("redis lock fail, err:%s, isLock:%v, key:%v", err.Error(), isLock, concurrentKey)
		return nil, fmt.Errorf("redis lock fail, err:%s, isLock:%v, key:%v", err.Error(), isLock, concurrentKey)
	}

	defer func() {
		unlockErr := l.Rdb.Unlock(context.Background(), concurrentKey, value)
		if unlockErr != nil {
			l.Slowf("redis unlock fail, key:%s, value:%s", concurrentKey, value)
		}
	}()

	//获取订单信息 根据订单号和appid查询
	orderInfo, err := l.payOrderModel.GetOneByOrderSnAndAppId(msg.OutOrderNo, msg.AppId)
	if err != nil {
		err = fmt.Errorf("获取订单失败！err=%v,order_code = %s", err, msg.OutOrderNo)
		util.CheckError(err.Error())
		return nil, err
	}
	if orderInfo.PayStatus != model.PmPayOrderTablePayStatusNo {
		notifyOrderHasDispose.CounterInc()
		err = fmt.Errorf("订单已处理")
		return nil, err
	}
	//修改数据库
	orderInfo.NotifyAmount = int(msg.TotalAmount)
	orderInfo.PayStatus = model.PmPayOrderTablePayStatusPaid
	orderInfo.ThirdOrderNo = msg.OrderId
	orderInfo.Currency = msg.Currency

	err = l.payOrderModel.UpdateNotify(orderInfo)
	if err != nil {
		err = fmt.Errorf("orderSn = %s, UpdateNotify，err:=%v", orderInfo.OrderSn, err)
		util.CheckError(err.Error())
		return nil, err
	}

	//回调业务方接口
	go func() {
		defer exception.Recover()
		respData, requestErr := util.HttpPost(orderInfo.NotifyUrl, originData, 5*time.Second)
		if requestErr != nil {
			l.Errorf("NotifyPayment-post, req:%+v, err:%v", originData, requestErr)
			CallbackBizFailNum.CounterInc()
			return
		}
		l.Slowf("NotifyPayment-post, req:%+v, respData:%s", originData, respData)
	}()

	resp := &types.DouyinResp{
		ErrNo:   0,
		ErrTips: "success",
	}
	return resp, nil
}

func (l *NotifyDouyinLogic) notifyRefund(req *http.Request, body []byte, msgJson string, originData interface{}) (*types.DouyinResp, error) {
	msg := new(douyin.RefundMsg)
	err := sonic.UnmarshalString(msgJson, msg)
	if err != nil {
		err = fmt.Errorf("unmarshalString fial, msgJson:%v, err:%v", msgJson, err)
		util.CheckError(err.Error())
		return nil, err
	}

	l.Slowf("notifyRefund, reqHeader:%v, msgJson:%s", req.Header, msgJson)

	payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(msg.AppId)
	if cfgErr != nil {
		err = fmt.Errorf("appid = %s, 读取抖音支付配置失败，err:=%v", msg.AppId, cfgErr)
		util.CheckError(err.Error())
		return nil, cfgErr
	}

	// 验签
	client := douyin.NewDouyinPay(payCfg.GetGeneralTradeConfig())
	err = client.VerifyNotify(req, body)
	if err != nil {
		l.Errorf("验签未通过，或者解密失败！err=%v", err)
		return &types.DouyinResp{
			ErrNo:   400,
			ErrTips: "验签未通过，或者解密失败",
		}, nil
	}

	// redis 并发控制
	concurrentKey, value := fmt.Sprintf("payGateway:refundNotify:douyin:%s", msg.OutRefundNo), uuid.New().String()
	isLock, err := l.Rdb.TryLockWithTimeout(context.Background(), concurrentKey, value, 1000)
	if err != nil || !isLock {
		l.Slowf("redis lock fail, err:%s, isLock:%v, key:%v", err.Error(), isLock, concurrentKey)
		return nil, fmt.Errorf("redis lock fail, err:%s, isLock:%v, key:%v", err.Error(), isLock, concurrentKey)
	}

	//修改数据库
	refundInfo, _ := l.refundOrderModel.GetInfo(msg.OutRefundNo)
	refundInfo.NotifyData = msgJson
	refundInfo.RefundedAt = int(msg.EventTime)
	if msg.Status == "SUCCESS" {
		refundInfo.RefundStatus = model.PmRefundOrderTableRefundStatusSuccess
	} else {
		refundInfo.RefundStatus = model.PmRefundOrderTableRefundStatusFail
	}
	_ = l.refundOrderModel.Update(msg.OutRefundNo, refundInfo)
	//回调业务方接口
	go func() {
		defer exception.Recover()
		respData, requestErr := util.HttpPost(refundInfo.NotifyUrl, originData, 5*time.Second)
		if requestErr != nil {
			CallbackBizFailNum.CounterInc()
			util.CheckError("NotifyRefund-post, req:%+v, err:%v", originData, requestErr)
			return
		}
		logx.Slowf("NotifyRefund-post, req:%+v, respData:%s", originData, respData)
	}()

	resp := &types.DouyinResp{
		ErrNo:   0,
		ErrTips: "success",
	}
	return resp, nil
}

func (l *NotifyDouyinLogic) notifyPreCreateRefund(req *http.Request, body []byte, msgJson string, originData interface{}) (*types.DouyinResp, error) {
	msg := new(douyin.PreCreateRefundMsg)
	err := sonic.UnmarshalString(msgJson, msg)
	if err != nil {
		err = fmt.Errorf("unmarshalString fial, msgJson:%v, err:%v", msgJson, err)
		util.CheckError(err.Error())
		return nil, err
	}

	l.Slowf("notifyPreCreateRefund, reqHeader:%v, msgJson:%s", req.Header, msgJson)

	payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(msg.AppId)
	if cfgErr != nil {
		err = fmt.Errorf("appid = %s, 读取抖音支付配置失败，err:=%v", msg.AppId, cfgErr)
		util.CheckError(err.Error())
		return nil, cfgErr
	}

	// 验签
	client := douyin.NewDouyinPay(payCfg.GetGeneralTradeConfig())
	err = client.VerifyNotify(req, body)
	if err != nil {
		l.Errorf("验签未通过，或者解密失败！err=%v", err)
		return &types.DouyinResp{
			ErrNo:   400,
			ErrTips: "验签未通过，或者解密失败",
		}, nil
	}

	/*
		并发控制
	*/

	/*
		业务逻辑
	*/

	/*
		回调业务方
	*/

	resp := &types.DouyinResp{
		ErrNo:   0,
		ErrTips: "success",
	}
	return resp, nil
}
