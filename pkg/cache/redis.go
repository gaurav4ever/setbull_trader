package cache

import (
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Host                  string        `yaml:"host"`
	Port                  string        `yaml:"port"`
	Database              int           `yaml:"database" json:"database,omitempty"`
	IdleConnectionTimeout time.Duration `yaml:"idleConnectionTimeout" json:"idle_connection_timeout,omitempty"`
	ConnectTimeout        time.Duration `yaml:"connectTimeout" json:"connect_timeout,omitempty"`
	ReadTimeout           time.Duration `yaml:"readTimeout" json:"read_timeout,omitempty"`
	WriteTimeout          time.Duration `yaml:"writeTimeout"  json:"write_timeout,omitempty"`
	PoolSize              int           `yaml:"poolSize"  json:"pool_size,omitempty"`
	MaxRetry              int           `yaml:"maxRetry"  json:"max_retry,omitempty"`
	MinIdleConns          int           `yaml:"minIdle"  json:"min_idle_conns,omitempty"`
	TTL                   time.Duration `yaml:"ttl"  json:"ttl,omitempty"`
	TCPNoDelay            bool          `yaml:"tcpNoDelay"  json:"tcp_no_delay,omitempty"`
	Disable               bool          `yaml:"disable"  json:"disable,omitempty"`
}

func NewRedisStore(cfg RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         cfg.Host + ":" + cfg.Port,
		MaxRetries:   cfg.MaxRetry,
		DialTimeout:  cfg.ConnectTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DB:           cfg.Database,
	})
}
