package inter

import (
	"net/http"

	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/logic/inter"

	"github.com/zeromicro/go-zero/rest/httpx"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/svc"
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/types"
)

func CrtUploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CrtUploadReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := inter.NewCrtUploadLogic(r.Context(), svcCtx)
		resp, err := l.CrtUpload(&req, r)
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.OkJson(w, resp)
		}
	}
}
