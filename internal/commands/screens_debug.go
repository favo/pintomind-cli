package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// The debug API runs remote console commands on a screen. Everything is a POST with the
// options in the body, so selectors and JavaScript stay out of URLs and access logs.

type debugPost struct {
	ID        int     `json:"id"`
	Type      string  `json:"type"`
	Title     *string `json:"title"`
	Duration  int     `json:"duration"`
	Remaining *int    `json:"remaining"`
	Ready     bool    `json:"ready"`
	Scheduled bool    `json:"scheduled"`
	Hidden    bool    `json:"hidden"`
	Path      *string `json:"path"`
}

type debugArea struct {
	Name    string      `json:"name"`
	Current *int        `json:"current"`
	Posts   []debugPost `json:"posts"`
}

type debugLoop struct {
	Running   bool `json:"running"`
	Suspended bool `json:"suspended"`
}

type debugLayout struct {
	Width  int         `json:"width"`
	Height int         `json:"height"`
	Loop   debugLoop   `json:"loop"`
	Areas  []debugArea `json:"areas"`
}

type debugStatus struct {
	ButlerVersion string    `json:"butler_version"`
	Loop          debugLoop `json:"loop"`
	Areas         []struct {
		Name    string     `json:"name"`
		Current *debugPost `json:"current"`
	} `json:"areas"`
}

type debugLogEntry struct {
	At      string `json:"at"`
	Level   string `json:"level"`
	Message any    `json:"message"`
}

type debugLogs struct {
	Total   int             `json:"total"`
	Entries []debugLogEntry `json:"entries"`
}

type debugRequest struct {
	URL        string `json:"url"`
	Type       string `json:"type"`
	Category   string `json:"category"`
	Status     *int   `json:"status"`
	Cache      string `json:"cache"`
	Size       *int   `json:"size"`
	DurationMs *int   `json:"duration_ms"`
	StartedAt  string `json:"started_at"`
	Error      bool   `json:"error"`
}

type debugNetwork struct {
	Total       int            `json:"total"`
	Transferred int            `json:"transferred"`
	FromCache   int            `json:"from_cache"`
	Entries     []debugRequest `json:"entries"`
}

