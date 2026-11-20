package commands

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/auth"
	"favo/pintomind-cli/internal/config"
)

func NewConnectionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connection",
		Short: "Manage connections",
	}
	cmd.AddCommand(newConfigAddCmd())
	cmd.AddCommand(newConfigRemoveCmd())
	cmd.AddCommand(newConfigListCmd())
	cmd.AddCommand(newConfigUseCmd())
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newUseAccountCmd(false))
	cmd.AddCommand(newUseAccountCmd(true))
	return cmd
}

func newConfigAddCmd() *cobra.Command {
	var baseURL string

	cmd := &cobra.Command{
		Use:   "add <name> <api-key>",
		Short: "Add a connection that uses an API key (see 'pintomind login' for OAuth)",
		Args:  cobra.ExactArgs(2),
		Example: `  pintomind connection add app.infoskjermen.no sk-xxx
  pintomind connection add develop sk-dev --url https://develop.infoskjermen.no
  pintomind connection use develop`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			apiKey := args[1]

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if existing, ok := cfg.Domains[name]; ok && existing.IsOAuth() {
				return fmt.Errorf("connection %q is an OAuth login; remove it first (pintomind connection remove %s) or pick another name", name, name)
			}

			url := baseURL
			if url == "" {
				url = "https://" + name
			}

			cfg.Domains[name] = config.Domain{
				APIKey:  apiKey,
				BaseURL: url,
			}
			if cfg.DefaultDomain == "" {
				cfg.DefaultDomain = name
			}

			if err := cfg.Save(); err != nil {
				return err
			}

			fmt.Printf("Added connection %q (%s)\n", name, url)
			if cfg.DefaultDomain == name {
				fmt.Println("Set as default connection.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&baseURL, "url", "", "Base URL (defaults to https://<name>)")
	return cmd
}

func newConfigRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a connection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			d, ok := cfg.Domains[name]
			if !ok {
				return fmt.Errorf("connection %q not found", name)
			}
			if d.IsOAuth() {
				if err := logoutConnection(name, d); err != nil {
					fmt.Fprintln(os.Stderr, "Warning:", err)
				}
				if cfg, err = config.Load(); err != nil {
					return err
				}
			}
			delete(cfg.Domains, name)
			if cfg.DefaultDomain == name {
				cfg.DefaultDomain = ""
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Printf("Removed connection %q\n", name)
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List connections",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Domains) == 0 {
				fmt.Println("No connections configured. Run: pintomind connection add app.infoskjermen.no <api-key>")
				return nil
			}
			names := make([]string, 0, len(cfg.Domains))
			for name := range cfg.Domains {
				names = append(names, name)
			}
			sort.Strings(names)
			rows := make([][]string, len(names))
			for i, name := range names {
				d := cfg.Domains[name]
				authKind := "api key"
				if d.IsOAuth() {
					authKind = "oauth"
					if d.KeychainItem == "" && d.Tokens == nil {
						authKind = "oauth (logged out)"
					}
				}
				rows[i] = []string{marker(name == cfg.DefaultDomain), name, d.BaseURL, authKind, d.ActingFor()}
			}
			printTable(cmd, []string{"", "NAME", "URL", "AUTH", "ACTS FOR"}, rows)
			return nil
		},
	}
}

func newConfigUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set the default connection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Domains[name]; !ok {
				return fmt.Errorf("connection %q not found — add it first with: pintomind connection add %s <api-key>", name, name)
			}
			cfg.DefaultDomain = name
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Printf("Default connection set to %q\n", name)
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the active connection",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			connectionOverride, _ := cmd.Root().PersistentFlags().GetString("connection")
			name, domain, err := cfg.ActiveDomain(connectionOverride)
			if err != nil {
				return err
			}
			fmt.Printf("Connection: %s\n", name)
			fmt.Printf("URL:     %s\n", domain.BaseURL)
			if domain.IsOAuth() {
				fmt.Printf("Auth:    OAuth login (client %s)\n", domain.ClientID)
				fmt.Printf("Tokens:  %s\n", auth.StorageDescription(domain))
				if acting := domain.ActingFor(); acting != "" {
					fmt.Printf("Acts for: %s\n", acting)
				} else {
					fmt.Println("Acts for: nothing yet — run: pintomind connection use-account")
				}
				return nil
			}
			masked := domain.APIKey
			if len(masked) > 8 {
				masked = masked[:4] + "..." + masked[len(masked)-4:]
			}
			fmt.Printf("API key: %s\n", masked)
			return nil
		},
	}
}

func newUseAccountCmd(network bool) *cobra.Command {
	kind, other := "account", "network"
	if network {
		kind, other = "network", "account"
	}
	return &cobra.Command{
		Use:   "use-" + kind + " [id]",
		Short: "Make an OAuth connection act for another " + kind + " (asks when no ID is given)",
		Long: "Switches what an OAuth connection acts for without logging in again. Requests then send\n" +
			"the " + map[bool]string{false: "Account-Id", true: "Network-Id"}[network] + " header instead of the " + other + "'s. " +
			"The choices are listed by 'pintomind " + kind + "s'.",
		Example: "  pintomind connection use-" + kind + " 42\n  pintomind connection use-" + kind + " --connection develop",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
				return fmt.Errorf("connection %q uses an API key, which always acts for its own %s; log in with: pintomind login", name, kind)
			}
			verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
			accounts, err := fetchMeAccounts(newUnscopedClient(name, d, verbose))
			if err != nil {
				return err
			}

			var choice accountChoice
			switch {
			case len(args) == 1:
				id, convErr := strconv.Atoi(args[0])
				if convErr != nil {
					return fmt.Errorf("%s ID must be a number", kind)
				}
				if network {
					choice, err = findNetwork(accounts, id)
				} else {
					choice, err = findAccount(accounts, id)
				}
			default:
				choice, err = pickAccount(accounts, d, kind)
			}
			if err == errNoChoice {
				printAccountChoices(cmd, accounts, d)
				return fmt.Errorf("pass the %s ID: pintomind connection use-%s <id>", kind, kind)
			}
			if err != nil {
				if isAborted(err) {
					fmt.Println("Aborted.")
					return nil
				}
				return err
			}
			if err := saveChoice(name, choice); err != nil {
				return err
			}
			choice.apply(&d)
			fmt.Printf("Connection %q now acts for %s.\n", name, d.ActingFor())
			return nil
		},
	}
}
