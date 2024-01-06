package notify

import (
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"net/http"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/logic/notify"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func NotifyWechatH5OrderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.WechatNotifyH5Req
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := notify.NewNotifyWechatH5OrderLogic(r.Context(), svcCtx)
		resp, err := l.NotifyWechatH5Order(&req, r)
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.OkJson(w, resp)
		}
	}
}