type debugElement struct {
	Element string `json:"element"`
	Path    string `json:"path"`
	Visible bool   `json:"visible"`
	Rect    struct {
		X      int `json:"x"`
		Y      int `json:"y"`
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"rect"`
	Text string `json:"text"`
}

type debugNode struct {
	Path       string            `json:"path"`
	Tag        string            `json:"tag"`
	Attributes map[string]string `json:"attributes"`
	Text       string            `json:"text"`
	ChildCount int               `json:"childCount"`
	Children   []debugNode       `json:"children"`
}

func newScreensDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug a screen: posts, logs, network, elements, JavaScript and playback",
		Long: "Runs remote console commands on a screen.\n\n" +
			"Needs an API key with the screens:debug scope. The screen must be online and run butler 0.17.0 or newer;\n" +
			"reload the screen to update it.\n\n" +
			"Elements can be given as a CSS selector or as a path such as @1/0/2, which dom and elements print.",
	}

	cmd.AddCommand(
		newDebugStatusCmd(),
		newDebugPostsCmd(),
		newDebugPostCmd(),
		newDebugLogsCmd(),
		newDebugNetworkCmd(),
		newDebugElementsCmd(),
		newDebugDomCmd(),
		newDebugHealthCmd(),
		newDebugEvalCmd(),
		newDebugLoopCmd(),
		newDebugPostActionCmd("show-post", "show_post", "Jump to a post on the screen"),
		newDebugPostActionCmd("hide-post", "hide_post", "Hide a post until the screen reloads"),
		newDebugPostActionCmd("unhide-post", "unhide_post", "Show a hidden post again"),
		newDebugHighlightCmd(),
	)
	return cmd
}

// runDebug posts a debug command and decodes the answer into out (or prints it with --json).
func runDebug(cmd *cobra.Command, screenID, command string, body map[string]any, out any) (bool, error) {
	// The arguments were fine, so a failure here is the screen's answer, not a usage problem
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	a := app(cmd)
	if body == nil {
		body = map[string]any{}
	}

	var raw json.RawMessage
	if err := a.Client.Post("/screens/"+screenID+"/debug/"+command, body, &raw); err != nil {
		return false, err
	}

	if a.JSONOutput {
		var v any
		_ = json.Unmarshal(raw, &v)
		printJSON(v)
		return true, nil
	}

	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return false, fmt.Errorf("unexpected answer from the screen: %w", err)
		}
	}
	return false, nil
}

// elementTarget turns an argument into a selector or a path body field.
func elementTarget(arg string) map[string]any {
	if strings.HasPrefix(arg, "@") {
		return map[string]any{"path": arg}
	}
	return map[string]any{"selector": arg}
}

func newDebugStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <screen-id>",
		Short: "Show what each area is showing right now",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var status debugStatus
			if printed, err := runDebug(cmd, args[0], "status", nil, &status); err != nil || printed {
				return err
			}

			fmt.Printf("%s · butler %s\n\n", loopState(status.Loop), status.ButlerVersion)
			rows := make([][]string, 0, len(status.Areas))
			for _, area := range status.Areas {
				if area.Current == nil {
					rows = append(rows, []string{area.Name, "-", "-", "-", "-"})
					continue
				}
				rows = append(rows, []string{area.Name, strconv.Itoa(area.Current.ID), area.Current.Type, title(area.Current.Title), remaining(area.Current)})
			}
			printTable(cmd, []string{"AREA", "POST", "TYPE", "TITLE", "LEFT"}, rows)
			return nil
		},
	}
}

func newDebugPostsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "posts <screen-id>",
		Short: "List the posts in every area and their state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var layout debugLayout
			if printed, err := runDebug(cmd, args[0], "posts", nil, &layout); err != nil || printed {
				return err
			}

			fmt.Printf("%s · %dx%d\n\n", loopState(layout.Loop), layout.Width, layout.Height)
			var rows [][]string
			for _, area := range layout.Areas {
				for _, post := range area.Posts {
					rows = append(rows, []string{area.Name, strconv.Itoa(post.ID), post.Type, title(post.Title), fmt.Sprintf("%ds", post.Duration), postState(area, post)})
				}
			}
			printTable(cmd, []string{"AREA", "ID", "TYPE", "TITLE", "DURATION", "STATE"}, rows)
			return nil
		},
	}
}

func newDebugPostCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "post <screen-id> <post-id>",
		Short: "Show the data the screen has for a post",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			postID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("post id must be a number")
			}
			var resp map[string]any
			if printed, err := runDebug(cmd, args[0], "post", map[string]any{"post_id": postID}, &resp); err != nil || printed {
				return err
			}
			printJSON(resp["data"])
			return nil
		},
	}
}

func newDebugLogsCmd() *cobra.Command {
	var limit int
	var level string
	var follow bool
	var interval time.Duration

	cmd := &cobra.Command{
		Use:   "logs <screen-id>",
		Short: "Show the most recent log lines on the screen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"limit": limit}
			if level != "" {
				body["level"] = level
			}

			seen := map[string]bool{}
			for {
				var logs debugLogs
				if printed, err := runDebug(cmd, args[0], "logs", body, &logs); err != nil || printed {
					return err
				}

				for _, entry := range logs.Entries {
					key := entry.At + entry.Level + formatLogMessage(entry.Message)
					if seen[key] {
						continue
					}
					seen[key] = true
					fmt.Println(formatLogEntry(entry))
				}

				if !follow {
					return nil
				}
				time.Sleep(interval)
			}
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 100, "Number of lines (max 200)")
	cmd.Flags().StringVar(&level, "level", "", "Only lines of this level: error, warn, info or log")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Keep printing new lines")
	cmd.Flags().DurationVar(&interval, "interval", 3*time.Second, "How often --follow asks the screen")
	return cmd
}

func newDebugNetworkCmd() *cobra.Command {
	var category, query, sortBy string
	var errorsOnly, desc bool
	var limit int

	cmd := &cobra.Command{
		Use:   "network <screen-id>",
		Short: "Show the screen's requests with cache status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"limit": limit, "sort": sortBy, "errors": errorsOnly}
			if desc {
				body["direction"] = "desc"
			}
			if category != "" {
				body["category"] = category
			}
			if query != "" {
				body["query"] = query
			}

			var network debugNetwork
			if printed, err := runDebug(cmd, args[0], "network", body, &network); err != nil || printed {
				return err
			}

			rows := make([][]string, len(network.Entries))
			for i, r := range network.Entries {
				rows[i] = []string{clock(r.StartedAt), intOrDash(r.Status), r.Type, r.Cache, sizeOrDash(r.Size), msOrDash(r.DurationMs), r.URL}
			}
			printTable(cmd, []string{"STARTED", "STATUS", "TYPE", "CACHE", "SIZE", "TIME", "URL"}, rows)
			fmt.Printf("\n%d requests · %s transferred · %d from cache\n", network.Total, humanBytes(network.Transferred), network.FromCache)
			return nil
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "data, script, style, image, media, font or other")
	cmd.Flags().BoolVar(&errorsOnly, "errors", false, "Only failed requests")
	cmd.Flags().StringVar(&query, "query", "", "Only URLs containing this text")
	cmd.Flags().StringVar(&sortBy, "sort", "started", "started, size, time, name or status")
	cmd.Flags().BoolVar(&desc, "desc", false, "Sort descending")
	cmd.Flags().IntVar(&limit, "limit", 500, "Number of requests (max 2000)")
	return cmd
}

func newDebugElementsCmd() *cobra.Command {
	var full bool

	cmd := &cobra.Command{
		Use:   "elements <screen-id> <selector|@path>",
		Short: "Show size, position and visibility of matching elements",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := elementTarget(args[1])
			if full {
				body["full"] = true
				var resp map[string]any
				if printed, err := runDebug(cmd, args[0], "elements", body, &resp); err != nil || printed {
					return err
				}
				printJSON(resp["elements"])
				return nil
			}

			var resp struct {
				Elements []debugElement `json:"elements"`
			}
			if printed, err := runDebug(cmd, args[0], "elements", body, &resp); err != nil || printed {
				return err
			}

			rows := make([][]string, len(resp.Elements))
			for i, e := range resp.Elements {
				visible := "hidden"
				if e.Visible {
					visible = "visible"
				}
				rows[i] = []string{e.Element, fmt.Sprintf("%dx%d", e.Rect.Width, e.Rect.Height), fmt.Sprintf("%d,%d", e.Rect.X, e.Rect.Y), visible, truncate(e.Text, 50)}
			}
			printTable(cmd, []string{"ELEMENT", "SIZE", "POSITION", "STATE", "TEXT"}, rows)
			return nil
		},
	}

	cmd.Flags().BoolVar(&full, "full", false, "Every computed style, box model and attributes of the first match (JSON)")
	return cmd
}

func newDebugDomCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dom <screen-id> [selector|@path]",
		Short: "Show an element and its children, to walk the element tree",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"path": "@"}
			if len(args) == 2 {
				body = elementTarget(args[1])
			}

			var node debugNode
			if printed, err := runDebug(cmd, args[0], "dom", body, &node); err != nil || printed {
				return err
			}

			fmt.Printf("%s  %s\n", node.Path, describeNode(node))
			for _, child := range node.Children {
				fmt.Printf("  %s  %s\n", child.Path, describeNode(child))
			}
			return nil
		},
	}
}

func newDebugHealthCmd() *cobra.Command {
	var fps bool

	cmd := &cobra.Command{
		Use:   "health <screen-id>",
		Short: "Show memory, network connection and optionally frames per second",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp map[string]any
			if printed, err := runDebug(cmd, args[0], "health", map[string]any{"fps": fps}, &resp); err != nil || printed {
				return err
			}

			fmt.Printf("butler %v, online since %v\n", resp["butler_version"], resp["online_since"])
			for _, section := range []string{"memory", "network", "fps"} {
				values, ok := resp[section].(map[string]any)
				if !ok {
					continue
				}
				fmt.Printf("\n%s\n", strings.ToUpper(section))
				keys := make([]string, 0, len(values))
				for k := range values {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Printf("  %-20s %v\n", k, values[k])
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&fps, "fps", false, "Also measure frames per second (about 2 seconds)")
	return cmd
}

func newDebugEvalCmd() *cobra.Command {
	var path, file string

	cmd := &cobra.Command{
		Use:   "eval <screen-id> [code]",
		Short: "Run JavaScript in the screen view",
		Long: "Runs JavaScript in the screen view and prints the result.\n\n" +
			"Give the code as an argument, with --file, or on stdin with -. Use return to get a value from several\n" +
			"lines. With --path, $0 is the element at that path, like in the browser's devtools.",
		Example: "  pintomind screens debug eval 42 'document.title'\n" +
			"  pintomind screens debug eval 42 --path @1/1/2 '$0.getBoundingClientRect()'\n" +
			"  echo 'return App.loop.isRunning' | pintomind screens debug eval 42 -",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := evalCode(args, file)
			if err != nil {
				return err
			}

			body := map[string]any{"code": code}
			if path != "" {
				body["path"] = path
			}

			var resp struct {
				Value any `json:"value"`
			}
			if printed, err := runDebug(cmd, args[0], "eval", body, &resp); err != nil || printed {
				return err
			}

			if s, ok := resp.Value.(string); ok {
				fmt.Println(s)
			} else {
				printJSON(resp.Value)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "Element path to use as $0, for example @1/1/2")
	cmd.Flags().StringVar(&file, "file", "", "Read the code from a file")
	return cmd
}

func evalCode(args []string, file string) (string, error) {
	switch {
	case file != "":
		data, err := os.ReadFile(file)
		return string(data), err
	case len(args) == 2 && args[1] == "-":
		data, err := io.ReadAll(os.Stdin)
		return string(data), err
	case len(args) == 2:
		return args[1], nil
	default:
		return "", fmt.Errorf("give the code as an argument, with --file, or on stdin with -")
	}
}

func newDebugLoopCmd() *cobra.Command {
	controls := []string{"status", "pause", "resume", "toggle", "next", "previous", "next-part", "previous-part"}

	return &cobra.Command{
		Use:       "loop <screen-id> [" + strings.Join(controls, "|") + "]",
		Short:     "Show or control playback",
		Args:      cobra.RangeArgs(1, 2),
		ValidArgs: controls,
		RunE: func(cmd *cobra.Command, args []string) error {
			control := "status"
			if len(args) == 2 {
				control = args[1]
			}

			var resp struct {
				debugLoop
				Post *debugPost `json:"post"`
			}
			body := map[string]any{"control": strings.ReplaceAll(control, "-", "_")}
			if printed, err := runDebug(cmd, args[0], "loop", body, &resp); err != nil || printed {
				return err
			}

			line := loopState(resp.debugLoop)
			if resp.Post != nil {
				line += fmt.Sprintf(" · area A shows %d %s %s (%s left)", resp.Post.ID, resp.Post.Type, title(resp.Post.Title), remaining(resp.Post))
			}
			fmt.Println(line)
			return nil
		},
	}
}

func newDebugPostActionCmd(use, command, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <screen-id> <post-id>",
		Short: short,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			postID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("post id must be a number")
			}
			var resp struct {
				Message string `json:"message"`
			}
			if printed, err := runDebug(cmd, args[0], command, map[string]any{"post_id": postID}, &resp); err != nil || printed {
				return err
			}
			fmt.Println(resp.Message)
			return nil
		},
	}
}

func newDebugHighlightCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "highlight <screen-id> <selector|@path>",
		Short: "Outline elements on the physical screen for five seconds",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Message string `json:"message"`
			}
			if printed, err := runDebug(cmd, args[0], "highlight", elementTarget(args[1]), &resp); err != nil || printed {
				return err
			}
			fmt.Println(resp.Message)
			return nil
		},
	}
}

func loopState(l debugLoop) string {
	switch {
	case l.Suspended:
		return "Suspended"
	case l.Running:
		return "Playing"
	default:
		return "Paused"
	}
}

func postState(area debugArea, post debugPost) string {
	var states []string
	if area.Current != nil && *area.Current == post.ID {
		states = append(states, "showing, "+remaining(&post)+" left")
	}
	if !post.Ready {
		states = append(states, "not ready")
	}
	if post.Hidden {
		states = append(states, "hidden from console")
	} else if !post.Scheduled {
		states = append(states, "hidden by schedule")
	}
	if len(states) == 0 {
		return "-"
	}
	return strings.Join(states, ", ")
}

func title(t *string) string {
	if t == nil || *t == "" {
		return "-"
	}
	return *t
}

func remaining(p *debugPost) string {
	if p.Remaining == nil {
		return "-"
	}
	return fmt.Sprintf("%ds", *p.Remaining)
}

func formatLogEntry(e debugLogEntry) string {
	return fmt.Sprintf("%s  %-5s  %s", clock(e.At), strings.ToUpper(e.Level), formatLogMessage(e.Message))
}

func formatLogMessage(m any) string {
	if s, ok := m.(string); ok {
		return s
	}
	data, _ := json.Marshal(m)
	return string(data)
}

func describeNode(n debugNode) string {
	label := "<" + n.Tag
	if id := n.Attributes["id"]; id != "" {
		label += " id=\"" + id + "\""
	}
	if class := n.Attributes["class"]; class != "" {
		label += " class=\"" + class + "\""
	}
	label += ">"
	if n.ChildCount > 0 {
		label += fmt.Sprintf(" (%d children)", n.ChildCount)
	}
	if n.Text != "" {
		label += "  " + truncate(n.Text, 60)
	}
	return label
}

func clock(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return "-"
	}
	return t.Local().Format("15:04:05")
}

func intOrDash(v *int) string {
	if v == nil {
		return "-"
	}
	return strconv.Itoa(*v)
}

func msOrDash(v *int) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%d ms", *v)
}

func sizeOrDash(v *int) string {
	if v == nil {
		return "-"
	}
	return humanBytes(*v)
}

func humanBytes(n int) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f kB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/1024/1024)
	}
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}
