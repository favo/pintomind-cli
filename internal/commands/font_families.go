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
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a font family",
	}
	cmd.AddCommand(newFontFamiliesCreateRemoteCSSCmd())
	cmd.AddCommand(newFontFamiliesCreateUploadedCmd())
	return cmd
}

func newFontFamiliesCreateRemoteCSSCmd() *cobra.Command {
	var name, fontURL, fontName string
	var suitableForBody, forceTextTransform bool

	cmd := &cobra.Command{
		Use:   "remote-css --name <name> --url <url>",
		Short: "Create a remote CSS font family (e.g. Google Fonts)",
		Example: `  pintomind font-families create remote-css --name "Inter" --url "https://fonts.googleapis.com/css2?family=Inter"
  pintomind font-families create remote-css --name "Roboto" --url "https://fonts.googleapis.com/css2?family=Roboto" --suitable-for-body`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			fontFamily := map[string]any{"name": name, "url": fontURL}
			if fontName != "" {
				fontFamily["font_name"] = fontName
			}
			if cmd.Flags().Changed("suitable-for-body") {
				fontFamily["suitable_for_body"] = suitableForBody
			}
			if cmd.Flags().Changed("force-text-transform") {
				fontFamily["force_text_transform"] = forceTextTransform
			}
			var resp map[string]any
			if err := a.Client.Post("/font_families", map[string]any{"type": "remote_css", "font_family": fontFamily}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Font family name (required)")
	cmd.Flags().StringVar(&fontURL, "url", "", "Hosted CSS URL, e.g. Google Fonts URL (required)")
	cmd.Flags().StringVar(&fontName, "font-name", "", "CSS font-family name override (picked from CSS if omitted)")
	cmd.Flags().BoolVar(&suitableForBody, "suitable-for-body", false, "Mark font as suitable for body text")
	cmd.Flags().BoolVar(&forceTextTransform, "force-text-transform", false, "Force text transform")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func newFontFamiliesCreateUploadedCmd() *cobra.Command {
	var name, fontNormal, fontBold, fontItalic, fontBoldItalic string
	var suitableForBody, forceTextTransform bool

	cmd := &cobra.Command{
		Use:   "uploaded --name <name> --font-normal <file> --font-bold <file>",
		Short: "Create a font family from uploaded font files (.ttf/.otf/.woff/.woff2)",
		Example: `  pintomind font-families create uploaded --name "MyFont" --font-normal ./MyFont-Regular.ttf --font-bold ./MyFont-Bold.ttf
  pintomind font-families create uploaded --name "MyFont" --font-normal ./MyFont-Regular.ttf --font-bold ./MyFont-Bold.ttf --font-italic ./MyFont-Italic.ttf --font-bold-italic ./MyFont-BoldItalic.ttf`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			fields := map[string]string{
				"type":              "uploaded",
				"font_family[name]": name,
			}
			if cmd.Flags().Changed("suitable-for-body") {
				fields["font_family[suitable_for_body]"] = fmt.Sprintf("%v", suitableForBody)
			}
			if cmd.Flags().Changed("force-text-transform") {
				fields["font_family[force_text_transform]"] = fmt.Sprintf("%v", forceTextTransform)
			}
			files := map[string]string{
				"font_family[font_normal]": fontNormal,
				"font_family[font_bold]":   fontBold,
			}
			if fontItalic != "" {
				files["font_family[font_italic]"] = fontItalic
			}
			if fontBoldItalic != "" {
				files["font_family[font_bold_italic]"] = fontBoldItalic
			}
			var resp map[string]any
			if err := a.Client.PostMultipart("/font_families", fields, files, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Font family name (required)")
	cmd.Flags().StringVar(&fontNormal, "font-normal", "", "Path to normal weight font file (required)")
	cmd.Flags().StringVar(&fontBold, "font-bold", "", "Path to bold weight font file (required)")
	cmd.Flags().StringVar(&fontItalic, "font-italic", "", "Path to italic font file (optional)")
	cmd.Flags().StringVar(&fontBoldItalic, "font-bold-italic", "", "Path to bold-italic font file (optional)")
	cmd.Flags().BoolVar(&suitableForBody, "suitable-for-body", false, "Mark font as suitable for body text")
	cmd.Flags().BoolVar(&forceTextTransform, "force-text-transform", false, "Force text transform")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("font-normal")
	_ = cmd.MarkFlagRequired("font-bold")
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

			hasFiles := cmd.Flags().Changed("font-normal") ||
				cmd.Flags().Changed("font-bold") ||
				cmd.Flags().Changed("font-italic") ||
				cmd.Flags().Changed("font-bold-italic")

			if hasFiles {
				// Use multipart when uploading font files
				fields := map[string]string{}
				files := map[string]string{}
				if cmd.Flags().Changed("name") {
					fields["font_family[name]"] = name
				}
				if cmd.Flags().Changed("suitable-for-body") {
					fields["font_family[suitable_for_body]"] = fmt.Sprintf("%v", suitableForBody)
				}
				if cmd.Flags().Changed("force-text-transform") {
					fields["font_family[force_text_transform]"] = fmt.Sprintf("%v", forceTextTransform)
				}
				if fontNormal != "" {
					files["font_family[font_normal]"] = fontNormal
				}
				if fontBold != "" {
					files["font_family[font_bold]"] = fontBold
				}
				if fontItalic != "" {
					files["font_family[font_italic]"] = fontItalic
				}
				if fontBoldItalic != "" {
					files["font_family[font_bold_italic]"] = fontBoldItalic
				}
				if len(fields)+len(files) == 0 {
					return fmt.Errorf("provide at least one field to update")
				}
				var resp map[string]any
				if err := a.Client.PatchMultipart("/font_families/"+args[0], fields, files, &resp); err != nil {
					return err
				}
				printJSON(resp)
				return nil
			}

			// No files — use JSON (works for both remote_css and name-only updates)
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
	cmd.Flags().StringVar(&fontNormal, "font-normal", "", "Path to normal weight font file (uploaded only; .ttf/.otf/.woff/.woff2)")
	cmd.Flags().StringVar(&fontBold, "font-bold", "", "Path to bold weight font file (uploaded only)")
	cmd.Flags().StringVar(&fontItalic, "font-italic", "", "Path to italic font file (uploaded only)")
	cmd.Flags().StringVar(&fontBoldItalic, "font-bold-italic", "", "Path to bold-italic font file (uploaded only)")
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
