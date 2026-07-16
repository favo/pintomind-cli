package commands

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

type Channel struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	OnlineScreens  int    `json:"online_screens"`
	OfflineScreens int    `json:"offline_screens"`
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
			applyPagination(cmd, q)

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
	addPaginationFlags(cmd)
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

type ChannelPost struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Area     string `json:"area"`
	Position *int   `json:"position"`
	Priority string `json:"priority"`
	Visible  *bool  `json:"visible"`
}

type ChannelPostsResponse struct {
	Total int           `json:"total"`
	Items []ChannelPost `json:"items"`
}

func newChannelsPostsCmd() *cobra.Command {
	var sortBy, fields string
	var visible, hidden bool

	cmd := &cobra.Command{
		Use:   "posts <channel-id>",
		Short: "List posts in a channel (account token required)",
		Example: `  pintomind channels posts 7
  pintomind channels posts 7 --visible                 # currently visible only
  pintomind channels posts 7 --hidden                  # currently hidden only
  pintomind channels posts 7 --sort-by position        # play order (also: area, created_at, updated_at)
  pintomind channels posts 7 --sort-by area,position:desc`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if visible && hidden {
				return fmt.Errorf("--visible and --hidden are mutually exclusive")
			}
			q := url.Values{}
			if visible {
				q.Set("visible", "true")
			}
			if hidden {
				q.Set("visible", "false")
			}
			if sortBy != "" {
				q.Set("sort_by", sortBy)
			}
			if fields != "" {
				q.Set("fields", fields)
			}
			applyPagination(cmd, q)

			if a.JSONOutput {
				var resp map[string]any
				if err := a.Client.Get("/channels/"+args[0]+"/posts", q, &resp); err != nil {
					return err
				}
				printJSON(resp)
				return nil
			}

			var resp ChannelPostsResponse
			if err := a.Client.Get("/channels/"+args[0]+"/posts", q, &resp); err != nil {
				return err
			}
			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, p := range resp.Items {
				pos := ""
				if p.Position != nil {
					pos = strconv.Itoa(*p.Position)
				}
				vis := ""
				if p.Visible != nil {
					vis = strconv.FormatBool(*p.Visible)
				}
				rows[i] = []string{
					strconv.Itoa(p.ID),
					p.Type,
					p.Title,
					p.Area,
					pos,
					p.Priority,
					vis,
				}
			}
			printTable(cmd, []string{"ID", "TYPE", "TITLE", "AREA", "POSITION", "PRIORITY", "VISIBLE"}, rows)
			return nil
		},
	}
	cmd.Flags().BoolVar(&visible, "visible", false, "Only publications that are currently visible")
	cmd.Flags().BoolVar(&hidden, "hidden", false, "Only publications that are currently hidden")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field: area, position, created_at, updated_at (append :desc to reverse; comma-separate for multiple)")
	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated fields to include (e.g. id,title,position,will_be_visible_at)")
	addPaginationFlags(cmd)
	return cmd
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
	var all bool

	cmd := &cobra.Command{
		Use:   "set-theme [channel-id] <theme-id>",
		Short: "Set the theme of a channel or all channels",
		Example: `  pintomind channels set-theme 3 12
  pintomind channels set-theme --all 12`,
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)

			var singleChannelID, themeIDStr string
			if all {
				if len(args) != 1 {
					return fmt.Errorf("expected <theme-id> when using --all")
				}
				themeIDStr = args[0]
			} else {
				if len(args) != 2 {
					return fmt.Errorf("expected <channel-id> <theme-id>")
				}
				singleChannelID = args[0]
				themeIDStr = args[1]
			}

			themeID, err := strconv.Atoi(themeIDStr)
			if err != nil {
				return fmt.Errorf("theme-id must be an integer")
			}

			channelIDs, err := resolveChannelIDs(cmd, singleChannelID, all)
			if err != nil {
				return err
			}

			body := map[string]any{"channel": map[string]any{"theme_id": themeID}}
			for _, id := range channelIDs {
				var resp map[string]any
				if err := a.Client.Patch("/channels/"+id, body, &resp); err != nil {
					return fmt.Errorf("channel %s: %w", id, err)
				}
				if a.JSONOutput {
					printJSON(resp)
				} else {
					fmt.Printf("Set theme %s on channel %s\n", themeIDStr, id)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Apply to all channels")
	return cmd
}
