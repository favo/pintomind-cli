package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/api"
	"favo/pintomind-cli/internal/appctx"
	"favo/pintomind-cli/internal/commands"
	"favo/pintomind-cli/internal/config"
)

var version = "dev"

func NewRootCmd() *cobra.Command {
	var accountOverride string
	var jsonOutput bool
	var verbose bool

	root := &cobra.Command{
		Use:     "pintomind",
		Short:   "CLI for the Pintomind / Infoskjermen API",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Skip app setup for config subcommands (no API client needed)
			if isConfigCmd(cmd) {
				return nil
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			domainName, domain, err := cfg.ActiveDomain(accountOverride)
			if err != nil {
				return err
			}

			client := api.New(domain.BaseURL, domain.APIKey)
			client.Verbose = verbose
			app := &appctx.App{
				Config:        cfg,
				Client:        client,
				ActiveAccount: domainName,
				JSONOutput:    jsonOutput,
				Verbose:       verbose,
			}
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
	}

	root.PersistentFlags().StringVar(&accountOverride, "account", "", "Override active account")
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show HTTP request and response details")

	root.AddCommand(commands.NewAPICmd())
	root.AddCommand(commands.NewVersionCmd(version))
	root.AddCommand(commands.NewSetupCmd(root))
	root.AddCommand(commands.NewConfigCmd())
	root.AddCommand(commands.NewMeCmd())
	root.AddCommand(commands.NewNetworkCmd())
	root.AddCommand(commands.NewScreensCmd())
	root.AddCommand(commands.NewChannelsCmd())
	root.AddCommand(commands.NewResourcesCmd())
	root.AddCommand(commands.NewMediaCollectionsCmd())
	root.AddCommand(commands.NewMediaCmd())
	root.AddCommand(commands.NewPostsCmd())
	root.AddCommand(commands.NewSchemasCmd())
	root.AddCommand(commands.NewThemesCmd())
	root.AddCommand(commands.NewColorPalettesCmd())
	root.AddCommand(commands.NewFontFamiliesCmd())

	root.InitDefaultCompletionCmd()

	return root
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// isConfigCmd returns true when cmd is or lives under the "config" command.
func isConfigCmd(cmd *cobra.Command) bool {
	c := cmd
	for c != nil {
		if c.Name() == "config" {
			return true
		}
		c = c.Parent()
	}
	return false
}
