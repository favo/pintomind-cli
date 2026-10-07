package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"favo/pintomind-cli/internal/appctx"
	"github.com/spf13/cobra"
)

// Pagination mirrors the `pagination` object returned by list endpoints.
type Pagination struct {
	CurrentPage int     `json:"current_page"`
	NextPage    *int    `json:"next_page"`
	PrevPage    *int    `json:"prev_page"`
	NextPageURL *string `json:"next_page_url"`
	PrevPageURL *string `json:"prev_page_url"`
	TotalPages  int     `json:"total_pages"`
	TotalCount  int     `json:"total_count"`
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// printTotal prints the "Total: N" header of a list command. It prefers
// pagination.total_count (all matching records) over the page-sized total and
// shows the current page when there is more than one.
func printTotal(total int, p *Pagination) {
	fmt.Println(totalLine(total, p))
	fmt.Println()
}

func totalLine(total int, p *Pagination) string {
	if p == nil {
		return fmt.Sprintf("Total: %d", total)
	}
	line := fmt.Sprintf("Total: %d", p.TotalCount)
	if p.TotalPages > 1 {
		line += fmt.Sprintf(" (page %d/%d, %d shown; use --page/--per-page for more)", p.CurrentPage, p.TotalPages, total)
	}
	return line
}

// selectColumns narrows a table to the columns backed by the API fields the
// user asked for with --fields, so unrequested columns don't show up as zero
// values. columnFields[i] lists the API fields that fill column i. With no
// --fields (or none matching a column) the table is returned unchanged.
func selectColumns(fields string, headers []string, columnFields [][]string, rows [][]string) ([]string, [][]string) {
	if strings.TrimSpace(fields) == "" {
		return headers, rows
	}
	requested := map[string]bool{}
	for _, f := range strings.Split(fields, ",") {
		requested[strings.TrimSpace(f)] = true
	}
	var keep []int
	for i, names := range columnFields {
		for _, name := range names {
			if requested[name] {
				keep = append(keep, i)
				break
			}
		}
	}
	if len(keep) == 0 {
		return headers, rows
	}
	pick := func(values []string) []string {
		out := make([]string, len(keep))
		for j, i := range keep {
			if i < len(values) {
				out[j] = values[i]
			}
		}
		return out
	}
	selectedRows := make([][]string, len(rows))
	for r, row := range rows {
		selectedRows[r] = pick(row)
	}
	return pick(headers), selectedRows
}

func printTable(cmd *cobra.Command, headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				if n := utf8.RuneCountInString(cell); n > widths[i] {
					widths[i] = n
				}
			}
		}
	}
	separator := make([]string, len(headers))
	for i, n := range widths {
		separator[i] = strings.Repeat("-", n)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	fmt.Fprintln(w, strings.Join(separator, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

func app(cmd *cobra.Command) *appctx.App {
	return appctx.FromContext(cmd.Context())
}
