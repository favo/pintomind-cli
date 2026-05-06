package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

type MediaBox struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	UUID      string `json:"uuid"`
	PostID    int    `json:"post_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type MediaBoxesResponse struct {
	Total int        `json:"total"`
	Items []MediaBox `json:"items"`
}

func NewMediaBoxesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media-boxes",
		Short: "Manage media boxes for plain post media slots",
	}
	cmd.AddCommand(newMediaBoxesListCmd())
	cmd.AddCommand(newMediaBoxesShowCmd())
	createCmd := newMediaBoxesCreateCmd()
	createCmd.AddCommand(newMediaBoxesCreateMediaCmd())
	createCmd.AddCommand(newMediaBoxesCreateIconCmd())
	createCmd.AddCommand(newMediaBoxesCreateEmojiCmd())
	createCmd.AddCommand(newMediaBoxesCreateGifCmd())
	createCmd.AddCommand(newMediaBoxesCreateUnsplashCmd())
	createCmd.AddCommand(newMediaBoxesCreateQRCodeCmd())
	cmd.AddCommand(createCmd)
	cmd.AddCommand(newMediaBoxesUpdateCmd())
	cmd.AddCommand(newMediaBoxesDeleteCmd())
	cmd.AddCommand(newSchemaSubCmd(map[string]string{
		"emoji":    "media_box_emoji",
		"gif":      "media_box_gif",
		"icon":     "media_box_icon",
		"media":    "media_box_image",
		"qr-code":  "media_box_qr_code",
		"unsplash": "media_box_unsplash",
	}))
	return cmd
}

func newMediaBoxesListCmd() *cobra.Command {
	var boxType, postID, sortBy string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List media boxes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			if boxType != "" {
				q.Set("type", boxType)
			}
			if postID != "" {
				q.Set("post_id", postID)
			}
			if sortBy != "" {
				q.Set("sort_by", sortBy)
			}
			applyPagination(cmd, q)

			var resp MediaBoxesResponse
			if err := a.Client.Get("/media_boxes", q, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, mb := range resp.Items {
				postID := ""
				if mb.PostID != 0 {
					postID = strconv.Itoa(mb.PostID)
				}
				rows[i] = []string{
					strconv.Itoa(mb.ID),
					mb.Type,
					mb.UUID,
					postID,
					mb.CreatedAt,
				}
			}
			printTable(cmd, []string{"ID", "TYPE", "UUID", "POST_ID", "CREATED_AT"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&boxType, "type", "", "Filter by media box type alias (comma-separated)")
	cmd.Flags().StringVar(&postID, "post-id", "", "Filter by owning post ID")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field (created_at or updated_at, e.g. created_at:desc)")
	addPaginationFlags(cmd)
	return cmd
}

func newMediaBoxesShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a media box",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/media_boxes/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newMediaBoxesCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create a media box",
	}
}

func newMediaBoxesUpdateCmd() *cobra.Command {
	var data string

	cmd := &cobra.Command{
		Use:   "update <id> --data '<json>'",
		Short: "Update a media box (type cannot be changed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var mediaBox map[string]any
			if err := json.Unmarshal([]byte(data), &mediaBox); err != nil {
				return fmt.Errorf("invalid JSON for --data: %w", err)
			}
			body := map[string]any{"media_box": mediaBox}
			var resp map[string]any
			if err := a.Client.Patch("/media_boxes/"+args[0], body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "Media box fields as JSON (required)")
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

func newMediaBoxesDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a media box",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if !force {
				fmt.Printf("Delete media box %s? Pass --force to skip this prompt.\n", args[0])
				var confirm string
				fmt.Print("Type 'yes' to confirm: ")
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if err := a.Client.Delete("/media_boxes/" + args[0]); err != nil {
				return err
			}
			fmt.Printf("Deleted media box %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newMediaBoxesCreateMediaCmd() *cobra.Command {
	var mediaID int
	var backgroundSize, backgroundFillType string
	var x, y, relativeSize float64

	cmd := &cobra.Command{
		Use:   "media --media-id <id>",
		Short: "Create a media box from a library media item",
		Example: `  pintomind media-boxes create media --media-id 42
  pintomind media-boxes create media --media-id 42 --background-size cover --x 0.5 --y 0.5`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			data := map[string]any{"media_id": mediaID}
			if backgroundSize != "" {
				data["background_size"] = backgroundSize
			}
			if backgroundFillType != "" {
				data["background_fill_type"] = backgroundFillType
			}
			if cmd.Flags().Changed("x") {
				data["x"] = x
			}
			if cmd.Flags().Changed("y") {
				data["y"] = y
			}
			if cmd.Flags().Changed("relative-size") {
				data["relative_size"] = relativeSize
			}
			return createMediaBoxAndPrint(a, "media", data)
		},
	}
	cmd.Flags().IntVar(&mediaID, "media-id", 0, "Media item ID (required)")
	cmd.Flags().StringVar(&backgroundSize, "background-size", "", "Background size: cover, contain, auto")
	cmd.Flags().StringVar(&backgroundFillType, "background-fill-type", "", "Background fill type")
	cmd.Flags().Float64Var(&x, "x", 0, "Horizontal focal point (0.0–1.0)")
	cmd.Flags().Float64Var(&y, "y", 0, "Vertical focal point (0.0–1.0)")
	cmd.Flags().Float64Var(&relativeSize, "relative-size", 0, "Relative size (0.0–1.0)")
	_ = cmd.MarkFlagRequired("media-id")
	return cmd
}

func newMediaBoxesCreateIconCmd() *cobra.Command {
	var iconName, iconType string
	var relativeSize float64

	cmd := &cobra.Command{
		Use:   "icon --icon-name <name> --icon-type <type>",
		Short: "Create a media box from an icon",
		Example: `  pintomind media-boxes create icon --icon-name rocket-launch --icon-type regular
  pintomind media-boxes create icon --icon-name rocket-launch --icon-type solid --relative-size 0.8`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			data := map[string]any{"icon_name": iconName, "icon_type": iconType}
			if cmd.Flags().Changed("relative-size") {
				data["relative_size"] = relativeSize
			}
			return createMediaBoxAndPrint(a, "icon", data)
		},
	}
	cmd.Flags().StringVar(&iconName, "icon-name", "", "Icon name (required)")
	cmd.Flags().StringVar(&iconType, "icon-type", "", "Icon type, e.g. regular or solid (required)")
	cmd.Flags().Float64Var(&relativeSize, "relative-size", 0, "Relative size (0.0–1.0)")
	_ = cmd.MarkFlagRequired("icon-name")
	_ = cmd.MarkFlagRequired("icon-type")
	return cmd
}

func newMediaBoxesCreateEmojiCmd() *cobra.Command {
	var emoji string
	var relativeSize float64

	cmd := &cobra.Command{
		Use:   "emoji --emoji <char>",
		Short: "Create a media box from an emoji character",
		Example: `  pintomind media-boxes create emoji --emoji '✨'
  pintomind media-boxes create emoji --emoji '🎂' --relative-size 0.8`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			data := map[string]any{"emoji": emoji}
			if cmd.Flags().Changed("relative-size") {
				data["relative_size"] = relativeSize
			}
			return createMediaBoxAndPrint(a, "emoji", data)
		},
	}
	cmd.Flags().StringVar(&emoji, "emoji", "", "Emoji character (required)")
	cmd.Flags().Float64Var(&relativeSize, "relative-size", 0, "Relative size (0.0–1.0)")
	_ = cmd.MarkFlagRequired("emoji")
	return cmd
}

func newMediaBoxesCreateGifCmd() *cobra.Command {
	var gifID, backgroundSize, backgroundFillType string
	var x, y, relativeSize float64

	cmd := &cobra.Command{
		Use:   "gif --gif-id <id>",
		Short: "Create a media box from a Giphy GIF",
		Example: `  pintomind media-boxes create gif --gif-id xT9IgG50Fb7Mi0only
  pintomind media-boxes create gif --gif-id xT9IgG50Fb7Mi0only --background-size contain --x 0.5 --y 0.5`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			data := map[string]any{"gif_id": gifID}
			if backgroundSize != "" {
				data["background_size"] = backgroundSize
			}
			if backgroundFillType != "" {
				data["background_fill_type"] = backgroundFillType
			}
			if cmd.Flags().Changed("x") {
				data["x"] = x
			}
			if cmd.Flags().Changed("y") {
				data["y"] = y
			}
			if cmd.Flags().Changed("relative-size") {
				data["relative_size"] = relativeSize
			}
			return createMediaBoxAndPrint(a, "gif", data)
		},
	}
	cmd.Flags().StringVar(&gifID, "gif-id", "", "Giphy GIF ID (required)")
	cmd.Flags().StringVar(&backgroundSize, "background-size", "", "Background size: cover, contain, auto")
	cmd.Flags().StringVar(&backgroundFillType, "background-fill-type", "", "Background fill type")
	cmd.Flags().Float64Var(&x, "x", 0, "Horizontal focal point (0.0–1.0)")
	cmd.Flags().Float64Var(&y, "y", 0, "Vertical focal point (0.0–1.0)")
	cmd.Flags().Float64Var(&relativeSize, "relative-size", 0, "Relative size (0.0–1.0)")
	_ = cmd.MarkFlagRequired("gif-id")
	return cmd
}

func newMediaBoxesCreateUnsplashCmd() *cobra.Command {
	var photoID, backgroundSize, backgroundFillType string
	var x, y, relativeSize float64

	cmd := &cobra.Command{
		Use:   "unsplash --photo-id <id>",
		Short: "Create a media box from an Unsplash photo",
		Example: `  pintomind media-boxes create unsplash --photo-id abc123
  pintomind media-boxes create unsplash --photo-id abc123 --background-size cover --x 0.3 --y 0.7`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			data := map[string]any{"photo_id": photoID}
			if backgroundSize != "" {
				data["background_size"] = backgroundSize
			}
			if backgroundFillType != "" {
				data["background_fill_type"] = backgroundFillType
			}
			if cmd.Flags().Changed("x") {
				data["x"] = x
			}
			if cmd.Flags().Changed("y") {
				data["y"] = y
			}
			if cmd.Flags().Changed("relative-size") {
				data["relative_size"] = relativeSize
			}
			return createMediaBoxAndPrint(a, "unsplash", data)
		},
	}
	cmd.Flags().StringVar(&photoID, "photo-id", "", "Unsplash photo ID (required)")
	cmd.Flags().StringVar(&backgroundSize, "background-size", "", "Background size: cover, contain, auto")
	cmd.Flags().StringVar(&backgroundFillType, "background-fill-type", "", "Background fill type")
	cmd.Flags().Float64Var(&x, "x", 0, "Horizontal focal point (0.0–1.0)")
	cmd.Flags().Float64Var(&y, "y", 0, "Vertical focal point (0.0–1.0)")
	cmd.Flags().Float64Var(&relativeSize, "relative-size", 0, "Relative size (0.0–1.0)")
	_ = cmd.MarkFlagRequired("photo-id")
	return cmd
}

func newMediaBoxesCreateQRCodeCmd() *cobra.Command {
	var qrURL string

	cmd := &cobra.Command{
		Use:     "qr-code --url <url>",
		Short:   "Create a media box displaying a QR code",
		Example: `  pintomind media-boxes create qr-code --url https://example.com`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			return createMediaBoxAndPrint(a, "qr_code", map[string]any{"url": qrURL})
		},
	}
	cmd.Flags().StringVar(&qrURL, "url", "", "URL to encode as QR code (required)")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}
