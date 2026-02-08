//go:build integration_vault

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/secrets"
)

func TestVaultSecretsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	addr := firstNonEmpty(os.Getenv("TEST_VAULT_ADDR"), os.Getenv("VAULT_ADDR"))
	token := firstNonEmpty(os.Getenv("TEST_VAULT_TOKEN"), os.Getenv("VAULT_TOKEN"))
	mount := firstNonEmpty(os.Getenv("TEST_VAULT_MOUNT"), "secret")

	if addr == "" || token == "" {
		t.Skip("Vault integration test requires TEST_VAULT_ADDR/TEST_VAULT_TOKEN")
	}

	if !vaultReady(addr) {
		t.Skip("Vault not reachable")
	}

	path := fmt.Sprintf("patient-price-discovery/tests/%d", time.Now().UnixNano())
	data := map[string]string{
		"OPENAI_API_KEY":        "vault-test-openai",
		"GEOLOCATION_API_KEY":   "vault-test-geo",
		"PROVIDER_LLM_API_KEY":  "vault-test-provider",
		"PROVIDER_LLM_ENDPOINT": "https://example.com",
	}

	err := writeVaultSecret(addr, token, mount, path, data)
	require.NoError(t, err)

	prevOpenAI := os.Getenv("OPENAI_API_KEY")
	prevGeo := os.Getenv("GEOLOCATION_API_KEY")
	prevProvider := os.Getenv("PROVIDER_LLM_API_KEY")
	defer restoreEnv("OPENAI_API_KEY", prevOpenAI)
	defer restoreEnv("GEOLOCATION_API_KEY", prevGeo)
	defer restoreEnv("PROVIDER_LLM_API_KEY", prevProvider)

	cfg := secrets.VaultConfig{
		Enabled:   true,
		Addr:      addr,
		Token:     token,
		Mount:     mount,
		Path:      path,
		KVVersion: 2,
		Timeout:   3 * time.Second,
		Overwrite: true,
	}

	result, err := secrets.ApplyVaultSecrets(context.Background(), cfg)
	require.NoError(t, err)
	require.GreaterOrEqual(t, result.Loaded, 3)
	require.Equal(t, "vault-test-openai", os.Getenv("OPENAI_API_KEY"))
	require.Equal(t, "vault-test-geo", os.Getenv("GEOLOCATION_API_KEY"))
	require.Equal(t, "vault-test-provider", os.Getenv("PROVIDER_LLM_API_KEY"))
}

func vaultReady(addr string) bool {
	client := http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(addr, "/")+"/v1/sys/health", nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusTooManyRequests
}

func writeVaultSecret(addr, token, mount, path string, data map[string]string) error {
	payload := map[string]interface{}{
		"data": data,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	base := strings.TrimRight(addr, "/")
	cleanMount := strings.Trim(mount, "/")
	cleanPath := strings.TrimLeft(path, "/")
	url := fmt.Sprintf("%s/v1/%s/data/%s", base, cleanMount, cleanPath)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vault-Token", token)

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("vault write failed: %s", resp.Status)
	}
	return nil
}

func restoreEnv(key, value string) {
	if value == "" {
		_ = os.Unsetenv(key)
	} else {
		_ = os.Setenv(key, value)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
