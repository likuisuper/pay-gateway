package notify

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"io/ioutil"
	"net/http"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyWechatUnifiedOrderLogic struct {
	logx.Logger
	ctx         context.Context
	svcCtx      *svc.ServiceContext
	orderModel  *model.OrderModel
	refundModel *model.RefundModel
}

func NewNotifyWechatUnifiedOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyWechatUnifiedOrderLogic {
	return &NotifyWechatUnifiedOrderLogic{
		Logger:      logx.WithContext(ctx),
		ctx:         ctx,
		svcCtx:      svcCtx,
		orderModel:  model.NewOrderModel(define.DbPayGateway),
		refundModel: model.NewRefundModel(define.DbPayGateway),
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
}

//orderInfo结构
type AttachInfo struct {
	OrderSn  string
	Amount   int
	Subject  string
	KsTypeId int
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
			return
		}
		if orderInfo.Status != model.PmPayOrderTablePayStatusNo {
			notifyOrderHasDispose.CounterInc()
			err = fmt.Errorf("订单已处理")
			return
		}
		//修改数据库
		orderInfo.Status = model.PmPayOrderTablePayStatusPaid
		orderInfo.PayType = 2
		orderInfo.PlatformTradeNo = data.OutTradeNo
		err = l.orderModel.UpdateNotify(orderInfo)
		if err != nil {
			err = fmt.Errorf("trade_no = %s, UpdateNotify，err:=%v", orderInfo.PlatformTradeNo, err)
			util.CheckError(err.Error())
			return
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

	return
}
