package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_TypesenseConfig(t *testing.T) {
	// Setup environment variables
	os.Setenv("TYPESENSE_URL", "http://test-typesense:8108")
	os.Setenv("TYPESENSE_API_KEY", "test-key")
	defer func() {
		os.Unsetenv("TYPESENSE_URL")
		os.Unsetenv("TYPESENSE_API_KEY")
	}()

	// Load config
	cfg, err := Load()
	assert.NoError(t, err)

	// Verify Typesense config
	assert.Equal(t, "http://test-typesense:8108", cfg.Typesense.URL)
	assert.Equal(t, "test-key", cfg.Typesense.APIKey)
}

func TestLoad_Defaults(t *testing.T) {
	// Ensure env vars are cleared
	os.Unsetenv("TYPESENSE_URL")
	os.Unsetenv("TYPESENSE_API_KEY")

	cfg, err := Load()
	assert.NoError(t, err)

	// Verify defaults
	assert.Equal(t, "http://localhost:8108", cfg.Typesense.URL)
	assert.Equal(t, "xyz", cfg.Typesense.APIKey)
}
