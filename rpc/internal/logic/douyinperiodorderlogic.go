package logic

import (
	"context"
	"time"

	douyin "gitee.com/zhuyunkj/pay-gateway/common/client/douyinGeneralTrade"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"

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
	if in.GetAction() == pb.DouyinPeriodOrderReqAction_DyPeriodActionQuery {
		// 查询签约情况
		return l.querySignOrder(in)
	}

	if in.GetAction() == pb.DouyinPeriodOrderReqAction_DyPeriodActionCancel {
		// 用户发起解约
		return l.terminateSign(in)
	}

	resp := pb.DouyinPeriodOrderResp{
		UserId: in.GetUserId(),
		IsSign: 0,
		Msg:    "不支持的操作类型",
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
	signResult, err := l.dyClient.QuerySignOrder(clientToken, periodModel.ThirdSignOrderNo)
	if err != nil || signResult == nil {
		l.Errorw("QuerySignOrder fail", logx.Field("err", err), logx.Field("authOrderId", periodModel.ThirdSignOrderNo))
		return &resp, nil
	}

	if signResult.ErrNo == 0 && signResult.UserSignData.Status == douyin.Dy_Sign_Status_Query_SERVING {
		// 已签约
		nextDecuctionTimeStr := time.Unix(signResult.UserSignData.SignTime/1000, 0).AddDate(0, 1, 0).Format("2006-01-02 15:04:05")
		updateData := map[string]interface{}{
			"pay_status":          model.Sign_Status_Success,
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
	}

	return &resp, nil
}

// 用户发起解约
func (l *DouyinPeriodOrderLogic) terminateSign(in *pb.DouyinPeriodOrderReq) (*pb.DouyinPeriodOrderResp, error) {
	resp := pb.DouyinPeriodOrderResp{
		UserId: in.GetUserId(),
		IsSign: 0,
		Msg:    "解约失败",
	}

	// 查询
	periodModel, err := l.payDyPeriodOrderModel.GetSignedByUserIdAndPkg(int(in.GetUserId()), in.GetPkg())
	if err != nil || periodModel == nil || periodModel.ID < 1 {
		l.Slowf("GetSignedByUserIdAndPkg error: %v, pkg: %s, userId :%d", err, in.GetPkg(), in.GetUserId())

		resp.Msg = "获取签约订单失败, err: " + err.Error()
		return &resp, nil
	}

	resp.IsSign = 1
	resp.Msg = "解约成功"

	return &resp, nil
}
