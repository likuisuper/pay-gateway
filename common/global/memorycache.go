package global

import (
	"encoding/json"

	"github.com/coocood/freecache"
	"github.com/zeromicro/go-zero/core/logx"
	"gitlab.muchcloud.com/consumer-project/zhuyun-core/util"
)

var MemoryCacheInstance *memoryCache

type memoryCache struct {
	FreeCache *freecache.Cache
}

func InitMemoryCacheInstance(size int) {
	size = size * 1024 * 1024
	MemoryCacheInstance = new(memoryCache)
	MemoryCacheInstance.FreeCache = freecache.NewCache(size)
}

//通过缓存取数据、没有就取dataFunc再设置缓存
/**
@cacheKey 缓存key
@expire 过期时间
@data 接受数据的指针
@dataFunc 取数据的方法闭包 错误返回err
*/
func (instance *memoryCache) GetDataWithCache(cacheKey string, expire int, data interface{}, dataFunc func() interface{}) (err error) {
	d, _ := instance.FreeCache.Get([]byte(cacheKey))
	if len(d) != 0 {
		_ = json.Unmarshal(d, data)
		return
	}

	funcRes := dataFunc()
	if err, ok := funcRes.(error); ok {
		return err
	}

	dataByte, _ := json.Marshal(funcRes)
	_ = json.Unmarshal(dataByte, data)

	freeCacheErr := instance.FreeCache.Set([]byte(cacheKey), dataByte, expire)
	if freeCacheErr != nil {
		util.CheckError("内存缓存设置失败 %v", freeCacheErr)
	}
	return
}

// 清除全部本地缓存
func (instance *memoryCache) ClearAll() {
	instance.FreeCache.Clear()
	logx.Info("清除内存缓存成功")
}
