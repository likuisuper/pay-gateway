package notify

import (
	"net/http"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/logic/notify"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// NotifyRefundWechatMiniHandler 小程序业务-微信商户退款回调通知
func NotifyRefundWechatMiniHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.WechatMiniRefundReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := notify.NewNotifyRefundWechatMiniLogic(r.Context(), svcCtx)
		resp, err := l.NotifyRefundWechatMini(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
