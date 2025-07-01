package notify

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/logic/notify"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/svc"
)

func NotifyAlipayNewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//var req types.EmptyReq
		//if err := httpx.Parse(r, &req); err != nil {
		//	httpx.Error(w, err)
		//	return
		//}

		l := notify.NewNotifyAlipayNewLogic(r.Context(), svcCtx)
		_, err := l.NotifyAlipayNew(r, w)
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
