package logic

import (
	"context"
	"gitee.com/zhuyunkj/pay-gateway/common/define"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"gitee.com/zhuyunkj/pay-gateway/rpc/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/rpc/pb/pb"
	"github.com/zeromicro/go-zero/core/logx"
	"reflect"
	"testing"
)

func TestOrderStatusLogic_OrderStatus(t *testing.T) {
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
		in *pb.OrderStatusReq
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantResp *pb.OrderStatusResp
		wantErr  bool
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
				in: &pb.OrderStatusReq{
					OrderSn:    "1210526885193322496",
					PayType:    pb.PayType_DouyinGeneralTrade,
					AppPkgName: "com.douyin.yunjutn",
				},
			},
			wantResp: nil,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &OrderStatusLogic{
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
			gotResp, err := l.OrderStatus(tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("OrderStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResp, tt.wantResp) {
				t.Errorf("OrderStatus() gotResp = %v, want %v", gotResp, tt.wantResp)
			}
		})
	}
}
