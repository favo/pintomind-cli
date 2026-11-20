package commands

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

func newScreensDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Disconnect a screen from the account (it shows a new code and can be connected again)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmAction(fmt.Sprintf("Disconnect screen %s from the account?", args[0]), force) {
				return nil
			}
			return runDelete(cmd, "/screens/"+args[0], "screen "+args[0])
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

type ProvisioningToken struct {
	ID        string `json:"id"`
	ScreenID  *int   `json:"screen_id"`
	NewScreen *struct {
		Name      *string `json:"name"`
		ChannelID *int    `json:"channel_id"`
	} `json:"new_screen"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

func newScreensProvisioningCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provisioning",
		Short: "One-time tokens a player redeems on first contact to become a screen",
	}

	var screenFilter int
	list := &cobra.Command{
		Use:   "list",
		Short: "List open provisioning tokens (never their secret)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			if screenFilter != 0 {
				q.Set("screen_id", strconv.Itoa(screenFilter))
			}
			if a.JSONOutput {
				return getJSON(cmd, "/provisioning_tokens", q)
			}
			var resp struct {
				Total int                 `json:"total"`
				Items []ProvisioningToken `json:"items"`
			}
			if err := a.Client.Get("/provisioning_tokens", q, &resp); err != nil {
				return err
			}
			printTotal(resp.Total, nil)
			rows := make([][]string, len(resp.Items))
			for i, t := range resp.Items {
				target := ""
				switch {
				case t.ScreenID != nil:
					target = "screen " + strconv.Itoa(*t.ScreenID)
				case t.NewScreen != nil:
					target = "new screen"
					if t.NewScreen.Name != nil && *t.NewScreen.Name != "" {
						target += " " + strconv.Quote(*t.NewScreen.Name)
					}
					if t.NewScreen.ChannelID != nil {
						target += fmt.Sprintf(" on channel %d", *t.NewScreen.ChannelID)
					}
				}
				rows[i] = []string{t.ID, target, t.ExpiresAt, t.CreatedAt}
			}
			printTable(cmd, []string{"ID", "BECOMES", "EXPIRES AT", "CREATED AT"}, rows)
			return nil
		},
	}
	list.Flags().IntVar(&screenFilter, "screen-id", 0, "Only tokens for this screen")

	var screenID, channelID, expiresIn int
	var newScreen bool
	var name string
	create := &cobra.Command{
		Use:   "create (--screen-id <id> | --new-screen [--name <name>] [--channel-id <id>])",
		Short: "Create a provisioning token (the secret is shown only now and works on this host only)",
		Example: `  pintomind screens provisioning create --screen-id 42
  pintomind screens provisioning create --new-screen --name Lobby --channel-id 7 --expires-in 86400`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body := map[string]any{}
			switch {
			case screenID != 0 && newScreen:
				return errors.New("--screen-id and --new-screen are mutually exclusive")
			case screenID != 0:
				body["screen_id"] = screenID
			case newScreen:
				ns := map[string]any{}
				setIfChanged(cmd, ns, "name", "name", name)
				setIfChanged(cmd, ns, "channel-id", "channel_id", channelID)
				body["new_screen"] = ns
			default:
				return errors.New("pass --screen-id or --new-screen")
			}
			setIfChanged(cmd, body, "expires-in", "expires_in", expiresIn)
			return sendJSON(cmd, "POST", "/provisioning_tokens", body)
		},
	}
	create.Flags().IntVar(&screenID, "screen-id", 0, "The screen the device becomes")
	create.Flags().BoolVar(&newScreen, "new-screen", false, "The device becomes a new screen")
	create.Flags().StringVar(&name, "name", "", "Name of the new screen")
	create.Flags().IntVar(&channelID, "channel-id", 0, "Channel of the new screen")
	create.Flags().IntVar(&expiresIn, "expires-in", 0, "Seconds until it expires (default 30 days; 1 hour to 90 days)")

	var force bool
	revoke := &cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke a provisioning token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmAction(fmt.Sprintf("Revoke provisioning token %s?", args[0]), force) {
				return nil
			}
			if err := app(cmd).Client.Delete("/provisioning_tokens/" + args[0]); err != nil {
				return err
			}
			fmt.Printf("Revoked provisioning token %s\n", args[0])
			return nil
		},
	}
	revoke.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	cmd.AddCommand(list, create, revoke)
	return cmd
}
