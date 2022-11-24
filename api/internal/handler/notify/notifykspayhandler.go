package notify

import (
	"net/http"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/logic/notify"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func NotifyKspayHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//var req types.EmptyReq
		//if err := httpx.Parse(r, &req); err != nil {
		//	httpx.Error(w, err)
		//	return
		//}

		l := notify.NewNotifyKspayLogic(r.Context(), svcCtx)
		_, err := l.NotifyKspay(r, w)
		if err != nil {
			httpx.Error(w, err)
		} else {
			//httpx.OkJson(w, resp)
		}
	}
}
