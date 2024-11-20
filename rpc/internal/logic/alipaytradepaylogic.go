package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	alipay2 "gitee.com/zhuyunkj/alipay/v3"
	"gitee.com/zhuyunkj/pay-gateway/common/clientMgr"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/common/types"
	"gitee.com/zhuyunkj/pay-gateway/common/utils"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/alarm"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"

	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AlipayTradePayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	OrderModel *model.OrderModel
}

var (
	SubscribeVipTradePayErr = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "SubscribeVipTradePayErr", nil, "订阅会员扣款失败", nil})}
)

func NewAlipayTradePayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayTradePayLogic {
	return &AlipayTradePayLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		OrderModel: model.NewOrderModel(define.DbPayGateway),
	}
}

// 支付宝：订阅扣款
func (l *AlipayTradePayLogic) AlipayTradePay(in *pb.AlipayTradePayReq) (*pb.AlipayCommonResp, error) {
	// todo: add your logic here and delete this line

	if in.OutTradeNo == "" || in.ExternalAgreementNo == "" {
		return nil, errors.New("参数异常")
	}

	tb, err := l.OrderModel.GetOneByOutTradeNo(in.OutTradeNo)
	if err != nil {
		logx.Errorf("订阅扣款： 获取订单失败 outTradeNo=%s err=%s", in.OutTradeNo, err.Error())
		return nil, errors.New("获取订单异常")
	}

	agreementSignParams := &alipay2.AgreementParams{
		AgreementNo: tb.AgreementNo,
	}

	product := types.Product{}
	err = json.Unmarshal([]byte(tb.ProductDesc), &product)
	if err != nil {
		logx.Errorf("订阅扣款： 解析订单商品详情 outTradeNo=%s err=%s", in.OutTradeNo, err.Error())
		return nil, errors.New("获取订单异常")
	}

	client, _, notifyUrl, err := clientMgr.GetAlipayClientByAppIdWithCache(tb.PayAppID)
	if err != nil {
		logx.Errorf("订阅扣款：获取支付宝客户端失败 outTradeNo=%s err=%s", in.OutTradeNo, err.Error())
		return nil, errors.New("扣款失败")
	}

	trade := alipay2.Trade{
		OutTradeNo:     utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
		TotalAmount:    fmt.Sprintf("%f", product.Amount),
		Subject:        product.TopText,
		ProductCode:    "GENERAL_WITHHOLDING",
		TimeoutExpress: "30m",
		NotifyURL:      notifyUrl,
	}
	tradePayApp := alipay2.TradePay{
		Trade:           trade,
		AgreementParams: agreementSignParams,
	}

	result, err := client.TradePay(tradePayApp)
	if err != nil {
		logx.Infof("订阅扣款：扣款失败 outTradeNo=%v, err=%s", result, err.Error())
		SubscribeVipTradePayErr.CounterInc()
		return &pb.AlipayCommonResp{
			Status: code.ALI_PAY_FAIL,
			Desc:   "扣款失败",
		}, nil
	} else {
		logx.Infof("%v", result)
		// 回调通知续约成功
		go func() {
			defer exception.Recover()
			dataMap := make(map[string]interface{})
			dataMap["notify_type"] = code.APP_NOTIFY_TYPE_PAY
			dataMap["external_agreement_no"] = in.ExternalAgreementNo

			headerMap := map[string]string{
				"App-Origin": tb.AppPkg,
			}

			err = utils.CallbackWithRetry(tb.AppNotifyUrl, headerMap, dataMap, 5*time.Second)
			if err != nil {
				desc := fmt.Sprintf("回调通知用户续约 异常, app_pkg=%s, out_trade_no=%s", tb.AppPkg, tb.OutTradeNo)
				alarm.ImmediateAlarm("notifyUserSignFeeErr", desc, alarm.ALARM_LEVEL_FATAL)
			}
		}()
		return &pb.AlipayCommonResp{
			Status: code.ALI_PAY_SUCCESS,
			Desc:   "扣款成功",
		}, nil
	}
}
