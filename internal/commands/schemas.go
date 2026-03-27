package commands

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
)

// schemaIDRe extracts schema IDs from the HTML links the /schemas endpoint returns,
// e.g. <a href='/api/v1/schemas/calendar_events'>calendar_events</a>
var schemaIDRe = regexp.MustCompile(`href='[^']+/schemas/([^']+)'`)

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
			// The /schemas endpoint returns HTML, not JSON.
			body, _, err := a.Client.DoRaw("GET", "/schemas", nil)
			if err != nil {
				return err
			}
			matches := schemaIDRe.FindAllSubmatch(body, -1)
			if len(matches) == 0 {
				fmt.Println("No schemas found.")
				return nil
			}
			for _, m := range matches {
				fmt.Println(string(m[1]))
			}
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
