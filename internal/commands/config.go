package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/config"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage domains and configuration",
	}
	cmd.AddCommand(newConfigAddCmd())
	cmd.AddCommand(newConfigRemoveCmd())
	cmd.AddCommand(newConfigListCmd())
	cmd.AddCommand(newConfigUseCmd())
	return cmd
}

func newConfigAddCmd() *cobra.Command {
	var apiKey string
	var baseURL string

	cmd := &cobra.Command{
		Use:   "add <domain>",
		Short: "Add a domain with its API key",
		Args:  cobra.ExactArgs(1),
		Example: `  pintomind config add app.infoskjermen.no --api-key sk-xxx
  pintomind config add develop --api-key sk-dev --url https://develop.infoskjermen.no
  pintomind config use develop`,
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			url := baseURL
			if url == "" {
				url = "https://" + domain
			}

			cfg.Domains[domain] = config.Domain{
				APIKey:  apiKey,
				BaseURL: url,
			}
			if cfg.DefaultDomain == "" {
				cfg.DefaultDomain = domain
			}

			if err := cfg.Save(); err != nil {
				return err
			}

			fmt.Printf("Added domain %q (%s)\n", domain, url)
			if cfg.DefaultDomain == domain {
				fmt.Println("Set as default domain.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for this domain (required)")
	cmd.Flags().StringVar(&baseURL, "url", "", "Base URL (defaults to https://<domain>, e.g. https://app.infoskjermen.no)")
	_ = cmd.MarkFlagRequired("api-key")
	return cmd
}

func newConfigRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <domain>",
		Short: "Remove a domain from config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Domains[domain]; !ok {
				return fmt.Errorf("domain %q not found", domain)
			}
			delete(cfg.Domains, domain)
			if cfg.DefaultDomain == domain {
				cfg.DefaultDomain = ""
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Printf("Removed domain %q\n", domain)
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured domains",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Domains) == 0 {
				fmt.Println("No domains configured. Run: pintomind config add app.infoskjermen.no --api-key <key>")
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
		Use:   "use <domain>",
		Short: "Set the default domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Domains[domain]; !ok {
				return fmt.Errorf("domain %q not found — add it first with: pintomind config add %s --api-key <key>", domain, domain)
			}
			cfg.DefaultDomain = domain
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Printf("Default domain set to %q\n", domain)
			return nil
		},
	}
}
