package config

import (
	"fmt"
	"time"

	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

var Config *Configuration

var validate = validator.New()

type Configuration struct {
	Server    ServerConfiguration    `mapstructure:"server"`
	Database  DatabaseConfiguration  `mapstructure:"database"`
	Producer  ProducerConfiguration  `mapstructure:"producer"`
	Tracing   TracingConfiguration   `mapstructure:"tracing"`
	Metrics   MetricsConfiguration   `mapstructure:"metrics"`
	RateLimit RateLimitConfiguration `mapstructure:"ratelimit"`
	Cors      CorsConfiguration      `mapstructure:"cors"`
}

type DatabaseConfiguration struct {
	Driver          string `mapstructure:"driver" validate:"required,oneof=postgres"`
	Dbname          string `mapstructure:"dbname" validate:"required"`
	Username        string `mapstructure:"username" validate:"required"`
	Password        string `mapstructure:"password" validate:"required"`
	Host            string `mapstructure:"host" validate:"required,hostname|ip"`
	Port            string `mapstructure:"port" validate:"required,numeric"`
	MaxOpenConns    int    `mapstructure:"maxOpenConns" default:"20" validate:"gte=1"`
	MaxIdleConns    int    `mapstructure:"maxIdleConns" default:"10" validate:"gte=0"`
	ConnMaxLifetime string `mapstructure:"connMaxLifetime" default:"30m"`
	MaxRetries      int    `mapstructure:"maxRetries" default:"3" validate:"gte=0"`
	// Legacy support: if provided as seconds in config under key maxLifetime
	LegacyMaxLifetimeSeconds int `mapstructure:"maxLifetime"`
}

type ProducerConfiguration struct {
	Host                 string `mapstructure:"host"`
	Port                 string `mapstructure:"port"`
	User                 string `mapstructure:"user"`
	Password             string `mapstructure:"password"`
	QueueName            string `mapstructure:"queueName"`
	Exchange             string `mapstructure:"exchange"`
	ConnectionRetries    int    `mapstructure:"connectionRetries" default:"5" validate:"gte=0"`
	ConnectionRetryDelay int    `mapstructure:"connectionRetryDelay" default:"2" validate:"gte=0"`
}

type TracingConfiguration struct {
	Enabled     bool   `mapstructure:"enabled"`
	Endpoint    string `mapstructure:"endpoint"`
	ServiceName string `mapstructure:"serviceName" default:"goreminder"`
	Insecure    bool   `mapstructure:"insecure"`
}

type MetricsConfiguration struct {
	Enabled bool   `mapstructure:"enabled"`
	Addr    string `mapstructure:"addr" default:":9090"`
}

type ServerConfiguration struct {
	Port   string `mapstructure:"port" default:"8080" validate:"required,numeric"`
	Secret string `mapstructure:"secret" default:"dev-secret"`
	Mode   string `mapstructure:"mode" default:"development" validate:"oneof=development production test"`
}

type RateLimitConfiguration struct {
	Enabled  bool   `mapstructure:"enabled" default:"false"`
	Requests int    `mapstructure:"requests" default:"100" validate:"gte=1"`
	Window   string `mapstructure:"window" default:"1m" validate:"required"`
}

type CorsConfiguration struct {
	Enabled          bool     `mapstructure:"enabled" default:"false"`
	AllowOrigins     []string `mapstructure:"allowOrigins" default:"[\"*\"]"`
	AllowMethods     []string `mapstructure:"allowMethods"`
	AllowHeaders     []string `mapstructure:"allowHeaders"`
	ExposeHeaders    []string `mapstructure:"exposeHeaders"`
	AllowCredentials bool     `mapstructure:"allowCredentials" default:"false"`
	MaxAge           int      `mapstructure:"maxAge" default:"3600" validate:"gte=0"`
}

// Setup configuration
func Setup(configPath string) error {
	var configuration Configuration

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	// Apply defaults to configuration before unmarshal so zero-values are prefilled
	if err := defaults.Set(&configuration); err != nil {
		return fmt.Errorf("unable to set default configuration values: %w", err)
	}

	// Unmarshal user-provided config
	err := viper.Unmarshal(&configuration)
	if err != nil {
		return fmt.Errorf("unable to decode into struct: %w", err)
	}

	// Additional derived/default logic not handled by tags
	// Ensure DB ConnMaxLifetime is a valid duration string
	if configuration.Database.ConnMaxLifetime == "" && configuration.Database.LegacyMaxLifetimeSeconds > 0 {
		configuration.Database.ConnMaxLifetime = fmt.Sprintf("%ds", configuration.Database.LegacyMaxLifetimeSeconds)
	}
	if configuration.Database.ConnMaxLifetime != "" {
		if _, err := time.ParseDuration(configuration.Database.ConnMaxLifetime); err != nil {
			return fmt.Errorf("invalid database.connMaxLifetime duration: %w", err)
		}
	}

	// Validate rate limit window duration
	if configuration.RateLimit.Window != "" {
		if _, err := time.ParseDuration(configuration.RateLimit.Window); err != nil {
			return fmt.Errorf("invalid ratelimit.window duration: %w", err)
		}
	}

	// Validate the resulting configuration
	if err := validate.Struct(configuration); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Save final config
	Config = &configuration

	return nil
}

// Get configuration data
func GetConfig() *Configuration {
	return Config
}
