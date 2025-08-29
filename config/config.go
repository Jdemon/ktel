package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config struct holds the application configuration.
type Config struct {
	AppName string `mapstructure:"appName" validate:"required"`
	Kafka   struct {
		Brokers           string `mapstructure:"brokers" validate:"required"`
		Topic             string `mapstructure:"topic" validate:"required"`
		GroupID           string `mapstructure:"groupId" validate:"required"`
		RebalanceStrategy string `mapstructure:"rebalanceStrategy"`
		TLS               struct {
			Enabled  bool   `mapstructure:"enabled"`
			CAFile   string `mapstructure:"caFile"`
			CertFile string `mapstructure:"certFile"`
			KeyFile  string `mapstructure:"keyFile"`
		} `mapstructure:"tls"`
		SASL struct {
			Enabled   bool   `mapstructure:"enabled"`
			Mechanism string `mapstructure:"mechanism"`
			Username  string `mapstructure:"username"`
			Password  string `mapstructure:"password"`
		} `mapstructure:"sasl"`
	} `mapstructure:"kafka"`
	Server struct {
		Port string `mapstructure:"port" validate:"required"`
	} `mapstructure:"server"`
	Otel struct {
		Enabled  bool `mapstructure:"enabled"`
		Exporter struct {
			Grpc struct {
				Endpoint string `mapstructure:"endpoint"`
			} `mapstructure:"grpc"`
		} `mapstructure:"exporter"`
	} `mapstructure:"otel"`
}

// New creates a new Config struct and loads configuration from a file and environment variables.
func New() (*Config, error) {
	v := viper.New()

	// Set default values
	v.SetDefault("server.port", "8080")
	v.SetDefault("appName", "kafka-consumer")
	v.SetDefault("kafka.groupId", "kafka-consumer-group")

	// Configure viper
	v.SetConfigName("ktel-config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("/app")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read configurations
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal configuration
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}
