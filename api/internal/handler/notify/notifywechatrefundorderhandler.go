package notify

import (
	"net/http"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/logic/notify"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func NotifyWechatRefundOrderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.WechatRefundReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}
		l := notify.NewNotifyWechatRefundOrderLogic(r.Context(), svcCtx)
		resp, err := l.NotifyWechatRefundOrder(&req,r)
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.OkJson(w, resp)
		}
	}
}
