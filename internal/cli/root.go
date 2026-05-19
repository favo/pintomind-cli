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
	var connectionOverride string
	var jsonOutput bool
	var verbose bool

	root := &cobra.Command{
		Use:     "pintomind",
		Short:   "CLI for the Pintomind / Infoskjermen API",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Check for updates on every command except `update` itself
			if !isUpdateCmd(cmd) {
				commands.CheckForUpdate(version)
			}
			// Skip API client setup for config/update subcommands
			if isConnectionCmd(cmd) || isUpdateCmd(cmd) {
				return nil
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			domainName, domain, err := cfg.ActiveDomain(connectionOverride)
			if err != nil {
				return err
			}

			client := api.New(domain.BaseURL, domain.APIKey)
			client.Verbose = verbose
			app := &appctx.App{
				Config:           cfg,
				Client:           client,
				ActiveConnection: domainName,
				JSONOutput:       jsonOutput,
				Verbose:          verbose,
			}
			cmd.SetContext(appctx.WithApp(cmd.Context(), app))
			return nil
		},
	}

	root.PersistentFlags().StringVar(&connectionOverride, "connection", "", "Override active connection")
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show HTTP request and response details")

	root.AddCommand(commands.NewAPICmd())
	root.AddCommand(commands.NewVersionCmd(version))
	root.AddCommand(commands.NewUpdateCmd(version))
	root.AddCommand(commands.NewPublishCmd())
	root.AddCommand(commands.NewSetupCmd(root))
	root.AddCommand(commands.NewConnectionCmd())
	root.AddCommand(commands.NewMeCmd())
	root.AddCommand(commands.NewNetworkCmd())
	root.AddCommand(commands.NewScreensCmd())
	root.AddCommand(commands.NewChannelsCmd())
	root.AddCommand(commands.NewResourcesCmd())
	root.AddCommand(commands.NewMediaCollectionsCmd())
	root.AddCommand(commands.NewMediaCmd())
	root.AddCommand(commands.NewTasksCmd())
	root.AddCommand(commands.NewMediaBoxesCmd())
	root.AddCommand(commands.NewIconsCmd())
	root.AddCommand(commands.NewPostsCmd())
	root.AddCommand(commands.NewPosterTemplatesCmd())
	root.AddCommand(commands.NewSchemasCmd())
	root.AddCommand(commands.NewThemesCmd())
	root.AddCommand(commands.NewColorPalettesCmd())
	root.AddCommand(commands.NewFontFamiliesCmd())
	root.AddCommand(commands.NewWebhooksCmd())

	root.InitDefaultCompletionCmd()

	return root
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// isConnectionCmd returns true when cmd is or lives under the "connection" command.
func isConnectionCmd(cmd *cobra.Command) bool {
	c := cmd
	for c != nil {
		if c.Name() == "connection" {
			return true
		}
		c = c.Parent()
	}
	return false
}

func isUpdateCmd(cmd *cobra.Command) bool {
	return cmd.Name() == "update" && cmd.Parent() != nil && cmd.Parent().Parent() == nil
}
