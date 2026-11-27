package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TokenSource interface {
	Token() (string, error)
	// Refresh replaces the access token stale after the API refused it.
	Refresh(stale string) (string, error)
}

type staticToken string

func (t staticToken) Token() (string, error) { return string(t), nil }

func (t staticToken) Refresh(string) (string, error) { return "", errNoRefresh }

var errNoRefresh = errors.New("token cannot be refreshed")

type Client struct {
	BaseURL    string
	Auth       TokenSource
	AccountID  int
	NetworkID  int
	HTTPClient *http.Client
	Verbose    bool
	// LastBody is the raw body of the last successful response, so --json can print what the API
	// sent rather than the fields a command's struct keeps
	LastBody []byte
}

func New(baseURL, apiKey string) *Client {
	return NewWithAuth(baseURL, staticToken(apiKey))
}

func NewWithAuth(baseURL string, auth TokenSource) *Client {
	// Normalise: strip trailing slash, ensure https scheme
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}
	return &Client{
		BaseURL: baseURL,
		Auth:    auth,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// send performs a request against the API, retrying once with a refreshed token on 401.
func (c *Client) send(hc *http.Client, method, path string, body []byte, contentType string) (int, []byte, error) {
	u := c.BaseURL + "/api/v1" + path
	if hc == nil {
		hc = c.HTTPClient
	}
	token, err := c.Auth.Token()
	if err != nil {
		return 0, nil, err
	}

	for attempt := 0; ; attempt++ {
		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}
		req, err := http.NewRequest(method, u, bodyReader)
		if err != nil {
			return 0, nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if c.NetworkID != 0 {
			req.Header.Set("Network-Id", strconv.Itoa(c.NetworkID))
		} else if c.AccountID != 0 {
			req.Header.Set("Account-Id", strconv.Itoa(c.AccountID))
		}

		if c.Verbose {
			fmt.Fprintf(debugWriter, "> %s %s\n", method, u)
			if c.NetworkID != 0 {
				fmt.Fprintf(debugWriter, "> Network-Id: %d\n", c.NetworkID)
			} else if c.AccountID != 0 {
				fmt.Fprintf(debugWriter, "> Account-Id: %d\n", c.AccountID)
			}
			if body != nil && strings.HasPrefix(contentType, "application/json") {
				fmt.Fprintf(debugWriter, "> body: %s\n", body)
			}
		}

		resp, err := hc.Do(req)
		if err != nil {
			return 0, nil, err
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if c.Verbose {
			fmt.Fprintf(debugWriter, "< %s\n", resp.Status)
		}
		if err != nil {
			return 0, nil, err
		}

		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			fresh, err := c.Auth.Refresh(token)
			if errors.Is(err, errNoRefresh) {
				return resp.StatusCode, data, nil
			}
			if err != nil {
				return 0, nil, err
			}
			token = fresh
			continue
		}
		return resp.StatusCode, data, nil
	}
}

func (c *Client) do(method, path string, body any, out any) error {
	var bodyBytes []byte
	contentType := ""
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return err
		}
		contentType = "application/json"
	}
	status, data, err := c.send(nil, method, path, bodyBytes, contentType)
	if err != nil {
		return err
	}
	return c.decodeResponse(status, data, out)
}

func (c *Client) decodeResponse(status int, data []byte, out any) error {
	if status < 400 {
		c.LastBody = data
	}
	if status >= 400 {
		return c.withHint(ResponseError(status, data), status)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// DoRaw performs a raw request and returns the response body bytes.
// Used by the `api` passthrough command.
func (c *Client) DoRaw(method, path string, body []byte) ([]byte, int, error) {
	contentType := ""
	if body != nil {
		contentType = "application/json"
	}
	status, data, err := c.send(nil, method, path, body, contentType)
	return data, status, err
}

func (c *Client) Get(path string, query url.Values, out any) error {
	if len(query) > 0 {
		path = path + "?" + query.Encode()
	}
	return c.do("GET", path, nil, out)
}

func (c *Client) Patch(path string, body any, out any) error {
	return c.do("PATCH", path, body, out)
}

func (c *Client) Put(path string, body any, out any) error {
	return c.do("PUT", path, body, out)
}

func (c *Client) Post(path string, body any, out any) error {
	return c.do("POST", path, body, out)
}

func (c *Client) Delete(path string) error {
	return c.do("DELETE", path, nil, nil)
}

func (c *Client) DeleteWithBody(path string, body any, out any) error {
	return c.do("DELETE", path, body, out)
}

// PostMultipart sends a multipart/form-data POST. fields are plain form values;
// files maps form field name → local file path.
func (c *Client) PostMultipart(path string, fields map[string]string, files map[string]string, out any) error {
	return c.doMultipart("POST", path, fields, files, out)
}

// PatchMultipart sends a multipart/form-data PATCH.
func (c *Client) PatchMultipart(path string, fields map[string]string, files map[string]string, out any) error {
	return c.doMultipart("PATCH", path, fields, files, out)
}

func (c *Client) doMultipart(method, path string, fields map[string]string, files map[string]string, out any) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for key, value := range fields {
		if err := w.WriteField(key, value); err != nil {
			return err
		}
	}
	for fieldName, filePath := range files {
		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("opening %s: %w", filePath, err)
		}
		defer f.Close() //nolint:gocritic
		fw, err := w.CreateFormFile(fieldName, filepath.Base(filePath))
		if err != nil {
			return err
		}
		if _, err := io.Copy(fw, f); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}

	uploadClient := *c.HTTPClient
	uploadClient.Timeout = 60 * time.Second
	status, data, err := c.send(&uploadClient, method, path, buf.Bytes(), w.FormDataContentType())
	if err != nil {
		return err
	}
	return c.decodeResponse(status, data, out)
}

