package inter

import (
	"context"
	"encoding/json"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/nacos-group/nacos-sdk-go/v2/common/http_agent"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"io/ioutil"
	"net/http"
)

type GetPayNodeListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPayNodeListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPayNodeListLogic {

	return &GetPayNodeListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPayNodeListLogic) GetPayNodeList(req *types.EmptyReq, request *http.Request) (resp *types.ResultResp, err error) {

	c := l.svcCtx.Config

	//scheme := "https"
	//if c.Mode != service.ProMode {
	//	scheme = "http"
	//}
	path := fmt.Sprintf("http://%s:%d%s", c.Nacos.NacosService[0].Ip, c.Nacos.NacosService[0].Port, "/nacos/v2/ns/instance/list")
	nacosResp, nacosErr := new(http_agent.HttpAgent).Get(path, nil, 5000, map[string]string{
		"namespaceId": c.Nacos.NamespaceId,
		"serviceName": "DEFAULT_GROUP@@payment.rpc",
		"username":    c.Nacos.Username,
		"password":    c.Nacos.Password,
	})
	if nacosErr != nil {
		logx.Errorf("Couldn't connect to the nacos API: %s", nacosErr.Error())
	}
	println(nacosResp)
	body, err := ioutil.ReadAll(nacosResp.Body)
	nacosService := new(model.Service)
	_ = json.Unmarshal(body, nacosService)

	nodeList := make([]string, 0)
	for _, h := range nacosService.Hosts {
		nodeHost := fmt.Sprintf("%s:%d", h.Ip, h.Port)
		nodeList = append(nodeList, nodeHost)
	}

	resp = &types.ResultResp{
		RequestId: util.GetUuid(),
		Status:    int64(codes.OK),
		Data:      nodeList,
	}

	return
}
