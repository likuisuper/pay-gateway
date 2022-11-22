package inter

import (
	"context"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/api/common/response"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/code"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	alipay2 "github.com/smartwalle/alipay/v3"
	"strconv"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	alipayFundTransUniTransferFailNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "alipayFundTransUniTransferFailNum", nil, "支付宝转账失败", nil})}
)

type AlipayFundTransUniTransferLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	appConfigModel       *model.PmAppConfigModel
	fundTransOrderModel  *model.PmFundTransOrderModel
	payConfigAlipayModel *model.PmPayConfigAlipayModel
}

func NewAlipayFundTransUniTransferLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayFundTransUniTransferLogic {
	return &AlipayFundTransUniTransferLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		fundTransOrderModel:  model.NewPmFundTransOrderModel(define.DbPayGateway),
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
	}
}

func (l *AlipayFundTransUniTransferLogic) AlipayFundTransUniTransfer(req *types.AlipayFundTransUniTransferReq) (resp *types.ResultResp, err error) {
	//检查金额 最大100
	amountFloat, err := strconv.ParseFloat(req.TransAmount, 64)
	if err != nil {
		err = fmt.Errorf("transAmount转化错误:%v", err)
		util.CheckError(err.Error())
		res := response.MakeResult(code.CODE_ERROR, err.Error(), nil)
		return &res, nil
	}
	if amountFloat < 0.1 || amountFloat > 100 {
		err = fmt.Errorf("转账金额必须在0.1~100之间，当前金额:%.2f", amountFloat)
		res := response.MakeResult(code.CODE_ERROR, err.Error(), nil)
		return &res, nil
	}

	//读取应用配置
	pkgCfg, err := l.appConfigModel.GetOneByPkgName(req.PkgName)
	if err != nil {
		util.CheckError("pkgName= %s, 读取应用配置失败，err:=%v", req.PkgName, err)
		res := response.MakeResult(code.CODE_ERROR, err.Error(), nil)
		return &res, nil
	}

	payCfg, cfgErr := l.payConfigAlipayModel.GetOneByAppID(pkgCfg.AlipayAppID)
	if cfgErr != nil {
		err = fmt.Errorf("pkgName= %s, 读取支付宝配置失败，err:=%v", req.PkgName, cfgErr)
		util.CheckError(err.Error())
		res := response.MakeResult(code.CODE_ERROR, err.Error(), nil)
		return &res, nil
	}

	// 将 key 的验证调整到初始化阶段
	payClient, err := client.GetAlipayClient(*payCfg.TransClientConfig())
	if err != nil {
		err = fmt.Errorf("pkgName= %s, 初使化支付错误，err:=%v", req.PkgName, err)
		util.CheckError(err.Error())
		res := response.MakeResult(code.CODE_ERROR, err.Error(), nil)
		return &res, nil
	}

	userInfo := &alipay2.PayeeInfo{
		Identity:     req.PayAccount,
		IdentityType: "ALIPAY_LOGON_ID",
		Name:         req.PayName,
	}
	fundTransUniTransfer := alipay2.FundTransUniTransfer{
		OutBizNo:    req.OrderNo,
		TransAmount: req.TransAmount,
		ProductCode: "TRANS_ACCOUNT_NO_PWD",
		BizScene:    "DIRECT_TRANSFER",
		OrderTitle:  req.OrderTitle,
		PayeeInfo:   userInfo,
		Remark:      req.Remark,
	}
	rest, err := payClient.FundTransUniTransfer(fundTransUniTransfer)
	if err != nil {
		err = fmt.Errorf("pkgName= %s, 支付宝转账失败，err:=%v", req.PkgName, err)
		logx.Errorf(err.Error())
		alipayFundTransUniTransferFailNum.CounterInc()
		res := response.MakeResult(code.CODE_ERROR, err.Error(), nil)
		return &res, nil
	}
	if rest.IsSuccess() == false {
		err = fmt.Errorf("调用转账失败 err: %s %s", rest.Content.SubCode, rest.Content.SubMsg)
		logx.Errorf(err.Error())
		alipayFundTransUniTransferFailNum.CounterInc()
		res := response.MakeResult(code.CODE_ERROR, err.Error(), nil)
		return &res, nil
	}

	//amount, _ := strconv.ParseFloat(req.TransAmount, 64)
	//orderInfo := &model.PmFundTransOrderTable{
	//	OrderSn:    req.OrderNo,
	//	AppPkgName: req.PkgName,
	//	Amount:     int(amount * 100),
	//	AliName:    req.PayName,
	//	AliAccount: req.PayAccount,
	//	PayAppId:   payCfg.AppID,
	//}
	//err = l.fundTransOrderModel.Create(orderInfo)
	//if err != nil {
	//	err = fmt.Errorf("fundTransOrderModel Create err: %v", err)
	//	util.CheckError(err.Error())
	//}

	res := response.MakeResult(code.CODE_OK, "", nil)
	return &res, nil
}
