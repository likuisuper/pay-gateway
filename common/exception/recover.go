package exception

import (
	"github.com/zeromicro/go-zero/core/logx"
	kv_m "gitlab.muchcloud.com/consumer-project/zhuyun-core/kv_monitor"
)

var (
	panicRecover = kv_m.Register{kv_m.Regist(&kv_m.Monitor{kv_m.CounterValue, kv_m.KvLabels{"kind": "common"}, "panicRecover", nil, "程序panic", nil})}
)

func Recover() {
	if msg := recover(); msg != nil {
		panicRecover.CounterInc()
		logx.Error("panic recover :", msg)
	}
}
