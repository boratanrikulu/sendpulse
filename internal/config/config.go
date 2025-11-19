package config

import (
	"fmt"
	"os"
	"time"

	"github.com/onrik/logrus/filename"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
)

var defaultAppName = "sendpulse"

var Logger *logrus.Logger

var Version string = "0.1.0"

type Cfg struct {
	AppName   string    `mapstructure:"app_name"`
	Server    Server    `mapstructure:"server"`
	Database  Database  `mapstructure:"database"`
	Messaging Messaging `mapstructure:"messaging"`
	Webhook   Webhook   `mapstructure:"webhook"`
}

type Server struct {
	Address string `mapstructure:"address"`
	Mode    Mode   `mapstructure:"mode"`
}

type Mode string

const (
	ModeDev  Mode = "dev"
	ModeProd Mode = "prod"
)

type Database struct {
	DSN string  `mapstructure:"dsn"`
	DB  *bun.DB `mapstructure:"-"`
}

type Messaging struct {
	Interval   time.Duration `mapstructure:"interval"`
	BatchSize  int           `mapstructure:"batch_size"`
	MaxRetries int           `mapstructure:"max_retries"`
	RetryDelay time.Duration `mapstructure:"retry_delay"`
	Enabled    bool          `mapstructure:"enabled"`
}

type Webhook struct {
	URL string `mapstructure:"url"`
}

func NewConfig(filepath string) (*Cfg, error) {
	cfg := &Cfg{}

	// Set defaults first
	cfg.setDefaults()

	// Read from yaml file if provided
	if filepath != "" {
		v := viper.New()
		v.SetConfigType("yaml")
		v.SetConfigFile(filepath)
		if err := v.ReadInConfig(); err != nil {
			Log().Warnf("reading config file: %v", err)
		} else {
			if err := v.Unmarshal(cfg); err != nil {
				return nil, fmt.Errorf("unmarshaling config: %w", err)
			}
		}
	}

	// Override config with environment variables
	cfg.loadFromEnv()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func (cfg *Cfg) setDefaults() {
	cfg.AppName = defaultAppName
	cfg.Server.Address = ":8080"
	cfg.Server.Mode = ModeDev
	cfg.Messaging.Interval = 2 * time.Minute
	cfg.Messaging.BatchSize = 2
	cfg.Messaging.MaxRetries = 3
	cfg.Messaging.RetryDelay = 2 * time.Second
	cfg.Messaging.Enabled = false
}

// loadFromEnv overrides config values with environment variables if they exist
func (cfg *Cfg) loadFromEnv() {
	const envPrefix = "SENDPULSE_"

	// App config
	if envAppName := os.Getenv(envPrefix + "APP_NAME"); envAppName != "" {
		cfg.AppName = envAppName
	}

	// Server config
	if envAddress := os.Getenv(envPrefix + "SERVER_ADDRESS"); envAddress != "" {
		cfg.Server.Address = envAddress
	}
	if envMode := os.Getenv(envPrefix + "SERVER_MODE"); envMode != "" {
		cfg.Server.Mode = Mode(envMode)
	}

	// Database config
	if envDSN := os.Getenv(envPrefix + "DATABASE_DSN"); envDSN != "" {
		cfg.Database.DSN = envDSN
	}

	// Webhook config
	if envURL := os.Getenv(envPrefix + "WEBHOOK_URL"); envURL != "" {
		cfg.Webhook.URL = envURL
	}

	// Messaging config
	if envEnabled := os.Getenv(envPrefix + "MESSAGING_ENABLED"); envEnabled != "" {
		cfg.Messaging.Enabled = envEnabled == "true"
	}
	if envInterval := os.Getenv(envPrefix + "MESSAGING_INTERVAL"); envInterval != "" {
		if duration, err := time.ParseDuration(envInterval); err == nil {
			cfg.Messaging.Interval = duration
		}
	}
	if envBatchSize := os.Getenv(envPrefix + "MESSAGING_BATCH_SIZE"); envBatchSize != "" {
		fmt.Sscanf(envBatchSize, "%d", &cfg.Messaging.BatchSize)
	}
	if envMaxRetries := os.Getenv(envPrefix + "MESSAGING_MAX_RETRIES"); envMaxRetries != "" {
		fmt.Sscanf(envMaxRetries, "%d", &cfg.Messaging.MaxRetries)
	}
	if envRetryDelay := os.Getenv(envPrefix + "MESSAGING_RETRY_DELAY"); envRetryDelay != "" {
		if duration, err := time.ParseDuration(envRetryDelay); err == nil {
			cfg.Messaging.RetryDelay = duration
		}
	}
}

func (cfg *Cfg) SetDB(db *bun.DB) *Cfg {
	cfg.Database.DB = db
	return cfg
}

func Log() *logrus.Logger {
	if Logger != nil {
		return Logger
	}

	Logger = logrus.New()
	Logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	Logger.Hooks.Add(filename.NewHook())
	return Logger
}

func (cfg *Cfg) validate() error {
	if cfg.Server.Mode != ModeProd && cfg.Server.Mode != ModeDev {
		return fmt.Errorf("server mode is required: %s is not a valid mode", cfg.Server.Mode)
	}

	if cfg.Database.DSN == "" {
		return fmt.Errorf("database DSN is required")
	}

	return nil
}
