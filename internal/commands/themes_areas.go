package commands

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var screenEffects = []string{"snow", "rain", "thunderstorm", "leaves", "hearts", "sparkles"}

type themeExtensionSummary struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
}

type themeAreaSummary struct {
	ID              int                     `json:"id"`
	Name            string                  `json:"name"`
	Identifier      string                  `json:"identifier"`
	ThemeExtensions []themeExtensionSummary `json:"theme_extensions"`
}

type themeAreasResponse struct {
	ID              int                                     `json:"id"`
	Name            string                                  `json:"name"`
	Areas           map[string]map[string]map[string]string `json:"areas"`
	ThemeAreas      []themeAreaSummary                      `json:"theme_areas"`
	ThemeExtensions []themeExtensionSummary                 `json:"theme_extensions"`
	PreviewURL      string                                  `json:"preview_url"`
}

func newThemesAreasCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "areas <theme-id>",
		Short: "Show a theme's areas, extensions (with ids) and area mapping",
		Long: `Show a theme's areas, extensions and area mapping.

The ids are what themes update takes in theme_areas_attributes / theme_extensions_attributes
to change an entry, or with "_destroy": true to remove it.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var theme themeAreasResponse
			if err := a.Client.Get("/themes/"+args[0], nil, &theme); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(map[string]any{"areas": theme.Areas, "theme_areas": theme.ThemeAreas, "theme_extensions": theme.ThemeExtensions})
				return nil
			}

			fmt.Printf("Theme %d: %s\n\nTheme areas\n", theme.ID, theme.Name)
			areaRows := make([][]string, len(theme.ThemeAreas))
			for i, area := range theme.ThemeAreas {
				areaRows[i] = []string{fmt.Sprint(area.ID), area.Identifier, area.Name, extensionList(area.ThemeExtensions)}
			}
			printTable(cmd, []string{"ID", "IDENTIFIER", "NAME", "EXTENSIONS"}, areaRows)

			fmt.Println("\nTheme extensions")
			extensionRows := make([][]string, len(theme.ThemeExtensions))
			for i, extension := range theme.ThemeExtensions {
				extensionRows[i] = []string{fmt.Sprint(extension.ID), extension.Type}
			}
			printTable(cmd, []string{"ID", "TYPE"}, extensionRows)

			fmt.Println("\nArea mapping")
			printTable(cmd, []string{"LAYOUTS", "AREA", "STYLE"}, mappingRows(theme.Areas))
			return nil
		},
	}
}

func extensionList(extensions []themeExtensionSummary) string {
	parts := make([]string, len(extensions))
	for i, extension := range extensions {
		parts[i] = fmt.Sprintf("%s (%d)", extension.Type, extension.ID)
	}
	return strings.Join(parts, ", ")
}

func mappingRows(areas map[string]map[string]map[string]string) [][]string {
	keys := make([]string, 0, len(areas))
	for key := range areas {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var rows [][]string
	for _, key := range keys {
		layoutAreas := make([]string, 0, len(areas[key]))
		for area := range areas[key] {
			layoutAreas = append(layoutAreas, area)
		}
		sort.Strings(layoutAreas)
		for _, area := range layoutAreas {
			rows = append(rows, []string{key, area, areas[key][area]["style"]})
		}
	}
	return rows
}

func newThemesSetAreasCmd() *cobra.Command {
	var data string

	cmd := &cobra.Command{
		Use:   "set-areas <theme-id> --data '<json>'",
		Short: "Replace the theme's area mapping (which layout area uses which theme area)",
		Long: `Replace the theme's area mapping.

Keys select layouts: "*:*:*" (all), "*:portrait:*" / "*:landscape:*" (every layout in one
orientation), "<template>:*:*", "<template>:<orientation>:*" or
"<template>:<orientation>:<extension>"; more specific keys win. Layout areas are A, B, C, D,
CLOCK, TICKER and LOGO; style is a theme area identifier, or "none" for the plain theme.
The whole mapping is replaced: include every key you want to keep.`,
		Args: cobra.ExactArgs(1),
		Example: `  pintomind themes set-areas 12 --data '{"*:*:*":{"B":{"style":"accent"}},"*:portrait:*":{"A":{"style":"glass"}}}'
  pintomind themes set-areas 12 --data @areas.json
  pintomind themes set-areas 12 --data '{}'   # every area shows the plain theme`,
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			areas, err := parseJSONObjectFlag("data", data)
			if err != nil {
				return err
			}
			var resp themeUpdateResponse
			if err := a.Client.Patch("/themes/"+args[0], map[string]any{"theme": map[string]any{"areas": areas}}, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}
			fmt.Printf("Set the area mapping of theme %s\n\n", args[0])
			printTable(cmd, []string{"LAYOUTS", "AREA", "STYLE"}, mappingRows(resp.Theme.Areas))
			return nil
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "Area mapping as JSON, @file or - for stdin (required)")
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

type themeUpdateResponse struct {
	Success bool               `json:"success"`
	Theme   themeAreasResponse `json:"theme"`
}

func newThemesSetEffectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-effect <theme-id> <" + strings.Join(screenEffects, "|") + "|none>",
		Short: "Set the ambient screen effect that runs while the theme is shown",
		Long: `Set the ambient screen effect that runs on every screen for as long as the theme is shown.
The effect draws over the whole screen, posts included; "none" removes it.`,
		Args:      cobra.ExactArgs(2),
		ValidArgs: append(append([]string{}, screenEffects...), "none"),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			effect := args[1]
			if effect == "none" {
				effect = ""
			} else if !slices.Contains(screenEffects, effect) {
				return fmt.Errorf("unknown effect %q: use one of %s or none", effect, strings.Join(screenEffects, ", "))
			}
			var resp map[string]any
			if err := a.Client.Patch("/themes/"+args[0], map[string]any{"theme": map[string]any{"screen_effect": effect}}, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}
			if effect == "" {
				fmt.Printf("Removed the screen effect from theme %s\n", args[0])
			} else {
				fmt.Printf("Theme %s now shows %s\n", args[0], effect)
			}
			return nil
		},
	}
}

func newThemesPreviewURLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "preview-url <theme-id>",
		Short: "Print a signed link to a preview of the theme (valid for an hour, no login needed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var theme themeAreasResponse
			if err := a.Client.Get("/themes/"+args[0], nil, &theme); err != nil {
				return err
			}
			if theme.PreviewURL == "" {
				return fmt.Errorf("the API gave no preview_url for theme %s", args[0])
			}
			fmt.Println(theme.PreviewURL)
			return nil
		},
	}
}
