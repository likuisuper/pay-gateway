package notify

import (
	"context"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/pay-gateway/db/mysql/model"
	"github.com/zeromicro/go-zero/core/logx"
	"net/http"
	"reflect"
	"testing"
)

func TestNotifyDouyinLogic_NotifyDouyin(t *testing.T) {
	type fields struct {
		Logger               logx.Logger
		ctx                  context.Context
		svcCtx               *svc.ServiceContext
		payOrderModel        *model.PmPayOrderModel
		payConfigTiktokModel *model.PmPayConfigTiktokModel
		refundOrderModel     *model.PmRefundOrderModel
	}
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantResp *types.DouyinResp
		wantErr  bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &NotifyDouyinLogic{
				Logger:               tt.fields.Logger,
				ctx:                  tt.fields.ctx,
				svcCtx:               tt.fields.svcCtx,
				payOrderModel:        tt.fields.payOrderModel,
				payConfigTiktokModel: tt.fields.payConfigTiktokModel,
				refundOrderModel:     tt.fields.refundOrderModel,
			}
			gotResp, err := l.NotifyDouyin(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NotifyDouyin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResp, tt.wantResp) {
				t.Errorf("NotifyDouyin() gotResp = %v, want %v", gotResp, tt.wantResp)
			}
		})
	}
}
