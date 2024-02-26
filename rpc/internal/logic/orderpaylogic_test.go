package logic

import (
	"context"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/config"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	"gitee.com/zhuyunkj/zhuyun-core/nacos"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"reflect"
	"testing"
)

var svcCtx *svc.ServiceContext

func init() {
	svcCtx = initLogic("./../../etc/nacos.yaml")
}

func initLogic(nacosConfigFile string) (ctx *svc.ServiceContext) {
	var c config.Config
	//conf.MustLoad(*configFile, &c)
	//从nacos获取配置
	var nacosConfig nacos.Config
	conf.MustLoad(nacosConfigFile, &nacosConfig)
	nacosClient, nacosErr := nacos.InitNacosClient(nacosConfig)
	if nacosErr != nil {
		logx.Errorf("初始化nacos客户端失败: " + nacosErr.Error())
		return
	}
	err := nacosClient.GetConfig(nacosConfig.DataId, nacosConfig.GroupId, &c)
	defer nacosClient.CloseClient()
	if err != nil {
		logx.Errorf("获取配置失败：" + err.Error())
		return
	}

	// 初始化数据库
	db.DBInit(c.Mysql, c.RedisConfig)
	ctx = svc.NewServiceContext(c)
	return ctx

}

func TestOrderPayLogic_OrderPay(t *testing.T) {
	type fields struct {
		ctx                  context.Context
		svcCtx               *svc.ServiceContext
		Logger               logx.Logger
		payOrderModel        *model.PmPayOrderModel
		appConfigModel       *model.PmAppConfigModel
		payConfigAlipayModel *model.PmPayConfigAlipayModel
		payConfigTiktokModel *model.PmPayConfigTiktokModel
		payConfigWechatModel *model.PmPayConfigWechatModel
		payConfigKsModel     *model.PmPayConfigKsModel
	}
	type args struct {
		in *pb.OrderPayReq
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantOut *pb.OrderPayResp
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				ctx:                  context.Background(),
				svcCtx:               svcCtx,
				Logger:               logx.WithContext(context.Background()),
				payOrderModel:        model.NewPmPayOrderModel(define.DbPayGateway),
				appConfigModel:       model.NewPmAppConfigModel(define.DbPayGateway),
				payConfigAlipayModel: model.NewPmPayConfigAlipayModel(define.DbPayGateway),
				payConfigTiktokModel: model.NewPmPayConfigTiktokModel(define.DbPayGateway),
				payConfigWechatModel: model.NewPmPayConfigWechatModel(define.DbPayGateway),
				payConfigKsModel:     model.NewPmPayConfigKsModel(define.DbPayGateway),
			},
			args: args{
				in: &pb.OrderPayReq{
					AppPkgName: "testPackageName2",
					OrderSn:    "testOrderId2",
					Amount:     1,
					Subject:    "testPayLocal",
					PayType:    pb.PayType_DouyinGeneralTrade,
					NotifyURL:  "testNotifyUrl",
					ReturnURL:  "testReturnUrl",
					WxOpenID:   "",
					DouyinGeneralTradeReq: &pb.DouyinGeneralTradeReq{
						SkuId:       "testSkuId1",
						Price:       1,
						Quantity:    1,
						Title:       "testSku",
						ImageList:   []string{"https://picnew12.photophoto.cn/20180606/618kuanghuanjiejinbishiliangtu-32308636_1.jpg"},
						Type:        pb.DouyinGeneralTradeReq_ContentRecharge,
						EntrySchema: nil,
						OrderEntrySchema: &pb.Schema{
							Path:   "pages/homePage/index",
							Params: "",
						},
						LimitPayWayList: nil,
					},
				},
			},
			wantOut: nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &OrderPayLogic{
				ctx:                  tt.fields.ctx,
				svcCtx:               tt.fields.svcCtx,
				Logger:               tt.fields.Logger,
				payOrderModel:        tt.fields.payOrderModel,
				appConfigModel:       tt.fields.appConfigModel,
				payConfigAlipayModel: tt.fields.payConfigAlipayModel,
				payConfigTiktokModel: tt.fields.payConfigTiktokModel,
				payConfigWechatModel: tt.fields.payConfigWechatModel,
				payConfigKsModel:     tt.fields.payConfigKsModel,
			}
			gotOut, err := l.OrderPay(tt.args.in)
			t.Log(err, gotOut)
			if (err != nil) != tt.wantErr {
				t.Errorf("OrderPay() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotOut, tt.wantOut) {
				t.Errorf("OrderPay() gotOut = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}
