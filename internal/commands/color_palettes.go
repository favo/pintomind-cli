package commands

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

type ColorPalette struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	TertiaryColor  string `json:"tertiary_color"`
}

type ColorPalettesResponse struct {
	Total int            `json:"total"`
	Items []ColorPalette `json:"items"`
}

func NewColorPalettesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "color-palettes",
		Aliases: []string{"palettes"},
		Short:   "Manage color palettes",
	}
	cmd.AddCommand(newColorPalettesListCmd())
	cmd.AddCommand(newColorPalettesShowCmd())
	cmd.AddCommand(newColorPalettesStatsCmd())
	cmd.AddCommand(newColorPalettesCreateCmd())
	cmd.AddCommand(newColorPalettesUpdateCmd())
	cmd.AddCommand(newColorPalettesDeleteCmd())
	cmd.AddCommand(newSchemaSubCmd(map[string]string{"color-palette": "color_palette"}))
	return cmd
}

func newColorPalettesListCmd() *cobra.Command {
	var sortBy string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List color palettes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			if sortBy != "" {
				q.Set("sort_by", sortBy)
			}
			applyPagination(cmd, q)

			var resp ColorPalettesResponse
			if err := a.Client.Get("/color_palettes", q, &resp); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, p := range resp.Items {
				rows[i] = []string{strconv.Itoa(p.ID), p.Name, p.PrimaryColor, p.SecondaryColor, p.TertiaryColor}
			}
			printTable(cmd, []string{"ID", "NAME", "PRIMARY", "SECONDARY", "TERTIARY"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field (e.g. name:asc)")
	addPaginationFlags(cmd)
	return cmd
}

func newColorPalettesShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a color palette",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/color_palettes/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newColorPalettesStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show color palette stats",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/color_palettes/stats", nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newColorPalettesCreateCmd() *cobra.Command {
	var name, primaryColor, secondaryColor, tertiaryColor string

	cmd := &cobra.Command{
		Use:   "create --name <name> --primary-color <hex>",
		Short: "Create a color palette",
		Example: `  pintomind color-palettes create --name "Brand" --primary-color "#1F8A8A" --secondary-color "#F5A623"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			palette := map[string]any{
				"name":          name,
				"primary_color": primaryColor,
			}
			if secondaryColor != "" {
				palette["secondary_color"] = secondaryColor
			}
			if tertiaryColor != "" {
				palette["tertiary_color"] = tertiaryColor
			}
			var resp map[string]any
			if err := a.Client.Post("/color_palettes", map[string]any{"color_palette": palette}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Palette name (required)")
	cmd.Flags().StringVar(&primaryColor, "primary-color", "", "Primary color hex (e.g. #1F8A8A) (required)")
	cmd.Flags().StringVar(&secondaryColor, "secondary-color", "", "Secondary color hex (optional)")
	cmd.Flags().StringVar(&tertiaryColor, "tertiary-color", "", "Tertiary color hex (optional)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("primary-color")
	return cmd
}

func newColorPalettesUpdateCmd() *cobra.Command {
	var name, primaryColor, secondaryColor, tertiaryColor string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a color palette",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			palette := map[string]any{}
			if cmd.Flags().Changed("name") {
				palette["name"] = name
			}
			if cmd.Flags().Changed("primary-color") {
				palette["primary_color"] = primaryColor
			}
			if cmd.Flags().Changed("secondary-color") {
				palette["secondary_color"] = secondaryColor
			}
			if cmd.Flags().Changed("tertiary-color") {
				palette["tertiary_color"] = tertiaryColor
			}
			if len(palette) == 0 {
				return fmt.Errorf("provide at least one field to update")
			}
			var resp map[string]any
			if err := a.Client.Patch("/color_palettes/"+args[0], map[string]any{"color_palette": palette}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Palette name")
	cmd.Flags().StringVar(&primaryColor, "primary-color", "", "Primary color hex")
	cmd.Flags().StringVar(&secondaryColor, "secondary-color", "", "Secondary color hex")
	cmd.Flags().StringVar(&tertiaryColor, "tertiary-color", "", "Tertiary color hex")
	return cmd
}

func newColorPalettesDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a color palette (themes using it are repointed before deletion)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if !force {
				fmt.Printf("Delete color palette %s? Themes using it will be repointed to another palette. Pass --force to skip this prompt.\n", args[0])
				var confirm string
				fmt.Print("Type 'yes' to confirm: ")
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if err := a.Client.Delete("/color_palettes/" + args[0]); err != nil {
				return err
			}
			fmt.Printf("Deleted color palette %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
