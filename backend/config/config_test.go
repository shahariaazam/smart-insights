package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Test default values
	t.Run("default values", func(t *testing.T) {
		cfg, err := Load()
		assert.NoError(t, err)
		assert.Equal(t, 8080, cfg.Port)
		assert.Equal(t, "development", cfg.Env)
	})

	// Test custom values
	t.Run("custom values", func(t *testing.T) {
		os.Setenv("PORT", "9090")
		os.Setenv("ENV", "production")
		defer func() {
			os.Unsetenv("PORT")
			os.Unsetenv("ENV")
		}()

		cfg, err := Load()
		assert.NoError(t, err)
		assert.Equal(t, 9090, cfg.Port)
		assert.Equal(t, "production", cfg.Env)
	})

	// Test invalid port
	t.Run("invalid port", func(t *testing.T) {
		os.Setenv("PORT", "invalid")
		defer os.Unsetenv("PORT")

		_, err := Load()
		assert.Error(t, err)
	})
}
