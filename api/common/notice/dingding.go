package notice

import (
	"context"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/bytedance/sonic"
	"github.com/zeromicro/go-zero/core/logx"
	"time"
)

type RobotSendReq struct {
	Msgtype    MsgType     `json:"msgtype"`
	At         *At         `json:"at,omitempty"`
	Link       *Link       `json:"link,omitempty"`
	Markdown   *Markdown   `json:"markdown,omitempty"`
	FeedCard   *FeedCard   `json:"feedCard,omitempty"`
	Text       *Text       `json:"text,omitempty"`
	ActionCard *ActionCard `json:"actionCard,omitempty"`
}

type MsgType string

const (
	MsgTypeMarkdown MsgType = "markdown"
)

type At struct {
	IsAtAll   string   `json:"isAtAll"`
	AtUserIds []string `json:"atUserIds"`
	AtMobiles []string `json:"atMobiles"`
}

type Link struct {
	MessageUrl string `json:"messageUrl"`
	PicUrl     string `json:"picUrl"`
	Text       string `json:"text"`
	Title      string `json:"title"`
}

type Markdown struct {
	Text  string `json:"text"`
	Title string `json:"title"`
}

type FeedCard struct {
	Links struct {
		PicURL     string `json:"picURL"`
		MessageURL string `json:"messageURL"`
		Title      string `json:"title"`
	} `json:"links"`
}

type Text struct {
	Content string `json:"content"`
}

type ActionCard struct {
	HideAvatar     string `json:"hideAvatar"`
	BtnOrientation string `json:"btnOrientation"`
	SingleTitle    string `json:"singleTitle"`
	Btns           []struct {
		ActionURL string `json:"actionURL"`
		Title     string `json:"title"`
	} `json:"btns"`
	Text      string `json:"text"`
	SingleURL string `json:"singleURL"`
	Title     string `json:"title"`
}

type RobotSendResp struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

// SendWebhookMsg 向钉钉发起通知
// 相关文档 https://open.dingtalk.com/document/orgapp/custom-bot-send-message-type?spm=ding_open_doc.document.0.0.43267f7fhFf0HW
func SendWebhookMsg(ctx context.Context, req *RobotSendReq, webhookUrl string) (*RobotSendResp, error) {
	result, err := util.HttpPost(webhookUrl, req, time.Second*3)
	if err != nil {
		logx.WithContext(ctx).Errorf("http.Do fail, err:%v, url:%s, req:%v", err, webhookUrl, req)
		return nil, err
	}

	resp := new(RobotSendResp)
	err = sonic.UnmarshalString(result, resp)
	if err != nil {
		logx.WithContext(ctx).Errorf("sonicUnmarshal fail, err:%v, result:%s", err, result)
		return nil, err
	}

	logx.WithContext(ctx).Slowf("sendWebhookMsg ok, req:%v, result:%s, webhookUrl:%s", req, result, webhookUrl)
	return resp, nil
}
