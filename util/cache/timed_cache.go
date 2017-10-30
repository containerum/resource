package cache

import (
	"sync"
	"time"
)

type timeValue struct {
	t time.Time
	v interface{}
}

type Timed struct {
	cache   map[interface{}]timeValue
	timeout time.Duration

	mu *sync.RWMutex
}

func NewTimed(timeout time.Duration) *Timed {
	c := &Timed{
		cache:   make(map[interface{}]timeValue),
		timeout: timeout,
		mu:      &sync.RWMutex{},
	}
	return c
}

func (c *Timed) Get(k interface{}) (v interface{}, cached bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tv, cached := c.cache[k]
	if !cached || time.Now().Sub(tv.t) > c.timeout {
		cached = false
		delete(c.cache, k)
	}

	return tv.v, true
}

func (c *Timed) Set(k interface{}, v interface{}) {
	tv := timeValue{time.Now(), v}
	c.mu.Lock()
	c.cache[k] = tv
	c.mu.Unlock()
}
