package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/pay-gateway/common/client"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/common/exception"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type NotifyBytedanceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	payOrderModel        *model.PmPayOrderModel
	payConfigTiktokModel *model.PmPayConfigTiktokModel
}

func NewNotifyBytedanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyBytedanceLogic {
	return &NotifyBytedanceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,

		payOrderModel:        model.NewPmPayOrderModel(define.DbPayGateway),
		payConfigTiktokModel: model.NewPmPayConfigTiktokModel(define.DbPayGateway),
	}
}

func (l *NotifyBytedanceLogic) NotifyBytedance(req *types.ByteDanceReq) (resp *types.ByteDanceResp, err error) {
	msgData := new(client.TikTokNotifyMsgData)
	err = json.Unmarshal([]byte(req.Msg), msgData)
	if err != nil {
		util.CheckError("json unmarshal err :%v", err)
		return
	}

	payCfg, cfgErr := l.payConfigTiktokModel.GetOneByAppID(msgData.Appid)
	if cfgErr != nil {
		err = fmt.Errorf("appid = %s, 读取抖音支付配置失败，err:=%v", msgData.Appid, cfgErr)
		util.CheckError(err.Error())
		return
	}

	payServer := client.NewTikTokPay(*payCfg.TransClientConfig())
	cliReq := &client.ByteDanceReq{
		Timestamp:    req.Timestamp,
		Nonce:        req.Nonce,
		Msg:          req.Msg,
		Type:         req.Type,
		MsgSignature: req.MsgSignature,
	}
	order, err := payServer.Notify(cliReq)
	if err != nil {
		logx.Errorf("验签未通过，或者解密失败！err=%v", err)
		err = errors.New(`{"code": "FAIL","message": "验签未通过，或者解密失败"}`)
		resp = &types.ByteDanceResp{
			ErrNo:   400,
			ErrTips: "验签未通过，或者解密失败",
		}
		return resp, nil
	}

	//获取订单信息
	orderInfo, err := l.payOrderModel.GetOneByCode(order.CpOrderno)
	if err != nil {
		err = fmt.Errorf("获取订单失败！err=%v,order_code = %s", err, order.CpOrderno)
		util.CheckError(err.Error())
		return
	}
	if orderInfo.PayStatus != model.PmPayOrderTablePayStatusNo {
		notifyOrderHasDispose.CounterInc()
		err = fmt.Errorf("订单已处理")
		return
	}
	//修改数据库
	orderInfo.NotifyAmount = order.TotalAmount
	orderInfo.PayStatus = model.PmPayOrderTablePayStatusPaid
	orderInfo.PayType = model.PmPayOrderTablePayTypeTiktokPayEc
	err = l.payOrderModel.UpdateNotify(orderInfo)
	if err != nil {
		err = fmt.Errorf("orderSn = %s, UpdateNotify，err:=%v", orderInfo.OrderSn, err)
		util.CheckError(err.Error())
		return
	}

	//回调业务方接口
	go func() {
		defer exception.Recover()
		_, _ = util.HttpPost(orderInfo.NotifyUrl, req, 5*time.Second)
	}()

	resp = &types.ByteDanceResp{
		ErrNo:   0,
		ErrTips: "success",
	}

	return
}
