package logic

import (
	"context"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	"gitee.com/zhuyunkj/zhuyun-core/util"

	"github.com/zeromicro/go-zero/core/logx"
)

type DyOrderRefundLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigTiktokModel *model.PmPayConfigTiktokModel
	refundOrderModel     *model.PmRefundOrderModel
}

func NewDyOrderRefundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DyOrderRefundLogic {
	return &DyOrderRefundLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigTiktokModel: model.NewPmPayConfigTiktokModel(define.DbPayGateway),
		refundOrderModel:     model.NewPmRefundOrderModel(define.DbPayGateway),
	}
}

// 抖音退款订单
func (l *DyOrderRefundLogic) DyOrderRefund(in *pb.DyOrderRefundReq) (out *pb.DyOrderRefundResp, err error) {
	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		//util.CheckError("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		err = fmt.Errorf("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		util.CheckError(err.Error())
		return
	}

	payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(pkgCfg.TiktokPayAppID)
	if cfgErr != nil {
		err = fmt.Errorf("pkgName= %s, 读取字节支付配置失败，err:=%v", in.AppPkgName, cfgErr)
		util.CheckError(err.Error())
		return
	}

	clientConfig := *payCfg.TransClientConfig()
	payClient := client.NewTikTokPay(clientConfig)
	resp, err := payClient.CreateRefundOrder(client.TikTokCreateRefundOrderReq{
		AppId:        clientConfig.AppId,
		OutOrderNo:   in.OutOrderNo,
		OutRefundNo:  in.OutRefundNo,
		Reason:       in.Reason,
		RefundAmount: int(in.RefundAmount),
	})
	if err != nil {
		return
	}

	//写入数据库
	_ = l.refundOrderModel.Create(&model.PmRefundOrderTable{
		AppID:        clientConfig.AppId,
		OutOrderNo:   in.GetOutOrderNo(),
		OutRefundNo:  in.GetOutRefundNo(),
		Reason:       in.GetReason(),
		RefundAmount: int(in.RefundAmount),
		NotifyUrl:    in.NotifyUrl,
		RefundNo:     in.OutRefundNo,
		RefundStatus: 0,
	})

	out = &pb.DyOrderRefundResp{
		ErrNo:   int64(resp.ErrNo),
		ErrTips: resp.ErrTips,
	}

	return
}
