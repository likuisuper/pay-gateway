package crontab

import (
	"net/http"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/logic/crontab"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func SupplementaryOrdersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SupplementaryOrdersReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := crontab.NewSupplementaryOrdersLogic(r.Context(), svcCtx)
		resp, err := l.SupplementaryOrders(&req)
		if err != nil {
			resp = &types.SupplementaryOrdersResp{
				ErrNo:   -1,
				ErrTips: err.Error(),
			}
		}
		httpx.OkJson(w, resp)
	}
}
