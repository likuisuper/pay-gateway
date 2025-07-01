package notify

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/logic/notify"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/svc"
)

func NotifyDouyinHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := notify.NewNotifyDouyinLogic(r.Context(), svcCtx)
		resp, err := l.NotifyDouyin(r)
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.OkJson(w, resp)
		}
	}
}
