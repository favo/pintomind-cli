package commands

import (
	"github.com/spf13/cobra"
)

func NewIconsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "icons",
		Short: "List available icon names, types, and categories (for icon media boxes)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/icons", nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}
