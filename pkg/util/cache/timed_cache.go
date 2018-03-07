package cache

import (
	"sync"
	"time"
)

type timeValue struct {
	t time.Time
	v interface{}
}

type timed struct {
	cache   map[interface{}]timeValue
	timeout time.Duration

	mu *sync.RWMutex
}

// NewTimed returns cache where resources have TTL. If TTL expired, it will not return value by key.
func NewTimed(timeout time.Duration) Cache {
	c := &timed{
		cache:   make(map[interface{}]timeValue),
		timeout: timeout,
		mu:      &sync.RWMutex{},
	}
	return c
}

func (c *timed) Get(k interface{}) (v interface{}, cached bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tv, cached := c.cache[k]
	if !cached || time.Now().Sub(tv.t) > c.timeout {
		cached = false
		delete(c.cache, k)
		return
	}

	v = tv.v
	return
}

func (c *timed) Set(k interface{}, v interface{}) {
	tv := timeValue{time.Now(), v}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[k] = tv
}
