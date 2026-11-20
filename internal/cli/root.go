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
	var accountOverride, networkOverride int

	root := &cobra.Command{
		Use:     "pintomind",
		Short:   "CLI for the Pintomind / Infoskjermen API",
		Version: version,
		// Most failures are API errors, not usage problems: don't dump the usage
		// text, and let Execute print the error exactly once.
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Check for updates on every command except `update` itself
			if !isUpdateCmd(cmd) {
				commands.CheckForUpdate(version)
			}
			// Skip API client setup for config/update/setup subcommands
			if isConnectionCmd(cmd) || isUpdateCmd(cmd) || isSetupCmd(cmd) || isAuthCmd(cmd) {
				return nil
			}
			if accountOverride != 0 && networkOverride != 0 {
				return fmt.Errorf("--account-id and --network-id are mutually exclusive")
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// Environment variables override the stored config when no explicit
			// --connection is given. Handy for CI and scripted/headless use.
			var client *api.Client
			var domainName string
			if connectionOverride == "" && os.Getenv("PINTOMIND_API_KEY") != "" {
				baseURL := os.Getenv("PINTOMIND_API_URL")
				if baseURL == "" {
					baseURL = "https://app.infoskjermen.no"
				}
				domainName = "env"
				client = api.New(baseURL, os.Getenv("PINTOMIND_API_KEY"))
				client.Verbose = verbose
			} else {
				name, domain, err := cfg.ActiveDomain(connectionOverride)
				if err != nil {
					return err
				}
				if accountOverride != 0 || networkOverride != 0 {
					if !domain.IsOAuth() {
						return fmt.Errorf("--account-id and --network-id need an OAuth connection (pintomind login); connection %q uses an API key", name)
					}
					domain.AccountID, domain.NetworkID = accountOverride, networkOverride
				}
				if domain.IsOAuth() && domain.AccountID == 0 && domain.NetworkID == 0 && cmd.Annotations[commands.NoAccountAnnotation] == "" {
					return fmt.Errorf("connection %q acts for no account yet — run: pintomind connection use-account (or use-network)", name)
				}
				domainName = name
				client = commands.NewClient(name, domain, verbose)
			}
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

	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return fmt.Errorf("%w\nRun '%s --help' for usage.", err, cmd.CommandPath())
	})

	root.PersistentFlags().StringVar(&connectionOverride, "connection", "", "Override active connection")
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show HTTP request and response details")
	root.PersistentFlags().IntVar(&accountOverride, "account-id", 0, "Act for this account for one command (OAuth connections)")
	root.PersistentFlags().IntVar(&networkOverride, "network-id", 0, "Act for this network for one command (OAuth connections)")

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
	addAPI(commands.NewEmojisCmd())
	addAPI(commands.NewUnsplashCmd())
	addAPI(commands.NewGiphyCmd())
	addAPI(commands.NewWebhooksCmd())
	addAPI(commands.NewTasksCmd())
	addAPI(commands.NewSchemasCmd())
	addAPI(commands.NewMeCmd())
	addAPI(commands.NewAccountsCmd())
	addAPI(commands.NewNetworksCmd())
	addAPI(commands.NewNetworkCmd())
	addAPI(commands.NewAPICmd())

	addTool(commands.NewLoginCmd())
	addTool(commands.NewLogoutCmd())
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
		fmt.Fprintln(os.Stderr, "Error:", err)
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

func isAuthCmd(cmd *cobra.Command) bool {
	return (cmd.Name() == "login" || cmd.Name() == "logout") && cmd.Parent() != nil && cmd.Parent().Parent() == nil
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
