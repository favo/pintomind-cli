package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

type Post struct {
	ID    int    `json:"id"`
	Type  string `json:"type"`
	Name  string `json:"name"`
	Title string `json:"title"`
	State string `json:"state"`
}

type PostsResponse struct {
	Total int    `json:"total"`
	Items []Post `json:"items"`
}

type Publication struct {
	ID         int    `json:"id"`
	ChannelID  int    `json:"channel_id"`
	Area       string `json:"area"`
	Position   *int   `json:"position"`
	FullScreen bool   `json:"full_screen"`
	Visible    bool   `json:"visible"`
}

type PublicationsResponse struct {
	Total int           `json:"total"`
	Items []Publication `json:"items"`
}

func NewPostsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "posts",
		Short: "Manage posts and channel publications (account token required)",
	}
	cmd.AddCommand(newPostsListCmd())
	cmd.AddCommand(newPostsShowCmd())
	createCmd := newPostsCreateCmd()
	createCmd.AddCommand(newPostsCreateImageCmd())
	createCmd.AddCommand(newPostsCreatePlainCmd())
	createCmd.AddCommand(newPostsCreateFeedCmd())
	createCmd.AddCommand(newPostsCreateCalendarCmd())
	createCmd.AddCommand(newPostsCreateIframeCmd())
	createCmd.AddCommand(newPostsCreateHTMLCmd())
	createCmd.AddCommand(newPostsCreatePosterCmd())
	createCmd.AddCommand(newPostsCreateCounterCmd())
	cmd.AddCommand(createCmd)
	cmd.AddCommand(newPostsUpdateCmd())
	cmd.AddCommand(newPostsDeleteCmd())
	cmd.AddCommand(newPostsPublicationsCmd())
	cmd.AddCommand(newPostsPublishCmd())
	cmd.AddCommand(newPostsUnpublishCmd())
	cmd.AddCommand(newPostsScheduleCmd())
	cmd.AddCommand(newPostsNotificationsCmd())
	cmd.AddCommand(newSchemaSubCmd(map[string]string{
		"calendar":    "post_calendar",
		"clock":       "post_clock",
		"counter":     "post_counter",
		"entur":       "post_entur",
		"feed":        "post_feed",
		"forecast":    "post_forecast",
		"iframe":      "post_iframe",
		"image":       "post_image",
		"plain":       "post_plain",
		"poster":      "post_poster",
		"power-price":   "post_power_price",
		"time-schedule": "post_time_schedule",
		"video":         "post_video",
		"world-clock":   "post_world_clock",
		"youtube":       "post_youtube",
	}))
	return cmd
}

