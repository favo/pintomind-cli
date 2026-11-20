package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"favo/pintomind-cli/internal/config"
)

type recordedRequest struct {
	path, auth, accountID, networkID string
}

func newFakeAPI(t *testing.T) (*httptest.Server, func() []recordedRequest) {
	t.Helper()
	var mu sync.Mutex
	var requests []recordedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, recordedRequest{
			path: r.URL.Path, auth: r.Header.Get("Authorization"),
			accountID: r.Header.Get("Account-Id"), networkID: r.Header.Get("Network-Id"),
		})
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/me/accounts":
			json.NewEncoder(w).Encode(map[string]any{
				"success":  true,
				"accounts": []any{map[string]any{"id": 3, "name": "Favo", "role": "account_owner"}},
				"networks": []any{map[string]any{"id": 7, "name": "Favo Partner", "role": "reseller", "account_id": 3, "can_manage": true}},
			})
		default:
			json.NewEncoder(w).Encode(map[string]any{"total": 0, "items": []any{}})
		}
	}))
	t.Cleanup(server.Close)
	return server, func() []recordedRequest {
		mu.Lock()
		defer mu.Unlock()
		out := requests
		requests = nil
		return out
	}
}

func setupOAuthConnection(t *testing.T, baseURL string) {
	t.Helper()
	t.Setenv("PINTOMIND_CONFIG_DIR", t.TempDir())
	t.Setenv("PINTOMIND_NO_KEYRING", "1")
	t.Setenv("PINTOMIND_API_KEY", "")
	if _, err := config.Update(func(cfg *config.Config) error {
		cfg.DefaultDomain = "dev"
		cfg.Domains["dev"] = config.Domain{
			Auth: config.AuthOAuth, BaseURL: baseURL, ClientID: "cli",
			Tokens: &config.Tokens{AccessToken: "at", RefreshToken: "rt", ExpiresAt: time.Now().Add(time.Hour)},
		}
		cfg.Domains["key"] = config.Domain{BaseURL: baseURL, APIKey: "sk-key"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func run(args ...string) error {
	root := NewRootCmd()
	root.SetArgs(args)
	return root.Execute()
}

func TestOAuthConnectionNeedsAnAccount(t *testing.T) {
	server, _ := newFakeAPI(t)
	setupOAuthConnection(t, server.URL)

	err := run("webhooks", "list")
	if err == nil || !strings.Contains(err.Error(), "use-account") {
		t.Fatalf("err = %v, want a hint to choose an account", err)
	}
	for _, cmd := range []string{"accounts", "networks"} {
		if err := run(cmd); err != nil {
			t.Fatalf("%s must work without an account: %v", cmd, err)
		}
	}
}

func TestUseNetworkSendsNetworkID(t *testing.T) {
	server, requests := newFakeAPI(t)
	setupOAuthConnection(t, server.URL)

	if err := run("connection", "use-network", "7"); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load()
	if d := cfg.Domains["dev"]; d.NetworkID != 7 || d.NetworkName != "Favo Partner" || d.AccountID != 0 {
		t.Fatalf("domain = %+v", d)
	}
	if got := requests(); len(got) != 1 || got[0].accountID != "" || got[0].networkID != "" {
		t.Fatalf("GET /me/accounts must name no account: %+v", got)
	}

	if err := run("webhooks", "list"); err != nil {
		t.Fatal(err)
	}
	got := requests()
	if len(got) != 1 || got[0].auth != "Bearer at" || got[0].networkID != "7" || got[0].accountID != "" {
		t.Fatalf("requests = %+v", got)
	}

	if err := run("--account-id", "3", "webhooks", "list"); err != nil {
		t.Fatal(err)
	}
	got = requests()
	if len(got) != 1 || got[0].accountID != "3" || got[0].networkID != "" {
		t.Fatalf("override: requests = %+v", got)
	}
}

func TestUseAccountRefusesUnknownAccountAndAPIKeys(t *testing.T) {
	server, _ := newFakeAPI(t)
	setupOAuthConnection(t, server.URL)

	if err := run("connection", "use-account", "99"); err == nil || !strings.Contains(err.Error(), "not one this login may act for") {
		t.Fatalf("err = %v", err)
	}
	if err := run("--connection", "key", "connection", "use-account", "3"); err == nil || !strings.Contains(err.Error(), "API key") {
		t.Fatalf("err = %v", err)
	}
	if err := run("--connection", "key", "--network-id", "7", "webhooks", "list"); err == nil || !strings.Contains(err.Error(), "OAuth connection") {
		t.Fatalf("err = %v", err)
	}
}

func TestAPIKeyConnectionSendsNoAccountHeaders(t *testing.T) {
	server, requests := newFakeAPI(t)
	setupOAuthConnection(t, server.URL)

	if err := run("--connection", "key", "webhooks", "list"); err != nil {
		t.Fatal(err)
	}
	got := requests()
	if len(got) != 1 || got[0].auth != "Bearer sk-key" || got[0].accountID != "" || got[0].networkID != "" {
		t.Fatalf("requests = %+v", got)
	}
}

func TestNetworkCommandPicksTheNetworkForOAuth(t *testing.T) {
	server, requests := newFakeAPI(t)
	setupOAuthConnection(t, server.URL)
	if err := run("connection", "use-account", "3"); err != nil {
		t.Fatal(err)
	}
	requests()

	if err := run("network"); err != nil {
		t.Fatal(err)
	}
	got := requests()
	last := got[len(got)-1]
	if last.path != "/api/v1/network" || last.networkID != "7" || last.accountID != "" {
		t.Fatalf("network with the login's only network: %+v", got)
	}

	if err := run("network", "9"); err != nil {
		t.Fatal(err)
	}
	if got := requests(); len(got) != 1 || got[0].networkID != "9" {
		t.Fatalf("network 9: %+v", got)
	}

	if err := run("--connection", "key", "network", "9"); err == nil || !strings.Contains(err.Error(), "API key") {
		t.Fatalf("API key with a network ID: %v", err)
	}
	if err := run("--connection", "key", "networks"); err == nil || !strings.Contains(err.Error(), "API key") {
		t.Fatalf("networks on an API key: %v", err)
	}
}

func TestOAuthEdgeCases(t *testing.T) {
	server, requests := newFakeAPI(t)
	setupOAuthConnection(t, server.URL)

	if err := run("api", "/me/accounts"); err != nil {
		t.Fatalf("api passthrough before an account is chosen: %v", err)
	}
	if got := requests(); len(got) != 1 || got[0].accountID != "" || got[0].networkID != "" {
		t.Fatalf("api passthrough requests = %+v", got)
	}

	if err := run("connection", "add", "dev", "sk-new"); err == nil || !strings.Contains(err.Error(), "OAuth login") {
		t.Fatalf("connection add over an OAuth connection: %v", err)
	}
	cfg, _ := config.Load()
	if !cfg.Domains["dev"].IsOAuth() {
		t.Fatal("connection add replaced the OAuth connection")
	}

	err := run("--account-id", "3", "screens", "debug", "status", "42")
	if err == nil || !strings.Contains(err.Error(), "screens:debug") {
		t.Fatalf("screens debug on OAuth: %v", err)
	}
	if got := requests(); len(got) != 0 {
		t.Fatalf("screens debug must not call the API on OAuth: %+v", got)
	}
}
