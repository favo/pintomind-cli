package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type themeCall struct {
	method, path string
	body         map[string]any
}

func newFakeThemesAPI(t *testing.T) (*httptest.Server, *[]themeCall) {
	t.Helper()
	var calls []themeCall
	theme := map[string]any{
		"id": 12, "name": "Acme", "preview_url": "https://example.test/theme_previews/abc",
		"areas":            map[string]any{"*:portrait:*": map[string]any{"A": map[string]any{"style": "glass"}}},
		"theme_areas":      []any{map[string]any{"id": 5, "name": "Glass", "identifier": "glass", "theme_extensions": []any{map[string]any{"id": 9, "type": "glass_extension"}}}},
		"theme_extensions": []any{map[string]any{"id": 3, "type": "forecast_extension"}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := themeCall{method: r.Method, path: r.URL.Path}
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			json.Unmarshal(raw, &call.body)
		}
		calls = append(calls, call)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/themes/12":
			json.NewEncoder(w).Encode(theme)
		case "PATCH /api/v1/themes/12", "POST /api/v1/themes":
			json.NewEncoder(w).Encode(map[string]any{"success": true, "theme": theme})
		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "not found"})
		}
	}))
	t.Cleanup(server.Close)
	return server, &calls
}

func themePayload(t *testing.T, calls []themeCall) map[string]any {
	t.Helper()
	for _, call := range calls {
		if call.method == "PATCH" || call.method == "POST" {
			theme, _ := call.body["theme"].(map[string]any)
			return theme
		}
	}
	t.Fatalf("no write in %v", calls)
	return nil
}

func TestThemesSetAreasSendsTheMapping(t *testing.T) {
	server, calls := newFakeThemesAPI(t)
	setupOAuthConnection(t, server.URL)

	if err := run("--account-id", "3", "themes", "set-areas", "12", "--data", `{"*:portrait:*":{"A":{"style":"glass"}}}`); err != nil {
		t.Fatal(err)
	}
	areas, _ := themePayload(t, *calls)["areas"].(map[string]any)
	if _, ok := areas["*:portrait:*"]; !ok || len(themePayload(t, *calls)) != 1 {
		t.Fatalf("theme payload = %v, want only areas with the portrait key", themePayload(t, *calls))
	}
}

func TestThemesSetEffect(t *testing.T) {
	server, calls := newFakeThemesAPI(t)
	setupOAuthConnection(t, server.URL)

	if err := run("--account-id", "3", "themes", "set-effect", "12", "snow"); err != nil {
		t.Fatal(err)
	}
	if got := themePayload(t, *calls)["screen_effect"]; got != "snow" {
		t.Fatalf("screen_effect = %v, want snow", got)
	}

	*calls = nil
	if err := run("--account-id", "3", "themes", "set-effect", "12", "none"); err != nil {
		t.Fatal(err)
	}
	if got, ok := themePayload(t, *calls)["screen_effect"]; !ok || got != "" {
		t.Fatalf("none should send an empty screen_effect, got %v", got)
	}

	*calls = nil
	err := run("--account-id", "3", "themes", "set-effect", "12", "fireworks")
	if err == nil || !strings.Contains(err.Error(), "snow") {
		t.Fatalf("err = %v, want it to list the effects", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("an unknown effect must not reach the API: %v", *calls)
	}
}

func TestThemesCreateReadsDataFromAFile(t *testing.T) {
	server, calls := newFakeThemesAPI(t)
	setupOAuthConnection(t, server.URL)
	path := filepath.Join(t.TempDir(), "theme.json")
	if err := os.WriteFile(path, []byte(`{"name":"Acme","theme_areas_attributes":[{"id":5,"_destroy":true}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := run("--account-id", "3", "themes", "create", "--data", "@"+path); err != nil {
		t.Fatal(err)
	}
	if got := themePayload(t, *calls)["name"]; got != "Acme" {
		t.Fatalf("name = %v", got)
	}
}

func TestThemesDataMustBeAnObject(t *testing.T) {
	server, calls := newFakeThemesAPI(t)
	setupOAuthConnection(t, server.URL)

	err := run("--account-id", "3", "themes", "update", "12", "--data", `["not","an","object"]`)
	if err == nil || !strings.Contains(err.Error(), "JSON object") {
		t.Fatalf("err = %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("calls = %v", *calls)
	}
}

func TestThemesAreasAndPreviewURLRead(t *testing.T) {
	server, calls := newFakeThemesAPI(t)
	setupOAuthConnection(t, server.URL)

	for _, args := range [][]string{{"themes", "areas", "12"}, {"themes", "areas", "12", "--json"}, {"themes", "preview-url", "12"}} {
		if err := run(append([]string{"--account-id", "3"}, args...)...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	for _, call := range *calls {
		if call.method != "GET" {
			t.Fatalf("read commands wrote: %v", call)
		}
	}
}
