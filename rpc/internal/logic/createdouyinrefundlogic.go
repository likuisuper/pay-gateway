package logic

import (
	"context"
	"errors"

	douyin "gitee.com/zhuyunkj/pay-gateway/common/client/douyinGeneralTrade"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"github.com/zeromicro/go-zero/core/logx"
)

var CreateDyRefundFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "CreateDyRefundFailNum", nil, "抖音创建退款订单异常", nil})}

type CreateDouyinRefundLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigTiktokModel *model.PmPayConfigTiktokModel
	refundOrderModel     *model.PmRefundOrderModel
	orderModel           *model.PmPayOrderModel
	periodOrderModel     *model.PmDyPeriodOrderModel
	dyClient             douyin.PayClient
}

func NewCreateDouyinRefundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDouyinRefundLogic {
	return &CreateDouyinRefundLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigTiktokModel: model.NewPmPayConfigTiktokModel(define.DbPayGateway),
		refundOrderModel:     model.NewPmRefundOrderModel(define.DbPayGateway),
		orderModel:           model.NewPmPayOrderModel(define.DbPayGateway),
		periodOrderModel:     model.NewPmDyPeriodOrderModel(define.DbPayGateway),
		dyClient:             douyin.PayClient{}, // 由于用不到支付相关的配置 直接初始化一个空的就是
	}
}

// CreateDouyinRefund 抖音退款 使用通用交易系统
func (l *CreateDouyinRefundLogic) CreateDouyinRefund(in *pb.CreateDouyinRefundReq) (*pb.CreateDouyinRefundResp, error) {

	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		CreateDyRefundFailNum.CounterInc()
		l.Errorf("CreateDouyinRefund pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		return nil, err
	}

	payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(pkgCfg.TiktokPayAppID)
	if cfgErr != nil {
		CreateDyRefundFailNum.CounterInc()
		l.Errorf("CreateDouyinRefund pkgName= %s, 读取字节支付配置失败，err:=%v", in.AppPkgName, cfgErr)
		return nil, cfgErr
	}

	if in.OrderSn == "" && in.OutOrderNo == "" {
		CreateDyRefundFailNum.CounterInc()
		l.Errorf("CreateDouyinRefund pkgName= %s, 订单号和抖音订单号不能同时为空", in.AppPkgName)
		return nil, errors.New("订单号和抖音订单号不能同时为空")
	}

	if in.IsPeriodProduct {
		return l.DyPeriodRefund(in, pkgCfg, payCfg)

	}
	return l.DyRefund(in, err, pkgCfg, payCfg)
}

