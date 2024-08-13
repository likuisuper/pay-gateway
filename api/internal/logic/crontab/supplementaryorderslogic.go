package crontab

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/logic/notify"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/api/common/notice"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	douyin "gitee.com/zhuyunkj/pay-gateway/common/client/douyinGeneralTrade"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/bytedance/sonic"
	"github.com/zeromicro/go-zero/core/logx"
)

const DingdingRobot = "https://oapi.dingtalk.com/robot/send?access_token=e59900c50124bc9353e9a83410de55c5c9a351bef988aa4a7fd61bbec00239ac"

var (
	orderSupplementaryErrNum     = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "notifyOrderSupplementaryErrNum", nil, "订单补偿失败", nil})}
	orderSupplementarySuccessNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "notifyOrderSupplementarySuccessNum", nil, "订单补偿成功", nil})}
)

type SupplementaryOrdersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	payOrderModel  *model.PmPayOrderModel
	appConfigModel *model.PmAppConfigModel

	//payConfigAlipayModel *model.PmPayConfigAlipayModel
	payConfigTiktokModel *model.PmPayConfigTiktokModel
	payConfigWechatModel *model.PmPayConfigWechatModel
	//payConfigKsModel     *model.PmPayConfigKsModel
}

func NewSupplementaryOrdersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SupplementaryOrdersLogic {
	return &SupplementaryOrdersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,

		payOrderModel:  model.NewPmPayOrderModel(define.DbPayGateway),
		appConfigModel: model.NewPmAppConfigModel(define.DbPayGateway),
		//payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
		payConfigTiktokModel: model.NewPmPayConfigTiktokModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
		//payConfigKsModel:     model.NewPmPayConfigKsModel(define.DbPayGateway),
	}
}

// SupplementaryOrders 定时任务补单逻辑
func (l *SupplementaryOrdersLogic) SupplementaryOrders(req *types.SupplementaryOrdersReq) (resp *types.SupplementaryOrdersResp, err error) {
	//请求参数处理，获取时间区间
	if req.Type != "lastDay" && req.Type != "lastTenMinute" {
		return nil, errors.New("invalid type")
	}
	startTime, endTime := l.getRequestParams(req)

	//获取待处理订单
	payList, err := l.payOrderModel.GetListByCreateTimeRange(startTime, endTime)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			l.Logger.Error("get list by create time range fail,err=%v", err)
		}
		return nil, errors.New("get list by create time range fail")
	}

	if len(payList) == 0 {
		l.Logger.Info("SupplementaryOrders: no order need supplementary")
		return nil, errors.New("no order need supplementary")
	}

	go util.SafeRun(func() {
		//重写ctx,防止超时
		ctx := context.Background()
		tracer := otel.GetTracerProvider().Tracer(trace.TraceName)
		ctx, span := tracer.Start(ctx, "GetRefundOrder", oteltrace.WithSpanKind(oteltrace.SpanKindServer))
		defer span.End()
		l.ctx = ctx
		l.Logger = logx.WithContext(ctx)

		totalNeedSupplementCount := len(payList)
		wxNeedSupplementCount := 0
		wxActualSupplementCount := 0
		douyinNeedSupplementCount := 0
		douyinActualSupplementCount := 0
		// 对需要补单的订单进行处理
		for _, payItem := range payList {
			//调用三方，触发回调
			pkgCfg, err := l.appConfigModel.GetOneByPkgName(payItem.AppPkgName)
			if err != nil {
				l.Logger.Errorf("读取应用配置失败 pkgName= %s, err:=%v", payItem.AppPkgName, err)
				continue
			}

			//小程序目前调过来的支付类型只有1，2，5，8 这4种，其中5为快手已弃用，2为抖音担保交易，已废弃，故实现1，，8即可
			switch payItem.PayType {
			case model.PmPayOrderTablePayTypeWechatPayUni: //model.PmPayOrderTablePayTypeWechatPayH5 ,暂时没用
				wxNeedSupplementCount++
				if err = l.handleWxOrder(payItem, pkgCfg.WechatPayAppID); err != nil {
					if !errors.Is(err, model.NoNeedSupplementaryError) {
						orderSupplementaryErrNum.CounterInc()
						l.Logger.Error(err.Error())
					}
				} else {
					orderSupplementarySuccessNum.CounterInc()
					wxActualSupplementCount++

				}
			case model.PmPayOrderTablePayTypeDouyinGeneralTrade: //抖音，通用交易
				douyinNeedSupplementCount++
				if err = l.handleDouyinOrder(payItem, pkgCfg.TiktokPayAppID); err != nil {
					if !errors.Is(err, model.NoNeedSupplementaryError) {
						orderSupplementaryErrNum.CounterInc()
						l.Logger.Error(err.Error())
					}
				} else {
					orderSupplementarySuccessNum.CounterInc()
					douyinActualSupplementCount++
				}
			}
			//case model.PmPayOrderTablePayTypeTiktokPayEc: //字节,担保交易，已废弃
			//	if err = l.handleBytedanceOrder(payItem, pkgCfg.TiktokPayAppID); err != nil {
			//		l.Logger.Error(err.Error())
			//	}
		}

		msg := fmt.Sprintf("订单补偿信息 :\n需要补单总数:%d;\n微信需要补单总数：%d,实际补单数目%d;\n抖音需要补单总数：%d,实际补单数目%d", totalNeedSupplementCount, wxNeedSupplementCount, wxActualSupplementCount, douyinNeedSupplementCount, douyinActualSupplementCount)
		l.Logger.Info(msg)
		if req.IsNotice == "1" {
			req := &notice.RobotSendReq{
				Msgtype: "text",
				Text: &notice.Text{
					Content: msg,
				},
			}
			_, err = notice.SendWebhookMsg(l.ctx, req, DingdingRobot)
			if err != nil {
				l.Errorf("dingDing notify fail, err:%v", err)
				return
			}
		}
	})

	return &types.SupplementaryOrdersResp{
		ErrNo:   0,
		ErrTips: "ok",
	}, nil
}

