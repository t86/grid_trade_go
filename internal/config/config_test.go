package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadSupportsSecretRefAccounts(t *testing.T) {
	raw := []byte(`
system:
  log_level: info
  secret_dir: /etc/grid-trade/accounts
accounts:
  - name: primary
    enabled: true
    market_types: ["spot", "futures_um"]
    secret_ref: primary
`)

	cfg, err := LoadBytes(raw)

	require.NoError(t, err)
	require.Equal(t, "/etc/grid-trade/accounts", cfg.System.SecretDir)
	require.True(t, cfg.Accounts[0].Enabled)
	require.Equal(t, "primary", cfg.Accounts[0].SecretRef)
}

func TestLoadRejectsMissingSecretRef(t *testing.T) {
	raw := []byte(`
accounts:
  - name: primary
    enabled: true
    market_types: ["spot"]
`)

	_, err := LoadBytes(raw)
	require.Error(t, err)
	require.Contains(t, err.Error(), "secret_ref")
}

func TestLoadFileReadsYAMLConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
system:
  log_level: info
  secret_dir: /etc/grid-trade/accounts
accounts:
  - name: primary
    enabled: true
    market_types: ["spot"]
    secret_ref: primary
`), 0o600))

	cfg, err := LoadFile(path)

	require.NoError(t, err)
	require.Equal(t, "primary", cfg.Accounts[0].Name)
}
