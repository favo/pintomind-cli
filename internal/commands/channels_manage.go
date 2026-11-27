package commands

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

func newChannelsCreateCmd() *cobra.Command {
	var name, channelType, template, orientation, extension string
	var themeID int

	cmd := &cobra.Command{
		Use:   "create --name <name>",
		Short: "Create a channel (needs a role that manages channels)",
		Example: `  pintomind channels create --name Lobby
  pintomind channels create --name Lobby --template ABC --orientation portrait --extension ticker
  pintomind channels create --name "Our app" --type go`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			channel := map[string]any{"name": name}
			setIfChanged(cmd, channel, "type", "type", channelType)
			setIfChanged(cmd, channel, "theme-id", "theme_id", themeID)
			setIfChanged(cmd, channel, "template", "template", template)
			setIfChanged(cmd, channel, "orientation", "orientation", orientation)
			setIfChanged(cmd, channel, "extension", "extension", extension)
			return sendJSON(cmd, "POST", "/channels", map[string]any{"channel": channel})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Channel name (required)")
	cmd.Flags().StringVar(&channelType, "type", "", "screen (default) or go")
	cmd.Flags().IntVar(&themeID, "theme-id", 0, "Theme ID (default: the account's standard theme)")
	addChannelLayoutFlags(cmd, &template, &orientation, &extension)
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func addChannelLayoutFlags(cmd *cobra.Command, template, orientation, extension *string) {
	cmd.Flags().StringVar(template, "template", "", "Area layout such as AB (screen channels; see 'channels layouts')")
	cmd.Flags().StringVar(orientation, "orientation", "", "landscape or portrait (screen channels)")
	cmd.Flags().StringVar(extension, "extension", "", "Ticker, clock and logo variant from 'channels layouts' (screen channels)")
}

func newChannelsUpdateCmd() *cobra.Command {
	var name, template, orientation, extension string
	var boardLabel, boardBackground, boardIcon string
	var themeID, boardPaletteID, boardMediaBoxID int

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a channel: name, theme, screen layout or Go header",
		Long: `Only the given fields change. Layout flags apply to screen channels, --board-* flags to Go
channels; a field that doesn't fit the channel type answers 422.`,
		Example: `  pintomind channels update 7 --name "Lobby (new)"
  pintomind channels update 7 --template AB --orientation landscape
  pintomind channels update 9 --board-label "News" --board-background-color primary-600`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			channel := map[string]any{}
			setIfChanged(cmd, channel, "name", "name", name)
			setIfChanged(cmd, channel, "theme-id", "theme_id", themeID)
			setIfChanged(cmd, channel, "template", "template", template)
			setIfChanged(cmd, channel, "orientation", "orientation", orientation)
			setIfChanged(cmd, channel, "extension", "extension", extension)
			setIfChanged(cmd, channel, "board-label", "board_label", boardLabel)
			setIfChanged(cmd, channel, "board-color-palette-id", "board_color_palette_id", boardPaletteID)
			setIfChanged(cmd, channel, "board-background-color", "board_background_color", boardBackground)
			setIfChanged(cmd, channel, "board-icon-color", "board_icon_color", boardIcon)
			setIfChanged(cmd, channel, "board-media-box-id", "board_media_box_id", boardMediaBoxID)
			if len(channel) == 0 {
				return errors.New("provide at least one field to update")
			}
			return sendJSON(cmd, "PATCH", "/channels/"+args[0], map[string]any{"channel": channel})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Channel name")
	cmd.Flags().IntVar(&themeID, "theme-id", 0, "Theme ID")
	addChannelLayoutFlags(cmd, &template, &orientation, &extension)
	cmd.Flags().StringVar(&boardLabel, "board-label", "", "Go header label")
	cmd.Flags().IntVar(&boardPaletteID, "board-color-palette-id", 0, "Go header colour palette ID")
	cmd.Flags().StringVar(&boardBackground, "board-background-color", "", "Go header background: palette colour (primary-600) or hex (#FFFFFF)")
	cmd.Flags().StringVar(&boardIcon, "board-icon-color", "", "Go header icon colour: palette colour or hex")
	cmd.Flags().IntVar(&boardMediaBoxID, "board-media-box-id", 0, "Go header media box ID")
	return cmd
}

func newChannelsDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a channel (soft-delete takes it off screens and schedules; a second call hard-deletes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if ok, err := confirmAction(fmt.Sprintf("Delete channel %s? Its screens lose it and it leaves their schedules.", args[0]), force); !ok {
				return err
			}
			return runDelete(cmd, "/channels/"+args[0], "channel "+args[0])
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newChannelsMovePostCmd() *cobra.Command {
	var area string
	var position int

	cmd := &cobra.Command{
		Use:   "move-post <channel-id> <post-id>",
		Short: "Move a published post to another area or position in the channel",
		Example: `  pintomind channels move-post 7 123 --position 0       # first in its area
  pintomind channels move-post 7 123 --area B
  pintomind channels move-post 7 123 --area F11            # full screen`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			publication := map[string]any{}
			setIfChanged(cmd, publication, "area", "area", area)
			setIfChanged(cmd, publication, "position", "position", position)
			if len(publication) == 0 {
				return errors.New("provide --area and/or --position")
			}
			return sendJSON(cmd, "PATCH", "/channels/"+args[0]+"/posts/"+args[1], map[string]any{"publication": publication})
		},
	}
	cmd.Flags().StringVar(&area, "area", "", "Area code, or F11 for full screen (default: the current area)")
	cmd.Flags().IntVar(&position, "position", 0, "Zero-based position in the area (default: keep the place, or first in a new area)")
	return cmd
}

func newChannelsPreviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "preview <id>",
		Short: "Get a signed preview link for a channel (valid for an hour, no login needed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return printPreview(cmd, "/channels/"+args[0]+"/preview", nil)
		},
	}
}

