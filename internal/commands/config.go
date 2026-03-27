package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/config"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage accounts and configuration",
	}
	cmd.AddCommand(newConfigAddCmd())
	cmd.AddCommand(newConfigRemoveCmd())
	cmd.AddCommand(newConfigListCmd())
	cmd.AddCommand(newConfigUseCmd())
	cmd.AddCommand(newConfigShowCmd())
	return cmd
}

func newConfigAddCmd() *cobra.Command {
	var baseURL string

	cmd := &cobra.Command{
		Use:   "add <name> <api-key>",
		Short: "Add an account with its API key",
		Args:  cobra.ExactArgs(2),
		Example: `  pintomind config add app.infoskjermen.no sk-xxx
  pintomind config add develop sk-dev --url https://develop.infoskjermen.no
  pintomind config use develop`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			apiKey := args[1]

			cfg, err := config.Load()
			if err != nil {
				return err
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

			fmt.Printf("Added account %q (%s)\n", name, url)
			if cfg.DefaultDomain == name {
				fmt.Println("Set as default account.")
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
		Short: "Remove an account from config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Domains[name]; !ok {
				return fmt.Errorf("account %q not found", name)
			}
			delete(cfg.Domains, name)
			if cfg.DefaultDomain == name {
				cfg.DefaultDomain = ""
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Printf("Removed account %q\n", name)
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured accounts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Domains) == 0 {
				fmt.Println("No accounts configured. Run: pintomind config add app.infoskjermen.no <api-key>")
				return nil
			}
			for name, d := range cfg.Domains {
				marker := "  "
				if name == cfg.DefaultDomain {
					marker = "* "
				}
				fmt.Printf("%s%s  %s\n", marker, name, d.BaseURL)
			}
			return nil
		},
	}
}

func newConfigUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set the default account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Domains[name]; !ok {
				return fmt.Errorf("account %q not found — add it first with: pintomind config add %s <api-key>", name, name)
			}
			cfg.DefaultDomain = name
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Printf("Default account set to %q\n", name)
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the active account and its configuration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			accountOverride, _ := cmd.Root().PersistentFlags().GetString("account")
			name, domain, err := cfg.ActiveDomain(accountOverride)
			if err != nil {
				return err
			}
			fmt.Printf("Account: %s\n", name)
			fmt.Printf("URL:     %s\n", domain.BaseURL)
			masked := domain.APIKey
			if len(masked) > 8 {
				masked = masked[:4] + "..." + masked[len(masked)-4:]
			}
			fmt.Printf("API key: %s\n", masked)
			return nil
		},
	}
}
