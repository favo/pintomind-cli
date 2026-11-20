package commands

import (
	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/api"
	"favo/pintomind-cli/internal/auth"
	"favo/pintomind-cli/internal/config"
)

// NewClient builds the API client for a stored connection, acting for its chosen account or
// network.
func NewClient(name string, d config.Domain, verbose bool) *api.Client {
	var client *api.Client
	if d.IsOAuth() {
		client = api.NewWithAuth(d.BaseURL, auth.NewSource(name, d))
		client.AccountID = d.AccountID
		client.NetworkID = d.NetworkID
	} else {
		client = api.New(d.BaseURL, d.APIKey)
	}
	client.Verbose = verbose
	return client
}

// newUnscopedClient acts for no account, as GET /me/accounts expects.
func newUnscopedClient(name string, d config.Domain, verbose bool) *api.Client {
	d.AccountID, d.NetworkID = 0, 0
	return NewClient(name, d, verbose)
}

const NoAccountAnnotation = "pintomind/no-account"

func activeIsOAuth(cmd *cobra.Command) bool {
	a := app(cmd)
	d, ok := a.Config.Domains[a.ActiveConnection]
	return ok && d.IsOAuth()
}
