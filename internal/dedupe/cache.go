package dedupe

import (
	"sync"
	"time"
)

type Cache struct {
	mu   sync.Mutex
	ttl  time.Duration
	seen map[string]time.Time
}

func New(ttl time.Duration) *Cache {
	return &Cache{ttl: ttl, seen: make(map[string]time.Time)}
}

func (c *Cache) Accept(key string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, t := range c.seen {
		if now.Sub(t) >= c.ttl { delete(c.seen, k) }
	}
	if t, ok := c.seen[key]; ok && now.Sub(t) < c.ttl { return false }
	c.seen[key] = now
	return true
}
