package cache

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
)

type API interface {
	Get(ctx context.Context, key string) (string, bool)
	SetWithDuration(ctx context.Context, key string, value string, duration time.Duration)
	Set(ctx context.Context, key string, value string)
}
type Manager struct {
	inmem *cache.Cache
	redis *redis.Client
}

func NewCacheManager(inmem *cache.Cache, redis *redis.Client) API {
	return &Manager{
		inmem: inmem,
		redis: redis,
	}
}

func (c *Manager) Get(ctx context.Context, key string) (string, bool) {
	logger := ctxzap.Extract(ctx)

	// get from in-mem cache
	cVal, present := c.inmem.Get(key)
	if !present {
		// get from redis
		rVal, err := c.redis.Get(ctx, key).Result()
		if (err != nil) && (err.Error() != "redis: nil") {
			logger.Sugar().Warnf("occurred while retrieving data from redis %v", err)
			return "", false
		}
		if len(rVal) == 0 {
			return rVal, false
		}
		return rVal, true
	}
	return cVal.(string), present
}

func (c *Manager) SetWithDuration(ctx context.Context, key string, value string, duration time.Duration) {
	logger := ctxzap.Extract(ctx)

	// set in mem
	c.inmem.Set(key, value, duration)

	// set in redis
	_, err := c.redis.Set(ctx, key, value, duration).Result()
	if err != nil {
		logger.Sugar().Errorf("occurred %v while saving data %v to redis for key %v", value, err, key)
	}
}

func (c *Manager) Set(ctx context.Context, key string, value string) {
	logger := ctxzap.Extract(ctx)
	// set in mem
	c.inmem.Set(key, value, time.Minute*10)

	// set in redis
	_, err := c.redis.Set(ctx, key, value, time.Minute*30).Result()
	if err != nil {
		logger.Sugar().Errorf("occurred %v while saving data %v to redis for key %v", err, value, key)
	}
}
