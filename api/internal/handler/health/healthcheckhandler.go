package health

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/logic/health"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/types"
)

func HealthCheckHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.HeaderReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := health.NewHealthCheckLogic(r.Context(), svcCtx)
		resp, err := l.HealthCheck(&req)
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.OkJson(w, resp)
		}
	}
}
