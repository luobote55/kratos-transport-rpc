package mqtt

import (
	"github.com/go-kratos/kratos/v2/log"
	"sync"
	"time"
)

type logFilter struct {
	cache
}

func (l *logFilter) Error(key string, err string) {
	_, ok := l.Get(key)
	if ok {
		return
	}
	l.Set(key, struct{}{})
	log.Error(err)
}

// 创建一个缓存结构体
type cache struct {
	sync.RWMutex
	data      map[string]interface{} // 缓存的数据
	timestamp map[string]int64       // 数据的时间戳
	timeout   time.Duration          // 超时时间
}

// 初始化缓存
func NewCache(timeout time.Duration) *cache {
	return &cache{
		data:      make(map[string]interface{}),
		timestamp: make(map[string]int64),
		timeout:   timeout,
	}
}

// 向缓存中添加数据
func (c *cache) Set(key string, value interface{}) {
	c.Lock()
	defer c.Unlock()
	c.data[key] = value
	c.timestamp[key] = time.Now().UnixNano() // 记录当前时间戳
}

// 从缓存中获取数据
func (c *cache) Get(key string) (interface{}, bool) {
	c.RLock()
	defer c.RUnlock()
	value, ok := c.data[key]
	if ok {
		// 判断数据是否过期
		timestamp, exist := c.timestamp[key]
		if !exist || time.Now().UnixNano()-timestamp > int64(c.timeout) {
			delete(c.data, key)
			delete(c.timestamp, key)
			return nil, false
		}
		return value, true
	}
	return nil, false
}
