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
	Server                             ServerConfig         `mapstructure:"server"`
	Trading                            TradingConfig        `mapstructure:"trading" yaml:"trading"`
	Dhan                               DhanConfig           `mapstructure:"dhan"`
	Upstox                             UpstoxConfig         `mapstructure:"upstox"`
	StockUniverse                      StockUniverseConfig  `mapstructure:"stock_universe"`
	HistoricalData                     HistoricalDataConfig `mapstructure:"historical_data"`
	MambaFilter                        MambaFilterConfig    `mapstructure:"mamba_filter" yaml:"mamba_filter"`
	OneMinCandleIngestionOffsetSeconds int                  `mapstructure:"one_min_candle_ingestion_offset_seconds" yaml:"one_min_candle_ingestion_offset_seconds"`
	Database                           struct {
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

type MarketConfig struct {
	ExcludeWeekends bool `yaml:"excludeWeekends"`
}

type TradingConfig struct {
	Market                  MarketConfig `yaml:"market"`
	FirstEntrySLPercent     float64      `yaml:"first_entry_sl_percentage"`
	FirstEntryRiskPerTrade  int          `yaml:"first_entry_risk_per_trade"`
	SecondEntrySLPercent    float64      `yaml:"second_entry_sl_percentage"`
	SecondEntryRiskPerTrade int          `yaml:"second_entry_risk_per_trade"`
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

type UpstoxConfig struct {
	ClientID     string `mapstructure:"client_id" yaml:"client_id"`
	ClientSecret string `mapstructure:"client_secret" yaml:"client_secret"`
	RedirectURI  string `mapstructure:"redirect_uri" yaml:"redirect_uri"`
	BasePath     string `mapstructure:"base_path" yaml:"base_path"`
}

type HistoricalDataConfig struct {
	MaxConcurrentRequests int           `yaml:"maxConcurrentRequests" json:"maxConcurrentRequests"`
	DefaultInterval       string        `yaml:"defaultInterval" json:"defaultInterval"`
	DefaultDaysToFetch    int           `yaml:"defaultDaysToFetch" json:"defaultDaysToFetch"`
	DefaultUserID         string        `yaml:"defaultUserID" json:"defaultUserID"`
	RetentionPeriodDays   int           `yaml:"retentionPeriodDays" json:"retentionPeriodDays"`
	BatchSize             int           `yaml:"batchSize" json:"batchSize"`
	EnableAutoCleanup     bool          `yaml:"enableAutoCleanup" json:"enableAutoCleanup"`
	CleanupInterval       time.Duration `yaml:"cleanupInterval" json:"cleanupInterval"`
}

// StockUniverseConfig contains configuration for the stock universe feature
type StockUniverseConfig struct {
	FilePath string `json:"file_path" yaml:"file_path"`
}

// Add new config type
type MambaFilterConfig struct {
	LookbackPeriod       int     `yaml:"lookback_period" json:"lookback_period"`
	MoveThresholdBullish float64 `yaml:"move_threshold_bullish" json:"move_threshold_bullish"`
	MoveThresholdBearish float64 `yaml:"move_threshold_bearish" json:"move_threshold_bearish"`
	MinSequenceLength    int     `yaml:"min_sequence_length" json:"min_sequence_length"`
	MaxGapDays           int     `yaml:"max_gap_days" json:"max_gap_days"`
	MinMambaRatio        float64 `yaml:"min_mamba_ratio" json:"min_mamba_ratio"`
	MinMambaDays         int     `yaml:"min_mamba_days" json:"min_mamba_days"`
	MoveAnalyzer         struct {
		StrengthThreshold float64 `yaml:"strength_threshold" json:"strength_threshold"`
		VolumeWeight      float64 `yaml:"volume_weight" json:"volume_weight"`
	} `yaml:"move_analyzer" json:"move_analyzer"`
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

	if config.Trading.Market.ExcludeWeekends {
		config.Trading.Market.ExcludeWeekends = true
	}

	// Validate Upstox configuration
	if err := config.ValidateUpstoxConfig(); err != nil {
		return nil, errors.Wrap(err, "invalid upstox configuration")
	}

	setDefaultHistoricalDataConfig(&config)

	return &config, nil
}

func setDefaultHistoricalDataConfig(config *Config) {
	if config.HistoricalData == (HistoricalDataConfig{}) {
		config.HistoricalData = HistoricalDataConfig{
			MaxConcurrentRequests: 5,
			DefaultInterval:       "1minute",
			DefaultDaysToFetch:    30,
			DefaultUserID:         "default_user",
			RetentionPeriodDays:   90,
			BatchSize:             1000,
			EnableAutoCleanup:     true,
			CleanupInterval:       24 * time.Hour,
		}
	}
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

// ValidateUpstoxConfig validates the Upstox configuration
func (c *Config) ValidateUpstoxConfig() error {
	if c.Upstox.ClientID == "" {
		return errors.New("upstox client_id is required")
	}
	if c.Upstox.ClientSecret == "" {
		return errors.New("upstox client_secret is required")
	}
	if c.Upstox.RedirectURI == "" {
		return errors.New("upstox redirect_uri is required")
	}
	if c.Upstox.BasePath == "" {
		c.Upstox.BasePath = "https://api.upstox.com" // Set default if not provided
	}
	return nil
}

// LoadStockUniverseConfig loads the stock universe configuration from the main config
func (c *Config) LoadStockUniverseConfig(cfg Config) (*StockUniverseConfig, error) {
	stockUniverseConfig := &StockUniverseConfig{
		FilePath: c.StockUniverse.FilePath,
	}

	// Set default file path if not specified
	if stockUniverseConfig.FilePath == "" {
		stockUniverseConfig.FilePath = "data/nse_upstox.json"
	}

	return stockUniverseConfig, nil
}

// Optionally, add getters for these fields if needed
func (c *Config) GetFirstEntrySLPercent() float64 {
	return c.Trading.FirstEntrySLPercent
}

func (c *Config) GetFirstEntryRiskPerTrade() int {
	return c.Trading.FirstEntryRiskPerTrade
}

func (c *Config) GetSecondEntrySLPercent() float64 {
	return c.Trading.SecondEntrySLPercent
}

func (c *Config) GetSecondEntryRiskPerTrade() int {
	return c.Trading.SecondEntryRiskPerTrade
}
