package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const AuthOAuth = "oauth"

type Tokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type Domain struct {
	APIKey  string `json:"api_key,omitempty"`
	BaseURL string `json:"base_url"`

	Auth        string `json:"auth,omitempty"`
	ClientID    string `json:"client_id,omitempty"`
	AccountID   int    `json:"account_id,omitempty"`
	AccountName string `json:"account_name,omitempty"`
	NetworkID   int    `json:"network_id,omitempty"`
	NetworkName string `json:"network_name,omitempty"`
	// The OS keychain item holding the tokens; empty when they are kept in Tokens.
	KeychainItem string  `json:"keychain_item,omitempty"`
	Tokens       *Tokens `json:"tokens,omitempty"`
}

func (d Domain) IsOAuth() bool {
	return d.Auth == AuthOAuth
}

func (d Domain) ActingFor() string {
	switch {
	case d.NetworkID != 0:
		return fmt.Sprintf("network %s (%d)", nameOr(d.NetworkName), d.NetworkID)
	case d.AccountID != 0:
		return fmt.Sprintf("account %s (%d)", nameOr(d.AccountName), d.AccountID)
	}
	return ""
}

func nameOr(name string) string {
	if name == "" {
		return "?"
	}
	return name
}

type Config struct {
	DefaultDomain string            `json:"default_account"`
	Domains       map[string]Domain `json:"accounts"`
}

func Dir() (string, error) {
	if dir := os.Getenv("PINTOMIND_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pintomind"), nil
}

func configPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
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
	// Write and rename so a parallel pintomind process never reads half a file.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Update reloads the config, applies fn and saves it, holding the config lock so concurrent
// pintomind processes don't overwrite each other's changes.
func Update(fn func(*Config) error) (*Config, error) {
	var cfg *Config
	err := WithLock("config", func() error {
		var err error
		if cfg, err = Load(); err != nil {
			return err
		}
		if err := fn(cfg); err != nil {
			return err
		}
		return cfg.Save()
	})
	return cfg, err
}

// ActiveDomain returns the domain config to use, respecting an override.
func (c *Config) ActiveDomain(override string) (string, Domain, error) {
	name := override
	if name == "" {
		name = c.DefaultDomain
	}
	if name == "" {
		return "", Domain{}, fmt.Errorf("no connection configured — run: pintomind login (or pintomind connection add <name> <api-key>)")
	}
	d, ok := c.Domains[name]
	if !ok {
		return "", Domain{}, fmt.Errorf("connection %q not found in config", name)
	}
	return name, d, nil
}
