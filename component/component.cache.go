package component

import (
	"fmt"
	"path/filepath"

	"github.com/asaskevich/govalidator"
	"github.com/micro-plat/hydra/conf"
	"github.com/qxnw/lib4go/cache"
	"github.com/qxnw/lib4go/concurrent/cmap"
)

//CacheTypeNameInVar 缓存在var配置中的类型名称
const CacheTypeNameInVar = "cache"

//CacheNameInVar 缓存名称在var配置中的末节点名称
const CacheNameInVar = "cache"

//IComponentCache Component Cache
type IComponentCache interface {
	GetCache(names ...string) (c cache.ICache, err error)
	GetCacheBy(tpName string, name string) (c cache.ICache, err error)
	SaveCacheObject(tpName string, name string, f func(c conf.IConf) (cache.ICache, error)) (bool, cache.ICache, error)
	Close() error
}

//StandardCache cache
type StandardCache struct {
	IContainer
	name     string
	cacheMap cmap.ConcurrentMap
}

//NewStandardCache 创建cache
func NewStandardCache(c IContainer, name ...string) *StandardCache {
	if len(name) > 0 {
		return &StandardCache{IContainer: c, name: name[0], cacheMap: cmap.New(2)}
	}
	return &StandardCache{IContainer: c, name: CacheNameInVar, cacheMap: cmap.New(2)}
}

//GetCache 获取缓存操作对象
func (s *StandardCache) GetCache(names ...string) (c cache.ICache, err error) {
	name := s.name
	if len(names) > 0 {
		name = names[0]
	}
	return s.GetCacheBy(CacheTypeNameInVar, name)
}

//GetCacheBy 根据类型获取缓存数据
func (s *StandardCache) GetCacheBy(tpName string, name string) (c cache.ICache, err error) {
	_, c, err = s.SaveCacheObject(tpName, name, func(chConf conf.IConf) (cache.ICache, error) {
		var chObjConf conf.CacheConf
		if err = chConf.Unmarshal(&chObjConf); err != nil {
			return nil, err
		}
		if b, err := govalidator.ValidateStruct(&chObjConf); !b {
			return nil, err
		}
		return cache.NewCache(chObjConf.Server, string(chConf.GetRaw()))
	})
	return c, err
}

//SaveCacheObject 缓存对象
func (s *StandardCache) SaveCacheObject(tpName string, name string, f func(c conf.IConf) (cache.ICache, error)) (bool, cache.ICache, error) {
	cacheConf, err := s.IContainer.GetVarConf(tpName, name)
	if err != nil {
		return false, nil, fmt.Errorf("%s %v", filepath.Join("/", s.GetPlatName(), "var", tpName, name), err)
	}
	key := fmt.Sprintf("%s/%s:%d", tpName, name, cacheConf.GetVersion())
	ok, ch, err := s.cacheMap.SetIfAbsentCb(key, func(input ...interface{}) (c interface{}, err error) {
		return f(cacheConf)
	})
	if err != nil {
		err = fmt.Errorf("创建cache失败:%s,err:%v", string(cacheConf.GetRaw()), err)
		return ok, nil, err
	}
	return ok, ch.(cache.ICache), err
}

//Close 关闭缓存连接
func (s *StandardCache) Close() error {
	s.cacheMap.RemoveIterCb(func(k string, v interface{}) bool {
		v.(cache.ICache).Close()
		return true
	})
	return nil
}
