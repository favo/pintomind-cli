package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &errBody)
		if errBody.Error != "" {
			return fmt.Errorf("API error %d: %s", resp.StatusCode, errBody.Error)
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

func (c *Client) Post(path string, body any, out any) error {
	return c.do("POST", path, body, out)
}

func (c *Client) Delete(path string) error {
	return c.do("DELETE", path, nil, nil)
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
