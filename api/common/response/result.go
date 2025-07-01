package response

import (
	"gitlab.muchcloud.com/consumer-project/pay-gateway/api/internal/types"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/util"
)

func MakeResult(status int, desc string, data interface{}) types.ResultResp {
	return types.ResultResp{
		RequestId: util.GetUuid(),
		Status:    int64(status),
		Desc:      desc,
		Data:      data,
	}
}
