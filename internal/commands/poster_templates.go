package commands

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

type PosterTemplate struct {
	ID             int      `json:"id"`
	Name           string   `json:"name"`
	AspectRatios   []string `json:"aspect_ratios"`
	ResourceID     int      `json:"resource_id"`
	RawPosterData  string   `json:"raw_poster_data"`
}

type PosterTemplatesResponse struct {
	Total int              `json:"total"`
	Items []PosterTemplate `json:"items"`
}

func NewPosterTemplatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "poster-templates",
		Short: "List poster templates shared via content networks (account token required)",
	}
	cmd.AddCommand(newPosterTemplatesListCmd())
	cmd.AddCommand(newPosterTemplatesShowCmd())
	return cmd
}

func newPosterTemplatesListCmd() *cobra.Command {
	var sortBy string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available poster templates",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			if sortBy != "" {
				q.Set("sort_by", sortBy)
			}
			applyPagination(cmd, q)

			var resp PosterTemplatesResponse
			if err := a.Client.Get("/poster_templates", q, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, t := range resp.Items {
				ratios := ""
				for j, r := range t.AspectRatios {
					if j > 0 {
						ratios += ","
					}
					ratios += r
				}
				rows[i] = []string{
					strconv.Itoa(t.ID),
					t.Name,
					ratios,
					strconv.Itoa(t.ResourceID),
				}
			}
			printTable(cmd, []string{"ID", "NAME", "ASPECT_RATIOS", "RESOURCE_ID"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field (e.g. name:asc)")
	addPaginationFlags(cmd)
	return cmd
}

func newPosterTemplatesShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a poster template (includes raw_poster_data)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			q := url.Values{}
			q.Set("ids", args[0])
			var resp PosterTemplatesResponse
			if err := a.Client.Get("/poster_templates", q, &resp); err != nil {
				return err
			}
			if len(resp.Items) == 0 {
				return fmt.Errorf("poster template %s not found", args[0])
			}
			printJSON(resp.Items[0])
			return nil
		},
	}
}
