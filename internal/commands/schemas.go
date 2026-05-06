package commands

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

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

// newSchemaSubCmd returns a "schema [type]" subcommand for a command group.
// keys maps user-friendly type names (e.g. "image") to API schema keys (e.g. "post_image").
// With a single entry and no arg the schema is fetched automatically.
// With multiple entries and no arg the available types are listed.
func newSchemaSubCmd(keys map[string]string) *cobra.Command {
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	return &cobra.Command{
		Use:   "schema [type]",
		Short: "Show the JSON schema for this resource type",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)

			var schemaKey string
			if len(args) == 0 {
				if len(keys) == 1 {
					schemaKey = sorted[0]
					schemaKey = keys[schemaKey]
				} else {
					fmt.Println("Available types:")
					for _, t := range sorted {
						fmt.Printf("  %s\n", t)
					}
					return nil
				}
			} else {
				key, ok := keys[args[0]]
				if !ok {
					return fmt.Errorf("unknown type %q — valid types: %s", args[0], strings.Join(sorted, ", "))
				}
				schemaKey = key
			}

			var resp map[string]any
			if err := a.Client.Get("/schemas/"+schemaKey, nil, &resp); err != nil {
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
