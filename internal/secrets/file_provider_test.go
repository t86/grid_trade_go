package secrets

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileProviderLoadsAccountSecret(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "primary.json"), []byte(`{
  "api_key": "api",
  "secret_key": "secret"
}`), 0o600))

	provider := NewFileProvider(dir)

	secret, err := provider.Load(context.Background(), "primary")

	require.NoError(t, err)
	require.Equal(t, "api", secret.APIKey)
	require.Equal(t, "secret", secret.SecretKey)
}

func TestFileProviderRejectsMissingSecretFields(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{"api_key":"api"}`), 0o600))

	provider := NewFileProvider(dir)

	_, err := provider.Load(context.Background(), "bad")

	require.ErrorContains(t, err, "secret_key")
	require.NotContains(t, err.Error(), "api")
}
