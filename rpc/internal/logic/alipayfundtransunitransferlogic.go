package logic

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	alipay2 "github.com/smartwalle/alipay/v3"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	alipayFundTransUniTransferFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "alipayFundTransUniTransferFailNum", nil, "支付宝转账失败", nil})}
)

type AlipayFundTransUniTransferLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	fundTransOrderModel  *model.PmFundTransOrderModel
	payConfigAlipayModel *model.PmPayConfigAlipayModel
}

func NewAlipayFundTransUniTransferLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayFundTransUniTransferLogic {
	return &AlipayFundTransUniTransferLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		fundTransOrderModel:  model.NewPmFundTransOrderModel(define.DbPayGateway),
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
	}
}

// 支付宝转出
func (l *AlipayFundTransUniTransferLogic) AlipayFundTransUniTransfer(in *pb.AlipayFundTransUniTransferReq) (res *pb.Empty, err error) {
	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(in.AppPkgName)
	if err != nil {
		util.CheckError("pkgName= %s, 读取应用配置失败，err:=%v", in.AppPkgName, err)
		err = errors.New("读取应用配置失败")
		return
	}

	payCfg, cfgErr := l.payConfigAlipayModel.GetOneByAppID(pkgCfg.AlipayAppID)
	if cfgErr != nil {
		err = fmt.Errorf("pkgName= %s, 读取支付宝配置失败，err:=%v", in.AppPkgName, cfgErr)
		util.CheckError(err.Error())
		return
	}

	// 将 key 的验证调整到初始化阶段
	payClient, err := client.GetAlipayClient(*payCfg.TransClientConfig())
	if err != nil {
		util.CheckError("pkgName= %s, 初使化支付错误，err:=%v", in.AppPkgName, err)
		return
	}
	userInfo := &alipay2.PayeeInfo{
		Identity:     in.PayeeInfo.Identity,
		IdentityType: in.PayeeInfo.IdentityType,
		Name:         in.PayeeInfo.Name,
	}
	fundTransUniTransfer := alipay2.FundTransUniTransfer{
		OutBizNo:       in.OrderSn,
		TransAmount:    in.TransAmount,
		ProductCode:    in.ProductCode,
		BizScene:       in.BizScene,
		OrderTitle:     in.OrderTitle,
		PayeeInfo:      userInfo,
		Remark:         in.Remark,
		BusinessParams: in.BusinessParams,
	}
	rest, err := payClient.FundTransUniTransfer(fundTransUniTransfer)
	if err != nil {
		err = fmt.Errorf("pkgName= %s, 支付宝转账失败，err:=%v", in.AppPkgName, err)
		logx.Errorf(err.Error())
		alipayFundTransUniTransferFailNum.CounterInc()
		return
	}

	if rest.IsSuccess() == false {
		err = fmt.Errorf("调用转账失败 err: %s %s", rest.Content.SubCode, rest.Content.SubMsg)
		logx.Errorf(err.Error())
		alipayFundTransUniTransferFailNum.CounterInc()
	}

	amount, _ := strconv.ParseFloat(in.TransAmount, 64)
	orderInfo := &model.PmFundTransOrderTable{
		OrderSn:    in.OrderSn,
		AppPkgName: in.AppPkgName,
		Amount:     int(amount * 100),
		AliName:    in.PayeeInfo.Name,
		AliAccount: in.PayeeInfo.Identity,
		PayAppId:   payCfg.AppID,
	}
	err = l.fundTransOrderModel.Create(orderInfo)
	if err != nil {
		err = fmt.Errorf("fundTransOrderModel Create err: %v", err)
		util.CheckError(err.Error())
		return
	}

	return &pb.Empty{}, nil
}
