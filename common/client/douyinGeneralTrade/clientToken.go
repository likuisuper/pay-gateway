package douyin

import (
	"errors"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/bytedance/sonic"
	"github.com/zeromicro/go-zero/core/logx"
	"log"
	"time"
)

type getClientTokenResp struct {
	Code int `json:"code"`
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
	Msg string `json:"msg"`
}

func getClientToken(url, appId string) (token string, err error) {
	defer func(start time.Time) {
		log.Printf("getClientToken timecost:%v, token:%s", time.Since(start), token)
		logx.Slowf("getClientToken timecost:%v, token:%s", time.Since(start), token)
	}(time.Now())

	result, err := util.HttpGet(url, map[string]string{
		"dyAppid": appId,
	}, nil)
	if err != nil {
		return "", err
	}

	resp := new(getClientTokenResp)
	err = sonic.Unmarshal(result, resp)
	if err != nil {
		return "", err
	}

	if resp.Code != 200 && resp.Data.Token == "" {
		return "", errors.New("invalid resp")
	}

	return resp.Data.Token, nil
}
