package logic

import (
	"context"
	douyin "gitee.com/zhuyunkj/pay-gateway/common/client/douyinGeneralTrade"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDouyinRefundLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigTiktokModel *model.PmPayConfigTiktokModel
	refundOrderModel     *model.PmRefundOrderModel
}

func NewCreateDouyinRefundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDouyinRefundLogic {
	return &CreateDouyinRefundLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigTiktokModel: model.NewPmPayConfigTiktokModel(define.DbPayGateway),
		refundOrderModel:     model.NewPmRefundOrderModel(define.DbPayGateway),
	}
}

// CreateDouyinRefund 抖音退款 使用通用交易系统
func (l *CreateDouyinRefundLogic) CreateDouyinRefund(in *pb.CreateDouyinRefundReq) (*pb.CreateDouyinRefundResp, error) {
	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		l.Errorf("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		return nil, err
	}

	payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(pkgCfg.TiktokPayAppID)
	if cfgErr != nil {
		l.Errorf("pkgName= %s, 读取字节支付配置失败，err:=%v", in.AppPkgName, cfgErr)
		return nil, cfgErr
	}

	clientConfig := payCfg.GetGeneralTradeConfig()
	payClient := douyin.NewDouyinPay(clientConfig)
	refundReq := &douyin.CreateRefundOrderReq{
		OrderId:     in.OutOrderNo, // todo: 需要使用抖音侧支付订单号 这个理论上要保存在中台支付的订单表上
		OutRefundNo: in.OutRefundNo,
		CpExtra:     "",
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
		ItemOrderDetail:   nil,
		RefundAll:         false,
	}

	clientToken, err := l.svcCtx.BaseAppConfigServerApi.GetDyClientToken(l.ctx, payCfg.AppID)
	if err != nil {
		l.Errorw("get douyin clientToken fail", logx.Field("err", err), logx.Field("appId", payCfg.AppID))
		return nil, err
	}

	refundResp, err := payClient.CreateRefundOrder(refundReq, clientToken)
	if err != nil || refundResp.ErrNo != 0 || refundResp.Data == nil {
		l.Errorf("createRefund fail, err:%v, req:%+v, resp:%v", err, refundReq, refundResp)
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
		RefundStatus: 0,
	}
	err = l.refundOrderModel.Create(refundOrder)
	if err != nil {
		l.Errorf("create refund order fail, err:%v, refundOrder:%+v", err, refundOrder)
	}

	return &pb.CreateDouyinRefundResp{
		RefundId:            refundResp.Data.RefundId,
		RefundAuditDeadline: refundResp.Data.RefundAuditDeadline,
	}, nil
}
