package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"favo/pintomind-cli/internal/config"
)

type fakeServer struct {
	*httptest.Server
	clientID    string
	challenge   string
	refreshes   atomic.Int32
	revoked     []string
	refreshSeen map[string]bool
	mu          sync.Mutex
}

func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()
	f := &fakeServer{clientID: "cli-client", refreshSeen: map[string]bool{}}
	mux := http.NewServeMux()
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)

	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 f.URL,
			"authorization_endpoint": f.URL + "/oauth/authorize",
			"token_endpoint":         f.URL + "/oauth/token",
			"revocation_endpoint":    f.URL + "/oauth/revoke",
			"authorization_response_iss_parameter_supported": true,
		})
	})
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("client_id") != f.clientID {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_client"})
			return
		}
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if r.Form.Get("code") != "the-code" || base64.RawURLEncoding.EncodeToString(sum[:]) != f.challenge {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"access_token": "at-0", "refresh_token": "rt-0", "expires_in": 3600})
		case "refresh_token":
			f.mu.Lock()
			reused := f.refreshSeen[r.Form.Get("refresh_token")]
			f.refreshSeen[r.Form.Get("refresh_token")] = true
			f.mu.Unlock()
			if reused || r.Form.Get("refresh_token") == "dead" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
				return
			}
			n := f.refreshes.Add(1)
			time.Sleep(20 * time.Millisecond)
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": fmt.Sprintf("at-%d", n), "refresh_token": fmt.Sprintf("rt-%d", n), "expires_in": 3600,
			})
		}
	})
	mux.HandleFunc("/oauth/revoke", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		f.mu.Lock()
		f.revoked = append(f.revoked, r.Form.Get("token"))
		f.mu.Unlock()
		w.Write([]byte("{}"))
	})
	return f
}

func setupConfig(t *testing.T, useKeyring bool) {
	t.Helper()
	t.Setenv("PINTOMIND_CONFIG_DIR", t.TempDir())
	if useKeyring {
		keyring.MockInit()
		t.Setenv("PINTOMIND_NO_KEYRING", "")
	} else {
		t.Setenv("PINTOMIND_NO_KEYRING", "1")
	}
}

