package config

import (
	"encoding/json"
	"setbull_trader/pkg/cache"
	"setbull_trader/pkg/database"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig `mapstructure:"server"`
	Dhan     DhanConfig   `mapstructure:"dhan"`
	Database struct {
		MasterDatasource struct {
			User     string `yaml:"user"`
			Password string `yaml:"password"`
			Host     string `yaml:"host"`
			Name     string `yaml:"name"`
		} `yaml:"masterDatasource"`
		SlaveDatasource struct {
			User     string `yaml:"user"`
			Password string `yaml:"password"`
			Host     string `yaml:"host"`
			Name     string `yaml:"name"`
		} `yaml:"slaveDatasource"`
		MaxIdleConnections    int           `yaml:"maxIdleConnections"`
		MaxOpenConnections    int           `yaml:"maxOpenConnections"`
		MaxConnectionLifeTime time.Duration `yaml:"maxConnectionLifetime"`
		MaxConnectionIdleTime time.Duration `yaml:"maxConnectionIdletime"`
		DisableTLS            bool          `yaml:"disableTLS"`
		Debug                 bool          `yaml:"debug"`
	} `yaml:"database"`
	Cache struct {
		Redis struct {
			Host                  string        `yaml:"host"`
			Port                  string        `yaml:"port"`
			Database              int           `yaml:"database" json:"database,omitempty"`
			IdleConnectionTimeout time.Duration `yaml:"idleConnectionTimeout" json:"idle_connection_timeout,omitempty"`
			ConnectTimeout        time.Duration `yaml:"connectTimeout"  json:"connect_timeout,omitempty"`
			ReadTimeout           time.Duration `yaml:"readTimeout"  json:"read_timeout,omitempty"`
			WriteTimeout          time.Duration `yaml:"writeTimeout"  json:"write_timeout,omitempty"`
			PoolSize              int           `yaml:"poolSize"  json:"pool_size,omitempty"`
			MaxRetry              int           `yaml:"maxRetry"  json:"max_retry,omitempty"`
			MinIdleConns          int           `yaml:"minIdle"  json:"min_idle_conns,omitempty"`
			TTL                   time.Duration `yaml:"ttl"  json:"ttl,omitempty"`
			TCPNoDelay            bool          `yaml:"tcpNoDelay"  json:"tcp_no_delay,omitempty"`
			Disable               bool          `yaml:"disable"  json:"disable,omitempty"`
		} `yaml:"redis" json:"redis,omitempty"`
		InMem struct {
			TTL        time.Duration `yaml:"ttl" json:"ttl,omitempty"`
			CleanUpTTL time.Duration `yaml:"cleanupttl" json:"cleanupttl,omitempty"`
		} `yaml:"inmem" json:"inmem,omitempty"`
	}
}

// ServerConfig represents the HTTP server configuration
type ServerConfig struct {
	Port         string `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// DhanConfig represents the Dhan API configuration
type DhanConfig struct {
	BaseURL     string `mapstructure:"base_url"`
	AccessToken string `mapstructure:"access_token"`
	ClientID    string `mapstructure:"client_id"`
}

// LoadConfig loads the application configuration from application.yaml
func LoadConfig() (*Config, error) {
	viper.SetConfigName("application.dev")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, errors.Wrap(err, "error reading config file")
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, errors.Wrap(err, "error unmarshalling config")
	}

	return &config, nil
}

func LoadDatabase(appCfg Config) (database.Config, error) {
	var cfg database.Config

	b, err := json.Marshal(appCfg.Database)
	if err != nil {
		return database.Config{}, err
	}

	err = json.Unmarshal(b, &cfg)
	if err != nil {
		return database.Config{}, err
	}

	return cfg, err
}

func LoadRedis(appCfg Config) (cache.RedisConfig, error) {
	var cfg cache.RedisConfig

	b, err := json.Marshal(appCfg.Cache.Redis)
	if err != nil {
		return cache.RedisConfig{}, err
	}

	err = json.Unmarshal(b, &cfg)
	if err != nil {
		return cache.RedisConfig{}, err
	}

	return cfg, nil
}

func LoadInMemoryCache(appCfg Config) (cache.InMemConfig, error) {
	var cfg cache.InMemConfig

	b, err := json.Marshal(appCfg.Cache.InMem)
	if err != nil {
		return cache.InMemConfig{}, err
	}

	err = json.Unmarshal(b, &cfg)
	if err != nil {
		return cache.InMemConfig{}, err
	}

	return cfg, nil
}
