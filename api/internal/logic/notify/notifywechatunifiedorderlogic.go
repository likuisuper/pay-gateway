package notify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/api/common/gocrypto"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyWechatUnifiedOrderLogic struct {
	logx.Logger
	ctx                  context.Context
	svcCtx               *svc.ServiceContext
	orderModel           *model.OrderModel
	refundModel          *model.RefundModel
	payConfigWechatModel *model.PmPayConfigWechatModel
}

func NewNotifyWechatUnifiedOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyWechatUnifiedOrderLogic {
	return &NotifyWechatUnifiedOrderLogic{
		Logger:               logx.WithContext(ctx),
		ctx:                  ctx,
		svcCtx:               svcCtx,
		orderModel:           model.NewOrderModel(define.DbPayGateway),
		refundModel:          model.NewRefundModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
	}
}

//微信支付回调解析
type wechatCallbackRepay struct {
	Appid      string `xml:"appid"`
	Attach     string `xml:"attach"`
	BankType   string `xml:"bank_type"`
	TotalFee   int    `xml:"total_fee"`
	TradeType  string `xml:"trade_type"`
	CashFee    int    `xml:"cash_fee"`
	OutTradeNo string `xml:"out_trade_no"`
	TimeEnd    string `xml:"time_end"`
	Sign       string `xml:"sign"`
	NonceStr   string `xml:"nonce_str"`
	SignType   string `xml:"sign_type"`
	ResultCode string `xml:"result_code"`
	ErrCode    string `xml:"err_code"`
	ErrCodeDes string `xml:"err_code_des"`
	MchId      string `xml:"mch_id"`
	ReqInfo    string `xml:"req_info"`
}

//orderInfo结构
type AttachInfo struct {
	OrderSn  string
	Amount   int
	Subject  string
	KsTypeId int
}

//微信退款回调解密内容
type wechatRefundReply struct {
	TransactionId       string `xml:"transaction_id"`
	OutTradeNo          string `xml:"out_trade_no"`
	RefundId            string `xml:"refund_id"`
	OutRefundNo         string `xml:"out_refund_no"`
	RefundFee           int    `xml:"refund_fee"`
	SettlementRefundFee int    `xml:"settlement_refund_fee"`
	RefundStatus        string `xml:"refund_status"`
	SuccessTime         string `xml:"success_time"`
	TotalFee            int    `xml:"total_fee"`
	CashRefundFee       string `xml:"cash_refund_fee"`
}

func (l *NotifyWechatUnifiedOrderLogic) NotifyWechatUnifiedOrder(r *http.Request) (resp *types.WeChatResp, err error) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logx.Errorf("获取请求体错误！err:=%v", err)
		return nil, err
	}
	logx.Slow("NotifyWechatUnifiedOrder:", string(body))
	var data wechatCallbackRepay
	err = xml.Unmarshal(body, &data)
	if err != nil {
		logx.Errorf("xml.Unmarshal err:=%v", err)
		return nil, err
	}
	//回调支付成功
	if data.ResultCode == "SUCCESS" {
		//退款回调
		if data.ReqInfo != "" {
			//解密
			reqInfo, _ := base64.StdEncoding.DecodeString(data.ReqInfo)
			//获取key
			payCfg, _ := l.payConfigWechatModel.GetOneByAppID(data.Appid)
			gocrypto.SetAesKey(strings.ToLower(util.Md5(payCfg.ApiKey)))
			plaintext, ecbErr := gocrypto.AesECBDecrypt(reqInfo)
			if ecbErr != nil {
				logx.Errorf("gocrypto.AesECBDecrypt err:=%v", ecbErr)
				return nil, err
			}
			strReq := strings.Replace(string(plaintext), "root", "xml", -1)
			var refundReply wechatRefundReply
			xmlErr := xml.Unmarshal([]byte(strReq), &refundReply)
			if xmlErr != nil {
				logx.Errorf("xml.Unmarshal err:=%v", xmlErr)
				return nil, err
			}
			logx.Infof("退款回调详情:%s", string(plaintext))

			orderInfo, _ := l.refundModel.GetOneByOutTradeRefundNo(refundReply.OutRefundNo)
			if orderInfo != nil {
				//修改订单退款状态
				l.orderModel.UpdateStatusByOutTradeNo(refundReply.OutTradeNo, code.ORDER_REFUNDED)
				orderInfo.RefundStatus = code.ORDER_SUCCESS
				orderInfo.RefundNo = refundReply.RefundId
				orderInfo.NotifyData = strReq
				//修改退款订单信息
				l.refundModel.Update(refundReply.OutRefundNo, orderInfo)
				// 回调退款成功
				go func() {
					defer exception.Recover()
					dataMap := make(map[string]interface{})
					dataMap["notify_type"] = code.APP_NOTIFY_TYPE_REFUND
					dataMap["out_trade_refund_no"] = refundReply.RefundId
					dataMap["out_trade_no"] = orderInfo.OutTradeNo
					dataMap["refund_out_side_app"] = false
					dataMap["refund_status"] = model.REFUND_STATUS_SUCCESS
					dataMap["refund_fee"] = refundReply.RefundFee
					_, _ = util.HttpPost(orderInfo.NotifyUrl, dataMap, 5*time.Second)
				}()

			} else {
				logx.Errorf("未获取到退款单信息:RefundId:%s,OutTradeNo:%s", refundReply.RefundId, refundReply.OutTradeNo)
			}
		} else {
			var attachInfo AttachInfo
			unJsonErr := json.Unmarshal([]byte(data.Attach), &attachInfo)
			if unJsonErr != nil {
				logx.Errorf("json.Unmarshal Attach  err:=%v", err)
				return nil, err
			}
			//获取订单信息
			orderInfo, dbErr := l.orderModel.GetOneByOutTradeNo(attachInfo.OrderSn)
			if dbErr != nil {
				dbErr = fmt.Errorf("获取订单失败！err=%v,order_code = %s", dbErr, attachInfo.OrderSn)
				util.CheckError(dbErr.Error())
				return nil, err
			}
			if orderInfo.PayAppID != data.Appid || orderInfo.Amount != data.CashFee {
				logx.Errorf("当前回调的订单信息不匹配", attachInfo.OrderSn)
				return nil, err
			}
			if orderInfo.Status != model.PmPayOrderTablePayStatusNo {
				notifyOrderHasDispose.CounterInc()
				err = fmt.Errorf("订单已处理")
				return nil, err
			}
			//修改数据库
			orderInfo.Status = model.PmPayOrderTablePayStatusPaid
			orderInfo.PayType = 2
			orderInfo.PlatformTradeNo = data.OutTradeNo
			err = l.orderModel.UpdateNotify(orderInfo)
			if err != nil {
				err = fmt.Errorf("trade_no = %s, UpdateNotify，err:=%v", orderInfo.PlatformTradeNo, err)
				util.CheckError(err.Error())
				return nil, err
			}
			//回调业务方接口
			go func() {
				defer exception.Recover()
				dataMap := make(map[string]interface{})
				dataMap["notify_type"] = code.APP_NOTIFY_TYPE_PAY
				dataMap["out_trade_no"] = orderInfo.OutTradeNo
				_, _ = util.HttpPost(orderInfo.AppNotifyUrl, dataMap, 5*time.Second)
			}()
		}
	} else {
		return &types.WeChatResp{
			Code:    data.ResultCode,
			Message: data.ErrCodeDes,
		}, nil
	}
	return &types.WeChatResp{
		Code:    "SUCCESS",
		Message: "OK",
	}, nil
}
