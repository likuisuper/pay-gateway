package utils

import (
	"errors"
	"fmt"
	"gitee.com/zhuyunkj/zhuyun-core/util"
	"github.com/google/uuid"
	"net"
)

func ToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch result := v.(type) {
	case string:
		return result
	case []byte:
		return string(result)
	default:
		return fmt.Sprint(result)
	}
}

func GetInterfaceIpv4Addr(interfaceName string) (ip net.IP, err error) {
	var (
		ief      *net.Interface
		addrs    []net.Addr
		ipv4Addr net.IP
	)
	if ief, err = net.InterfaceByName(interfaceName); err != nil { // get interface
		return
	}
	if addrs, err = ief.Addrs(); err != nil { // get addresses
		return
	}
	for _, addr := range addrs { // get ipv4 address
		if ipv4Addr = addr.(*net.IPNet).IP.To4(); ipv4Addr != nil {
			break
		}
	}
	if ipv4Addr == nil {
		return nil, errors.New(fmt.Sprintf("interface %s dosen't have an ipv4 address\n", interfaceName))
	}
	return ipv4Addr, nil
}

// 生成唯一订单号
func GenerateOrderCode(machineNo, workerNo int64) (orderCode string) {
	//获取唯一订单号
	worker, err := util.CreateWorker(machineNo, workerNo)
	if err != nil {
		//如果雪花算法有问题。直接使用google算法获取uuid
		orderCode = uuid.NewString()
	} else {
		orderCode, err = worker.GetId()
		if err != nil {
			//如果雪花算法有问题。直接使用google算法获取uuid
			orderCode = uuid.NewString()
		}
	}
	return
}