func newPostsListCmd() *cobra.Command {
	var postType, sortBy, fields string
	var archived, deleted bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List posts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			if postType != "" {
				q.Set("type", postType)
			}
			if sortBy != "" {
				q.Set("sort_by", sortBy)
			}
			if fields != "" {
				q.Set("fields", fields)
			}
			if archived {
				q.Set("archived", "true")
			}
			if deleted {
				q.Set("deleted", "true")
			}
			applyPagination(cmd, q)

			var resp PostsResponse
			if err := a.Client.Get("/posts", q, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, p := range resp.Items {
				rows[i] = []string{
					strconv.Itoa(p.ID),
					p.Type,
					firstNonEmpty(p.Name, p.Title),
					p.State,
				}
			}
			printTable(cmd, []string{"ID", "TYPE", "NAME", "STATE"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&postType, "type", "", "Filter by post type alias (comma-separated for multiple)")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field (e.g. created_at:desc)")
	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated fields to include")
	cmd.Flags().BoolVar(&archived, "archived", false, "Show archived posts only")
	cmd.Flags().BoolVar(&deleted, "deleted", false, "Show soft-deleted posts only")
	addPaginationFlags(cmd)
	return cmd
}

func newPostsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a post",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/posts/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newPostsCreateCmd() *cobra.Command {
	var postType, data, sourceID string

	cmd := &cobra.Command{
		Use:   "create <type> [flags]",
		Short: "Create a post (use a type subcommand below)",
		Long: `Create a post.

Pick the subcommand for the post type you want — each one has friendly flags
(--name, --image, --channel-id, etc.) and an interactive mode (-i):

  pintomind posts create image -i
  pintomind posts create plain --heading "<p>Hi</p>" --body "<p>World</p>"
  pintomind posts create feed --url https://example.com/feed.xml --name "News"
  pintomind posts create poster --source-id 42 --name "My poster"

Advanced: pass raw API JSON with --type and --data (skips validation helpers).`,
		Example: `  pintomind posts create image --name "Spring" --image ./a.jpg --channel-id 7
  pintomind posts create plain -i
  pintomind posts create feed --url https://example.com/feed.xml

  # Advanced (raw JSON):
  pintomind posts create --type image --data '{"name":"Spring","images":[{"media_id":42}]}'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if postType == "" {
				return fmt.Errorf("specify a post type subcommand (e.g. `posts create image`) or pass --type with --data — see `posts create --help`")
			}
			a := app(cmd)
			body := map[string]any{"type": postType}
			if data != "" {
				var post map[string]any
				if err := json.Unmarshal([]byte(data), &post); err != nil {
					return fmt.Errorf("invalid JSON for --data: %w", err)
				}
				body["post"] = post
			}
			if sourceID != "" {
				body["source_id"] = sourceID
			}
			var resp map[string]any
			if err := a.Client.Post("/posts", body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&postType, "type", "", "Advanced: post type alias for raw JSON mode (use a subcommand instead)")
	cmd.Flags().StringVar(&data, "data", "", "Advanced: raw post fields as JSON (paired with --type)")
	cmd.Flags().StringVar(&sourceID, "source-id", "", "Duplicate a network poster template (poster type only)")
	return cmd
}

func newPostsUpdateCmd() *cobra.Command {
	var data string

	cmd := &cobra.Command{
		Use:   "update <id> --data '<json>'",
		Short: "Update a post (type cannot be changed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var post map[string]any
			if err := json.Unmarshal([]byte(data), &post); err != nil {
				return fmt.Errorf("invalid JSON for --data: %w", err)
			}
			body := map[string]any{"post": post}
			var resp map[string]any
			if err := a.Client.Patch("/posts/"+args[0], body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "Post fields as JSON (required)")
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

func newPostsDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a post (soft-delete; second call hard-deletes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if !force {
				fmt.Printf("Delete post %s? This soft-deletes (run again to hard-delete). Pass --force to skip this prompt.\n", args[0])
				var confirm string
				fmt.Print("Type 'yes' to confirm: ")
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if err := a.Client.Delete("/posts/" + args[0]); err != nil {
				return err
			}
			fmt.Printf("Deleted post %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newPostsPublicationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "publications <post-id>",
		Short: "List a post's channel publications",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp PublicationsResponse
			if err := a.Client.Get("/posts/"+args[0]+"/publications", nil, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, p := range resp.Items {
				pos := ""
				if p.Position != nil {
					pos = strconv.Itoa(*p.Position)
				}
				rows[i] = []string{
					strconv.Itoa(p.ID),
					strconv.Itoa(p.ChannelID),
					p.Area,
					pos,
					strconv.FormatBool(p.FullScreen),
					strconv.FormatBool(p.Visible),
				}
			}
			printTable(cmd, []string{"ID", "CHANNEL_ID", "AREA", "POSITION", "FULLSCREEN", "VISIBLE"}, rows)
			return nil
		},
	}
}

func newPostsPublishCmd() *cobra.Command {
	var channelIDs []int
	var area string
	var fullScreen bool
	var position int

	cmd := &cobra.Command{
		Use:   "publish <post-id> --channel-id N [--channel-id N ...]",
		Short: "Publish a post to one or more channels",
		Example: `  pintomind posts publish 123 --channel-id 7
  pintomind posts publish 123 --channel-id 7 --channel-id 12
  pintomind posts publish 123 --channel-id 7 --area F11
  pintomind posts publish 123 --channel-id 7 --full-screen
  pintomind posts publish 123 --channel-id 7 --position 0   # first in the area's play order`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if len(channelIDs) == 0 {
				return fmt.Errorf("at least one --channel-id is required")
			}
			publication := map[string]any{"channel_ids": channelIDs}
			if area != "" {
				publication["area"] = area
			}
			if fullScreen {
				publication["full_screen"] = true
			}
			if position >= 0 {
				publication["position"] = position
			}
			body := map[string]any{"publication": publication}
			var resp map[string]any
			if err := a.Client.Post("/posts/"+args[0]+"/publications", body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().IntSliceVar(&channelIDs, "channel-id", nil, "Channel ID to publish to (repeatable; required)")
	cmd.Flags().StringVar(&area, "area", "", "Channel area for all channels (defaults to the post's area; pass F11 for fullscreen)")
	cmd.Flags().BoolVar(&fullScreen, "full-screen", false, "Publish as fullscreen to all channels (alternative to --area F11)")
	cmd.Flags().IntVar(&position, "position", -1, "0-based play-order position within the channel area (0 = first; omit for channel default)")
	_ = cmd.MarkFlagRequired("channel-id")
	return cmd
}

func newPostsUnpublishCmd() *cobra.Command {
	var channelIDs []int
	var force bool

	cmd := &cobra.Command{
		Use:   "unpublish <post-id> [--channel-id N ...]",
		Short: "Unpublish a post from specific channels (or all channels if none specified)",
		Example: `  pintomind posts unpublish 123                          # all channels
  pintomind posts unpublish 123 --channel-id 7           # just channel 7
  pintomind posts unpublish 123 --channel-id 7 --channel-id 12 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			target := fmt.Sprintf("%d channel(s)", len(channelIDs))
			if len(channelIDs) == 0 {
				target = "ALL channels"
			}
			if !force {
				fmt.Printf("Unpublish post %s from %s? Pass --force to skip this prompt.\n", args[0], target)
				var confirm string
				fmt.Print("Type 'yes' to confirm: ")
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			var body any
			if len(channelIDs) > 0 {
				body = map[string]any{"channel_ids": channelIDs}
			}
			var resp map[string]any
			if err := a.Client.DeleteWithBody("/posts/"+args[0]+"/publications", body, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}
			deleted := 0
			if n, ok := resp["deleted_count"].(float64); ok {
				deleted = int(n)
			}
			fmt.Printf("Unpublished post %s from %d channel(s)\n", args[0], deleted)
			return nil
		},
	}
	cmd.Flags().IntSliceVar(&channelIDs, "channel-id", nil, "Channel ID to unpublish from (repeatable; omit to unpublish from all)")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
