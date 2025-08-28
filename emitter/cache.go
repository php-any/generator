package emitter

import "sync"

// CacheImpl 简单缓存：键值缓存与集合去重

type CacheImpl struct {
	mu   sync.RWMutex
	data map[string]interface{}
	set  map[string]map[string]bool // group -> key
}

func NewCache() *CacheImpl {
	return &CacheImpl{
		data: make(map[string]interface{}),
		set:  make(map[string]map[string]bool),
	}
}

func (c *CacheImpl) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data[key]
	return v, ok
}

func (c *CacheImpl) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

func (c *CacheImpl) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *CacheImpl) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = map[string]interface{}{}
	c.set = map[string]map[string]bool{}
}

// 集合操作：用于import去重/文件去重
func (c *CacheImpl) SetAdd(group, key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.set[group] == nil {
		c.set[group] = map[string]bool{}
	}
	if c.set[group][key] {
		return false
	}
	c.set[group][key] = true
	return true
}

func (c *CacheImpl) SetHas(group, key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.set[group] != nil && c.set[group][key]
}
