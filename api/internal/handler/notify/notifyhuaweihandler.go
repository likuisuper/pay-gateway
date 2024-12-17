package notify

import (
	"net/http"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/logic/notify"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func NotifyHuaweiHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.HuaweiReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := notify.NewNotifyHuaweiLogic(r.Context(), svcCtx)
		l.NotifyHuawei(&req)
		// 通过HTTP状态码来标识华为应用内支付服务器通知您的应用服务器是否发送成功：
		// 如果通知发送成功，则发送HTTP 200，不需要返回响应体。
		// 如果通知发送失败，则通过发送HTTP 40X或者HTTP 50X，告知华为应用内支付服务器进行重试，华为应用内支付服务器会在一段时间内重试多次。
		httpx.OkJson(w, nil)
	}
}