// handleWxOrder 微信订单处理
func (l *SupplementaryOrdersLogic) handleWxOrder(orderInfo *model.PmPayOrderTable, appId string) error {
	payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(appId)
	if cfgErr != nil {
		return fmt.Errorf("handleWxOrder: 读取微信支付配置失败 pkgName= %s, err:=%v", orderInfo.AppPkgName, cfgErr)
	}

	//查询微信订单状态
	payConf := payCfg.TransClientConfig()
	payClient := client.NewWeChatCommPay(*payConf)
	transaction, err := payClient.GetOrderStatus(orderInfo.OrderSn)
	if err != nil {
		return fmt.Errorf("handleWxOrder:查询微信订单失败, orderSn=%s, err=%v", orderInfo.OrderSn, err)
	}

	if *transaction.TradeState == "SUCCESS" {
		isSupplementary, err := l.payOrderModel.QueryAfterUpdate(*transaction.OutTradeNo, appId, *transaction.TransactionId, int(*transaction.Amount.PayerTotal))
		if err != nil {
			return err
		}

		if isSupplementary { //成功补单
			_, err = util.HttpPost(orderInfo.NotifyUrl, transaction, 5*time.Second)
			if err != nil {
				notify.CallbackBizFailNum.CounterInc()
				return fmt.Errorf("handleWxOrder:callback notify_url failed , transaction:%+v, err:%v", transaction, err)
			}
			//正常处理
			return nil
		}

		//这种情况不存在
	}

	//订单状态未完成，等待下次脚本刷新
	return model.NoNeedSupplementaryError
}

// handleDouyinOrder 抖音订单回调处理
func (l *SupplementaryOrdersLogic) handleDouyinOrder(orderInfo *model.PmPayOrderTable, appId string) error {
	payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(appId)
	if cfgErr != nil {
		return fmt.Errorf("handleDouyinOrder: pkgName= %s, 读取抖音支付配置失败，err:=%v", orderInfo.AppPkgName, cfgErr)
	}

	douyinPayConfig := payCfg.GetGeneralTradeConfig()
	payClient := douyin.NewDouyinPay(douyinPayConfig)

	clientToken, err := l.svcCtx.BaseAppConfigServerApi.GetDyClientToken(l.ctx, douyinPayConfig.AppId)
	if err != nil {
		l.Errorw("get douyin client token fail", logx.Field("err", err), logx.Field("appId", douyinPayConfig.AppId))
		return err
	}

	douyinOrder, err := payClient.QueryOrder("", orderInfo.OrderSn, clientToken)
	if err != nil {
		return fmt.Errorf("handleDouyinOrder:查询抖音订单失败, orderSn=%s, err=%v", orderInfo.OrderSn, err)
	}
	douyinOrderData := douyinOrder.Data
	if douyinOrderData != nil && douyinOrderData.PayStatus == "SUCCESS" {

		isSupplementary, err := l.payOrderModel.QueryAfterUpdate(douyinOrderData.OutOrderNo, appId, douyinOrderData.OrderId, int(douyinOrderData.TotalAmount))
		if err != nil {
			return err
		}

		if isSupplementary { //成功补单
			//回调业务方接口
			msg, _ := sonic.MarshalString(douyin.GeneralTradeMsg{
				OutOrderNo: orderInfo.OrderSn,
			})
			req := &types.ByteDanceReq{
				Msg: msg,
			}
			_, err = util.HttpPost(orderInfo.NotifyUrl, req, 5*time.Second)
			if err != nil {
				notify.CallbackBizFailNum.CounterInc()
				return fmt.Errorf("\"handleDouyinOrder:callback notify_url failed , req:%+v, err:%v", req, err)
			}
			//正常处理
			return nil
		}

		//这种情况不存在
	}

	return model.NoNeedSupplementaryError
}

