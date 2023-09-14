package notify

import (
	"context"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"net/http"
	"net/url"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

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

	agreementNo := r.Form.Get("agreement_no")
	externalAgreementNo := r.Form.Get("external_agreement_no")
	outTradeNo := r.Form.Get("out_trade_no")
	userid := r.Form.Get("user_id")

	if agreementNo == "" || externalAgreementNo == "" || userid == "" || outTradeNo == "" {
		logx.Errorf("签约回调参数异常, %s", bodyData)
		return
	}

	if err != nil {
		logx.Errorf(err.Error())
		notifyAlipayErrNum.CounterInc()
		return
	}

	order, err := l.orderModel.GetOneByOutTradeNo(outTradeNo)
	if err != nil {
		logx.Errorf("获取订单详情失败: %v", err.Error())
		notifyAlipayErrNum.CounterInc()
		return
	}

	order.AgreementNo = agreementNo
	order.ExternalAgreementNo = externalAgreementNo
	err = l.orderModel.UpdateNotify(order)
	if err != nil {
		logx.Errorf("更新订单详情失败: %v", err.Error())
		notifyAlipayErrNum.CounterInc()
		return
	}

	go func() {
		defer exception.Recover()
		dataMap := l.transFormDataToMap(bodyData)
		_, _ = util.HttpPost(order.AppNotifyUrl, dataMap, 5*time.Second)
	}()
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
