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
	cmd.AddCommand(newMediaBoxesCreateCmd())
	cmd.AddCommand(newMediaBoxesUpdateCmd())
	cmd.AddCommand(newMediaBoxesDeleteCmd())
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
	var boxType, data string

	cmd := &cobra.Command{
		Use:   "create --type <type> --data '<json>'",
		Short: "Create a media box",
		Example: `  pintomind media-boxes create --type media --data '{"media_id":42,"background_size":"cover"}'
  pintomind media-boxes create --type icon --data '{"icon_name":"rocket-launch","icon_type":"regular"}'
  pintomind media-boxes create --type emoji --data '{"emoji":"✨"}'
  pintomind media-boxes create --type qr_code --data '{"url":"https://example.com"}'
  pintomind media-boxes create --type unsplash --data '{"photo_id":"abc123"}'
  pintomind media-boxes create --type gif --data '{"gif_id":"xT9IgG50Fb7Mi0only"}'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var mediaBox map[string]any
			if err := json.Unmarshal([]byte(data), &mediaBox); err != nil {
				return fmt.Errorf("invalid JSON for --data: %w", err)
			}
			body := map[string]any{
				"type":      boxType,
				"media_box": mediaBox,
			}
			var resp map[string]any
			if err := a.Client.Post("/media_boxes", body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&boxType, "type", "", "Media box type alias: media, icon, emoji, gif, unsplash, or qr_code (required)")
	cmd.Flags().StringVar(&data, "data", "", "Media box fields as JSON (required)")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("data")
	return cmd
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
