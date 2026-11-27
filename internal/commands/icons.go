package commands

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func NewIconsCmd() *cobra.Command {
	var category string
	cmd := &cobra.Command{
		Use:   "icons [query]",
		Short: "List icon names, types and categories for icon media boxes (filter by name part or --category)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getJSON(cmd, "/icons", lookupQuery(args, category))
		},
	}
	cmd.Flags().StringVar(&category, "category", "", "Only icons in this category")
	return cmd
}

func lookupQuery(args []string, category string) url.Values {
	q := url.Values{}
	if len(args) == 1 {
		q.Set("q", args[0])
	}
	setStringIf(q, "category", category)
	return q
}

func NewEmojisCmd() *cobra.Command {
	var category string
	cmd := &cobra.Command{
		Use:   "emojis [query]",
		Short: "Find emojis for emoji media boxes (media_box.emoji); a short preset without a query",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			q := lookupQuery(args, category)
			if a.JSONOutput {
				return getJSON(cmd, "/emojis", q)
			}
			var resp struct {
				Items []struct {
					Emoji    string `json:"emoji"`
					Name     string `json:"name"`
					Category string `json:"category"`
				} `json:"items"`
				Categories []string `json:"categories"`
			}
			if err := a.Client.Get("/emojis", q, &resp); err != nil {
				return err
			}
			rows := make([][]string, len(resp.Items))
			for i, e := range resp.Items {
				rows[i] = []string{e.Emoji, e.Name, e.Category}
			}
			printTable(cmd, []string{"EMOJI", "NAME", "CATEGORY"}, rows)
			if len(resp.Categories) > 0 {
				fmt.Printf("\nCategories: %s\n", strings.Join(resp.Categories, ", "))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&category, "category", "", "Only emojis in this category")
	return cmd
}

func NewUnsplashCmd() *cobra.Command {
	var page int
	cmd := &cobra.Command{
		Use:   "unsplash [query]",
		Short: "Search Unsplash photos for unsplash media boxes (media_box.photo_id); popular photos without a query",
		Long:  "Search Unsplash photos. Credit the photographer (AUTHOR) wherever the photo is shown.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			q := lookupQuery(args, "")
			if page > 0 {
				q.Set("page", strconv.Itoa(page))
			}
			if a.JSONOutput {
				return getJSON(cmd, "/unsplash", q)
			}
			var resp struct {
				Items []struct {
					ID          string `json:"id"`
					Width       int    `json:"width"`
					Height      int    `json:"height"`
					Description string `json:"description"`
					AuthorName  string `json:"author_name"`
					URL         string `json:"url"`
				} `json:"items"`
				Page       int `json:"page"`
				TotalPages int `json:"total_pages"`
			}
			if err := a.Client.Get("/unsplash", q, &resp); err != nil {
				return err
			}
			rows := make([][]string, len(resp.Items))
			for i, p := range resp.Items {
				rows[i] = []string{p.ID, fmt.Sprintf("%dx%d", p.Width, p.Height), p.AuthorName, truncate(strings.TrimSpace(p.Description), 50)}
			}
			printTable(cmd, []string{"PHOTO ID", "SIZE", "AUTHOR", "DESCRIPTION"}, rows)
			if resp.TotalPages > 1 {
				fmt.Printf("\nPage %d of %d (--page for more)\n", resp.Page, resp.TotalPages)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "Page number")
	return cmd
}

func NewGiphyCmd() *cobra.Command {
	var offset int
	cmd := &cobra.Command{
		Use:   "giphy [query]",
		Short: "Search Giphy for gif media boxes (media_box.gif_id); featured GIFs without a query",
		Long:  `Search Giphy. Show "Powered by GIPHY" wherever the results are shown.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			q := lookupQuery(args, "")
			if offset > 0 {
				q.Set("offset", strconv.Itoa(offset))
			}
			if a.JSONOutput {
				return getJSON(cmd, "/giphy", q)
			}
			var resp struct {
				Items []struct {
					ID     string `json:"id"`
					Title  string `json:"title"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"items"`
				NextOffset *int `json:"next_offset"`
			}
			if err := a.Client.Get("/giphy", q, &resp); err != nil {
				return err
			}
			rows := make([][]string, len(resp.Items))
			for i, g := range resp.Items {
				rows[i] = []string{g.ID, fmt.Sprintf("%dx%d", g.Width, g.Height), truncate(g.Title, 60)}
			}
			printTable(cmd, []string{"GIF ID", "SIZE", "TITLE"}, rows)
			if resp.NextOffset != nil {
				fmt.Printf("\nMore: --offset %d\n", *resp.NextOffset)
			}
			fmt.Println("Powered by GIPHY")
			return nil
		},
	}
	cmd.Flags().IntVar(&offset, "offset", 0, "Offset for more results (next_offset of the previous page)")
	return cmd
}

func newPostsPlainLayoutsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plain-layouts",
		Short: "List the layouts (variation) a plain post can have",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			if a.JSONOutput {
				return getJSON(cmd, "/plain_layouts", nil)
			}
			var resp struct {
				Items []struct {
					Name       string `json:"name"`
					MediaCount int    `json:"media_count"`
					Heading    bool   `json:"heading"`
					Body       bool   `json:"body"`
				} `json:"items"`
			}
			if err := a.Client.Get("/plain_layouts", nil, &resp); err != nil {
				return err
			}
			rows := make([][]string, len(resp.Items))
			for i, l := range resp.Items {
				rows[i] = []string{l.Name, strconv.Itoa(l.MediaCount), check(l.Heading), check(l.Body)}
			}
			printTable(cmd, []string{"VARIATION", "MEDIA", "HEADING", "BODY"}, rows)
			return nil
		},
	}
}
