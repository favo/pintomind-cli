package commands

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

type Channel struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	OnlineScreens int    `json:"online_screens"`
	OfflineScreens int   `json:"offline_screens"`
}

type ChannelsResponse struct {
	Total int       `json:"total"`
	Items []Channel `json:"items"`
}

func NewChannelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channels",
		Short: "Manage channels",
	}
	cmd.AddCommand(newChannelsListCmd())
	cmd.AddCommand(newChannelsShowCmd())
	cmd.AddCommand(newChannelsPostsCmd())
	cmd.AddCommand(newChannelsStatsCmd())
	cmd.AddCommand(newChannelsSetThemeCmd())
	return cmd
}

func newChannelsListCmd() *cobra.Command {
	var fields string
	var sortBy string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List channels",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			if fields != "" {
				q.Set("fields", fields)
			}
			if sortBy != "" {
				q.Set("sort_by", sortBy)
			}

			var resp ChannelsResponse
			if err := a.Client.Get("/channels", q, &resp); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, c := range resp.Items {
				rows[i] = []string{
					strconv.Itoa(c.ID),
					c.Name,
					c.Type,
					strconv.Itoa(c.OnlineScreens),
					strconv.Itoa(c.OfflineScreens),
				}
			}
			printTable(cmd, []string{"ID", "NAME", "TYPE", "ONLINE", "OFFLINE"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated fields to include")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field (e.g. name, name:desc)")
	return cmd
}

func newChannelsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/channels/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newChannelsPostsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "posts <channel-id>",
		Short: "List posts in a channel (account token required)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/channels/"+args[0]+"/posts", nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newChannelsStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats [channel-id]",
		Short: "Show channel stats (requires channels:read:stats scope)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			path := "/channels/stats"
			if len(args) == 1 {
				path = "/channels/" + args[0] + "/stats"
			}
			var resp map[string]any
			if err := a.Client.Get(path, nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newChannelsSetThemeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-theme <channel-id> <theme-id>",
		Short: "Set the theme of a channel",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			themeID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("theme-id must be an integer")
			}
			body := map[string]any{"channel": map[string]any{"theme_id": themeID}}
			var resp map[string]any
			if err := a.Client.Patch("/channels/"+args[0], body, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
			} else {
				fmt.Printf("Set theme %s on channel %s\n", args[1], args[0])
			}
			return nil
		},
	}
}
