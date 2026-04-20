package config

import (
	"errors"

	"gopkg.in/yaml.v3"
)

type Config struct {
	System     SystemConfig     `yaml:"system"`
	Accounts   []AccountConfig  `yaml:"accounts"`
	Strategies []StrategyConfig `yaml:"strategies"`
}

type SystemConfig struct {
	LogLevel string `yaml:"log_level"`
	HTTPAddr string `yaml:"http_addr"`
}

type AccountConfig struct {
	Name         string   `yaml:"name"`
	MarketTypes  []string `yaml:"market_types"`
	APIKeyEnv    string   `yaml:"api_key_env"`
	SecretKeyEnv string   `yaml:"secret_key_env"`
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

func (c Config) Validate() error {
	if len(c.Accounts) == 0 {
		return errors.New("at least one account is required")
	}
	for _, account := range c.Accounts {
		if account.APIKeyEnv == "" {
			return errors.New("api_key_env is required")
		}
		if account.SecretKeyEnv == "" {
			return errors.New("secret_key_env is required")
		}
	}
	return nil
}
