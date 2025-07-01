package inter

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/logic/inter"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/types"
)

func HandleComplainHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ComplainReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := inter.NewHandleComplainLogic(r.Context(), svcCtx)
		resp, err := l.HandleComplain(&req)
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.OkJson(w, resp)
		}
	}
}
