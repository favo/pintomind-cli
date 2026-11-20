package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"favo/pintomind-cli/internal/config"
)

// ErrLoginGone means the refresh token was refused: the login expired, was revoked or the
// password changed. Only a new login helps.
var ErrLoginGone = errors.New("the login has expired or was revoked")

type Metadata struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	RevocationEndpoint    string `json:"revocation_endpoint"`
	IssParameterSupported bool   `json:"authorization_response_iss_parameter_supported"`
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

func NormalizeBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	return baseURL
}

func Discover(ctx context.Context, baseURL string) (*Metadata, error) {
	baseURL = NormalizeBaseURL(baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/.well-known/oauth-authorization-server", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discovering OAuth endpoints on %s: %w", baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovering OAuth endpoints on %s: HTTP %d", baseURL, resp.StatusCode)
	}
	var m Metadata
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("decoding OAuth metadata: %w", err)
	}
	if m.AuthorizationEndpoint == "" || m.TokenEndpoint == "" {
		return nil, fmt.Errorf("%s publishes no OAuth authorization or token endpoint", baseURL)
	}
	if m.Issuer == "" {
		m.Issuer = baseURL
	}
	return &m, nil
}

type PKCE struct {
	Verifier  string
	Challenge string
}

func NewPKCE() (PKCE, error) {
	verifier, err := randomString(32)
	if err != nil {
		return PKCE{}, err
	}
	sum := sha256.Sum256([]byte(verifier))
	return PKCE{Verifier: verifier, Challenge: base64.RawURLEncoding.EncodeToString(sum[:])}, nil
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

type LoginOptions struct {
	BaseURL   string
	ClientID  string
	NoBrowser bool
	// Prompt receives the authorization URL; it is always shown so a user can open it by hand.
	Prompt func(authorizeURL string)
}

// Login runs the authorization code flow with PKCE and a loopback redirect (RFC 8252 §7.3).
func Login(ctx context.Context, opts LoginOptions) (config.Tokens, error) {
	baseURL := NormalizeBaseURL(opts.BaseURL)
	meta, err := Discover(ctx, baseURL)
	if err != nil {
		return config.Tokens{}, err
	}
	pkce, err := NewPKCE()
	if err != nil {
		return config.Tokens{}, err
	}
	state, err := randomString(24)
	if err != nil {
		return config.Tokens{}, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return config.Tokens{}, fmt.Errorf("listening for the login redirect: %w", err)
	}
	defer listener.Close()
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", listener.Addr().(*net.TCPAddr).Port)

	authorizeURL := buildAuthorizeURL(meta.AuthorizationEndpoint, url.Values{
		"response_type":         {"code"},
		"client_id":             {opts.ClientID},
		"redirect_uri":          {redirectURI},
		"state":                 {state},
		"code_challenge":        {pkce.Challenge},
		"code_challenge_method": {"S256"},
		"resource":              {baseURL + "/api/v1"},
	})

	results := make(chan callbackResult, 1)
	server := &http.Server{
		Handler:           callbackHandler(state, meta, results),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() { _ = server.Serve(listener) }()
	defer server.Close()

	if opts.Prompt != nil {
		opts.Prompt(authorizeURL)
	}
	if !opts.NoBrowser {
		_ = openBrowser(authorizeURL)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	var result callbackResult
	select {
	case result = <-results:
	case <-ctx.Done():
		return config.Tokens{}, fmt.Errorf("gave up waiting for the browser login: %w", ctx.Err())
	}
	if result.err != nil {
		return config.Tokens{}, result.err
	}

	return requestTokens(ctx, meta.TokenEndpoint, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {result.code},
		"redirect_uri":  {redirectURI},
		"client_id":     {opts.ClientID},
		"code_verifier": {pkce.Verifier},
	})
}

func buildAuthorizeURL(endpoint string, params url.Values) string {
	sep := "?"
	if strings.Contains(endpoint, "?") {
		sep = "&"
	}
	return endpoint + sep + params.Encode()
}

type callbackResult struct {
	code string
	err  error
}

func callbackHandler(state string, meta *Metadata, results chan<- callbackResult) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		var result callbackResult
		switch {
		case q.Get("state") != state:
			result.err = errors.New("the login redirect had the wrong state; try again")
		case q.Get("iss") != "" && q.Get("iss") != meta.Issuer,
			q.Get("iss") == "" && meta.IssParameterSupported:
			result.err = fmt.Errorf("the login redirect came from issuer %q, expected %q", q.Get("iss"), meta.Issuer)
		case q.Get("error") != "":
			result.err = fmt.Errorf("login refused: %s", describeOAuthError(q.Get("error"), q.Get("error_description")))
		case q.Get("code") == "":
			result.err = errors.New("the login redirect carried no authorization code")
		default:
			result.code = q.Get("code")
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		message := "You are logged in to the Pintomind CLI. You can close this window."
		if result.err != nil {
			w.WriteHeader(http.StatusBadRequest)
			message = "Login failed: " + result.err.Error()
		}
		fmt.Fprintf(w, `<!doctype html><meta charset="utf-8"><title>Pintomind CLI</title>`+
			`<body style="font-family:system-ui;margin:4rem auto;max-width:32rem"><p>%s</p></body>`,
			html.EscapeString(message))

		select {
		case results <- result:
		default:
		}
	})
	return mux
}

