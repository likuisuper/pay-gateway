package notify

import (
	"context"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	jsoniter "github.com/json-iterator/go"
	"io/ioutil"
	"net/http"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyWechatRefundOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	orderModel           *model.OrderModel
	refundModel          *model.RefundModel
	payConfigWechatModel *model.PmPayConfigWechatModel
}

func NewNotifyWechatRefundOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyWechatRefundOrderLogic {
	return &NotifyWechatRefundOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		orderModel:           model.NewOrderModel(define.DbPayGateway),
		refundModel:          model.NewRefundModel(define.DbPayGateway),
		payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
	}
}

func (l *NotifyWechatRefundOrderLogic) NotifyWechatRefundOrder(req *types.WechatRefundReq,r *http.Request) (resp *types.WeChatResp, err error) {

	header := r.Header
	logx.Slow("微信退款回调请求头", header)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logx.Errorf("获取请求体错误！err:=%v", err)
		return nil, err
	}
	logx.Slow("NotifyWechatRefundOrder:", string(body))
	appId := req.Appid
	logx.Slowf("WechatNotifyRefund AppId: %s", appId)
	payCfg, err := l.payConfigWechatModel.GetOneByAppID(appId)
	if err != nil {
		err = fmt.Errorf("pkgName= %s, 读取微信支付配置失败，err:=%v", "all", err)
		util.CheckError(err.Error())
		return nil,err
	}
	var wxCli *client.WeChatCommPay
	wxCli = client.NewWeChatCommPay(*payCfg.TransClientConfig())
	transaction, err := wxCli.RefundNotify(r)
	if err != nil {
		err = fmt.Errorf("解析及验证内容失败！err=%v ", err)
		logx.Errorf(err.Error())
		return nil,err
	}
	jsonStr, _ := jsoniter.MarshalToString(transaction)
	logx.Slowf("wechat支付回调异常: %s", jsonStr)
	return nil,err
}
