package logic

import (
	"context"
	"time"

	douyin "gitlab.muchcloud.com/consumer-project/pay-gateway/common/client/douyinGeneralTrade"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DouyinPeriodOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	payDyPeriodOrderModel *model.PmDyPeriodOrderModel
	dyClient              douyin.PayClient
}

func NewDouyinPeriodOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DouyinPeriodOrderLogic {
	return &DouyinPeriodOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		payDyPeriodOrderModel: model.NewPmDyPeriodOrderModel(define.DbPayGateway),
		dyClient:              douyin.PayClient{}, // 由于用不到支付相关的配置 直接初始化一个空的就是
	}
}

// 抖音周期代扣相关查询和修改
func (l *DouyinPeriodOrderLogic) DouyinPeriodOrder(in *pb.DouyinPeriodOrderReq) (*pb.DouyinPeriodOrderResp, error) {
	// 记录日志
	l.Sloww("DouyinPeriodOrder", logx.Field("in", in), logx.Field("action", in.GetAction()))

	if in.GetAction() == pb.DouyinPeriodOrderReqAction_DyPeriodActionQuery {
		// 查询单个签约情况
		return l.querySignOrder(in)
	}

	if in.GetAction() == pb.DouyinPeriodOrderReqAction_DyPeriodActionCancel {
		// 用户发起解约
		return l.terminateSign(in)
	}

	if in.GetAction() == pb.DouyinPeriodOrderReqAction_DyPeriodActionGetPayList {
		// 获取今日可以扣款的签约订单信息
		return l.getSignedPayList()
	}

	if in.GetAction() == pb.DouyinPeriodOrderReqAction_DyPeriodActionUpdateNextTime {
		// 更新下一期抖音签约代扣扣款时间
		return l.updateNexDecuctionTime(in)
	}

	resp := pb.DouyinPeriodOrderResp{
		UserId: in.GetUserId(),
		IsSign: 0,
		Msg:    "不支持的操作类型",
	}
	return &resp, nil
}

// 更新下一期抖音签约代扣扣款时间
func (l *DouyinPeriodOrderLogic) updateNexDecuctionTime(in *pb.DouyinPeriodOrderReq) (*pb.DouyinPeriodOrderResp, error) {
	resp := pb.DouyinPeriodOrderResp{}

	mdl := model.NewPmDyPeriodOrderModel(define.DbPayGateway)
	tbl, err := mdl.GetById(in.GetPmDyPeriodOrderId())
	if err != nil {
		l.Errorf("GetById failed: %v", err)
		resp.Msg = "NewPmDyPeriodOrderModel GetById err : " + err.Error()
		return &resp, nil
	}

	err = mdl.UpdateSomeData(int(in.GetPmDyPeriodOrderId()), map[string]interface{}{
		"next_decuction_time": tbl.NextDecuctionTime.AddDate(0, 1, 0).Format("2006-01-02 15:04:05"),
	})
	if err != nil {
		l.Errorf("UpdateSomeData failed: %v", err)
		resp.Msg = "NewPmDyPeriodOrderModel UpdateSomeData err : " + err.Error()
		return &resp, nil
	}

	resp.Msg = "success"
	return &resp, nil
}

func (l *DouyinPeriodOrderLogic) getSignedPayList() (*pb.DouyinPeriodOrderResp, error) {
	resp := pb.DouyinPeriodOrderResp{}

	list, err := model.NewPmDyPeriodOrderModel(define.DbPayGateway).GetSignedPayList()
	if err != nil {
		l.Errorf("getSignedPayList failed: %v", err)
		return &resp, nil
	}

	payConfigTiktokModel := model.NewPmPayConfigTiktokModel(define.DbPayGateway)

	for _, v := range list {
		merchantUid := ""
		appConfig, err := payConfigTiktokModel.GetOneByAppID(v.PayAppId)
		if err == nil && appConfig != nil {
			merchantUid = appConfig.SignPayMerchantUid
		}

		resp.SignedList = append(resp.SignedList, &pb.DySignedOrderInfo{
			OrderSn:           v.OrderSn,
			AppPkg:            v.AppPkgName,
			UserId:            int64(v.UserId),
			NextDecuctionTime: v.NextDecuctionTime.Format("2006-01-02 15:04:05"),
			DySignNo:          v.ThirdSignOrderNo,
			NotifyUrl:         v.NotifyUrl,
			MerchantUid:       merchantUid,
			PkId:              int64(v.ID),
		})
	}

	return &resp, nil
}

