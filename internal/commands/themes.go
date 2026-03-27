package commands

import (
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
		Short: "List and inspect themes",
	}
	cmd.AddCommand(newThemesListCmd())
	cmd.AddCommand(newThemesShowCmd())
	cmd.AddCommand(newThemesStatsCmd())
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
