package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"favo/pintomind-cli/internal/appctx"

	"github.com/spf13/cobra"
)

type Theme struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ThemesResponse struct {
	Total int     `json:"total"`
	Items []Theme `json:"items"`
}

func NewThemesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "themes",
		Short: "Manage themes",
	}
	cmd.AddCommand(newThemesListCmd())
	cmd.AddCommand(newThemesShowCmd())
	cmd.AddCommand(newThemesStatsCmd())
	cmd.AddCommand(newThemesCreateCmd())
	cmd.AddCommand(newThemesUpdateCmd())
	cmd.AddCommand(newThemesSetMediaBoxCmd("background"))
	cmd.AddCommand(newThemesSetMediaBoxCmd("logo"))
	cmd.AddCommand(newThemesDeleteCmd())
	cmd.AddCommand(newSchemaSubCmd(map[string]string{"theme": "theme"}))
	return cmd
}

func newThemesListCmd() *cobra.Command {
	var sortBy string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List themes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			if sortBy != "" {
				q.Set("sort_by", sortBy)
			}
			applyPagination(cmd, q)

			var resp ThemesResponse
			if err := a.Client.Get("/themes", q, &resp); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, t := range resp.Items {
				rows[i] = []string{strconv.Itoa(t.ID), t.Name}
			}
			printTable(cmd, []string{"ID", "NAME"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field (e.g. name:asc)")
	addPaginationFlags(cmd)
	return cmd
}

func newThemesShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a theme",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/themes/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newThemesStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show theme stats",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/themes/stats", nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newThemesCreateCmd() *cobra.Command {
	var data string

	cmd := &cobra.Command{
		Use:     "create --data '<json>'",
		Short:   "Create a theme",
		Example: `  pintomind themes create --data '{"name":"My theme","font_family_header_id":1,"font_family_body_id":2,"background_color":"#1a1a2e","text_color":"#ffffff"}'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var theme map[string]any
			if err := json.Unmarshal([]byte(data), &theme); err != nil {
				return fmt.Errorf("invalid JSON for --data: %w", err)
			}
			body := map[string]any{"theme": theme}
			var resp map[string]any
			if err := a.Client.Post("/themes", body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "Theme attributes as JSON (required)")
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

func newThemesUpdateCmd() *cobra.Command {
	var data string

	cmd := &cobra.Command{
		Use:   "update <id> --data '<json>'",
		Short: "Update a theme",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var theme map[string]any
			if err := json.Unmarshal([]byte(data), &theme); err != nil {
				return fmt.Errorf("invalid JSON for --data: %w", err)
			}
			body := map[string]any{"theme": theme}
			var resp map[string]any
			if err := a.Client.Patch("/themes/"+args[0], body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "Theme attributes as JSON (required)")
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

// newThemesSetMediaBoxCmd builds `themes set-background` / `themes set-logo`. Both attach a
// MediaBox to the theme: either an existing box id, or a box created on the fly from an
// uploaded file, a media library item, an Unsplash photo, or a Giphy gif.
func newThemesSetMediaBoxCmd(slot string) *cobra.Command {
	var mediaBoxID, mediaID, mediaCollection int
	var file, photoID, gifID, backgroundSize string
	var x, y float64
	var clear bool

	field := "background_media_id"
	if slot == "logo" {
		field = "logo_id"
	}

	cmd := &cobra.Command{
		Use:   "set-" + slot + " <theme-id>",
		Short: "Set or clear the theme " + slot,
		Long: `Set or clear the theme ` + slot + `.

--file uploads the file (or URL) to the default ` + slot + ` media collection, wraps it in a
media box and attaches it. --media-id, --unsplash-photo-id and --gif-id create the box from
something that already exists. --media-box attaches a box you built yourself.`,
		Args: cobra.ExactArgs(1),
		Example: `  pintomind themes set-` + slot + ` 12 --file ./` + slot + `.png
  pintomind themes set-` + slot + ` 12 --media-id 42
  pintomind themes set-` + slot + ` 12 --media-box 4711
  pintomind themes set-` + slot + ` 12 --clear`,
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)

			sources := 0
			for _, used := range []bool{clear, mediaBoxID > 0, file != "", mediaID > 0, photoID != "", gifID != ""} {
				if used {
					sources++
				}
			}
			if sources != 1 {
				return fmt.Errorf("pass exactly one of --file, --media-box, --media-id, --unsplash-photo-id, --gif-id or --clear")
			}

			var value any
			switch {
			case clear:
				value = nil
			case mediaBoxID > 0:
				value = mediaBoxID
			default:
				if file != "" {
					id, err := uploadThemeMedia(a, file, slot, mediaCollection)
					if err != nil {
						return err
					}
					mediaID = id
				}

				boxType, data := "media", map[string]any{}
				switch {
				case mediaID > 0:
					data["media_id"] = mediaID
				case photoID != "":
					boxType = "unsplash"
					data["photo_id"] = photoID
				case gifID != "":
					boxType = "gif"
					data["gif_id"] = gifID
				}
				if backgroundSize != "" {
					data["background_size"] = backgroundSize
				}
				if cmd.Flags().Changed("x") {
					data["x"] = x
				}
				if cmd.Flags().Changed("y") {
					data["y"] = y
				}

				id, err := createMediaBoxID(a, boxType, data)
				if err != nil {
					return err
				}
				if !a.JSONOutput {
					fmt.Printf("Created %s media box %d\n", boxType, id)
				}
				value = id
			}

			body := map[string]any{"theme": map[string]any{field: value}}
			var resp map[string]any
			if err := a.Client.Patch("/themes/"+args[0], body, &resp); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(resp)
				return nil
			}
			if clear {
				fmt.Printf("Cleared %s on theme %s\n", slot, args[0])
			} else {
				fmt.Printf("Set %s of theme %s to media box %v\n", slot, args[0], value)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&file, "file", "", "File path or URL to upload, wrap in a media box and attach")
	cmd.Flags().IntVar(&mediaCollection, "media-collection", 0, "Media collection ID for --file uploads (defaults to the default "+slot+" collection)")
	cmd.Flags().IntVar(&mediaBoxID, "media-box", 0, "Attach an existing MediaBox by id")
	cmd.Flags().IntVar(&mediaID, "media-id", 0, "Create a media box from this media library item and attach it")
	cmd.Flags().StringVar(&photoID, "unsplash-photo-id", "", "Create an Unsplash media box from this photo id and attach it")
	cmd.Flags().StringVar(&gifID, "gif-id", "", "Create a Giphy media box from this gif id and attach it")
	cmd.Flags().StringVar(&backgroundSize, "background-size", "", "Background size for a created box: cover, contain, auto")
	cmd.Flags().Float64Var(&x, "x", 0, "Horizontal focal point for a created box (0.0–1.0)")
	cmd.Flags().Float64Var(&y, "y", 0, "Vertical focal point for a created box (0.0–1.0)")
	cmd.Flags().BoolVar(&clear, "clear", false, "Remove the current "+slot)
	return cmd
}

// uploadThemeMedia uploads a file or URL to the media collection for the given theme slot
// ("background" or "logo") and returns the resulting media id.
func uploadThemeMedia(a *appctx.App, input, slot string, collectionID int) (int, error) {
	if collectionID == 0 {
		id, err := findDefaultCollection(a, slot)
		if err != nil {
			return 0, err
		}
		collectionID = id
	}

	mediaIDs, err := uploadFileToCollection(a, input, strconv.Itoa(collectionID))
	if err != nil {
		return 0, err
	}
	if len(mediaIDs) == 0 {
		return 0, fmt.Errorf("upload finished without returning a media id")
	}
	if !a.JSONOutput {
		fmt.Printf("Uploaded media %d to collection %d\n", mediaIDs[0], collectionID)
	}
	return mediaIDs[0], nil
}

func newThemesDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a theme (hard-delete; affected channels are reset to the account's standard theme)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if !force {
				fmt.Printf("Delete theme %s? Channels using it will be reset to the standard theme. Pass --force to skip this prompt.\n", args[0])
				var confirm string
				fmt.Print("Type 'yes' to confirm: ")
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if err := a.Client.Delete("/themes/" + args[0]); err != nil {
				return err
			}
			fmt.Printf("Deleted theme %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
