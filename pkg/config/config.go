package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

var Config *Configuration

type Configuration struct {
	Server   ServerConfiguration
	Database DatabaseConfiguration
	Producer ProducerConfiguration
	Tracing  TracingConfiguration
	Metrics  MetricsConfiguration
}

type DatabaseConfiguration struct {
	Driver          string
	Dbname          string
	Username        string // TODO: simplify to User
	Password        string
	Host            string
	Port            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime string //`yaml:"conn_max_lifetime" env:"POSTGRES_CONN_MAX_LIFETIME"`
	MaxRetries      int    //`yaml:"max_retries" env:"MAX_RETRIES"`
}

type ProducerConfiguration struct {
	Host                 string
	Port                 string // TODO: int?
	User                 string
	Password             string
	QueueName            string
	Exchange             string
	ConnectionRetries    int
	ConnectionRetryDelay time.Duration
}

type TracingConfiguration struct {
	Enabled     bool
	Endpoint    string
	ServiceName string
	Insecure    bool
}

type MetricsConfiguration struct {
	Enabled bool
	Addr    string
}

type ServerConfiguration struct {
	Port   string
	Secret string
	Mode   string
}

// Setup configuration
func Setup(configPath string) error {
	var configuration *Configuration

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	err := viper.Unmarshal(&configuration)
	if err != nil {
		return fmt.Errorf("unable to decode into struct: %w", err)
	}

	Config = configuration

	return nil
}

// Get configuration data
func GetConfig() *Configuration {
	return Config
}
