package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	Verbose    bool
}

func New(baseURL, apiKey string) *Client {
	// Normalise: strip trailing slash, ensure https scheme
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) do(method, path string, body any, out any) error {
	u := c.BaseURL + "/api/v1" + path

	var bodyBytes []byte
	var bodyReader io.Reader
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.Verbose {
		fmt.Fprintf(debugWriter, "> %s %s\n", method, u)
		if bodyBytes != nil {
			fmt.Fprintf(debugWriter, "> body: %s\n", bodyBytes)
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if c.Verbose {
		fmt.Fprintf(debugWriter, "< %s\n", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		if message := apiErrorMessage(resp.StatusCode, data); message != "" {
			return fmt.Errorf("API error %d: %s", resp.StatusCode, message)
		}
		return fmt.Errorf("API error %d", resp.StatusCode)
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
	u := c.BaseURL + "/api/v1" + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.Verbose {
		fmt.Fprintf(debugWriter, "> %s %s\n", method, u)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if c.Verbose {
		fmt.Fprintf(debugWriter, "< %s\n", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	return data, resp.StatusCode, err
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

	u := c.BaseURL + "/api/v1" + path
	req, err := http.NewRequest(method, u, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", w.FormDataContentType())

	if c.Verbose {
		fmt.Fprintf(debugWriter, "> %s %s (multipart)\n", method, u)
	}

	uploadClient := *c.HTTPClient
	uploadClient.Timeout = 60 * time.Second
	resp, err := uploadClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if c.Verbose {
		fmt.Fprintf(debugWriter, "< %s\n", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		if message := apiErrorMessage(resp.StatusCode, data); message != "" {
			return fmt.Errorf("API error %d: %s", resp.StatusCode, message)
		}
		return fmt.Errorf("API error %d", resp.StatusCode)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
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
		for _, key := range []string{"error", "message", "errors", "detail", "details"} {
			if raw, ok := value[key]; ok {
				if key == "errors" {
					return validationErrorsMessage(raw)
				}
				if message := errorMessageFromPayload(raw); message != "" {
					return message
				}
			}
		}
	}
	return ""
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
