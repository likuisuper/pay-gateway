package notify

import (
	"context"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"net/http"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	notifyAlipayErrNum = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "notifyAlipayErrNum", nil, "支付宝回调失败", nil})}
)

type NotifyAlipayLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	payOrderModel        *model.PmPayOrderModel
	payConfigAlipayModel *model.PmPayConfigAlipayModel
}

func NewNotifyAlipayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyAlipayLogic {
	return &NotifyAlipayLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,

		payOrderModel:        model.NewPmPayOrderModel(define.DbPayGateway),
		payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
	}
}

func (l *NotifyAlipayLogic) NotifyAlipay(r *http.Request, w http.ResponseWriter) (resp *types.EmptyReq, err error) {
	//data, err := ioutil.ReadAll(r.Body)
	//if err != nil {
	//	logx.Errorf("NotifyAlipay err: %v", err)
	//	notifyAlipayErrNum.CounterInc()
	//	return
	//}
	//logx.Slowf("NotifyAlipay body: %s", string(data))

	//appId := jsoniter.Get(data, "app_id").ToString()
	//payCfg, err := l.payConfigAlipayModel.GetOneByAppID(appId)
	//if err != nil {
	//	err = fmt.Errorf("pkgName= %s, 读取支付配置失败，err:=%v", "all", err)
	//	util.CheckError(err.Error())
	//	return
	//}
	//
	//payClient, err := client.GetAlipayClient(*payCfg.TransClientConfig())
	//if err != nil {
	//	util.CheckError("pkgName= %s, 初使化支付错误，err:=%v", "all", err)
	//	return
	//}
	//payClient.VerifySign()

	err = r.ParseForm()
	if err != nil {
		logx.Errorf("NotifyAlipay err: %v", err)
		notifyAlipayErrNum.CounterInc()
		return
	}
	appId := r.Form.Get("app_id")
	logx.Slowf(appId)

	return
}
