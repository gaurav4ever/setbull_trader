package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

type InMemConfig struct {
	TTL        time.Duration `json:"ttl,omitempty"`
	CleanUpTTL time.Duration `json:"cleanupttl,omitempty"`
}

func NewInMemoryCache(cfg InMemConfig) *cache.Cache {
	return cache.New(cfg.TTL, cfg.CleanUpTTL)
}