func openBrowser(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func requestTokens(ctx context.Context, tokenEndpoint string, form url.Values) (config.Tokens, error) {
	status, body, err := postForm(ctx, tokenEndpoint, form)
	if err != nil {
		return config.Tokens{}, err
	}
	var tr tokenResponse
	_ = json.Unmarshal(body, &tr)
	if status != http.StatusOK {
		if tr.Error == "invalid_grant" && form.Get("grant_type") == "refresh_token" {
			return config.Tokens{}, ErrLoginGone
		}
		if tr.Error != "" {
			return config.Tokens{}, fmt.Errorf("token request failed: %s", describeOAuthError(tr.Error, tr.ErrorDescription))
		}
		return config.Tokens{}, fmt.Errorf("token request failed: HTTP %d", status)
	}
	if tr.AccessToken == "" || tr.RefreshToken == "" {
		return config.Tokens{}, errors.New("token response lacks an access or refresh token")
	}
	expiresIn := tr.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	return config.Tokens{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second).UTC().Truncate(time.Second),
	}, nil
}

func RefreshTokens(ctx context.Context, baseURL, clientID, refreshToken string) (config.Tokens, error) {
	meta, err := Discover(ctx, baseURL)
	if err != nil {
		return config.Tokens{}, err
	}
	return requestTokens(ctx, meta.TokenEndpoint, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	})
}

// Revoke ends the login the refresh token belongs to (RFC 7009).
func Revoke(ctx context.Context, baseURL, clientID, refreshToken string) error {
	meta, err := Discover(ctx, baseURL)
	if err != nil {
		return err
	}
	endpoint := meta.RevocationEndpoint
	if endpoint == "" {
		endpoint = NormalizeBaseURL(baseURL) + "/oauth/revoke"
	}
	status, body, err := postForm(ctx, endpoint, url.Values{
		"token":           {refreshToken},
		"token_type_hint": {"refresh_token"},
		"client_id":       {clientID},
	})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		var tr tokenResponse
		_ = json.Unmarshal(body, &tr)
		if tr.Error != "" {
			return fmt.Errorf("revoking the login failed: %s", describeOAuthError(tr.Error, tr.ErrorDescription))
		}
		return fmt.Errorf("revoking the login failed: HTTP %d", status)
	}
	return nil
}

func postForm(ctx context.Context, endpoint string, form url.Values) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, body, err
}

func describeOAuthError(code, description string) string {
	if description != "" {
		return fmt.Sprintf("%s (%s)", description, code)
	}
	return code
}
