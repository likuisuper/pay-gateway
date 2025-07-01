package logic

import (
	"context"
	"fmt"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/client"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/pb/pb"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/util"

	"github.com/zeromicro/go-zero/core/logx"
)

type WechatMiniRefundQueryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigWechatModel *model.PmPayConfigWechatModel
	refundOrderModel     *model.PmRefundOrderModel
	orderModel           *model.PmPayOrderModel
}

func NewWechatMiniRefundQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WechatMiniRefundQueryLogic {
	return &WechatMiniRefundQueryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
		refundOrderModel:     model.NewPmRefundOrderModel(define.DbPayGateway),
		orderModel:           model.NewPmPayOrderModel(define.DbPayGateway),
	}
}

// 小程序-微信的退款单详情
func (l *WechatMiniRefundQueryLogic) WechatMiniRefundQuery(in *pb.WechatMiniRefundQueryReq) (out *pb.WechatMiniRefundQueryResp, err error) {

	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		//util.CheckError("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		err = fmt.Errorf("WechatMiniRefundQuery pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		util.CheckError(err.Error())
		return
	}

	payCfg, cfgErr := l.payConfigWechatModel.GetOneByAppID(pkgCfg.WechatPayAppID)
	if cfgErr != nil {
		err = fmt.Errorf("WechatMiniRefundQuery pkgName= %s, 读取微信支付配置失败，err:=%v", in.AppPkgName, cfgErr)
		util.CheckError(err.Error())
		return
	}

	clientConfig := *payCfg.TransClientConfig()
	payClient := client.NewWeChatCommPay(clientConfig)
	resp, err := payClient.MiniRefundOrderQuery(&client.MiniRefundOrderQuery{
		OutRefundNo: in.OutRefundNo,
	})

	if err != nil {
		err = fmt.Errorf("WechatMiniRefundQuery 退款查询失败:OutRefundNo = %s .err =%v ", in.OutRefundNo, err)
		util.CheckError(err.Error())
		return nil, err
	}

	refundInfo, err := l.refundOrderModel.GetInfo(in.OutRefundNo)

	if err == nil && refundInfo.ID > 0 && pkgCfg.WechatPayAppID == refundInfo.AppID {
		//存在记录, 判断下是否已被回调通知处理，未处理的再次更新一遍
		//SUCCESS: 退款成功
		//CLOSED: 退款关闭
		//PROCESSING: 退款处理中
		//ABNORMAL: 退款异常，
		if refundInfo.RefundStatus != model.PmRefundOrderTableRefundStatusSuccess && *resp.Status == "SUCCESS" {
			refundInfo.RefundStatus = model.PmRefundOrderTableRefundStatusSuccess
			l.refundOrderModel.Update(in.OutRefundNo, refundInfo)
		}

		if refundInfo.RefundStatus != model.PmRefundOrderTableRefundStatusFail && *resp.Status == "ABNORMAL" {
			refundInfo.RefundStatus = model.PmRefundOrderTableRefundStatusFail
			l.refundOrderModel.Update(in.OutRefundNo, refundInfo)
		}

	}

	out = &pb.WechatMiniRefundQueryResp{
		RefundId:    *resp.RefundId,
		OutRefundNo: *resp.OutRefundNo,
		SuccessTime: resp.SuccessTime.Format("2006-01-02 15:04:05"), //todo 待打包的机器上go版本升级后可以代替：resp.SuccessTime.Format(time.DateTime),
		CreateTime:  resp.CreateTime.Format("2006-01-02 15:04:05"),
		Status:      string(*resp.Status),
	}

	return
}