// withHint says what to do about a 403 that comes from what the connection acts for.
func (c *Client) withHint(err error, status int) error {
	if status != http.StatusForbidden {
		return err
	}
	message := strings.ToLower(err.Error())
	switch {
	case c.NetworkID != 0 && strings.Contains(message, "account-bearer"):
		return fmt.Errorf("%w\nThe connection acts for network %d, and this needs an account: pass --account-id <id> or run 'pintomind connection use-account'", err, c.NetworkID)
	case strings.Contains(message, "single sign-on"):
		return fmt.Errorf("%w\nRun 'pintomind login' and log in through the account's SSO on the login page", err)
	}
	return err
}

// ResponseError builds the error returned for an HTTP status >= 400, including
// whatever details the JSON error body carries.
func ResponseError(statusCode int, data []byte) error {
	if message := apiErrorMessage(statusCode, data); message != "" {
		return fmt.Errorf("API error %d: %s", statusCode, message)
	}
	return fmt.Errorf("API error %d", statusCode)
}

func apiErrorMessage(statusCode int, data []byte) string {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return ""
	}

	var payload any
	if err := json.Unmarshal(data, &payload); err == nil {
		if message := errorMessageFromPayload(payload); message != "" {
			return message
		}
		if statusCode == http.StatusUnprocessableEntity {
			return string(data)
		}
		return ""
	}

	if statusCode == http.StatusUnprocessableEntity {
		return string(data)
	}
	return ""
}

func errorMessageFromPayload(payload any) string {
	switch value := payload.(type) {
	case string:
		return strings.TrimSpace(value)
	case []any:
		return joinErrorMessages(value)
	case map[string]any:
		var parts []string
		add := func(message string) {
			if message == "" {
				return
			}
			for _, existing := range parts {
				if existing == message {
					return
				}
			}
			parts = append(parts, message)
		}
		for _, key := range []string{"error", "message", "detail", "details"} {
			if raw, ok := value[key]; ok {
				if message := errorMessageFromPayload(raw); message != "" {
					add(message)
					break
				}
			}
		}
		if raw, ok := value["errors"]; ok {
			add(validationErrorsMessage(raw))
		}
		if raw, ok := value["unpermitted_params"]; ok {
			if params := formatParams(raw); params != "" {
				add("unpermitted params: " + params)
			}
		}
		if raw, ok := value["permitted_params"]; ok {
			if params := formatParams(raw); params != "" {
				add("permitted params: " + params)
			}
		}
		return strings.Join(parts, "; ")
	}
	return ""
}

// formatParams renders a Rails-style permitted/unpermitted params structure
// (strings, arrays and nested objects) compactly, e.g. "media[name, description]".
func formatParams(payload any) string {
	switch value := payload.(type) {
	case nil:
		return ""
	case string:
		return value
	case []any:
		items := make([]string, 0, len(value))
		for _, item := range value {
			if s := formatParams(item); s != "" {
				items = append(items, s)
			}
		}
		return strings.Join(items, ", ")
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		items := make([]string, 0, len(keys))
		for _, key := range keys {
			if inner := formatParams(value[key]); inner != "" {
				items = append(items, key+"["+inner+"]")
			} else {
				items = append(items, key)
			}
		}
		return strings.Join(items, ", ")
	default:
		return fmt.Sprint(value)
	}
}

func validationErrorsMessage(payload any) string {
	switch value := payload.(type) {
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		messages := make([]string, 0, len(keys))
		for _, key := range keys {
			message := errorMessageFromPayload(value[key])
			if message == "" {
				continue
			}
			if key == "base" {
				messages = append(messages, message)
				continue
			}
			messages = append(messages, fmt.Sprintf("%s: %s", key, message))
		}
		return strings.Join(messages, "; ")
	default:
		return errorMessageFromPayload(payload)
	}
}

func joinErrorMessages(values []any) string {
	messages := make([]string, 0, len(values))
	for _, value := range values {
		if message := errorMessageFromPayload(value); message != "" {
			messages = append(messages, message)
		}
	}
	return strings.Join(messages, "; ")
}

func (c *Client) PutDirectUpload(uploadURL string, headers map[string]string, body io.Reader, contentLength int64) error {
	req, err := http.NewRequest("PUT", uploadURL, body)
	if err != nil {
		return err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if contentLength >= 0 {
		req.ContentLength = contentLength
	}

	if c.Verbose {
		fmt.Fprintf(debugWriter, "> PUT %s\n", uploadURL)
	}

	uploadClient := *c.HTTPClient
	uploadClient.Timeout = 0
	resp, err := uploadClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if c.Verbose {
		fmt.Fprintf(debugWriter, "< %s\n", resp.Status)
	}

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if len(data) > 0 {
			return fmt.Errorf("direct upload error %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
		}
		return fmt.Errorf("direct upload error %d", resp.StatusCode)
	}
	return nil
}
