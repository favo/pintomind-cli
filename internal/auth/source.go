package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"favo/pintomind-cli/internal/config"
)

const refreshMargin = time.Minute

// Source hands out the access token of an OAuth connection and refreshes it when it expires or
// the API answers 401.
type Source struct {
	Name     string
	BaseURL  string
	ClientID string

	mu     sync.Mutex
	tokens *config.Tokens
}

func NewSource(name string, d config.Domain) *Source {
	return &Source{Name: name, BaseURL: d.BaseURL, ClientID: d.ClientID}
}

func (s *Source) Token() (string, error) {
	s.mu.Lock()
	tokens := s.tokens
	s.mu.Unlock()
	if tokens == nil {
		t, err := s.load()
		if err != nil {
			return "", err
		}
		tokens = &t
		s.set(t)
	}
	if time.Until(tokens.ExpiresAt) > refreshMargin {
		return tokens.AccessToken, nil
	}
	return s.Refresh(tokens.AccessToken)
}

// Refresh replaces the access token stale. Refreshing happens behind one lock across processes:
// a refresh token that was already rotated counts as theft and revokes the login, so whoever
// waited re-reads the store and takes the token another process got.
func (s *Source) Refresh(stale string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var access string
	err := config.WithLock("refresh-"+s.Name, func() error {
		current, err := s.load()
		if err != nil {
			return err
		}
		if current.AccessToken != stale && time.Until(current.ExpiresAt) > refreshMargin {
			s.tokens = &current
			access = current.AccessToken
			return nil
		}

		fresh, err := RefreshTokens(context.Background(), s.BaseURL, s.ClientID, current.RefreshToken)
		if errors.Is(err, ErrLoginGone) {
			_ = DeleteTokens(s.Name)
			return fmt.Errorf("%w — log in again: pintomind login --name %s", ErrLoginGone, s.Name)
		}
		if err != nil {
			return err
		}
		// The new refresh token must be stored before the new access token is used: once it is,
		// the old refresh token stops working.
		if err := SaveTokens(s.Name, fresh); err != nil {
			return fmt.Errorf("saving the refreshed login: %w", err)
		}
		s.tokens = &fresh
		access = fresh.AccessToken
		return nil
	})
	return access, err
}

func (s *Source) set(t config.Tokens) {
	s.mu.Lock()
	s.tokens = &t
	s.mu.Unlock()
}

func (s *Source) load() (config.Tokens, error) {
	cfg, err := config.Load()
	if err != nil {
		return config.Tokens{}, err
	}
	d, ok := cfg.Domains[s.Name]
	if !ok {
		return config.Tokens{}, fmt.Errorf("connection %q not found", s.Name)
	}
	t, err := LoadTokens(d)
	if errors.Is(err, errNotLoggedIn) {
		return t, fmt.Errorf("connection %q is logged out — run: pintomind login --name %s", s.Name, s.Name)
	}
	return t, err
}
