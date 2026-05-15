package api

import (
	"fmt"
	"io"
	"net/http"
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
