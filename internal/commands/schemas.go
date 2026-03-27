package commands

import (
	"github.com/spf13/cobra"
)

func NewSchemasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schemas",
		Short: "Inspect resource type schemas",
	}
	cmd.AddCommand(newSchemasListCmd())
	cmd.AddCommand(newSchemasShowCmd())
	return cmd
}

func newSchemasListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available schema keys",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/schemas", nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newSchemasShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show the JSON schema for a resource type (e.g. calendar_events, feed_items)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/schemas/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}
