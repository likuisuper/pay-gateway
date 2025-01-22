package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

var CallbackBizFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "callbackBizFailNum", nil, "网关回调业务异常", nil})}
var CallbackRefundFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "CallbackRefundFailNum", nil, "网关回调退款业务异常", nil})}

type NotifyDouyinLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	payOrderModel         *model.PmPayOrderModel
	payConfigTiktokModel  *model.PmPayConfigTiktokModel
	refundOrderModel      *model.PmRefundOrderModel
	payDyPeriodOrderModel *model.PmDyPeriodOrderModel

	Rdb *cache.RedisInstance
}

func NewNotifyDouyinLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyDouyinLogic {
	return &NotifyDouyinLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,

		payOrderModel:         model.NewPmPayOrderModel(define.DbPayGateway),
		payConfigTiktokModel:  model.NewPmPayConfigTiktokModel(define.DbPayGateway),
		refundOrderModel:      model.NewPmRefundOrderModel(define.DbPayGateway),
		payDyPeriodOrderModel: model.NewPmDyPeriodOrderModel(define.DbPayGateway),
		Rdb:                   db.WithRedisDBContext(define.DbPayGateway),
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

	defer req.Body.Close()

	data := new(douyin.GeneralTradeCallbackData)
	err = sonic.Unmarshal(body, data)
	if err != nil {
		l.Errorf("NotifyDouyin, unmarshal body fail, err:%s, body: %s", err.Error(), string(body))
		return &types.DouyinResp{
			ErrNo:   500,
			ErrTips: "unmarshal body fail",
		}, nil
	}

	l.Slowf("NotifyDouyin raw body: %s", string(body))

	switch data.Type {
	case douyin.EventPayment:
		return l.notifyPayment(req, body, data.Msg, data)
	case douyin.EventRefund:
		return l.notifyRefund(req, body, data.Msg, data)
	case douyin.EventSettle:
		// 该类型线上未接入，后续需要再实现对应逻辑
		return &types.DouyinResp{
			ErrNo:   0,
			ErrTips: "success",
		}, nil
	case douyin.EventPreCreateRefund:
		// 退款申请回调
		return l.notifyPreCreateRefund(req, body, data.Msg, data)
	case douyin.EventSignCallback:
		// 抖音周期代扣签约回调
		l.handleSignCallback(data.Msg)
		resp := &types.DouyinResp{
			ErrNo:   0,
			ErrTips: "success",
			Data:    nil,
		}
		return resp, nil
	}

	l.Errorf("NotifyDouyin invalid msg type:%s, data:%v, raw body", data.Type, data, string(body))
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
	if err != nil || orderInfo == nil || orderInfo.ID < 1 {
		err = fmt.Errorf("获取订单失败 err=%v, order_code:%s, appId:%s", err, msg.OutOrderNo, msg.AppId)
		util.CheckError(err.Error())
		return nil, err
	}

	if orderInfo.PayStatus != model.PmPayOrderTablePayStatusNo {
		notifyOrderHasDispose.CounterInc()
		err = fmt.Errorf("订单已处理")
		return nil, err
	}

	if msg.Status == "SUCCESS" {
		orderInfo.PayStatus = model.PmPayOrderTablePayStatusPaid
		//修改数据库
		orderInfo.NotifyAmount = int(msg.TotalAmount)
	} else if msg.Status == "CANCEL" {
		orderInfo.PayStatus = model.PmPayOrderTablePayStatusCancel
	} else {
		l.Slowf("douyin支付回调异常: %s", msgJson)
		return nil, nil
	}
	orderInfo.ThirdOrderNo = msg.OrderId
	err = l.payOrderModel.UpdateNotify(orderInfo)
	if err != nil {
		err = fmt.Errorf("orderSn = %s, UpdateNotify，err:=%v", orderInfo.OrderSn, err)
		util.CheckError(err.Error())
		return nil, err
	}

	//回调业务方接口
	go func() {
		defer exception.Recover()
		headMap := map[string]string{
			"App-Origin": orderInfo.AppPkgName,
		}
		respData, requestErr := util.HttpPostWithHeader(orderInfo.NotifyUrl, originData, headMap, 5*time.Second)
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

// 抖音退款回调
func (l *NotifyDouyinLogic) notifyRefund(req *http.Request, body []byte, msgJson string, originData interface{}) (*types.DouyinResp, error) {
	msg := new(douyin.RefundMsg)
	err := sonic.UnmarshalString(msgJson, msg)
	if err != nil {
		err = fmt.Errorf("notifyRefund unmarshalString fial, msgJson:%v, err:%v", msgJson, err)
		util.CheckError(err.Error())
		CallbackRefundFailNum.CounterInc()
		return nil, err
	}

	l.Slowf("notifyRefund, reqHeader:%v, msgJson:%s", req.Header, msgJson)

	payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(msg.AppId)
	if cfgErr != nil {
		err = fmt.Errorf("notifyRefund appid = %s, 读取抖音支付配置失败，err:=%v", msg.AppId, cfgErr)
		util.CheckError(err.Error())
		CallbackRefundFailNum.CounterInc()
		return nil, cfgErr
	}

	// 验签
	client := douyin.NewDouyinPay(payCfg.GetGeneralTradeConfig())
	err = client.VerifyNotify(req, body)
	if err != nil {
		l.Errorf("notifyRefund 验签未通过，或者解密失败！err=%v", err)
		CallbackRefundFailNum.CounterInc()
		return &types.DouyinResp{
			ErrNo:   400,
			ErrTips: "验签未通过，或者解密失败",
		}, nil
	}

	// redis 并发控制
	concurrentKey, value := fmt.Sprintf("payGateway:refundNotify:douyin:%s", msg.OutRefundNo), uuid.New().String()
	isLock, err := l.Rdb.TryLockWithTimeout(context.Background(), concurrentKey, value, 1000)
	if err != nil || !isLock {
		l.Slowf("notifyRefund redis lock fail, err:%s, isLock:%v, key:%v", err.Error(), isLock, concurrentKey)
		return nil, fmt.Errorf("redis lock fail, err:%s, isLock:%v, key:%v", err.Error(), isLock, concurrentKey)
	}
	defer func() {
		unlockErr := l.Rdb.Unlock(context.Background(), concurrentKey, value)
		if unlockErr != nil {
			l.Slowf("redis unlock fail, key:%s, value:%s", concurrentKey, value)
		}
	}()
	//修改数据库
	refundInfo, err := l.refundOrderModel.GetInfoByRefundNo(msg.RefundId)
	if err != nil {
		CallbackRefundFailNum.CounterInc()
		l.Errorf("notifyRefund 获取退款订单失败！err=%v,order_code = %s", err, msg.RefundId)
		return &types.DouyinResp{
			ErrNo:   400,
			ErrTips: "获取退款订单失败",
		}, nil
	}
	//根据支付网关的退款单号查询 创建退款订单超时未拿到抖音侧退款单号 还需更新退款单号信息
	if refundInfo.ID == 0 {
		refundInfo, err = l.refundOrderModel.GetInfo(msg.OutRefundNo)
		if err != nil || refundInfo.ID == 0 {
			CallbackRefundFailNum.CounterInc()
			l.Errorf("notifyRefund 获取退款订单失败！err=%v,order_code = %s", err, msg.RefundId)
			return &types.DouyinResp{
				ErrNo:   400,
				ErrTips: "获取退款订单失败",
			}, nil
		}
	}
	//判断改退款订单是否已被处理过
	if refundInfo.RefundStatus != model.PmRefundOrderTableRefundStatusApply {
		l.Slowf("notifyRefund 退款订单已被处理过！order_code = %s", msg.RefundId)
		resp := &types.DouyinResp{
			ErrNo:   0,
			ErrTips: "success",
		}
		return resp, nil
	}

	//查询订单的包名信息
	orderInfo, err := l.payOrderModel.GetOneByThirdOrderNoAndAppId(msg.OrderId, msg.AppId)
	if err != nil || orderInfo.ID == 0 {
		CallbackRefundFailNum.CounterInc()
		l.Errorf("notifyRefund 获取订单失败！err=%v,order_code = %s", err, msg.RefundId)
		return &types.DouyinResp{
			ErrNo:   400,
			ErrTips: "获取订单失败",
		}, nil
	}

	refundInfo.NotifyData = msgJson
	refundInfo.RefundedAt = msg.EventTime
	if refundInfo.RefundNo == "" {
		//创建退款订单超时未拿到抖音侧退款单号 还需更新退款单号信息
		refundInfo.RefundNo = msg.RefundId
	}
	if msg.Status == "SUCCESS" {
		refundInfo.RefundStatus = model.PmRefundOrderTableRefundStatusSuccess
	} else {
		refundInfo.RefundStatus = model.PmRefundOrderTableRefundStatusFail
	}
	err = l.refundOrderModel.Update(msg.OutRefundNo, refundInfo)
	if err != nil {
		CallbackRefundFailNum.CounterInc()
		l.Errorf("notifyRefund 更新退款订单失败！err=%v,order_code = %s", err, msg.RefundId)
		return &types.DouyinResp{
			ErrNo:   400,
			ErrTips: "更新退款订单失败",
		}, nil
	}
	//回调业务方接口
	go func() {
		defer exception.Recover()
		headMap := map[string]string{
			"App-Origin": orderInfo.AppPkgName,
		}
		respData, requestErr := util.HttpPostWithHeader(refundInfo.NotifyUrl, originData, headMap, 5*time.Second)
		if requestErr != nil {
			CallbackRefundFailNum.CounterInc()
			CallbackBizFailNum.CounterInc()
			util.CheckError("notifyRefund NotifyRefund-post, req:%+v, err:%v", originData, requestErr)
			l.Errorf("notifyRefund NotifyRefund-post, req:%+v, err:%v, url:%v", originData, requestErr, refundInfo.NotifyUrl)
			return
		}
		l.Slowf("notifyRefund NotifyRefund-post, req:%+v, respData:%s", originData, respData)
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
		Data:    nil,
	}
	return resp, nil
}

// 抖音签约回调结构体
type DySignCallbackNotify struct {
	AppId          string `json:"app_id"`            // 小程序 app_id
	Status         string `json:"status"`            // 签约结果状态，目前有四种状态： "SUCCESS" （用户签约成功 ） •"TIME_OUT" （用户未签约，订单超时关单） •"CANCEL" (解约成功)	•"DONE" （服务完成，已到期）
	AuthOrderId    string `json:"auth_order_id"`     // 平台侧签约单的单号，长度<=64byte
	OutAuthOrderNo string `json:"out_auth_order_no"` // 开发者侧签约单的单号，长度<=64byte
	EventTime      int64  `json:"event_time"`        // 用户签约成功/签约取消/解约成功的时间戳，单位为毫秒
}

const (
	Dy_Sign_Status_SUCCESS  = "SUCCESS"  // 用户签约成功
	Dy_Sign_Status_TIME_OUT = "TIME_OUT" // 用户未签约，订单超时关单
	Dy_Sign_Status_CANCEL   = "CANCEL"   // 解约成功
	Dy_Sign_Status_DONE     = "DONE"     // 服务完成，已到期(按照解约处理 ?? 这个状态需要观察数据)
)

// 抖音周期代扣签约回调处理
// https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/server/payment/management-capacity/periodic-deduction/sign/sign-callback
func (l *NotifyDouyinLogic) handleSignCallback(msg string) {
	// msg 字段内容示例
	//签约成功回调示例
	// {
	// 	"app_id": "ttcfdbb96650e33350",
	// 	"status": "SUCCESS",
	// 	"auth_order_id": "ad72432423423",
	// 	"out_auth_order_no": "out_order_no_1",
	// 	"event_time": 1698128528000
	// }

	//超时取消回调示例
	// {
	// 	"app_id": "ttcfdbb96650e33350",
	// 	"status": "TIME_OUT",
	// 	"auth_order_id": "ad72432423423",
	// 	"out_auth_order_no": "out_order_no_1",
	// 	"event_time": 1698128528000
	// }

	//用户解约回调示例
	// {
	// 	"app_id": "ttcfdbb96650e33350",
	// 	"status": "CANCEL",
	// 	"auth_order_id": "ad72432423423",
	// 	"out_auth_order_no": "out_order_no_1",
	// 	"cancel_source": 1,
	// 	"event_time": 1698128528000
	// }

	//服务完成回调示例
	// {
	// 	"app_id": "ttcfdbb96650e33350",
	// 	"status": "DONE",
	// 	"auth_order_id": "ad72432423423",
	// 	"out_auth_order_no": "out_order_no_1",
	// 	"event_time": 1698128528000
	// }

	var signResult DySignCallbackNotify
	err := json.Unmarshal([]byte(msg), &signResult)
	if err != nil {
		l.Errorf("json.Unmarshal error: %v", err)
		return
	}

	if signResult.Status == Dy_Sign_Status_DONE {
		// 记录一下日志 这个怎么处理 暂时还不知道
		l.Slowf("服务完成已到期 orderNo: %s , appId: %s , eventTime: %d ", signResult.OutAuthOrderNo, signResult.AppId, signResult.EventTime)
	} else if signResult.Status == Dy_Sign_Status_TIME_OUT {
		// 记录一下日志
		l.Slowf("签约单超时 orderNo: %s , appId: %s , eventTime: %d ", signResult.OutAuthOrderNo, signResult.AppId, signResult.EventTime)
	} else if signResult.Status == Dy_Sign_Status_SUCCESS {
		// 签约成功 查询记录是否存在
		tbl, err := l.payDyPeriodOrderModel.GetOneByOrderSnAndAppId(signResult.OutAuthOrderNo, signResult.AppId)
		if err != nil || tbl == nil || tbl.ID < 1 {
			l.Errorf("签约成功 查询记录出错 err: %s , orderNo: %s , appId: %s , eventTime: %d ", err.Error(), signResult.OutAuthOrderNo, signResult.AppId, signResult.EventTime)
			return
		}

		// 修改状态为签约成功
		updateData := map[string]interface{}{
			"sign_status":         1,
			"sign_date":           time.Unix(signResult.EventTime/1000, 0).Format("2006-01-02 15:04:05"),
			"third_sign_order_no": signResult.AuthOrderId, // 抖音的签约单号
		}
		l.payDyPeriodOrderModel.UpdateSomeData(tbl.ID, updateData)
	} else if signResult.Status == Dy_Sign_Status_CANCEL {
		// 解约成功 查询记录是否存在
		tbl, err := l.payDyPeriodOrderModel.GetOneByOrderSnAndAppId(signResult.OutAuthOrderNo, signResult.AppId)
		if err != nil || tbl == nil || tbl.ID < 1 {
			l.Errorf("解约成功 查询记录出错 err: %s , orderNo: %s , appId: %s , eventTime: %d ", err.Error(), signResult.OutAuthOrderNo, signResult.AppId, signResult.EventTime)
			return
		}

		// 修改状态为解约成功
		updateData := map[string]interface{}{
			"sign_status":           2,
			"unsign_date":           time.Unix(signResult.EventTime/1000, 0).Format("2006-01-02 15:04:05"),
			"third_unsign_order_no": signResult.AuthOrderId, // 抖音的签约单号
		}
		l.payDyPeriodOrderModel.UpdateSomeData(tbl.ID, updateData)
	}
}
