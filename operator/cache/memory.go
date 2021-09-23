package cache

import (
	"sync"
	"time"
)

func NewMemory(initSize, maxSize uint) *Memory {
	m := Memory{}
	m.cache = make(map[string]item, initSize)
	return &m
}

type Memory struct {
	cache   map[string]item
	maxSize int
	mutex   sync.RWMutex
}

type item struct {
	data      interface{}
	timestamp time.Time
}

func (m *Memory) Get(key string) (interface{}, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	item, ok := m.cache[key]
	return item.data, ok
}

func (m *Memory) Add(key string, data interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.cache) == m.maxSize {
		now := time.Now()
		oldest := ""
		for i, _ := range m.cache {
			if m.cache[i].timestamp.Before(now) {
				oldest = i
			}
		}
		delete(m.cache, oldest)
	}

	e := item{
		data:      data,
		timestamp: time.Now(),
	}

	m.cache[key] = e
}