// DyPeriodRefund 抖音代扣退款
func (l *CreateDouyinRefundLogic) DyPeriodRefund(in *pb.CreateDouyinRefundReq, pkgCfg *model.PmAppConfigTable, payCfg *model.PmPayConfigTiktokTable) (*pb.CreateDouyinRefundResp, error) {
	//查询订单数否存在
	periodOrderInfo, err := l.periodOrderModel.GetOneByOrderSnAndAppId(in.OrderSn, pkgCfg.TiktokPayAppID)
	if err != nil || periodOrderInfo == nil || periodOrderInfo.ID < 1 {
		CreateDyRefundFailNum.CounterInc()
		l.Errorf("CreateDyPeriodRefund pkgName= %s, order_sn: %v 获取抖音代扣订单失败 err:=%v", in.AppPkgName, in.OrderSn, err)
		return nil, err
	}

	//解约
	terminateRes, err := NewDouyinPeriodOrderLogic(l.ctx, l.svcCtx).terminateSign(&pb.DouyinPeriodOrderReq{
		Action:            pb.DouyinPeriodOrderReqAction_DyPeriodActionCancel,
		Pkg:               periodOrderInfo.AppPkgName,
		UserId:            int64(periodOrderInfo.UserId),
		PmDyPeriodOrderId: int64(periodOrderInfo.ID),
	})
	if err != nil {
		CreateDyRefundFailNum.CounterInc()
		l.Errorf("CreateDyPeriodRefund pkgName= %s, order_sn: %v 解约失败 err:=%v", in.AppPkgName, in.OrderSn, err)
		return nil, err
	}
	if !terminateRes.IsUnsignSuccess {
		CreateDyRefundFailNum.CounterInc()
		l.Errorf("CreateDyPeriodRefund pkgName= %s, order_sn: %v 解约失败 err:=%v,res:%v", in.AppPkgName, in.OrderSn, err, terminateRes)
		return nil, err
	}

	clientToken, err := l.svcCtx.BaseAppConfigServerApi.GetDyClientToken(l.ctx, periodOrderInfo.PayAppId)
	if err != nil || clientToken == "" {
		l.Errorw("CreateDyPeriodRefund get douyin client token fail", logx.Field("err", err), logx.Field("appId", periodOrderInfo.PayAppId))
		return nil, errors.New("获取抖音支付token失败")
	}

	//请求退款
	refundNo, err := l.dyClient.CreateSignRefund(clientToken, in.OutRefundNo, periodOrderInfo.ThirdOrderNo, payCfg.NotifyUrl, in.RefundReason, in.RefundAmount)
	if err != nil {
		CreateDyRefundFailNum.CounterInc()
		l.Errorf("CreateDyPeriodRefund pkgName= %s, order_sn: %v 创建抖音退款订单失败 err:=%v", in.AppPkgName, in.OrderSn, err)
		return nil, err
	}

	//退款表新增记录
	l.Slowf("CreateDyPeriodRefund createRefund success, req:%+v,refundResp:%+v", in, refundNo)
	//写入数据库
	refundOrder := &model.PmRefundOrderTable{
		AppID:        pkgCfg.TiktokPayAppID,
		OutOrderNo:   in.OutOrderNo,
		OutRefundNo:  in.OutRefundNo,
		Reason:       in.RefundReason,
		RefundAmount: int(in.RefundAmount),
		NotifyUrl:    periodOrderInfo.NotifyUrl, //退款回调地址和支付回调地址一致
		RefundNo:     refundNo,
		RefundStatus: model.PmRefundOrderTableRefundStatusApply,
	}
	err = l.refundOrderModel.Create(refundOrder)
	if err != nil {
		CreateDyRefundFailNum.CounterInc()
		l.Errorf("CreateDyPeriodRefund create refund order fail, err:%v, refundOrder:%+v", err, refundOrder)
	}

	//更新订单状态(解约逻辑中已修改)

	//返回三方退款订单号
	return &pb.CreateDouyinRefundResp{
		RefundId: refundNo,
	}, nil
}

