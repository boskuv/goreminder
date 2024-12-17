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
	assert.Equal(t, 30, config.Config.Database.MaxLifetime)
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
	assert.Contains(t, err.Error(), "unable to decode into struct")
}
