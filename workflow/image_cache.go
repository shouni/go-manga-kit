package workflow

import (
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

type imageCache struct {
	cache    *ttlcache.Cache[string, any]
	started  bool
	mu       sync.Mutex
	stopOnce sync.Once
}

func newImageCache(defaultExpiration time.Duration) *imageCache {
	c := ttlcache.New[string, any](
		ttlcache.WithTTL[string, any](defaultExpiration),
		ttlcache.WithDisableTouchOnHit[string, any](),
	)

	return &imageCache{cache: c}
}

func (c *imageCache) Start() {
	if c == nil || c.cache == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		return
	}
	c.started = true
	go c.cache.Start()
}

func (c *imageCache) Get(key string) (any, bool) {
	item := c.cache.Get(key)
	if item == nil {
		return nil, false
	}

	return item.Value(), true
}

func (c *imageCache) Set(key string, value any, ttl time.Duration) {
	c.cache.Set(key, value, ttl)
}

func (c *imageCache) Stop() {
	if c == nil || c.cache == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return
	}

	c.started = false
	c.cache.Stop()
}
