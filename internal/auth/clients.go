package auth

import (
	"fmt"
	"net/url"
	"strings"
)

// The Pintomind CLI's OAuth client per host; each environment registers its own
// (bin/rails oauth:clients:upsert CONFIG=config/oauth_clients.yml in the backend).
var clientIDs = map[string]string{
	"develop.infoskjermen.no": "tzSgaAM46bAgS2Ej9lCaN8UhyASG74mNoptwAodM05o",
}

func Host(baseURL string) string {
	u, err := url.Parse(NormalizeBaseURL(baseURL))
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func ClientIDFor(baseURL string) (string, error) {
	host := Host(baseURL)
	if id := clientIDs[host]; id != "" {
		return id, nil
	}
	return "", fmt.Errorf("the CLI knows no OAuth client for %s — pass --client-id", host)
}
