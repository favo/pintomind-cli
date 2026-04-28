package commands

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

type FontFamily struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type FontFamiliesResponse struct {
	Total int          `json:"total"`
	Items []FontFamily `json:"items"`
}

func NewFontFamiliesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "font-families",
		Aliases: []string{"fonts"},
		Short:   "Manage font families",
	}
	cmd.AddCommand(newFontFamiliesListCmd())
	cmd.AddCommand(newFontFamiliesShowCmd())
	cmd.AddCommand(newFontFamiliesStatsCmd())
	cmd.AddCommand(newFontFamiliesCreateCmd())
	cmd.AddCommand(newFontFamiliesUpdateCmd())
	cmd.AddCommand(newFontFamiliesDeleteCmd())
	return cmd
}

func newFontFamiliesListCmd() *cobra.Command {
	var sortBy, fontType string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List font families",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			if fontType != "" {
				q.Set("type", fontType)
			}
			if sortBy != "" {
				q.Set("sort_by", sortBy)
			}
			applyPagination(cmd, q)

			var resp FontFamiliesResponse
			if err := a.Client.Get("/font_families", q, &resp); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, f := range resp.Items {
				rows[i] = []string{strconv.Itoa(f.ID), f.Type, f.Name, f.URL}
			}
			printTable(cmd, []string{"ID", "TYPE", "NAME", "URL"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&fontType, "type", "", "Filter by type: remote_css, uploaded, standard (comma-separated)")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field (e.g. name:asc)")
	addPaginationFlags(cmd)
	return cmd
}

func newFontFamiliesShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a font family",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/font_families/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newFontFamiliesStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show font family stats",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/font_families/stats", nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newFontFamiliesCreateCmd() *cobra.Command {
	var fontType, name, fontURL, fontName string
	var suitableForBody, forceTextTransform bool
	var fontNormal, fontBold, fontItalic, fontBoldItalic string

	cmd := &cobra.Command{
		Use:   "create --type <type> --name <name>",
		Short: "Create a font family",
		Example: `  # Remote CSS (e.g. Google Fonts)
  pintomind font-families create --type remote_css --name "Inter" --url "https://fonts.googleapis.com/css2?family=Inter"

  # Uploaded font (signed_ids from 'pintomind media upload' / direct_uploads)
  pintomind font-families create --type uploaded --name "MyFont" --font-normal <signed_id> --font-bold <signed_id>`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			fontFamily := map[string]any{"name": name}

			switch fontType {
			case "remote_css":
				if fontURL == "" {
					return fmt.Errorf("--url is required for remote_css fonts")
				}
				fontFamily["url"] = fontURL
				if fontName != "" {
					fontFamily["font_name"] = fontName
				}
			case "uploaded":
				if fontNormal == "" || fontBold == "" {
					return fmt.Errorf("--font-normal and --font-bold are required for uploaded fonts")
				}
				fontFamily["font_normal"] = fontNormal
				fontFamily["font_bold"] = fontBold
				if fontItalic != "" {
					fontFamily["font_italic"] = fontItalic
				}
				if fontBoldItalic != "" {
					fontFamily["font_bold_italic"] = fontBoldItalic
				}
			default:
				return fmt.Errorf("--type must be remote_css or uploaded")
			}

			if cmd.Flags().Changed("suitable-for-body") {
				fontFamily["suitable_for_body"] = suitableForBody
			}
			if cmd.Flags().Changed("force-text-transform") {
				fontFamily["force_text_transform"] = forceTextTransform
			}

			body := map[string]any{
				"type":        fontType,
				"font_family": fontFamily,
			}
			var resp map[string]any
			if err := a.Client.Post("/font_families", body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&fontType, "type", "", "Font type: remote_css or uploaded (required)")
	cmd.Flags().StringVar(&name, "name", "", "Font family name (required)")
	cmd.Flags().StringVar(&fontURL, "url", "", "Hosted CSS URL (remote_css only, e.g. Google Fonts URL)")
	cmd.Flags().StringVar(&fontName, "font-name", "", "CSS font-family name override (remote_css only; picked from CSS if omitted)")
	cmd.Flags().StringVar(&fontNormal, "font-normal", "", "signed_id for normal weight font file (uploaded only, required)")
	cmd.Flags().StringVar(&fontBold, "font-bold", "", "signed_id for bold weight font file (uploaded only, required)")
	cmd.Flags().StringVar(&fontItalic, "font-italic", "", "signed_id for italic font file (uploaded only, optional)")
	cmd.Flags().StringVar(&fontBoldItalic, "font-bold-italic", "", "signed_id for bold-italic font file (uploaded only, optional)")
	cmd.Flags().BoolVar(&suitableForBody, "suitable-for-body", false, "Mark font as suitable for body text")
	cmd.Flags().BoolVar(&forceTextTransform, "force-text-transform", false, "Force text transform")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newFontFamiliesUpdateCmd() *cobra.Command {
	var name, fontURL, fontName string
	var suitableForBody, forceTextTransform bool
	var fontNormal, fontBold, fontItalic, fontBoldItalic string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a font family (type cannot be changed after creation)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			fontFamily := map[string]any{}
			if cmd.Flags().Changed("name") {
				fontFamily["name"] = name
			}
			if cmd.Flags().Changed("url") {
				fontFamily["url"] = fontURL
			}
			if cmd.Flags().Changed("font-name") {
				fontFamily["font_name"] = fontName
			}
			if cmd.Flags().Changed("font-normal") {
				fontFamily["font_normal"] = fontNormal
			}
			if cmd.Flags().Changed("font-bold") {
				fontFamily["font_bold"] = fontBold
			}
			if cmd.Flags().Changed("font-italic") {
				fontFamily["font_italic"] = fontItalic
			}
			if cmd.Flags().Changed("font-bold-italic") {
				fontFamily["font_bold_italic"] = fontBoldItalic
			}
			if cmd.Flags().Changed("suitable-for-body") {
				fontFamily["suitable_for_body"] = suitableForBody
			}
			if cmd.Flags().Changed("force-text-transform") {
				fontFamily["force_text_transform"] = forceTextTransform
			}
			if len(fontFamily) == 0 {
				return fmt.Errorf("provide at least one field to update")
			}
			var resp map[string]any
			if err := a.Client.Patch("/font_families/"+args[0], map[string]any{"font_family": fontFamily}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Font family name")
	cmd.Flags().StringVar(&fontURL, "url", "", "Hosted CSS URL (remote_css only)")
	cmd.Flags().StringVar(&fontName, "font-name", "", "CSS font-family name override (remote_css only)")
	cmd.Flags().StringVar(&fontNormal, "font-normal", "", "signed_id for normal weight font file (uploaded only)")
	cmd.Flags().StringVar(&fontBold, "font-bold", "", "signed_id for bold weight font file (uploaded only)")
	cmd.Flags().StringVar(&fontItalic, "font-italic", "", "signed_id for italic font file (uploaded only)")
	cmd.Flags().StringVar(&fontBoldItalic, "font-bold-italic", "", "signed_id for bold-italic font file (uploaded only)")
	cmd.Flags().BoolVar(&suitableForBody, "suitable-for-body", false, "Mark font as suitable for body text")
	cmd.Flags().BoolVar(&forceTextTransform, "force-text-transform", false, "Force text transform")
	return cmd
}

func newFontFamiliesDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a font family (themes using it are reset to the system default)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if !force {
				fmt.Printf("Delete font family %s? Themes using it will be reset to the system default font (figtree). Pass --force to skip this prompt.\n", args[0])
				var confirm string
				fmt.Print("Type 'yes' to confirm: ")
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if err := a.Client.Delete("/font_families/" + args[0]); err != nil {
				return err
			}
			fmt.Printf("Deleted font family %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