// DyRefund 抖音退款
func (l *CreateDouyinRefundLogic) DyRefund(in *pb.CreateDouyinRefundReq, err error, pkgCfg *model.PmAppConfigTable, payCfg *model.PmPayConfigTiktokTable) (*pb.CreateDouyinRefundResp, error) {
	//查询订单是否存在
	payOrderInfo, err := l.orderModel.GetOneByOrderSnAndAppId(in.OrderSn, pkgCfg.TiktokPayAppID)
	if err != nil || payOrderInfo == nil || payOrderInfo.ID < 1 {
		CreateDyRefundFailNum.CounterInc()
		l.Errorf("CreateDouyinRefund pkgName= %s, order_sn: %v 获取抖音支付订单失败 err:=%v", in.AppPkgName, in.OrderSn, err)
		return nil, err
	}

	if in.OutOrderNo == "" && in.OrderSn != "" {
		in.OutOrderNo = payOrderInfo.ThirdOrderNo
	}

	currency := ""
	if payOrderInfo.Currency == "DYDIAMOND" {
		currency = "DIAMOND"
	}

	clientConfig := payCfg.GetGeneralTradeConfig()
	payClient := douyin.NewDouyinPay(clientConfig)

	refundReq := &douyin.CreateRefundOrderReq{
		OrderId:     in.OutOrderNo,
		OutRefundNo: in.OutRefundNo,
		CpExtra:     "extra_info",
		OrderEntrySchema: douyin.Schema{
			Path:   in.GetOrderEntrySchema().GetPath(),
			Params: in.GetOrderEntrySchema().GetParams(),
		},
		NotifyUrl: clientConfig.NotifyUrl,
		RefundReason: []*douyin.RefundReason{
			{
				Code: 999,
				Text: "其他",
			},
		},
		RefundTotalAmount: in.RefundAmount,
		RefundAll:         in.RefundAll,
		Currency:          currency,
	}

	itemOrderDetail := make([]*douyin.ItemOrderDetail, 0)

	//是否是全额退款
	if !in.RefundAll {
		clientToken, err := l.svcCtx.BaseAppConfigServerApi.GetDyClientToken(l.ctx, pkgCfg.TiktokPayAppID)
		if err != nil {
			CreateDyRefundFailNum.CounterInc()
			l.Errorf("CreateDouyinRefund pkgName= %s get douyin client token fail", in.AppPkgName, err)
			return nil, err
		}

		//获取抖音侧订单信息
		douyinOrder, err := payClient.QueryOrder(in.GetOutOrderNo(), in.GetOrderSn(), clientToken)
		if err != nil || douyinOrder == nil || douyinOrder.Data == nil || len(douyinOrder.Data.ItemOrderList) == 0 {
			CreateDyRefundFailNum.CounterInc()
			l.Errorf("CreateDouyinRefund pkgName=%s, 读取抖音支付订单失败 err:=%v", in.AppPkgName, err)
			return nil, err
		}

		itemOrderDetail = append(itemOrderDetail, &douyin.ItemOrderDetail{
			ItemOrderId:  douyinOrder.Data.ItemOrderList[0].ItemOrderId,
			RefundAmount: in.RefundAmount,
		})
		refundReq.ItemOrderDetail = itemOrderDetail
		refundReq.OrderId = douyinOrder.Data.OrderId
	}

	clientToken, err := l.svcCtx.BaseAppConfigServerApi.GetDyClientToken(l.ctx, payCfg.AppID)
	if err != nil {
		CreateDyRefundFailNum.CounterInc()
		l.Errorw("get douyin clientToken fail", logx.Field("err", err), logx.Field("appId", payCfg.AppID))
		return nil, err
	}

	refundResp, err := payClient.CreateRefundOrder(refundReq, clientToken)
	if err != nil {
		CreateDyRefundFailNum.CounterInc()
		l.Errorf("CreateDouyinRefund createRefund fail, err:%v, req:%+v, resp:%+v", err, refundReq, refundResp)
		return nil, err
	}

	if refundResp.ErrNo != 0 {
		CreateDyRefundFailNum.CounterInc()
		l.Errorf("CreateDouyinRefund createRefund fail, req:%+v, resp:%+v", refundReq, refundResp)
		return nil, errors.New(refundResp.ErrMsg)
	}

	l.Slowf("CreateDouyinRefund createRefund success, req:%+v,refundResp:%+v", refundReq, refundResp)
	//写入数据库
	refundOrder := &model.PmRefundOrderTable{
		AppID:        clientConfig.AppId,
		OutOrderNo:   in.OutOrderNo,
		OutRefundNo:  in.OutRefundNo,
		Reason:       in.RefundReason,
		RefundAmount: int(in.RefundAmount),
		NotifyUrl:    payOrderInfo.NotifyUrl, //退款回调地址和支付回调地址一致
		RefundNo:     refundResp.Data.RefundId,
		RefundStatus: model.PmRefundOrderTableRefundStatusApply,
	}
	err = l.refundOrderModel.Create(refundOrder)
	if err != nil {
		CreateDyRefundFailNum.CounterInc()
		l.Errorf("CreateDouyinRefund create refund order fail, err:%v, refundOrder:%+v", err, refundOrder)
	}

	return &pb.CreateDouyinRefundResp{
		RefundId:            refundResp.Data.RefundId,
		RefundAuditDeadline: refundResp.Data.RefundAuditDeadline,
	}, nil
}
