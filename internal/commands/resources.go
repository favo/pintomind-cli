package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"github.com/spf13/cobra"
)

func NewResourcesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resources",
		Short: "Manage resources",
	}
	cmd.AddCommand(newResourcesListCmd())
	cmd.AddCommand(newResourcesShowCmd())
	cmd.AddCommand(newResourcesStatsCmd())
	cmd.AddCommand(newResourcesCreateCmd())
	cmd.AddCommand(newResourcesUpdateCmd())
	cmd.AddCommand(newResourcesAppendCmd())
	cmd.AddCommand(newResourcesRefreshCmd())
	cmd.AddCommand(newResourcesDeleteCmd())
	return cmd
}

func newResourcesListCmd() *cobra.Command {
	var resourceType string
	var sortBy string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List resources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			if resourceType != "" {
				q.Set("type", resourceType)
			}
			if sortBy != "" {
				q.Set("sort_by", sortBy)
			}
			var resp map[string]any
			if err := a.Client.Get("/resources", q, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&resourceType, "type", "", "Filter by resource type alias")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field (e.g. name:asc)")
	return cmd
}

func newResourcesShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a resource",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/resources/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newResourcesStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show resource stats",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/resources/stats", nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newResourcesCreateCmd() *cobra.Command {
	var resourceType string
	var data string

	cmd := &cobra.Command{
		Use:   "create --type <type> --data '<json>'",
		Short: "Create a resource",
		Example: `  pintomind resources create --type text_slide --data '{"title":"Hello","body":"World"}'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var resource map[string]any
			if err := json.Unmarshal([]byte(data), &resource); err != nil {
				return fmt.Errorf("invalid JSON for --data: %w", err)
			}
			body := map[string]any{
				"type":     resourceType,
				"resource": resource,
			}
			var resp map[string]any
			if err := a.Client.Post("/resources", body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&resourceType, "type", "", "Resource type alias (required)")
	cmd.Flags().StringVar(&data, "data", "", "Resource fields as JSON (required)")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

func newResourcesUpdateCmd() *cobra.Command {
	var data string

	cmd := &cobra.Command{
		Use:   "update <id> --data '<json>'",
		Short: "Update a resource",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resource map[string]any
			if err := json.Unmarshal([]byte(data), &resource); err != nil {
				return fmt.Errorf("invalid JSON for --data: %w", err)
			}
			body := map[string]any{"resource": resource}
			var resp map[string]any
			if err := a.Client.Patch("/resources/"+args[0], body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "Resource fields as JSON (required)")
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

func newResourcesAppendCmd() *cobra.Command {
	var items string

	cmd := &cobra.Command{
		Use:   "append <id> --items '<json-array>'",
		Short: "Append items to a resource",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var itemList []any
			if err := json.Unmarshal([]byte(items), &itemList); err != nil {
				return fmt.Errorf("invalid JSON array for --items: %w", err)
			}
			body := map[string]any{"resource": map[string]any{"items": itemList}}
			var resp map[string]any
			if err := a.Client.Patch("/resources/"+args[0]+"/append", body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&items, "items", "", "Items to append as JSON array (required)")
	_ = cmd.MarkFlagRequired("items")
	return cmd
}

func newResourcesRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh <id>",
		Short: "Refresh an external resource",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/resources/"+args[0]+"/refresh", nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newResourcesDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a resource (soft-delete; second call hard-deletes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if !force {
				fmt.Printf("Delete resource %s? This soft-deletes (run again to hard-delete). Pass --force to skip this prompt.\n", args[0])
				var confirm string
				fmt.Print("Type 'yes' to confirm: ")
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if err := a.Client.Delete("/resources/" + args[0]); err != nil {
				return err
			}
			fmt.Printf("Deleted resource %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
