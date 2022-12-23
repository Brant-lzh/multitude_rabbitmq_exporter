package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"time"
)

var cache *cacheRegistry

type cacheRegistry struct {
	mp    map[string]*multiRegistry
	mutex *sync.RWMutex
}

func initCacheRegistry() {
	cache = &cacheRegistry{
		mp:    make(map[string]*multiRegistry),
		mutex: &sync.RWMutex{},
	}
}

func (c *cacheRegistry) Get(key string) (*multiRegistry, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	result, found := c.mp[key]
	return result, found
}

func (c *cacheRegistry) Set(key string, value *multiRegistry) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.mp[key] = value
}

func (c *cacheRegistry) Count() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.mp)
}

type multiRegistry struct {
	reg        *prometheus.Registry
	createTime time.Time
	updateTime time.Time
}

func NewMultiRegistry(cs prometheus.Collector) *multiRegistry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(cs)
	return &multiRegistry{
		reg:        registry,
		createTime: time.Now(),
		updateTime: time.Now(),
	}
}

func (m *multiRegistry) SetUpdateTime(time time.Time) {
	m.updateTime = time
}
