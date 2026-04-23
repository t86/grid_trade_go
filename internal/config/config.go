package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	System     SystemConfig     `yaml:"system"`
	Accounts   []AccountConfig  `yaml:"accounts"`
	Strategies []StrategyConfig `yaml:"strategies"`
}

type SystemConfig struct {
	LogLevel  string `yaml:"log_level"`
	HTTPAddr  string `yaml:"http_addr"`
	SecretDir string `yaml:"secret_dir"`
}

type AccountConfig struct {
	Name        string   `yaml:"name"`
	Enabled     bool     `yaml:"enabled"`
	MarketTypes []string `yaml:"market_types"`
	SecretRef   string   `yaml:"secret_ref"`
}

type StrategyConfig struct {
	ID     string `yaml:"id"`
	Type   string `yaml:"type"`
	Symbol string `yaml:"symbol"`
	Market string `yaml:"market"`
}

func LoadBytes(raw []byte) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func LoadFile(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	return LoadBytes(raw)
}

func (c Config) Validate() error {
	if len(c.Accounts) == 0 {
		return errors.New("at least one account is required")
	}
	names := map[string]struct{}{}
	for _, account := range c.Accounts {
		if account.Name == "" {
			return errors.New("account name is required")
		}
		if _, exists := names[account.Name]; exists {
			return fmt.Errorf("duplicate account name %q", account.Name)
		}
		names[account.Name] = struct{}{}
		if !account.Enabled {
			continue
		}
		if account.SecretRef == "" {
			return errors.New("secret_ref is required")
		}
		if len(account.MarketTypes) == 0 {
			return errors.New("market_types is required")
		}
	}
	return nil
}