// getRequestParams 参数处理
func (l *SupplementaryOrdersLogic) getRequestParams(req *types.SupplementaryOrdersReq) (startTime, endTime time.Time) {
	now := time.Now()
	if req.Type == "lastDay" {
		yesterday := now.AddDate(0, 0, -1)
		startTime = getNatureDayTime(yesterday)
		endTime = getNatureDayTime(now)
	} else { //默认10分钟
		startMinute, err := strconv.Atoi(req.StartMinute)
		if err != nil {
			startMinute = 10
		}
		endMinute, err := strconv.Atoi(req.EndMinute)
		if err != nil {
			endMinute = 1
		}

		if startMinute < endMinute || startMinute-endMinute > 720 { //最多跑1天的订单量
			startMinute = 10
			endMinute = 1
		}

		startTime = now.Add(-time.Duration(startMinute) * time.Minute)
		endTime = now.Add(-time.Duration(endMinute) * time.Minute)
	}

	return
}

// getNatureDayTime 获取自然日时间
func getNatureDayTime(ts time.Time) time.Time {
	dayTime := time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, ts.Location())
	return dayTime
}

// httpPost 调试使用
//func httpPost(url string, data interface{}, timeout time.Duration) (string, error) {
//	jsonStr, _ := json.Marshal(data)
//	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
//	if err != nil {
//		return "", err
//	}
//	req.Header.Add("content-type", "application/json")
//
//	defer req.Body.Close()
//
//	httpClient := &http.Client{Timeout: timeout}
//	resp, err := httpClient.Do(req)
//	if err != nil {
//		return "", err
//	}
//	defer resp.Body.Close()
//	result, err := io.ReadAll(resp.Body)
//	return string(result), err
//}

// handleTiktokOrder 抖音支付，担保交易，已废弃
//func (l *SupplementaryOrdersLogic) handleBytedanceOrder(orderInfo *model.PmPayOrderTable, appId string) error {
//	payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(appId)
//	if cfgErr != nil {
//		return fmt.Errorf("pkgName= %s, 读取字节支付配置失败，err:=%v", orderInfo.AppPkgName, cfgErr)
//	}
//
//	//查询抖音订单状态
//	payConf := payCfg.TransClientConfig()
//	payCli := client.NewTikTokPay(*payConf)
//
//	tiktokOrderInfo, err := payCli.GetOrderStatus(orderInfo.OrderSn)
//	if err != nil {
//		return fmt.Errorf("查询字节订单失败, orderSn=%s, err=%v", orderInfo.OrderSn, err)
//	}
//
//	if tiktokOrderInfo.OrderStatus == "SUCCESS" {
//		currentOrderInfo, err := l.payOrderModel.GetOneByCode(orderInfo.OrderSn)
//		if err != nil {
//			return fmt.Errorf("获取订单失败！err=%v,order_code = %s", err, orderInfo.OrderSn)
//		}
//
//		if currentOrderInfo.PayStatus != model.PmPayOrderTablePayStatusNo { //任务执行期间，已触发了回调
//			return nil
//		}
//
//		currentOrderInfo.NotifyAmount = tiktokOrderInfo.TotalFee
//		currentOrderInfo.PayStatus = model.PmPayOrderTablePayStatusPaid
//		err = l.payOrderModel.UpdateNotify(currentOrderInfo)
//		if err != nil {
//			return fmt.Errorf("orderSn = %s, UpdateNotify，err:=%v", orderInfo.OrderSn, err)
//		}
//
//		msg, _ := sonic.MarshalString(douyin.GeneralTradeMsg{
//			OutOrderNo: orderInfo.OrderSn,
//		})
//		req := &types.ByteDanceReq{
//			Msg: msg,
//		}
//		_, requestErr := util.HttpPost(orderInfo.NotifyUrl, req, 5*time.Second)
//		if requestErr != nil {
//			return fmt.Errorf("NotifyPayment-post, req:%+v, err:%v", orderInfo.OrderSn, err)
//		}
//	}
//	//订单状态未完成，等待下次脚本刷新
//	return nil
//}
