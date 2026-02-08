package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type VaultConfig struct {
	Enabled   bool
	Addr      string
	Token     string
	Namespace string
	Mount     string
	Path      string
	KVVersion int
	Timeout   time.Duration
	Overwrite bool
}

type VaultResult struct {
	Enabled bool
	Path    string
	Loaded  int
	Skipped int
}

func LoadVaultConfigFromEnv(pathOverride string) VaultConfig {
	enabled := strings.EqualFold(os.Getenv("VAULT_ENABLED"), "true")
	mount := os.Getenv("VAULT_MOUNT")
	if mount == "" {
		mount = "secret"
	}
	kvVersion := 2
	if val := os.Getenv("VAULT_KV_VERSION"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			kvVersion = parsed
		}
	}
	path := pathOverride
	if path == "" {
		path = os.Getenv("VAULT_PATH")
	}
	timeout := 5 * time.Second
	if val := os.Getenv("VAULT_TIMEOUT_MS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			timeout = time.Duration(parsed) * time.Millisecond
		}
	}
	overwrite := strings.EqualFold(os.Getenv("VAULT_OVERWRITE"), "true")

	return VaultConfig{
		Enabled:   enabled,
		Addr:      os.Getenv("VAULT_ADDR"),
		Token:     os.Getenv("VAULT_TOKEN"),
		Namespace: os.Getenv("VAULT_NAMESPACE"),
		Mount:     mount,
		Path:      path,
		KVVersion: kvVersion,
		Timeout:   timeout,
		Overwrite: overwrite,
	}
}

func ApplyVaultSecrets(ctx context.Context, cfg VaultConfig) (VaultResult, error) {
	if !cfg.Enabled {
		return VaultResult{Enabled: false}, nil
	}

	if cfg.Addr == "" || cfg.Token == "" || cfg.Path == "" {
		return VaultResult{Enabled: true, Path: cfg.Path}, errors.New("vault configuration incomplete (VAULT_ADDR, VAULT_TOKEN, VAULT_PATH)")
	}

	url, err := buildVaultURL(cfg.Addr, cfg.Mount, cfg.Path, cfg.KVVersion)
	if err != nil {
		return VaultResult{Enabled: true, Path: cfg.Path}, err
	}

	client := &http.Client{Timeout: cfg.Timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return VaultResult{Enabled: true, Path: cfg.Path}, err
	}
	req.Header.Set("X-Vault-Token", cfg.Token)
	if cfg.Namespace != "" {
		req.Header.Set("X-Vault-Namespace", cfg.Namespace)
	}

	resp, err := client.Do(req)
	if err != nil {
		return VaultResult{Enabled: true, Path: cfg.Path}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return VaultResult{Enabled: true, Path: cfg.Path}, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return VaultResult{Enabled: true, Path: cfg.Path}, fmt.Errorf("vault fetch failed: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return VaultResult{Enabled: true, Path: cfg.Path}, err
	}

	data, err := extractVaultData(payload, cfg.KVVersion)
	if err != nil {
		return VaultResult{Enabled: true, Path: cfg.Path}, err
	}

	loaded := 0
	skipped := 0
	for key, value := range data {
		if !cfg.Overwrite && os.Getenv(key) != "" {
			skipped++
			continue
		}
		if err := os.Setenv(key, stringifyVaultValue(value)); err != nil {
			return VaultResult{Enabled: true, Path: cfg.Path, Loaded: loaded, Skipped: skipped}, err
		}
		loaded++
	}

	return VaultResult{
		Enabled: true,
		Path:    cfg.Path,
		Loaded:  loaded,
		Skipped: skipped,
	}, nil
}

func buildVaultURL(addr, mount, path string, kvVersion int) (string, error) {
	addr = strings.TrimRight(addr, "/")
	mount = strings.Trim(mount, "/")
	path = strings.TrimLeft(path, "/")
	if addr == "" || mount == "" || path == "" {
		return "", errors.New("vault address, mount, and path must be set")
	}
	if kvVersion == 1 {
		return fmt.Sprintf("%s/v1/%s/%s", addr, mount, path), nil
	}
	return fmt.Sprintf("%s/v1/%s/data/%s", addr, mount, path), nil
}

func extractVaultData(payload map[string]interface{}, kvVersion int) (map[string]interface{}, error) {
	if kvVersion == 1 {
		if data, ok := payload["data"].(map[string]interface{}); ok {
			return data, nil
		}
		return nil, errors.New("vault response missing data for KV v1")
	}

	if data, ok := payload["data"].(map[string]interface{}); ok {
		if inner, ok := data["data"].(map[string]interface{}); ok {
			return inner, nil
		}
	}
	return nil, errors.New("vault response missing data for KV v2")
}

func stringifyVaultValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case nil:
		return ""
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(encoded)
	}
}
