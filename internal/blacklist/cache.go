package blacklist

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

type Cache struct {
	mu  sync.Mutex
	lru *lru.Cache[string, int64] // key -> unix nano expiry
}

func New(size int) (*Cache, error) {
	c, err := lru.New[string, int64](size)
	if err != nil {
		return nil, err
	}
	return &Cache{lru: c}, nil
}

func (c *Cache) IsBlacklisted(key string, now time.Time) bool {
	nowN := now.UnixNano()
	c.mu.Lock()
	defer c.mu.Unlock()
	exp, ok := c.lru.Get(key)
	if !ok {
		return false
	}
	if exp <= nowN {
		c.lru.Remove(key)
		return false
	}
	return true
}

func (c *Cache) Blacklist(key string, until time.Time) {
	c.mu.Lock()
	c.lru.Add(key, until.UnixNano())
	c.mu.Unlock()
}
