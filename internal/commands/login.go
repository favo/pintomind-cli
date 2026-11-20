package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/auth"
	"favo/pintomind-cli/internal/config"
)

const defaultLoginURL = "https://app.infoskjermen.no"

func NewLoginCmd() *cobra.Command {
	var name, baseURL, clientID string
	var accountID, networkID int
	var noBrowser bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with your user in the browser (OAuth) and add or renew a connection",
		Long: `Opens the browser on the server's login page and waits for it to send you back to the
CLI on 127.0.0.1. The login belongs to your user: afterwards you choose which account or network
the connection acts for, and can switch later with 'connection use-account' / 'use-network'.

The tokens go in the OS keychain when one is available, else in the config file (mode 0600).
API key connections ('connection add') keep working alongside.`,
		Example: `  pintomind login
  pintomind login --url https://app.pintomind.com
  pintomind login --name develop --url https://develop.infoskjermen.no
  pintomind login --network 7`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if accountID != 0 && networkID != 0 {
				return errors.New("--account and --network are mutually exclusive")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			existing, exists := cfg.Domains[name]
			if exists && !existing.IsOAuth() {
				return fmt.Errorf("connection %q uses an API key; pick another --name or remove it first", name)
			}
			if baseURL == "" {
				baseURL = existing.BaseURL
			}
			if baseURL == "" {
				baseURL = defaultLoginURL
			}
			baseURL = auth.NormalizeBaseURL(baseURL)
			if name == "" {
				name = auth.Host(baseURL)
				existing, exists = cfg.Domains[name]
				if exists && !existing.IsOAuth() {
					return fmt.Errorf("connection %q uses an API key; pass --name to name the new connection", name)
				}
			}
			if exists && auth.Host(existing.BaseURL) != auth.Host(baseURL) {
				return fmt.Errorf("connection %q is for %s; pick another --name for %s", name, existing.BaseURL, baseURL)
			}
			if clientID == "" && exists {
				clientID = existing.ClientID
			}
			if clientID == "" {
				if clientID, err = auth.ClientIDFor(baseURL); err != nil {
					return err
				}
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()
			tokens, err := auth.Login(ctx, auth.LoginOptions{
				BaseURL:   baseURL,
				ClientID:  clientID,
				NoBrowser: noBrowser,
				Prompt: func(authorizeURL string) {
					fmt.Fprintf(os.Stderr, "Log in to %s in the browser. If it doesn't open, visit:\n\n  %s\n\nWaiting for the login…\n", baseURL, authorizeURL)
				},
			})
			if err != nil {
				return err
			}

			var previous config.Tokens
			if exists {
				previous, _ = auth.LoadTokens(existing)
			}
			cfg, err = config.Update(func(cfg *config.Config) error {
				d := cfg.Domains[name]
				d.Auth = config.AuthOAuth
				d.BaseURL = baseURL
				d.ClientID = clientID
				d.APIKey = ""
				cfg.Domains[name] = d
				if cfg.DefaultDomain == "" {
					cfg.DefaultDomain = name
				}
				return nil
			})
			if err != nil {
				return err
			}
			if err := auth.SaveTokens(name, tokens); err != nil {
				return fmt.Errorf("saving the login: %w", err)
			}
			if previous.RefreshToken != "" && previous.RefreshToken != tokens.RefreshToken {
				_ = auth.Revoke(context.Background(), baseURL, clientID, previous.RefreshToken)
			}

			cfg, _ = config.Load()
			d := cfg.Domains[name]
			fmt.Printf("Logged in to %s as connection %q (tokens in the %s).\n", baseURL, name, auth.StorageDescription(d))

			accounts, err := fetchMeAccounts(newUnscopedClient(name, d, false))
			if err != nil {
				return fmt.Errorf("listing your accounts: %w", err)
			}
			var choice accountChoice
			switch {
			case networkID != 0:
				choice, err = findNetwork(accounts, networkID)
			case accountID != 0:
				choice, err = findAccount(accounts, accountID)
			default:
				choice, err = pickAccount(accounts, d, "")
			}
			if errors.Is(err, errNoChoice) {
				fmt.Println("Choose what the connection acts for:")
				printAccountChoices(cmd, accounts, d)
				fmt.Printf("\nthen run: pintomind connection use-account <id> --connection %s (or use-network)\n", name)
				return nil
			}
			if err != nil {
				return err
			}
			if err := saveChoice(name, choice); err != nil {
				return err
			}
			choice.apply(&d)
			fmt.Printf("Acting for %s.\n", d.ActingFor())
			if cfg.DefaultDomain != name {
				fmt.Printf("Default connection is still %q; switch with: pintomind connection use %s\n", cfg.DefaultDomain, name)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Connection name (default: the server's host name)")
	cmd.Flags().StringVar(&baseURL, "url", "", "Server URL (default: "+defaultLoginURL+", or the connection's URL)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID (default: the CLI's client for the host)")
	cmd.Flags().IntVar(&accountID, "account", 0, "Act for this account ID without asking")
	cmd.Flags().IntVar(&networkID, "network", 0, "Act for this network ID without asking")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Print the login URL instead of opening the browser")
	return cmd
}

func NewLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out an OAuth connection: revoke its login and remove the tokens",
		Long: `Revokes the login on the server and removes the stored tokens. The connection itself
stays, so 'pintomind login --name <connection>' logs in again with the same settings. Use
'connection remove' to forget it entirely.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			override, _ := cmd.Root().PersistentFlags().GetString("connection")
			name, d, err := cfg.ActiveDomain(override)
			if err != nil {
				return err
			}
			if !d.IsOAuth() {
				return fmt.Errorf("connection %q uses an API key; remove it with: pintomind connection remove %s", name, name)
			}
			if err := logoutConnection(name, d); err != nil {
				return err
			}
			fmt.Printf("Logged out connection %q.\n", name)
			return nil
		},
	}
}

// logoutConnection revokes the login and forgets its tokens. The tokens are removed even when
// the server can't be reached, and the revoke error is reported then.
func logoutConnection(name string, d config.Domain) error {
	tokens, err := auth.LoadTokens(d)
	if err != nil {
		_ = auth.DeleteTokens(name)
		return nil
	}
	revokeErr := auth.Revoke(context.Background(), d.BaseURL, d.ClientID, tokens.RefreshToken)
	if err := auth.DeleteTokens(name); err != nil {
		return err
	}
	if revokeErr != nil {
		return fmt.Errorf("removed the local tokens, but %w", revokeErr)
	}
	return nil
}
