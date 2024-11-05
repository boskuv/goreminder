package config

import (
	"fmt"

	"github.com/spf13/viper"
)

var Config *Configuration

type Configuration struct {
	Server   ServerConfiguration
	Database DatabaseConfiguration
}

type DatabaseConfiguration struct {
	Driver       string
	Dbname       string
	Username     string
	Password     string
	Host         string
	Port         string
	MaxLifetime  int
	MaxOpenConns int
	MaxIdleConns int
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
