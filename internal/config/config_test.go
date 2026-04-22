package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadRejectsMissingAccountSecrets(t *testing.T) {
	raw := []byte(`
system:
  log_level: info
accounts:
  - name: primary
    market_types: ["spot"]
`)

	_, err := LoadBytes(raw)
	require.Error(t, err)
	require.Contains(t, err.Error(), "api_key_env")
}

func TestLoadFileReadsYAMLConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
system:
  log_level: info
accounts:
  - name: primary
    market_types: ["spot"]
    api_key_env: BINANCE_API_KEY
    secret_key_env: BINANCE_SECRET_KEY
`), 0o600))

	cfg, err := LoadFile(path)

	require.NoError(t, err)
	require.Equal(t, "primary", cfg.Accounts[0].Name)
}
