package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Domain struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

type Config struct {
	DefaultDomain string            `json:"default_account"`
	Domains       map[string]Domain `json:"accounts"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pintomind", "config.json"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{Domains: make(map[string]Domain)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if cfg.Domains == nil {
		cfg.Domains = make(map[string]Domain)
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ActiveDomain returns the domain config to use, respecting an override.
func (c *Config) ActiveDomain(override string) (string, Domain, error) {
	name := override
	if name == "" {
		name = c.DefaultDomain
	}
	if name == "" {
		return "", Domain{}, fmt.Errorf("no account configured — run: pintomind config add app.infoskjermen.no <api-key>")
	}
	d, ok := c.Domains[name]
	if !ok {
		return "", Domain{}, fmt.Errorf("account %q not found in config", name)
	}
	return name, d, nil
}