// 查询签约情况
func (l *DouyinPeriodOrderLogic) querySignOrder(in *pb.DouyinPeriodOrderReq) (*pb.DouyinPeriodOrderResp, error) {
	resp := pb.DouyinPeriodOrderResp{
		UserId: in.GetUserId(),
		IsSign: 0,
		Msg:    "未签约",
	}

	// 查询最新一单状态 目前一个用户在每个扣款周期内只能发起一笔代扣单。要是同一周期内已有处理中或成功的代扣单，就无法再发起新订单
	// 查询
	periodModel, _ := l.payDyPeriodOrderModel.GetSignedByUserIdAndPkg(int(in.GetUserId()), in.GetPkg(), model.Sign_Status_Success)
	if periodModel != nil && periodModel.ID > 0 {
		// 已签约
		resp.IsSign = 1
		resp.Msg = "已签约"
		resp.NextDecuctionTime = periodModel.NextDecuctionTime.Format("2006-01-02 15:04:05")
		resp.DeductionAmount = int64(periodModel.Amount) // 单位分
	}

	periodModel, err := l.payDyPeriodOrderModel.GetSignedByUserIdAndPkg(int(in.GetUserId()), in.GetPkg(), model.Sign_Status_Wait)
	if err != nil && periodModel == nil || periodModel.ID < 1 {
		// 查询失败
		l.Errorf("querySignOrder failed: %v, userId: %d, pkg: %s ", err, in.GetUserId(), in.GetPkg())
		return &resp, nil
	}

	clientToken, err := l.svcCtx.BaseAppConfigServerApi.GetDyClientToken(l.ctx, periodModel.PayAppId)
	if err != nil || clientToken == "" {
		l.Errorw("get douyin client token fail", logx.Field("err", err), logx.Field("appId", periodModel.PayAppId))
		return &resp, nil
	}

	// 再查询一下抖音服务确认是否签约
	signResult, err := l.dyClient.QuerySignOrder(clientToken, periodModel.SignNo)
	if err != nil || signResult == nil {
		l.Errorw("QuerySignOrder fail", logx.Field("err", err), logx.Field("authOrderId", periodModel.SignNo))
		return &resp, nil
	}

	if signResult.ErrNo != 0 {
		l.Errorw("dyClient.QuerySignOrder failed", logx.Field("signResult", signResult))
		return &resp, nil
	}

	if signResult.UserSignData.Status == douyin.Dy_Sign_Status_Query_SERVING {
		// 已签约
		nextDecuctionTimeStr := time.Unix(signResult.UserSignData.SignTime/1000, 0).AddDate(0, 1, 0).Format("2006-01-02 15:04:05")
		updateData := map[string]interface{}{
			"sign_status":         model.Sign_Status_Success,
			"sign_date":           time.Unix(signResult.UserSignData.SignTime/1000, 0).Format("2006-01-02 15:04:05"), // 签约时间
			"next_decuction_time": nextDecuctionTimeStr,                                                              // 下次扣款时间
			"third_sign_order_no": signResult.UserSignData.OutAuthOrderNo,                                            // 抖音平台返回的签约单号
		}
		// 修改数据库
		err = l.payDyPeriodOrderModel.UpdateSomeData(periodModel.ID, updateData)
		// 记录日志
		l.Sloww("payDyPeriodOrderModel.UpdateSomeData", logx.Field("id", periodModel.ID), logx.Field("updateData", updateData), logx.Field("err", err))

		resp.IsSign = 1
		resp.Msg = "已签约"
		resp.NextDecuctionTime = nextDecuctionTimeStr
		resp.DeductionAmount = int64(periodModel.Amount) // 单位分
	} else if signResult.UserSignData.Status == douyin.Dy_Sign_Status_Query_CANCEL || signResult.UserSignData.Status == douyin.Dy_Sign_Status_Query_DONE {
		signStatus := model.Sign_Status_Cancel
		if signResult.UserSignData.Status == douyin.Dy_Sign_Status_Query_DONE {
			signStatus = model.Sign_Status_Done
		}

		// 修改数据库 更新签约状态
		updateData := map[string]interface{}{
			"sign_status": signStatus,
		}
		err = l.payDyPeriodOrderModel.UpdateSomeData(periodModel.ID, updateData)
		// 记录日志
		l.Sloww("payDyPeriodOrderModel.UpdateSomeData", logx.Field("id", periodModel.ID), logx.Field("updateData", updateData), logx.Field("err", err))
	}

	return &resp, nil
}

