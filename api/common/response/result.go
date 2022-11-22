package response

import (
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/zhuyun-core/util"
)

func MakeResult(status int, desc string, data interface{}) types.ResultResp {
	return types.ResultResp{
		RequestId: util.GetUuid(),
		Status:    int64(status),
		Desc:      desc,
		Data:      data,
	}
}
