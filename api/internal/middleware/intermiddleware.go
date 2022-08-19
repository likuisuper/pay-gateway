package middleware

import (
	"net/http"
	"strings"
)

type InterMiddleware struct {
}

func NewInterMiddleware() *InterMiddleware {
	return &InterMiddleware{}
}

func (m *InterMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//allowCidr := "172.30.0.0/16"
		//ip := r.Header.Get("X-Forwarded-For")
		//allow := m.isBelong(ip, allowCidr)
		//if allow {
		//	next(w, r)
		//	return
		//}
		//err := errors.New("not allow")
		//logx.Errorf("InterMiddleware err: %v", err)
		//httpx.Error(w, err)
		//return

		next(w, r)
	}
}

//判断网段合法
func (m *InterMiddleware) isBelong(ip, cidr string) bool {
	ipStrList := strings.Split(ip, ".")
	cidrList := strings.Split(cidr, ".")
	if len(ipStrList) != 4 || len(ipStrList) != 4 {
		return false
	}
	for i := 0; i < 2; i++ {
		if ipStrList[i] != cidrList[i] {
			return false
		}
	}
	return true
}