// 用户发起解约
func (l *DouyinPeriodOrderLogic) terminateSign(in *pb.DouyinPeriodOrderReq) (*pb.DouyinPeriodOrderResp, error) {
	resp := pb.DouyinPeriodOrderResp{
		UserId:          in.GetUserId(),
		IsUnsignSuccess: false,
		Msg:             "你未签约",
	}

	// 查询
	periodModel, _ := l.payDyPeriodOrderModel.GetSignedByUserIdAndPkg(int(in.GetUserId()), in.GetPkg(), model.Sign_Status_Success)
	if periodModel == nil || periodModel.ID < 1 {
		return &resp, nil
	}

	clientToken, err := l.svcCtx.BaseAppConfigServerApi.GetDyClientToken(l.ctx, periodModel.PayAppId)
	if err != nil || clientToken == "" {
		l.Errorw("get douyin client token fail", logx.Field("err", err), logx.Field("appId", periodModel.PayAppId))
		return &resp, nil
	}

	// 再查询一下抖音服务确认是否签约
	signResult, err := l.dyClient.QuerySignOrder(clientToken, periodModel.SignNo)
	if err != nil || signResult == nil {
		l.Errorw("QuerySignOrder fail", logx.Field("err", err), logx.Field("authOrderId", periodModel.SignNo))
		return &resp, nil
	}

	if signResult.ErrNo != 0 {
		// 记录一下日志就可以了
		l.Errorw("dyClient.QuerySignOrder failed", logx.Field("signResult", signResult))
	}

	// "err_no":20000,"err_msg":"订单不存在"
	if signResult.ErrNo == 20000 || (len(signResult.UserSignData.Status) > 0 && signResult.UserSignData.Status != douyin.Dy_Sign_Status_Query_SERVING) {
		// 修改数据库
		updateData := map[string]interface{}{
			"sign_status": model.Sign_Status_Cancel,
			"unsign_date": time.Now().Format("2006-01-02 15:04:05"),
		}
		err = l.payDyPeriodOrderModel.UpdateSomeData(periodModel.ID, updateData)

		// 记录日志
		l.Sloww("payDyPeriodOrderModel.UpdateSomeData", logx.Field("id", periodModel.ID), logx.Field("updateData", updateData), logx.Field("err", err))

		resp.IsUnsignSuccess = true
		resp.Msg = "解约成功"
		return &resp, nil
	}

	// 已签约 开始解约
	unsignResult, err := l.dyClient.TerminateSign(clientToken, periodModel.ThirdSignOrderNo)
	if err != nil || unsignResult == nil {
		l.Errorw("TerminateSign fail", logx.Field("err", err), logx.Field("authOrderId", periodModel.ThirdSignOrderNo))
		resp.Msg = "解约失败请稍后再试"
		return &resp, nil
	}

	// 修改数据库
	updateData := map[string]interface{}{
		"sign_status": model.Sign_Status_Cancel,
		"unsign_date": time.Now().Format("2006-01-02 15:04:05"),
	}
	err = l.payDyPeriodOrderModel.UpdateSomeData(periodModel.ID, updateData)
	// 记录日志
	l.Sloww("payDyPeriodOrderModel.UpdateSomeData", logx.Field("id", periodModel.ID), logx.Field("updateData", updateData), logx.Field("err", err))

	resp.IsUnsignSuccess = true
	resp.Msg = "解约成功"

	return &resp, nil
}
