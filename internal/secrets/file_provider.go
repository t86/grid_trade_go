package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileProvider struct {
	baseDir string
}

func NewFileProvider(baseDir string) FileProvider {
	return FileProvider{baseDir: baseDir}
}

func (p FileProvider) Load(_ context.Context, ref string) (AccountSecret, error) {
	if ref == "" {
		return AccountSecret{}, errors.New("secret load failed: ref is required")
	}
	if filepath.Base(ref) != ref || strings.Contains(ref, "..") {
		return AccountSecret{}, errors.New("secret load failed: invalid ref")
	}

	raw, err := os.ReadFile(filepath.Join(p.baseDir, ref+".json"))
	if err != nil {
		return AccountSecret{}, fmt.Errorf("secret load failed: %w", err)
	}

	var payload struct {
		APIKey    string `json:"api_key"`
		SecretKey string `json:"secret_key"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return AccountSecret{}, fmt.Errorf("secret load failed: %w", err)
	}
	if payload.APIKey == "" {
		return AccountSecret{}, errors.New("secret load failed: api_key is required")
	}
	if payload.SecretKey == "" {
		return AccountSecret{}, errors.New("secret load failed: secret_key is required")
	}

	return AccountSecret{
		APIKey:    payload.APIKey,
		SecretKey: payload.SecretKey,
	}, nil
}