func printPreview(cmd *cobra.Command, path string, body any) error {
	a := app(cmd)
	var resp map[string]any
	if err := a.Client.Post(path, body, &resp); err != nil {
		return err
	}
	if a.JSONOutput {
		printJSON(resp)
		return nil
	}
	previewURL, _ := resp["preview_url"].(string)
	if previewURL == "" {
		printJSON(resp)
		return nil
	}
	fmt.Println(previewURL)
	if expires, ok := resp["expires_at"].(string); ok && expires != "" {
		fmt.Printf("Expires: %s\n", expires)
	}
	return nil
}

type ChannelSearchResult struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Label   *string `json:"label"`
	Link    *string `json:"link"`
	Account *struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"account"`
	Distance *int `json:"distance"`
}

func newChannelsSearchCmd() *cobra.Command {
	var query, fields string
	var lat, lng float64

	cmd := &cobra.Command{
		Use:   "search [query]",
		Long:  "The query matches the channel name or the name of the account owning it.",
		Short: "Search publicly shared Go channels across all accounts (channels:search scope)",
		Example: `  pintomind channels search kirke
  pintomind channels search --lat 60.39 --lng 5.32     # nearest first`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if len(args) == 1 {
				query = args[0]
			}
			q := url.Values{}
			setStringIf(q, "q", query)
			if cmd.Flags().Changed("lat") || cmd.Flags().Changed("lng") {
				if !cmd.Flags().Changed("lat") || !cmd.Flags().Changed("lng") {
					return errors.New("--lat and --lng go together")
				}
				q.Set("lat", strconv.FormatFloat(lat, 'f', -1, 64))
				q.Set("lng", strconv.FormatFloat(lng, 'f', -1, 64))
			}
			if q.Get("q") == "" && q.Get("lat") == "" {
				return errors.New("give a search query or --lat and --lng")
			}
			setStringIf(q, "fields", fields)
			applyPagination(cmd, q)

			var resp struct {
				Total      int                   `json:"total"`
				Pagination *Pagination           `json:"pagination"`
				Items      []ChannelSearchResult `json:"items"`
			}
			if a.JSONOutput || fields != "" {
				return getJSON(cmd, "/channels/search", q)
			}
			if err := a.Client.Get("/channels/search", q, &resp); err != nil {
				return err
			}
			printTotal(resp.Total, resp.Pagination)
			rows := make([][]string, len(resp.Items))
			for i, c := range resp.Items {
				account, distance := "", ""
				if c.Account != nil {
					account = c.Account.Name
				}
				if c.Distance != nil {
					distance = strconv.Itoa(*c.Distance)
				}
				rows[i] = []string{strconv.Itoa(c.ID), c.Name, account, distance}
			}
			printTable(cmd, []string{"ID", "NAME", "ACCOUNT", "DISTANCE"}, rows)
			return nil
		},
	}
	cmd.Flags().Float64Var(&lat, "lat", 0, "Latitude; with --lng lists the nearest channels")
	cmd.Flags().Float64Var(&lng, "lng", 0, "Longitude")
	addFieldsFlag(cmd, &fields)
	addPaginationFlags(cmd, 200)
	return cmd
}

type ChannelLayout struct {
	Template    string `json:"template"`
	Orientation string `json:"orientation"`
	Extension   string `json:"extension"`
	Favorite    bool   `json:"favorite"`
	Ticker      bool   `json:"ticker"`
	Clock       bool   `json:"clock"`
	Logo        bool   `json:"logo"`
}

func newChannelsLayoutsCmd() *cobra.Command {
	var template, orientation string
	var favorites bool

	cmd := &cobra.Command{
		Use:   "layouts",
		Short: "List the layouts a screen channel can have (template, orientation, extension)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var resp struct {
				Total int             `json:"total"`
				Items []ChannelLayout `json:"items"`
			}
			if a.JSONOutput {
				return getJSON(cmd, "/channel_layouts", nil)
			}
			if err := a.Client.Get("/channel_layouts", nil, &resp); err != nil {
				return err
			}
			var rows [][]string
			for _, l := range resp.Items {
				if template != "" && l.Template != template || orientation != "" && l.Orientation != orientation || favorites && !l.Favorite {
					continue
				}
				rows = append(rows, []string{l.Template, l.Orientation, l.Extension, check(l.Ticker), check(l.Clock), check(l.Logo), check(l.Favorite)})
			}
			printTable(cmd, []string{"TEMPLATE", "ORIENTATION", "EXTENSION", "TICKER", "CLOCK", "LOGO", "FAVORITE"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&template, "template", "", "Only this template")
	cmd.Flags().StringVar(&orientation, "orientation", "", "Only this orientation")
	cmd.Flags().BoolVar(&favorites, "favorites", false, "Only the layouts the web offers first")
	return cmd
}

func check(b bool) string {
	if b {
		return "✓"
	}
	return ""
}
