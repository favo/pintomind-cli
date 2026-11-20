package api

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientShowsValidationErrorsFor422(t *testing.T) {
	client := newTestClient(http.StatusUnprocessableEntity, `{"errors":{"name":["can't be blank"],"base":["cannot publish archived post"]}}`)
	err := client.Post("/posts", map[string]any{"post": map[string]any{}}, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	got := err.Error()
	for _, want := range []string{
		"API error 422:",
		"cannot publish archived post",
		"name: can't be blank",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("error = %q, want to contain %q", got, want)
		}
	}
}

func TestClientShowsRawBodyFor422WithoutKnownJSONMessage(t *testing.T) {
	client := newTestClient(http.StatusUnprocessableEntity, `{"title":["is too short"]}`)
	err := client.Post("/posts", map[string]any{"post": map[string]any{}}, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	want := `API error 422: {"title":["is too short"]}`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestClientPreservesExistingErrorMessage(t *testing.T) {
	client := newTestClient(http.StatusForbidden, `{"error":"forbidden"}`)
	err := client.Get("/me", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	want := "API error 403: forbidden"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestClientShowsUnpermittedParamsFor422(t *testing.T) {
	client := newTestClient(http.StatusUnprocessableEntity, `{"success":false,"error":"Unpermitted parameters","unpermitted_params":["foo"],"permitted_params":{"media":["name","description",{"settings":["a"]}]}}`)
	err := client.Patch("/media/1", map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	want := "API error 422: Unpermitted parameters; unpermitted params: foo; permitted params: media[name, description, settings[a]]"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestClientShowsErrorAndErrorsArray(t *testing.T) {
	client := newTestClient(http.StatusUnprocessableEntity, `{"success":false,"error":"Invalid","errors":["Name can't be blank","Url is invalid"]}`)
	err := client.Post("/resources", map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	want := "API error 422: Invalid; Name can't be blank; Url is invalid"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func newTestClient(statusCode int, body string) *Client {
	client := New("https://example.test", "token")
	client.HTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: statusCode,
				Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
				Header: http.Header{
					"Content-Type": {"application/json"},
				},
				Body:    io.NopCloser(strings.NewReader(body)),
				Request: req,
			}, nil
		}),
	}
	return client
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type rotatingToken struct {
	current   string
	refreshes int
}

func (r *rotatingToken) Token() (string, error) { return r.current, nil }

func (r *rotatingToken) Refresh(stale string) (string, error) {
	r.refreshes++
	r.current = "fresh"
	return r.current, nil
}

func TestClientRefreshesOnceOn401AndSendsAccountHeaders(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization")+"|"+r.Header.Get("Account-Id")+"|"+r.Header.Get("Network-Id"))
		if r.Header.Get("Authorization") != "Bearer fresh" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	auth := &rotatingToken{current: "expired"}
	client := NewWithAuth(server.URL, auth)
	client.AccountID = 42
	var out map[string]any
	if err := client.Post("/posts", map[string]any{"a": 1}, &out); err != nil {
		t.Fatal(err)
	}
	if auth.refreshes != 1 || out["ok"] != true {
		t.Fatalf("refreshes = %d, out = %v", auth.refreshes, out)
	}
	want := []string{"Bearer expired|42|", "Bearer fresh|42|"}
	if strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Fatalf("requests = %v, want %v", seen, want)
	}

	seen = nil
	client.AccountID, client.NetworkID = 0, 7
	if err := client.Get("/me", nil, nil); err != nil {
		t.Fatal(err)
	}
	if seen[0] != "Bearer fresh||7" {
		t.Fatalf("network request = %q", seen[0])
	}
}

func TestClientWithAPIKeyDoesNotRetry401(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid key"}`))
	}))
	defer server.Close()

	err := New(server.URL, "sk-x").Get("/me", nil, nil)
	if err == nil || err.Error() != "API error 401: invalid key" || calls != 1 {
		t.Fatalf("err = %v, calls = %d", err, calls)
	}
}