func addConnection(t *testing.T, name, baseURL, clientID string) {
	t.Helper()
	if _, err := config.Update(func(cfg *config.Config) error {
		cfg.Domains[name] = config.Domain{Auth: config.AuthOAuth, BaseURL: baseURL, ClientID: clientID}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestLoginRunsPKCELoopbackFlow(t *testing.T) {
	f := newFakeServer(t)

	tokens, err := Login(context.Background(), LoginOptions{
		BaseURL:   f.URL,
		ClientID:  f.clientID,
		NoBrowser: true,
		Prompt: func(authorizeURL string) {
			u, _ := url.Parse(authorizeURL)
			q := u.Query()
			if q.Get("code_challenge_method") != "S256" || q.Get("resource") != f.URL+"/api/v1" || q.Get("scope") != "" {
				t.Errorf("unexpected authorize params: %v", q)
			}
			redirect := q.Get("redirect_uri")
			if !strings.HasPrefix(redirect, "http://127.0.0.1:") || !strings.HasSuffix(redirect, "/callback") {
				t.Errorf("redirect_uri = %q", redirect)
			}
			f.challenge = q.Get("code_challenge")
			go func() {
				resp, err := http.Get(redirect + "?" + url.Values{"code": {"the-code"}, "state": {q.Get("state")}, "iss": {f.URL}}.Encode())
				if err == nil {
					resp.Body.Close()
				}
			}()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "at-0" || tokens.RefreshToken != "rt-0" || time.Until(tokens.ExpiresAt) < 50*time.Minute {
		t.Fatalf("tokens = %+v", tokens)
	}
}

func TestLoginRefusesWrongStateAndIssuer(t *testing.T) {
	for name, params := range map[string]func(state, iss string) url.Values{
		"state": func(_, iss string) url.Values {
			return url.Values{"code": {"the-code"}, "state": {"forged"}, "iss": {iss}}
		},
		"issuer": func(state, _ string) url.Values {
			return url.Values{"code": {"the-code"}, "state": {state}, "iss": {"https://evil.example"}}
		},
		"no iss": func(state, _ string) url.Values { return url.Values{"code": {"the-code"}, "state": {state}} },
		"refused": func(state, iss string) url.Values {
			return url.Values{"error": {"access_denied"}, "state": {state}, "iss": {iss}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFakeServer(t)
			_, err := Login(context.Background(), LoginOptions{
				BaseURL: f.URL, ClientID: f.clientID, NoBrowser: true,
				Prompt: func(authorizeURL string) {
					u, _ := url.Parse(authorizeURL)
					q := u.Query()
					go func() {
						resp, err := http.Get(q.Get("redirect_uri") + "?" + params(q.Get("state"), f.URL).Encode())
						if err == nil {
							resp.Body.Close()
						}
					}()
				},
			})
			if err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestLoginCancelledInBrowser(t *testing.T) {
	f := newFakeServer(t)
	pages := make(chan string, 1)
	_, err := Login(context.Background(), LoginOptions{
		BaseURL: f.URL, ClientID: f.clientID, NoBrowser: true,
		Prompt: func(authorizeURL string) {
			u, _ := url.Parse(authorizeURL)
			q := u.Query()
			go func() {
				resp, err := http.Get(q.Get("redirect_uri") + "?" + url.Values{
					"error": {"access_denied"}, "state": {q.Get("state")}, "iss": {f.URL},
				}.Encode())
				if err != nil {
					pages <- err.Error()
					return
				}
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				pages <- string(body)
			}()
		},
	})
	if !errors.Is(err, ErrLoginCancelled) {
		t.Fatalf("err = %v, want ErrLoginCancelled", err)
	}
	if page := <-pages; !strings.Contains(page, "Login cancelled") {
		t.Fatalf("browser page = %q", page)
	}
}

func TestSaveTokensUsesKeychainWhenAvailable(t *testing.T) {
	setupConfig(t, true)
	addConnection(t, "dev", "https://example.test", "c")
	if err := SaveTokens("dev", config.Tokens{AccessToken: "a", RefreshToken: "r"}); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load()
	d := cfg.Domains["dev"]
	if d.KeychainItem != "dev" || d.Tokens != nil {
		t.Fatalf("domain = %+v, want tokens in the keychain only", d)
	}
	got, err := LoadTokens(d)
	if err != nil || got.RefreshToken != "r" {
		t.Fatalf("LoadTokens = %+v, %v", got, err)
	}

	if err := DeleteTokens("dev"); err != nil {
		t.Fatal(err)
	}
	if _, err := keyring.Get(keyringService, "dev"); err == nil {
		t.Fatal("keychain item survived DeleteTokens")
	}
}

func TestSaveTokensFallsBackToConfig(t *testing.T) {
	setupConfig(t, false)
	addConnection(t, "dev", "https://example.test", "c")
	if err := SaveTokens("dev", config.Tokens{AccessToken: "a", RefreshToken: "r"}); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load()
	if d := cfg.Domains["dev"]; d.KeychainItem != "" || d.Tokens == nil || d.Tokens.RefreshToken != "r" {
		t.Fatalf("domain = %+v, want tokens in the config", d)
	}
}

func TestSourceRefreshesExpiredTokenAndStoresRotation(t *testing.T) {
	f := newFakeServer(t)
	setupConfig(t, false)
	addConnection(t, "dev", f.URL, f.clientID)
	SaveTokens("dev", config.Tokens{AccessToken: "old", RefreshToken: "rt-start", ExpiresAt: time.Now().Add(-time.Minute)})

	cfg, _ := config.Load()
	src := NewSource("dev", cfg.Domains["dev"])
	token, err := src.Token()
	if err != nil {
		t.Fatal(err)
	}
	if token != "at-1" {
		t.Fatalf("token = %q, want at-1", token)
	}
	cfg, _ = config.Load()
	if got := cfg.Domains["dev"].Tokens.RefreshToken; got != "rt-1" {
		t.Fatalf("stored refresh token = %q, want the rotated rt-1", got)
	}
}

func TestConcurrentRefreshesShareOneRefresh(t *testing.T) {
	f := newFakeServer(t)
	setupConfig(t, false)
	addConnection(t, "dev", f.URL, f.clientID)
	SaveTokens("dev", config.Tokens{AccessToken: "stale", RefreshToken: "rt-start", ExpiresAt: time.Now().Add(time.Hour)})

	cfg, _ := config.Load()
	// Separate sources stand in for separate processes; they share only the store and the lock.
	var wg sync.WaitGroup
	results := make([]string, 6)
	errs := make([]error, 6)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = NewSource("dev", cfg.Domains["dev"]).Refresh("stale")
		}(i)
	}
	wg.Wait()
	for i := range results {
		if errs[i] != nil || results[i] != "at-1" {
			t.Fatalf("refresh %d = %q, %v", i, results[i], errs[i])
		}
	}
	if n := f.refreshes.Load(); n != 1 {
		t.Fatalf("server saw %d refreshes, want 1", n)
	}
}

func TestRefreshWithDeadLoginClearsTokens(t *testing.T) {
	f := newFakeServer(t)
	setupConfig(t, false)
	addConnection(t, "dev", f.URL, f.clientID)
	SaveTokens("dev", config.Tokens{AccessToken: "stale", RefreshToken: "dead", ExpiresAt: time.Now().Add(time.Hour)})

	cfg, _ := config.Load()
	_, err := NewSource("dev", cfg.Domains["dev"]).Refresh("stale")
	if err == nil || !strings.Contains(err.Error(), "pintomind login --name dev") {
		t.Fatalf("err = %v", err)
	}
	cfg, _ = config.Load()
	if cfg.Domains["dev"].Tokens != nil {
		t.Fatal("tokens of a dead login were kept")
	}
}

func TestRevokeSendsRefreshToken(t *testing.T) {
	f := newFakeServer(t)
	if err := Revoke(context.Background(), f.URL, f.clientID, "rt-x"); err != nil {
		t.Fatal(err)
	}
	if len(f.revoked) != 1 || f.revoked[0] != "rt-x" {
		t.Fatalf("revoked = %v", f.revoked)
	}
}

func TestClientIDFor(t *testing.T) {
	if id, err := ClientIDFor("https://develop.infoskjermen.no/"); err != nil || id == "" {
		t.Fatalf("develop: %q, %v", id, err)
	}
	if _, err := ClientIDFor("https://unknown.example"); err == nil || !strings.Contains(err.Error(), "--client-id") {
		t.Fatalf("unknown host: %v", err)
	}
}
