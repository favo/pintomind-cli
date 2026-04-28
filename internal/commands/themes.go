package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

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
	cmd.AddCommand(newThemesDeleteCmd())
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
	var themeType string
	var data string

	cmd := &cobra.Command{
		Use:   "create --data '<json>'",
		Short: "Create a theme",
		Example: `  pintomind themes create --data '{"name":"My theme","font_family_header_id":1,"font_family_body_id":2,"color_palette_id":3}'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var theme map[string]any
			if err := json.Unmarshal([]byte(data), &theme); err != nil {
				return fmt.Errorf("invalid JSON for --data: %w", err)
			}
			body := map[string]any{"theme": theme}
			if themeType != "" {
				body["type"] = themeType
			}
			var resp map[string]any
			if err := a.Client.Post("/themes", body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&themeType, "type", "", "Theme type alias (default: modern)")
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
