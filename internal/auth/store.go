package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/zalando/go-keyring"

	"favo/pintomind-cli/internal/config"
)

const keyringService = "pintomind-cli"

func keyringDisabled() bool {
	return os.Getenv("PINTOMIND_NO_KEYRING") != ""
}

// LoadTokens reads the tokens of an OAuth connection from the keychain or the config.
func LoadTokens(d config.Domain) (config.Tokens, error) {
	if d.KeychainItem != "" {
		secret, err := keyring.Get(keyringService, d.KeychainItem)
		if err != nil {
			if errors.Is(err, keyring.ErrNotFound) {
				return config.Tokens{}, errNotLoggedIn
			}
			return config.Tokens{}, fmt.Errorf("reading the login from the keychain: %w", err)
		}
		var t config.Tokens
		if err := json.Unmarshal([]byte(secret), &t); err != nil {
			return config.Tokens{}, fmt.Errorf("decoding the login from the keychain: %w", err)
		}
		return t, nil
	}
	if d.Tokens == nil || d.Tokens.RefreshToken == "" {
		return config.Tokens{}, errNotLoggedIn
	}
	return *d.Tokens, nil
}

var errNotLoggedIn = errors.New("not logged in")

// SaveTokens stores the tokens of connection name: in the OS keychain when one is available,
// else in config.json (mode 0600).
func SaveTokens(name string, t config.Tokens) error {
	secret, err := json.Marshal(t)
	if err != nil {
		return err
	}
	_, err = config.Update(func(cfg *config.Config) error {
		d, ok := cfg.Domains[name]
		if !ok {
			return fmt.Errorf("connection %q not found", name)
		}
		item := name
		if !keyringDisabled() && keyring.Set(keyringService, item, string(secret)) == nil {
			d.KeychainItem = item
			d.Tokens = nil
		} else {
			if d.KeychainItem != "" {
				_ = keyring.Delete(keyringService, d.KeychainItem)
			}
			d.KeychainItem = ""
			d.Tokens = &t
		}
		cfg.Domains[name] = d
		return nil
	})
	return err
}

// DeleteTokens forgets the login of connection name, keeping the connection itself.
func DeleteTokens(name string) error {
	_, err := config.Update(func(cfg *config.Config) error {
		d, ok := cfg.Domains[name]
		if !ok {
			return nil
		}
		removeKeychainItem(d)
		d.KeychainItem = ""
		d.Tokens = nil
		cfg.Domains[name] = d
		return nil
	})
	return err
}

func removeKeychainItem(d config.Domain) {
	if d.KeychainItem != "" {
		_ = keyring.Delete(keyringService, d.KeychainItem)
	}
}

func StorageDescription(d config.Domain) string {
	switch {
	case d.KeychainItem != "":
		return "OS keychain"
	case d.Tokens != nil:
		return "config file"
	}
	return "none (logged out)"
}
