package middleware

import (
	"errors"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"strconv"
	"strings"
)

type InterMiddleware struct {
}

func NewInterMiddleware() *InterMiddleware {
	return &InterMiddleware{}
}

func (m *InterMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowCidr := "172.30.0.0/16"
		ip := r.Header.Get("X-Forwarded-For")
		if m.isBelong(ip, allowCidr) {
			next(w, r)
		}
		err := errors.New("not allow")
		httpx.Error(w, err)
	}
}

//判断网段合法
func (m *InterMiddleware) isBelong(ip, cidr string) bool {
	ipAddr := strings.Split(ip, `.`)
	if len(ipAddr) < 4 {
		return false
	}
	cidrArr := strings.Split(cidr, `/`)
	if len(cidrArr) < 2 {
		return false
	}
	var tmp = make([]string, 0)
	for key, value := range strings.Split(`255.255.255.0`, `.`) {
		iint, _ := strconv.Atoi(value)

		iint2, _ := strconv.Atoi(ipAddr[key])

		tmp = append(tmp, strconv.Itoa(iint&iint2))
	}
	return strings.Join(tmp, `.`) == cidrArr[0]
}
