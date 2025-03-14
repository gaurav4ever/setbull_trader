package config

import (
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Dhan   DhanConfig   `mapstructure:"dhan"`
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
