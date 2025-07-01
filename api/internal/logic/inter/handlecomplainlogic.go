package inter

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
	alipay2 "gitlab.muchcloud.com/consumer-project/alipay"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/common/response"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/types"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/client"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/code"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/util"
)

type HandleComplainLogic struct {
	logx.Logger
	ctx                  context.Context
	svcCtx               *svc.ServiceContext
	payConfigAlipayModel *model.PmPayConfigAlipayModel
}

func NewHandleComplainLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HandleComplainLogic {
	return &HandleComplainLogic{
		Logger:               logx.WithContext(ctx),
		ctx:                  ctx,
		svcCtx:               svcCtx,
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
	}
}

func (l *HandleComplainLogic) HandleComplain(req *types.ComplainReq) (resp *types.ResultResp, err error) {
	// 将 key 的验证调整到初始化阶段
	payCfg, cfgErr := l.payConfigAlipayModel.GetOneByAppID(req.AppId)
	if cfgErr != nil {
		err = fmt.Errorf("pkgName= %s, 读取支付宝配置失败，err:=%v", req.AppId, cfgErr)
		util.CheckError(err.Error())
		res := response.MakeResult(code.CODE_ERROR, err.Error(), nil)
		return &res, nil
	}
	payClient, err := client.GetAlipayClient(*payCfg.TransClientConfig())
	if err != nil {
		err = fmt.Errorf("pkgName= %s, 初始化客户端失败，err:=%v", req.AppId, err)
		util.CheckError(err.Error())
		res := response.MakeResult(code.CODE_ERROR, err.Error(), nil)
		return &res, nil
	}

	p := alipay2.ComplainList{
		GmtComplaintStart: req.StartTime,
		GmtComplaintEnd:   req.EndTime,
		PageSize:          2,
		CurrentPageNum:    1,
	}
	rest, err := payClient.GetComplainList(p)
	if err != nil {
		res := response.MakeResult(code.CODE_ERROR, err.Error(), nil)
		return &res, nil
	}
	data := map[string]interface{}{
		"totalNum": rest.AlipaySecurityRiskComplaintInfoBatchqueryResponse.TotalSize,
	}
	res := response.MakeResult(code.CODE_OK, "操作成功", data)
	return &res, nil
}
