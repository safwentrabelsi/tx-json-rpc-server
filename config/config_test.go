package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	os.Clearenv()

	t.Run("when NETWORK and INFURA_PROJECT_ID are not set, return error", func(t *testing.T) {
		err := LoadConfig()
		require.Error(t, err)
	})

	t.Run("when NETWORK and INFURA_PROJECT_ID are set, load config without error", func(t *testing.T) {
		os.Setenv("NETWORK", "test_network")
		os.Setenv("INFURA_PROJECT_ID", "test_project_id")

		err := LoadConfig()
		require.NoError(t, err)
		require.Equal(t, "test_network", GetConfig().Network())
		require.Equal(t, "test_project_id", GetConfig().InfuraKey())
	})

	t.Run("when optional env variables are not set, load config with default values", func(t *testing.T) {
		os.Setenv("NETWORK", "test_network")
		os.Setenv("INFURA_PROJECT_ID", "test_project_id")

		err := LoadConfig()
		require.NoError(t, err)

		cfg := GetConfig()

		require.Equal(t, "INFO", cfg.LogLevel())
		require.Equal(t, "localhost:8080", cfg.Addr())
	})

	t.Run("when optional env variables are set, load config with those values", func(t *testing.T) {
		os.Setenv("NETWORK", "test_network")
		os.Setenv("INFURA_PROJECT_ID", "test_project_id")
		os.Setenv("LOG_LEVEL", "DEBUG")
		os.Setenv("HOST", "test_host")
		os.Setenv("PORT", "9090")

		err := LoadConfig()
		require.NoError(t, err)

		cfg := GetConfig()

		require.Equal(t, "DEBUG", cfg.LogLevel())
		require.Equal(t, "test_host:9090", cfg.Addr())
	})
}
