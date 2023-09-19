package notify

import (
	"context"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"net/url"

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
	// todo: add your logic here and delete this line

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

//formdata数据转成map
func (l *NotifyAlipaySignLogic) transFormDataToMap(formData string) (dataMap map[string]interface{}) {
	dataMap = make(map[string]interface{}, 0)
	values, _ := url.ParseQuery(formData)
	for key, datas := range values {
		if len(datas) > 0 {
			dataMap[key] = datas[0]
		}
	}
	return
}
