package notify

import (
	"context"
	"net/http"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyAlipaySignLogic struct {
	logx.Logger
	ctx        context.Context
	svcCtx     *svc.ServiceContext
	orderModel *model.OrderModel
}

func NewNotifyAlipaySignLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyAlipaySignLogic {
	return &NotifyAlipaySignLogic{
		Logger:     logx.WithContext(ctx),
		ctx:        ctx,
		svcCtx:     svcCtx,
		orderModel: model.NewOrderModel(define.DbPayGateway),
	}
}

func (l *NotifyAlipaySignLogic) NotifyAlipaySign(r *http.Request, w http.ResponseWriter) (resp *types.EmptyReq, err error) {
	err = r.ParseForm()
	if err != nil {
		logx.Errorf("NotifyAlipay err: %v", err)
		notifyAlipayErrNum.CounterInc()
		return
	}

	bodyData := r.Form.Encode()
	logx.Slowf("NotifyAlipay form %s", bodyData)

	httpx.OkJson(w, "success")

	return
}
