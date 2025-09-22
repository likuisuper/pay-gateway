package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	alipay2 "gitlab.muchcloud.com/consumer-project/alipay"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/clientMgr"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/define"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/types"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/common/utils"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/db/mysql/model"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AlipayH5PayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger

	orderModel *model.OrderModel
}

func NewAlipayH5PayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayH5PayLogic {
	return &AlipayH5PayLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),

		orderModel: model.NewOrderModel(define.DbPayGateway),
	}
}

// 支付宝h5支付，目前只支持普通商品，不支持订阅商品
func (l *AlipayH5PayLogic) AlipayH5Pay(in *pb.AlipayPageSignReq) (*pb.AlipayPageSignResp, error) {
	payClient, payAppId, notifyUrl, err := clientMgr.GetAlipayClientByAppPkgWithCache2(in.GetAppPkgName())
	if err != nil {
		l.Errorf("clientMgr.GetAlipayClientByAppPkgWithCache2 err: %v, pkg: %s", err, in.GetAppPkgName())
		return nil, err
	}

	var amount string
	var productType, intAmount int

	product := types.Product{}
	err = json.Unmarshal([]byte(in.ProductDesc), &product)
	if err != nil {
		parseProductDescErr.CounterInc()
		logx.Errorf("创建订单异常：商品信息错误 err = %s product = %s", err.Error(), in.ProductDesc)
		return nil, errors.New("商品信息错误")
	}

	productType = product.ProductType
	intAmount = int(product.Amount * 100)
	amount = fmt.Sprintf("%.2f", product.Amount)
	if intAmount <= 0 {
		logx.Errorf("创建订单异常：商品金额异常 product = %s", in.ProductDesc)
		return nil, errors.New("商品信息错误")
	}

	orderInfo := model.OrderTable{
		AppPkg:       in.AppPkgName,
		UserID:       int(in.UserId),
		OutTradeNo:   utils.GenerateOrderCode(l.svcCtx.Config.SnowFlake.MachineNo, l.svcCtx.Config.SnowFlake.WorkerNo),
		PayType:      model.PmPayOrderTablePayTypeAlipay,
		Status:       0,
		PayAppID:     payAppId,
		AppNotifyUrl: in.NotifyURL,
		Amount:       intAmount,
		ProductDesc:  in.ProductDesc,
		ProductType:  productType,
		ProductID:    int(in.ProductId),
	}

	trade := alipay2.Trade{
		ProductCode:     "QUICK_WAP_WAY",                                         // 产品码固定参数
		Subject:         in.Subject,                                              // 订单标题
		OutTradeNo:      orderInfo.OutTradeNo,                                    // 商户订单号
		TotalAmount:     amount,                                                  // 订单总金额, 单位元
		TimeExpire:      time.Now().Add(time.Hour).Format("2006-01-02 15:04:05"), // 默认1个小时超时
		NotifyURL:       notifyUrl,
		PassbackParams:  orderInfo.OutTradeNo,
		MerchantOrderNo: orderInfo.OutTradeNo,
	}

	appPay := alipay2.TradeWapPay{
		Trade: trade,
	}
	bytes, err := json.Marshal(appPay)
	logx.Slowf("请求支付宝h5支付的参数: %v, err: %v, notifyUrl: %s", string(bytes), err, notifyUrl)

	result, err := payClient.TradeWapPay(appPay)
	if err != nil {
		logx.Errorf("创建支付宝h5订单异常, 生成支付宝加签串失败, err: %s", err.Error())
		return nil, errors.New("创建h5订单异常")
	}

	err = l.orderModel.Create(&orderInfo)
	if err != nil {
		logx.Errorf("创建订单异常：创建订单表失败， err = %s", err.Error())
		return nil, errors.New("创建h5订单异常")
	}

	return &pb.AlipayPageSignResp{
		URL:        result.String(), // 返回完整的url地址
		OutTradeNo: orderInfo.OutTradeNo,
	}, nil
}
