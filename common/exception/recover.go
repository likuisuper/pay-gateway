package exception

import (
	kv_m "gitee.com/zhuyunkj/zhuyun-core/kv_monitor"
	"github.com/zeromicro/go-zero/core/logx"
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
