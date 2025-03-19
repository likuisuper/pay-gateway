package logic

import (
	"context"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db"
	"gorm.io/gorm"
	"strings"

	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AutoPkgAddLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	PayGateWayDB *gorm.DB
}

func NewAutoPkgAddLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AutoPkgAddLogic {
	return &AutoPkgAddLogic{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		PayGateWayDB: db.WithDBContext(define.DbPayGateway),
	}
}

// AutoPkgAdd 自动创建应用
func (l *AutoPkgAddLogic) AutoPkgAdd(in *pb.AutoPkgAddReq) (*pb.AutoPkgAddResp, error) {
	if len(in.SqlList) == 0 {
		return &pb.AutoPkgAddResp{
			Status:  1,
			FailMsg: "sqlList 不能为空",
		}, nil
	}
	var failAppIds, failMsg []string
	for _, v := range in.SqlList {
		err := l.PayGateWayDB.Exec(v).Error
		if err != nil {
			failAppIds = append(failAppIds, v)
			failMsg = append(failMsg, fmt.Sprintf("%s失败原因:%s", v, err))
			logx.Errorf("AutoPkgAdd fail, sql= %s, error= %v", v, err)
			continue
		}
	}
	if len(failAppIds) > 0 {
		return &pb.AutoPkgAddResp{
			Status:  1,
			FailMsg: strings.Join(failMsg, "m"),
			AppIds:  failAppIds,
		}, nil
	}
	return &pb.AutoPkgAddResp{}, nil
}
