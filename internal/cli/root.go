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
			// Skip API client setup for config/update/setup subcommands
			if isConnectionCmd(cmd) || isUpdateCmd(cmd) || isSetupCmd(cmd) {
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

	const (
		groupAPI  = "api"
		groupTool = "tool"
	)
	root.AddGroup(
		&cobra.Group{ID: groupAPI, Title: "Available commands:"},
		&cobra.Group{ID: groupTool, Title: "Local commands:"},
	)

	addAPI := func(c *cobra.Command) {
		c.GroupID = groupAPI
		root.AddCommand(c)
	}
	addTool := func(c *cobra.Command) {
		c.GroupID = groupTool
		root.AddCommand(c)
	}

	addAPI(commands.NewScreensCmd())
	addAPI(commands.NewChannelsCmd())
	addAPI(commands.NewPostsCmd())
	addAPI(commands.NewMediaCmd())
	addAPI(commands.NewMediaCollectionsCmd())
	addAPI(commands.NewMediaBoxesCmd())
	addAPI(commands.NewResourcesCmd())
	addAPI(commands.NewPosterTemplatesCmd())
	addAPI(commands.NewThemesCmd())
	addAPI(commands.NewColorPalettesCmd())
	addAPI(commands.NewFontFamiliesCmd())
	addAPI(commands.NewIconsCmd())
	addAPI(commands.NewWebhooksCmd())
	addAPI(commands.NewTasksCmd())
	addAPI(commands.NewSchemasCmd())
	addAPI(commands.NewMeCmd())
	addAPI(commands.NewNetworkCmd())
	addAPI(commands.NewAPICmd())

	addTool(commands.NewConnectionCmd())
	addTool(commands.NewSetupCmd(root))
	addTool(commands.NewUpdateCmd(version))
	addTool(commands.NewVersionCmd(version))

	root.SetHelpCommandGroupID(groupTool)
	root.SetCompletionCommandGroupID(groupTool)
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

// isSetupCmd returns true when cmd is or lives under the "setup" command.
func isSetupCmd(cmd *cobra.Command) bool {
	c := cmd
	for c != nil {
		if c.Name() == "setup" {
			return true
		}
		c = c.Parent()
	}
	return false
}
