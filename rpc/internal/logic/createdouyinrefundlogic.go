package logic

import (
	"context"
	"errors"
	douyin "gitee.com/zhuyunkj/pay-gateway/common/client/douyinGeneralTrade"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	jsoniter "github.com/json-iterator/go"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDouyinRefundLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigTiktokModel *model.PmPayConfigTiktokModel
	refundOrderModel     *model.PmRefundOrderModel
	orderModel           *model.PmPayOrderModel
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
	}
}

// CreateDouyinRefund 抖音退款 使用通用交易系统
func (l *CreateDouyinRefundLogic) CreateDouyinRefund(in *pb.CreateDouyinRefundReq) (*pb.CreateDouyinRefundResp, error) {
	params, _ := jsoniter.Marshal(in)
	l.Slowf("CreateDouyinRefund params: %v", string(params))

	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		l.Errorf("CreateDouyinRefund pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		return nil, err
	}

	payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(pkgCfg.TiktokPayAppID)
	if cfgErr != nil {
		l.Errorf("CreateDouyinRefund pkgName= %s, 读取字节支付配置失败，err:=%v", in.AppPkgName, cfgErr)
		return nil, cfgErr
	}

	if in.OrderSn == "" && in.OutOrderNo == "" {
		l.Errorf("CreateDouyinRefund pkgName= %s, 订单号和抖音订单号不能同时为空", in.AppPkgName)
		return nil, errors.New("订单号和抖音订单号不能同时为空")
	}

	if in.OutOrderNo == "" && in.OrderSn != "" {
		//查询抖音侧订单号
		payOrderInfo, err := l.orderModel.GetOneByCode(in.OrderSn)
		if err != nil {
			l.Errorf("CreateDouyinRefund pkgName= %s, 读取抖音支付订单失败，err:=%v", in.AppPkgName, err)
			return nil, err
		}
		in.OutOrderNo = payOrderInfo.ThirdOrderNo
	}

	clientConfig := payCfg.GetGeneralTradeConfig()
	payClient := douyin.NewDouyinPay(clientConfig)

	itemOrderDetail := make([]*douyin.ItemOrderDetail, 0)
	//是否是全额退款
	if !in.RefundAll {
		clientToken, err := l.svcCtx.BaseAppConfigServerApi.GetDyClientToken(l.ctx, pkgCfg.TiktokPayAppID)
		if err != nil {
			l.Errorf("CreateDouyinRefund pkgName= %s get douyin client token fail", in.AppPkgName, err)
			return nil, err
		}

		//获取抖音侧订单信息 OutOrderNo等于抖音侧的oriderID
		douyinOrder, err := payClient.QueryOrder(in.OutOrderNo, "", clientToken)
		if err != nil {
			l.Errorf("CreateDouyinRefund pkgName= %s, 读取抖音支付订单失败，err:=%v", in.AppPkgName, err)
			return nil, err
		}
		itemOrderDetail = append(itemOrderDetail, &douyin.ItemOrderDetail{
			ItemOrderId:  douyinOrder.Data.ItemOrderList[0].SkuId,
			RefundAmount: in.RefundAmount,
		})

	}

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
				Text: in.RefundReason,
			},
		},
		RefundTotalAmount: in.RefundAmount,
		ItemOrderDetail:   itemOrderDetail,
		RefundAll:         in.RefundAll,
	}

	clientToken, err := l.svcCtx.BaseAppConfigServerApi.GetDyClientToken(l.ctx, payCfg.AppID)
	if err != nil {
		l.Errorw("get douyin clientToken fail", logx.Field("err", err), logx.Field("appId", payCfg.AppID))
		return nil, err
	}

	refundResp, err := payClient.CreateRefundOrder(refundReq, clientToken)
	if err != nil || refundResp.ErrNo != 0 || refundResp.Data == nil {
		l.Errorf("CreateDouyinRefund createRefund fail, err:%v, req:%+v, resp:%v", err, refundReq, refundResp)
		return nil, err
	}

	//写入数据库
	refundOrder := &model.PmRefundOrderTable{
		AppID:        clientConfig.AppId,
		OutOrderNo:   in.OutOrderNo,
		OutRefundNo:  in.OutRefundNo,
		Reason:       in.RefundReason,
		RefundAmount: int(in.RefundAmount),
		NotifyUrl:    in.RefundNotifyUrl,
		RefundNo:     refundResp.Data.RefundId,
		RefundStatus: model.PmRefundOrderTableRefundStatusApply,
	}
	err = l.refundOrderModel.Create(refundOrder)
	if err != nil {
		l.Errorf("CreateDouyinRefund create refund order fail, err:%v, refundOrder:%+v", err, refundOrder)
	}

	return &pb.CreateDouyinRefundResp{
		RefundId:            refundResp.Data.RefundId,
		RefundAuditDeadline: refundResp.Data.RefundAuditDeadline,
	}, nil
}
