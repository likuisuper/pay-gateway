package inter

import (
	"context"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/codes"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitee.com/zhuyunkj/pay-gateway/api/internal/svc"
	"gitee.com/zhuyunkj/pay-gateway/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
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

	config := clientv3.Config{
		Endpoints:   l.svcCtx.Config.Etcd.Host,
		DialTimeout: 5 * time.Second,
	}
	etcdCli, err := clientv3.New(config)
	if err != nil {
		return
	}
	defer etcdCli.Close()

	res, err := etcdCli.Get(l.ctx, "payment.rpc", clientv3.WithPrefix())
	if err != nil {
		return
	}

	nodeList := make([]string, 0)
	for _, kv := range res.Kvs {
		v := string(kv.Value)
		vList := strings.Split(v, ":")
		apiHost := vList[0] + ":" + strconv.Itoa(l.svcCtx.Config.Port)
		nodeList = append(nodeList, apiHost)
	}

	resp = &types.ResultResp{
		RequestId: util.GetUuid(),
		Status:    int64(codes.OK),
		Data:      nodeList,
	}

	return
}
