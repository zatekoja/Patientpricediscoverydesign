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
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_MODEL")

	cfg, err := Load()
	assert.NoError(t, err)

	// Verify defaults
	assert.Equal(t, "http://localhost:8108", cfg.Typesense.URL)
	assert.Equal(t, "xyz", cfg.Typesense.APIKey)
	assert.Equal(t, "", cfg.OpenAI.APIKey)
	assert.Equal(t, "gpt-4o-mini", cfg.OpenAI.Model)
}

func TestLoad_OpenAIConfig(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-openai-key")
	os.Setenv("OPENAI_MODEL", "gpt-4o-mini")
	defer func() {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("OPENAI_MODEL")
	}()

	cfg, err := Load()
	assert.NoError(t, err)

	assert.Equal(t, "test-openai-key", cfg.OpenAI.APIKey)
	assert.Equal(t, "gpt-4o-mini", cfg.OpenAI.Model)
}
