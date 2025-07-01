package notify

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/logic/notify"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/svc"
)

func NotifyAlipayHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//var req types.EmptyReq
		//if err := httpx.Parse(r, &req); err != nil {
		//	httpx.Error(w, err)
		//	return
		//}

		l := notify.NewNotifyAlipayLogic(r.Context(), svcCtx)
		_, err := l.NotifyAlipay(r, w)
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
