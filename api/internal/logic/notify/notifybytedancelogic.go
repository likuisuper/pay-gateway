package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	jsoniter "github.com/json-iterator/go"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyBytedanceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	payOrderModel        *model.PmPayOrderModel
	payConfigTiktokModel *model.PmPayConfigTiktokModel
	refundOrderModel     *model.PmRefundOrderModel
}

func NewNotifyBytedanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyBytedanceLogic {
	return &NotifyBytedanceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,

		payOrderModel:        model.NewPmPayOrderModel(define.DbPayGateway),
		payConfigTiktokModel: model.NewPmPayConfigTiktokModel(define.DbPayGateway),
		refundOrderModel:     model.NewPmRefundOrderModel(define.DbPayGateway),
	}
}

func (l *NotifyBytedanceLogic) NotifyBytedance(req *types.ByteDanceReq) (resp *types.ByteDanceResp, err error) {
	logx.Slowf("NotifyBytedance, req:%+v", req)

	if req.Type == "payment" {
		resp, err = l.NotifyPayment(req)
		return
	}
	if req.Type == "refund" {
		resp, err = l.NotifyRefund(req)
	}

	resp = &types.ByteDanceResp{
		ErrNo:   0,
		ErrTips: "success",
	}
	return
}

//支付成功回调
func (l *NotifyBytedanceLogic) NotifyPayment(req *types.ByteDanceReq) (resp *types.ByteDanceResp, err error) {
	msgData := new(client.TikTokNotifyMsgData)
	err = json.Unmarshal([]byte(req.Msg), msgData)
	if err != nil {
		util.CheckError("json unmarshal err :%v", err)
		return
	}

	payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(msgData.Appid)
	if cfgErr != nil {
		err = fmt.Errorf("appid = %s, 读取抖音支付配置失败，err:=%v", msgData.Appid, cfgErr)
		util.CheckError(err.Error())
		return
	}

	payServer := client.NewTikTokPay(*payCfg.TransClientConfig())
	cliReq := &client.ByteDanceReq{
		Timestamp:    req.Timestamp,
		Nonce:        req.Nonce,
		Msg:          req.Msg,
		Type:         req.Type,
		MsgSignature: req.MsgSignature,
	}
	order, err := payServer.Notify(cliReq)
	if err != nil {
		logx.Errorf("验签未通过，或者解密失败！err=%v", err)
		err = errors.New(`{"code": "FAIL","message": "验签未通过，或者解密失败"}`)
		resp = &types.ByteDanceResp{
			ErrNo:   400,
			ErrTips: "验签未通过，或者解密失败",
		}
		return resp, nil
	}

	if order.Status != "SUCCESS" {
		jsonStr, _ := jsoniter.MarshalToString(cliReq)
		logx.Slowf("bytedance支付回调异常: %s", jsonStr)
		return
	}

	//获取订单信息
	orderInfo, err := l.payOrderModel.GetOneByCode(order.CpOrderno)
	if err != nil {
		err = fmt.Errorf("获取订单失败！err=%v,order_code = %s", err, order.CpOrderno)
		util.CheckError(err.Error())
		return
	}
	if orderInfo.PayStatus != model.PmPayOrderTablePayStatusNo {
		notifyOrderHasDispose.CounterInc()
		err = fmt.Errorf("订单已处理")
		return
	}
	//修改数据库
	orderInfo.NotifyAmount = order.TotalAmount
	orderInfo.PayStatus = model.PmPayOrderTablePayStatusPaid
	orderInfo.PayType = model.PmPayOrderTablePayTypeTiktokPayEc
	err = l.payOrderModel.UpdateNotify(orderInfo)
	if err != nil {
		err = fmt.Errorf("orderSn = %s, UpdateNotify，err:=%v", orderInfo.OrderSn, err)
		util.CheckError(err.Error())
		return
	}

	//回调业务方接口
	go func() {
		defer exception.Recover()
		respData, requestErr := util.HttpPost(orderInfo.NotifyUrl, req, 5*time.Second)
		if requestErr != nil {
			util.CheckError("NotifyPayment-post, req:%+v, err:%v", req, requestErr)
			return
		}
		logx.Slowf("NotifyPayment-post, req:%+v, respData:%s", req, respData)
	}()

	resp = &types.ByteDanceResp{
		ErrNo:   0,
		ErrTips: "success",
	}
	return
}

//退款回调
func (l *NotifyBytedanceLogic) NotifyRefund(req *types.ByteDanceReq) (resp *types.ByteDanceResp, err error) {
	refundData := new(client.TikTokNotifyMsgRefundData)
	err = json.Unmarshal([]byte(req.Msg), refundData)
	if err != nil {
		util.CheckError("json unmarshal err :%v", err)
		return
	}

	payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(refundData.Appid)
	if cfgErr != nil {
		err = fmt.Errorf("appid = %s, 读取抖音支付配置失败，err:=%v", refundData.Appid, cfgErr)
		util.CheckError(err.Error())
		return
	}
	payServer := client.NewTikTokPay(*payCfg.TransClientConfig())
	//签名核对
	timestamp, _ := strconv.Atoi(req.Timestamp)
	notifySing := payServer.NotifySign(timestamp, req.Nonce, req.Msg)
	if notifySing != req.MsgSignature {
		logx.Errorf("回调签名错误, req:%+v", req)
		return nil, errors.New("回调签名错误")
	}
	//修改数据库
	refundInfo, _ := l.refundOrderModel.GetInfo(refundData.CpRefundno)
	refundInfo.NotifyData = req.Msg
	refundInfo.RefundedAt = refundData.RefundedAt
	if refundData.Status == "SUCCESS" {
		refundInfo.RefundStatus = model.PmRefundOrderTableRefundStatusSuccess
	} else {
		refundInfo.RefundStatus = model.PmRefundOrderTableRefundStatusFail
	}
	_ = l.refundOrderModel.Update(refundData.CpRefundno, refundInfo)
	//回调业务方接口
	go func() {
		defer exception.Recover()
		dataMap := map[string]interface{}{
			"dy_notify_data": req,
			"refund_info":    refundInfo,
		}
		respData, requestErr := util.HttpPost(refundInfo.NotifyUrl, dataMap, 5*time.Second)
		if requestErr != nil {
			util.CheckError("NotifyRefund-post, req:%+v, err:%v", dataMap, requestErr)
			return
		}
		logx.Slowf("NotifyRefund-post, req:%+v, respData:%s", dataMap, respData)
	}()

	resp = &types.ByteDanceResp{
		ErrNo:   0,
		ErrTips: "success",
	}
	return
}
