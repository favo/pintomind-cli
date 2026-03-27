package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api [METHOD] <path>",
		Short: "Make a raw API request",
		Long: `Make a raw request to the Pintomind API. METHOD defaults to GET.
The path should start with / and is relative to /api/v1.

Pass a JSON body via stdin for POST/PATCH requests.`,
		Example: `  pintomind api /screens
  pintomind api /screens/42
  pintomind api GET /channels?sort_by=name
  echo '{"screen":{"command":"reload"}}' | pintomind api PATCH /screens/42`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)

			method := "GET"
			path := args[0]
			if len(args) == 2 {
				method = strings.ToUpper(args[0])
				path = args[1]
			}

			var body []byte
			if method == "POST" || method == "PATCH" || method == "PUT" {
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) == 0 {
					body, _ = os.ReadFile("/dev/stdin")
				}
			}

			data, status, err := a.Client.DoRaw(method, path, body)
			if err != nil {
				return err
			}

			if !a.JSONOutput && a.Verbose {
				fmt.Fprintf(os.Stderr, "< HTTP %d\n", status)
			}

			os.Stdout.Write(data)
			if len(data) > 0 && data[len(data)-1] != '\n' {
				fmt.Println()
			}
			return nil
		},
	}
}
