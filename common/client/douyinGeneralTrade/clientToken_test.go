package douyin

import (
	redisdb "gitee.com/zhuyunkj/zhuyun-core/cache"
	"os"
)

func init() {
	/*
		- ComPrefix: payment
		    MaxActive: 100 #最大的激活连接数,支持最高并发数
		    MaxIdle: 5  #最大的空闲连接数
		    IdleTimeout: 180 #最大的空闲连接等待时间，超过此时间后，空闲连接将被关闭
		    Address: 120.79.85.139:6379
		    Passwd: 123456@2021bcd
		    DBs:
		      - Name: pay_gateway
		        DBIndex: 12
		        Prefix: pay_gateway
	*/
	err := redisdb.InitRedis([]*redisdb.RedisConfigs{
		{
			RedisCommon: redisdb.RedisCommon{
				ComPrefix:   "payment",
				MaxActive:   100,
				MaxIdle:     5,
				IdleTimeout: 180,
				Address:     "120.79.85.139:6379",
				Passwd:      "123456@2021bcd",
			},
			DBs: []redisdb.RedisDB{
				{
					Name:    "pay_gateway",
					DBIndex: 12,
					Prefix:  "pay_gateway",
				},
			},
		},
	})
	if err != nil {
		os.Exit(1)
	}
}
