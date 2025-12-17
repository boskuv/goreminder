package config_test

import (
	"os"
	"testing"

	"github.com/boskuv/goreminder/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestSetup_Success(t *testing.T) {
	// Prepare a temporary config file
	content := []byte(`
server:
  port: "8080"
  secret: "testsecret"
  mode: "development"
database:
  driver: "postgres"
  dbname: "testdb"
  username: "testuser"
  password: "testpass"
  host: "localhost"
  port: "5432"
  maxLifetime: 30
  maxOpenConns: 10
  maxIdleConns: 5
`)

	tempFile, err := os.CreateTemp("", "testconfig*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(content)
	assert.NoError(t, err)
	tempFile.Close()

	// Call Setup with the test config file
	err = config.Setup(tempFile.Name())
	assert.NoError(t, err)
	assert.NotNil(t, config.Config)

	// Validate ServerConfiguration
	assert.Equal(t, "8080", config.Config.Server.Port)
	assert.Equal(t, "testsecret", config.Config.Server.Secret)
	assert.Equal(t, "development", config.Config.Server.Mode)

	// Validate DatabaseConfiguration
	assert.Equal(t, "postgres", config.Config.Database.Driver)
	assert.Equal(t, "testdb", config.Config.Database.Dbname)
	assert.Equal(t, "testuser", config.Config.Database.Username)
	assert.Equal(t, "testpass", config.Config.Database.Password)
	assert.Equal(t, "localhost", config.Config.Database.Host)
	assert.Equal(t, "5432", config.Config.Database.Port)
	assert.Equal(t, 30, config.Config.Database.LegacyMaxLifetimeSeconds)
	assert.Equal(t, 10, config.Config.Database.MaxOpenConns)
	assert.Equal(t, 5, config.Config.Database.MaxIdleConns)
}

func TestSetup_InvalidPath(t *testing.T) {
	// Test with an invalid file path
	err := config.Setup("nonexistent.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reading config file")
}

func TestSetup_InvalidFormat(t *testing.T) {
	// Create a temporary file with invalid format
	tempFile, err := os.CreateTemp("", "invalidconfig*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write([]byte("invalid yaml content"))
	assert.NoError(t, err)
	tempFile.Close()

	// Call Setup with the invalid config file
	err = config.Setup(tempFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal errors")
}

func TestSetup_EnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("GOREMINDER_SERVER_PORT", "9090")
	os.Setenv("GOREMINDER_SERVER_SECRET", "env-secret")
	os.Setenv("GOREMINDER_DATABASE_HOST", "env-host")
	os.Setenv("GOREMINDER_DATABASE_PORT", "5433")
	defer func() {
		os.Unsetenv("GOREMINDER_SERVER_PORT")
		os.Unsetenv("GOREMINDER_SERVER_SECRET")
		os.Unsetenv("GOREMINDER_DATABASE_HOST")
		os.Unsetenv("GOREMINDER_DATABASE_PORT")
	}()

	// Prepare a temporary config file with different values
	content := []byte(`
server:
  port: "8080"
  secret: "yaml-secret"
  mode: "development"
database:
  driver: "postgres"
  dbname: "testdb"
  username: "testuser"
  password: "testpass"
  host: "yaml-host"
  port: "5432"
  maxOpenConns: 10
  maxIdleConns: 5
`)

	tempFile, err := os.CreateTemp("", "testconfig*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(content)
	assert.NoError(t, err)
	tempFile.Close()

	// Call Setup - env variables should override YAML values
	err = config.Setup(tempFile.Name())
	assert.NoError(t, err)
	assert.NotNil(t, config.Config)

	// Environment variables should override YAML values
	assert.Equal(t, "9090", config.Config.Server.Port)         // from env
	assert.Equal(t, "env-secret", config.Config.Server.Secret) // from env
	assert.Equal(t, "env-host", config.Config.Database.Host)   // from env
	assert.Equal(t, "5433", config.Config.Database.Port)       // from env

	// Values not set in env should come from YAML
	assert.Equal(t, "development", config.Config.Server.Mode)    // from yaml
	assert.Equal(t, "testdb", config.Config.Database.Dbname)     // from yaml
	assert.Equal(t, "testuser", config.Config.Database.Username) // from yaml
}

func TestSetup_EnvironmentVariablesOnly(t *testing.T) {
	// Set all required environment variables
	os.Setenv("GOREMINDER_SERVER_PORT", "8080")
	os.Setenv("GOREMINDER_SERVER_SECRET", "env-secret")
	os.Setenv("GOREMINDER_SERVER_MODE", "production")
	os.Setenv("GOREMINDER_DATABASE_DRIVER", "postgres")
	os.Setenv("GOREMINDER_DATABASE_DBNAME", "env-db")
	os.Setenv("GOREMINDER_DATABASE_USERNAME", "env-user")
	os.Setenv("GOREMINDER_DATABASE_PASSWORD", "env-pass")
	os.Setenv("GOREMINDER_DATABASE_HOST", "env-host")
	os.Setenv("GOREMINDER_DATABASE_PORT", "5432")
	defer func() {
		os.Unsetenv("GOREMINDER_SERVER_PORT")
		os.Unsetenv("GOREMINDER_SERVER_SECRET")
		os.Unsetenv("GOREMINDER_SERVER_MODE")
		os.Unsetenv("GOREMINDER_DATABASE_DRIVER")
		os.Unsetenv("GOREMINDER_DATABASE_DBNAME")
		os.Unsetenv("GOREMINDER_DATABASE_USERNAME")
		os.Unsetenv("GOREMINDER_DATABASE_PASSWORD")
		os.Unsetenv("GOREMINDER_DATABASE_HOST")
		os.Unsetenv("GOREMINDER_DATABASE_PORT")
	}()

	// Call Setup with empty config path - should work with env only
	err := config.Setup("")
	assert.NoError(t, err)
	assert.NotNil(t, config.Config)

	// All values should come from environment
	assert.Equal(t, "8080", config.Config.Server.Port)
	assert.Equal(t, "env-secret", config.Config.Server.Secret)
	assert.Equal(t, "production", config.Config.Server.Mode)
	assert.Equal(t, "postgres", config.Config.Database.Driver)
	assert.Equal(t, "env-db", config.Config.Database.Dbname)
	assert.Equal(t, "env-user", config.Config.Database.Username)
	assert.Equal(t, "env-pass", config.Config.Database.Password)
	assert.Equal(t, "env-host", config.Config.Database.Host)
	assert.Equal(t, "5432", config.Config.Database.Port)
}
