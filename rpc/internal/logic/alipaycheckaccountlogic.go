package logic

import (
	"context"
	"errors"
	"fmt"

	"github.com/zeromicro/go-zero/core/service"
	alipay2 "gitlab.muchcloud.com/consumer-project/alipay"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/client"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/code"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	kv_m "gitlab.muchcloud.com/consumer-project/zhuyun-core/kv_monitor"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/util"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	aliPayAccountVerifyFailNum           = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "aliPayAccountVerifyFailNum", nil, "账号转账验证失败数量", nil})}
	aliPayAccountVerifyTranferSuccessNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "aliPayAccountVerifyTranferSuccessNum", nil, "账号验证转账金额 0.01 成功数量", nil})}
)

type AlipayCheckAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	appConfigModel       *model.PmAppConfigModel
	payConfigAlipayModel *model.PmPayConfigAlipayModel
}

func NewAlipayCheckAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayCheckAccountLogic {
	return &AlipayCheckAccountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
	}
}

// 支付宝转出账号校验
func (l *AlipayCheckAccountLogic) AlipayCheckAccount(in *pb.AlipayCheckAccountReq) (res *pb.AlipayCheckAccountResp, err error) {
	res = new(pb.AlipayCheckAccountResp)

	//测试环境不支付
	if l.svcCtx.Config.Mode != service.ProMode {
		res.Status = code.ALI_PAY_SUCCESS
		res.Desc = "测试不验证"
		return res, nil
	}

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

	payData := alipay2.FundTransToAccountTransfer{
		OutBizNo:      util.MakeOrderNo(int(in.UserId)),
		PayeeType:     "ALIPAY_LOGONID",
		PayeeAccount:  in.Account,
		Amount:        "0.01",
		PayeeRealName: in.Name,
		Remark:        "活动验证转账",
	}
	rest, err := payClient.FundTransToAccountTransfer(payData)
	if err != nil {
		aliPayAccountVerifyFailNum.CounterInc()
		res.Status = code.ALI_PAY_FAIL
		res.Desc = err.Error()
		util.CheckError("pkgname= %s, 账号验证转账 0.01 错误，err:=%v", in.GetAppPkgName(), err)
		return res, nil
	}

	logx.Infof("pkgname= %s, 调用请求支付账号验证:%v", in.GetAppPkgName(), rest.Content)
	if rest.IsSuccess() {
		aliPayAccountVerifyTranferSuccessNum.CounterInc()
		res.Status = code.ALI_PAY_SUCCESS
		logx.Infof("pkgname= %s, 账号验证转账金额 0.01 成功, account= %s, name= %s, userid= %d ", in.GetAppPkgName(), in.Account, in.Name, in.UserId)
		//支付成功
	} else {
		if rest.Content.SubCode == "EXCEED_LIMIT_SM_MIN_AMOUNT" {
			res.Status = code.ALI_PAY_SUCCESS
		} else {
			aliPayAccountVerifyFailNum.CounterInc()
			res.Status = code.ALI_PAY_FAIL
			res.Desc = rest.Content.SubMsg
		}
	}
	return
}
