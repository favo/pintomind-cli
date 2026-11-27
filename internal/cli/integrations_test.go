package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newFakeIntegrationsAPI(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/integrations/12/available_resources":
			json.NewEncoder(w).Encode(map[string]any{"resources": []any{
				map[string]any{"title": "Made", "type": "microsoft_calendar", "resource_id": 40, "ok": true},
				map[string]any{"title": "Møterom", "type": "microsoft_calendar", "token": "tok-2"},
			}})
		case "POST /api/v1/integrations/12/resources":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			calls = append(calls, "token="+body["token"])
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"success": true, "resource": map[string]any{"id": 41}})
		case "DELETE /api/v1/integrations/12":
			json.NewEncoder(w).Encode(map[string]any{"success": true})
		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "not found"})
		}
	}))
	t.Cleanup(server.Close)
	return server, &calls
}

func TestIntegrationsAddResourceByNumber(t *testing.T) {
	server, calls := newFakeIntegrationsAPI(t)
	setupOAuthConnection(t, server.URL)

	if err := run("--account-id", "3", "integrations", "add-resource", "12", "2"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(*calls, "\n"); !strings.Contains(got, "POST /api/v1/integrations/12/resources\ntoken=tok-2") {
		t.Fatalf("calls = %s", got)
	}
}

func TestIntegrationsAddResourceRefusesAMadeOffer(t *testing.T) {
	server, _ := newFakeIntegrationsAPI(t)
	setupOAuthConnection(t, server.URL)

	err := run("--account-id", "3", "integrations", "add-resource", "12", "1")
	if err == nil || !strings.Contains(err.Error(), "resource 40") {
		t.Fatalf("err = %v, want it to name the existing resource", err)
	}
}

func TestIntegrationsDeleteWithResourcesSendsForce(t *testing.T) {
	server, calls := newFakeIntegrationsAPI(t)
	setupOAuthConnection(t, server.URL)

	if err := run("--account-id", "3", "integrations", "delete", "12", "--with-resources", "--force"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(*calls, "\n"); !strings.Contains(got, "DELETE /api/v1/integrations/12?force=true") {
		t.Fatalf("calls = %s", got)
	}
}
