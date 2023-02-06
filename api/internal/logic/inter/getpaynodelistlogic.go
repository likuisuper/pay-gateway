package inter

import (
	"context"
	"encoding/json"
	"fmt"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
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

	path := fmt.Sprintf("http://%s:%d%s", c.Nacos.NacosService[0].Ip, c.Nacos.NacosService[0].Port, "/nacos/v2/ns/instance/list")
	path += fmt.Sprintf("?namespaceId=%s&serviceName=DEFAULT_GROUP@@payment.rpc&username=%s&password=%s", c.Nacos.NamespaceId, c.Nacos.Username, c.Nacos.Password)
	body, nacosErr := util.HttpGet(path, map[string]string{}, map[string]string{})

	if nacosErr != nil {
		logx.Errorf("Couldn't connect to the nacos API: %s", nacosErr.Error())
	}
	//body, err := ioutil.ReadAll(nacosResp.Body)
	nacosService := new(model.Service)
	err = json.Unmarshal(body, nacosService)
	if err != nil {
		logx.Errorf("Unmarshal err: %s, dataStr: %s", err.Error(), string(body))
	}

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
