package config

import (
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
