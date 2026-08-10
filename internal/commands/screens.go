package commands

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

type ScreenChannel struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Screen struct {
	ID            int            `json:"id"`
	Name          string         `json:"name"`
	Online        bool           `json:"online"`
	OnlineSince   string         `json:"online_since"`
	OfflineSince  string         `json:"offline_since"`
	LastStartupAt string         `json:"last_startup_at"`
	Channel       *ScreenChannel `json:"channel"`
}

// screenTableHeaders and screenTableRows render the shared screen list/watch table.
var screenTableHeaders = []string{"ID", "NAME", "STATUS", "LAST_STARTUP", "CHANNEL"}

func screenTableRows(items []Screen) [][]string {
	rows := make([][]string, len(items))
	for i, s := range items {
		status := "offline"
		since := s.OfflineSince
		if s.Online {
			status = "online"
			since = s.OnlineSince
		}
		if d, ok := durationSince(since); ok {
			status += " for " + humanDuration(d)
		}
		startup := "-"
		if d, ok := durationSince(s.LastStartupAt); ok {
			startup = humanDuration(d) + " ago"
		}
		ch := "-"
		if s.Channel != nil {
			ch = fmt.Sprintf("%s (%d)", s.Channel.Name, s.Channel.ID)
		}
		rows[i] = []string{strconv.Itoa(s.ID), s.Name, status, startup, ch}
	}
	return rows
}

// durationSince parses an API timestamp and returns the elapsed time.
func durationSince(ts string) (time.Duration, bool) {
	if ts == "" {
		return 0, false
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return 0, false
	}
	return time.Since(t), true
}

// humanDuration formats a duration as a rough human-readable amount ("2 hours", "3 days").
func humanDuration(d time.Duration) string {
	plural := func(n int, unit string) string {
		if n == 1 {
			return fmt.Sprintf("1 %s", unit)
		}
		return fmt.Sprintf("%d %ss", n, unit)
	}
	switch {
	case d < time.Minute:
		return "less than a minute"
	case d < time.Hour:
		return plural(int(d.Minutes()), "minute")
	case d < 24*time.Hour:
		return plural(int(d.Hours()), "hour")
	case d < 30*24*time.Hour:
		return plural(int(d.Hours()/24), "day")
	case d < 365*24*time.Hour:
		return plural(int(d.Hours()/(24*30)), "month")
	default:
		return plural(int(d.Hours()/(24*365)), "year")
	}
}

type ScreensResponse struct {
	Total int      `json:"total"`
	Items []Screen `json:"items"`
}

func NewScreensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "screens",
		Short: "Manage screens",
	}

	const (
		groupCommands = "commands"
		groupActions  = "actions"
		groupEffects  = "effects"
	)
	cmd.AddGroup(
		&cobra.Group{ID: groupCommands, Title: "Available commands:"},
		&cobra.Group{ID: groupActions, Title: "Available actions:"},
		&cobra.Group{ID: groupEffects, Title: "Fun effects:"},
	)

	addTo := func(groupID string, cmds ...*cobra.Command) {
		for _, c := range cmds {
			c.GroupID = groupID
			cmd.AddCommand(c)
		}
	}

	// Info & management
	addTo(groupCommands,
		newScreensListCmd(),
		newScreensShowCmd(),
		newScreensConnectCmd(),
		newScreensUpdateCmd(),
		newScreensStatsCmd(),
		newScreensWatchCmd(),
		newScreensWaitOnlineCmd(),
		newScreensSetChannelCmd(),
		newScreensTempChannelCmd(),
	)

	// Device + playback actions
	addTo(groupActions,
		newScreenActionCmd("reload", "Reload the screen", "reload", ""),
		newScreenActionCmd("reboot", "Reboot the screen", "reboot", ""),
		newScreenActionCmd("clear-cache", "Clear the screen cache", "clear_cache", ""),
		newScreenActionCmd("upgrade-firmware", "Upgrade screen firmware", "upgrade_firmware", ""),
		newScreenActionCmd("identify", "Identify the screen (shows overlay)", "identify", ""),
		newScreensToggleNightModeCmd(),
		newScreenActionCmd("next", "Skip to next content", "remote_control", "next"),
		newScreenActionCmd("previous", "Go to previous content", "remote_control", "previous"),
		newScreenActionCmd("play", "Play content", "remote_control", "play"),
		newScreenActionCmd("pause", "Pause content", "remote_control", "pause"),
		newScreenActionCmd("toggle-play", "Toggle play/pause", "remote_control", "toggle_play"),
		newScreenActionCmd("forwards", "Skip forwards", "remote_control", "forwards"),
		newScreenActionCmd("backwards", "Skip backwards", "remote_control", "backwards"),
	)

	// Fun effects
	addTo(groupEffects,
		newScreenActionCmd("confetti-fire", "Trigger confetti fire effect", "trigger_effect", "confetti_fire"),
		newScreenActionCmd("confetti-fireworks", "Trigger confetti fireworks effect", "trigger_effect", "confetti_fireworks"),
		newScreenActionCmd("school-parade", "Trigger school parade effect", "trigger_effect", "confetti_school_parade"),
		newScreenActionCmd("snow", "Trigger snow effect", "trigger_effect", "snow"),
	)

	return cmd
}

// newScreenActionCmd builds a subcommand that sends a fixed API command+signal to one or more screens.
func newScreenActionCmd(name, short, apiCommand, signal string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name + " [id]",
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, _ := cmd.Flags().GetString("ids")
			all, _ := cmd.Flags().GetBool("all")

			singleID := ""
			if len(args) == 1 {
				singleID = args[0]
			}

			targetIDs, bulk, err := resolveScreenIDs(cmd, singleID, ids, all)
			if err != nil {
				return err
			}

			screen := map[string]any{"command": apiCommand}
			if signal != "" {
				screen["signal"] = signal
			}
			body := map[string]any{"screen": screen}

			label := name
			if signal != "" {
				label = signal
			}

			target := targetIDs
			if bulk {
				target = "screens " + targetIDs
			} else {
				target = "screen " + targetIDs
			}

			return sendScreenPatch(cmd, targetIDs, bulk, body,
				fmt.Sprintf("Sent %q to %s", label, target))
		},
	}
	addTargetFlags(cmd)
	return cmd
}

func newScreensToggleNightModeCmd() *cobra.Command {
	var on, off bool

	cmd := &cobra.Command{
		Use:   "toggle-night-mode [id]",
		Short: "Toggle night mode on the screen (--on / --off to force a state)",
		Example: `  pintomind screens toggle-night-mode 42          # toggle current state
  pintomind screens toggle-night-mode 42 --on    # force night mode on
  pintomind screens toggle-night-mode --all --off`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if on && off {
				return fmt.Errorf("--on and --off are mutually exclusive")
			}

			ids, _ := cmd.Flags().GetString("ids")
			all, _ := cmd.Flags().GetBool("all")

			singleID := ""
			if len(args) == 1 {
				singleID = args[0]
			}

			targetIDs, bulk, err := resolveScreenIDs(cmd, singleID, ids, all)
			if err != nil {
				return err
			}

			screen := map[string]any{"command": "toggle_night_mode"}
			label := "toggle-night-mode"
			if on {
				screen["state"] = true
				label = "night mode on"
			}
			if off {
				screen["state"] = false
				label = "night mode off"
			}
			body := map[string]any{"screen": screen}

			target := "screen " + targetIDs
			if bulk {
				target = "screens " + targetIDs
			}

			return sendScreenPatch(cmd, targetIDs, bulk, body,
				fmt.Sprintf("Sent %q to %s", label, target))
		},
	}
	cmd.Flags().BoolVar(&on, "on", false, "Force night mode on instead of toggling")
	cmd.Flags().BoolVar(&off, "off", false, "Force night mode off instead of toggling")
	addTargetFlags(cmd)
	return cmd
}

func newScreensListCmd() *cobra.Command {
	var online, offline bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List screens",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			if online {
				q.Set("online", "true")
			}
			if offline {
				q.Set("offline", "true")
			}
			applyPagination(cmd, q)

			var resp ScreensResponse
			if err := a.Client.Get("/screens", q, &resp); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			printTable(cmd, screenTableHeaders, screenTableRows(resp.Items))
			return nil
		},
	}
	cmd.Flags().BoolVar(&online, "online", false, "Show only online screens")
	cmd.Flags().BoolVar(&offline, "offline", false, "Show only offline screens")
	addPaginationFlags(cmd)
	return cmd
}

func newScreensShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a screen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/screens/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newScreensUpdateCmd() *cobra.Command {
	var name, notes string

	cmd := &cobra.Command{
		Use:   "update <id> [--name <name>] [--notes <notes>]",
		Short: "Update a screen's name and notes",
		Example: `  pintomind screens update 42 --name "Lobby screen"
  pintomind screens update 42 --notes "Behind the reception desk"
  pintomind screens update 42 --name "Lobby screen" --notes ""   # empty --notes clears them`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			screen := map[string]any{}
			if cmd.Flags().Changed("name") {
				screen["name"] = name
			}
			if cmd.Flags().Changed("notes") {
				screen["notes"] = notes
			}
			if len(screen) == 0 {
				return fmt.Errorf("pass --name and/or --notes")
			}
			var resp map[string]any
			if err := a.Client.Patch("/screens/"+args[0], map[string]any{"screen": screen}, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}
			fmt.Printf("Updated screen %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Screen name (blank names are ignored by the API)")
	cmd.Flags().StringVar(&notes, "notes", "", "Screen notes (pass an empty string to clear)")
	return cmd
}

func newScreensConnectCmd() *cobra.Command {
	var name, notes string
	var channelID int

	cmd := &cobra.Command{
		Use:   "connect <code>",
		Short: "Connect a new screen using the code shown on it",
		Long: `Connect a new screen to your account using the short-lived code displayed on it.

A valid code can only be used once. Optionally pass --channel-id to show a
channel on the screen immediately, --name to name the screen (a name is
generated otherwise), and --notes to add notes. Limited to 10 attempts per
API key in a 10-minute window.`,
		Example: `  pintomind screens connect 12ABC
  pintomind screens connect 12ABC --channel-id 17
  pintomind screens connect 12ABC --channel-id 17 --name "Lobby screen" --notes "Behind the reception desk"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			screen := map[string]any{"code": args[0]}
			if channelID > 0 {
				screen["channel_id"] = channelID
			}
			if name != "" {
				screen["name"] = name
			}
			if notes != "" {
				screen["notes"] = notes
			}
			var resp map[string]any
			if err := a.Client.Post("/screens", map[string]any{"screen": screen}, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}
			id, _ := resp["id"].(float64)
			name, _ := resp["name"].(string)
			fmt.Printf("Connected screen %d (%s)\n", int(id), name)
			if channelID > 0 {
				fmt.Printf("Showing channel %d\n", channelID)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Screen name (generated if omitted)")
	cmd.Flags().StringVar(&notes, "notes", "", "Screen notes")
	cmd.Flags().IntVar(&channelID, "channel-id", 0, "Channel to show on the screen immediately")
	return cmd
}

func newScreensStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show screen online/offline counts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/screens/stats", nil, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}
			if online, ok := resp["online"]; ok {
				fmt.Printf("Online:  %v\n", online)
			}
			if offline, ok := resp["offline"]; ok {
				fmt.Printf("Offline: %v\n", offline)
			}
			return nil
		},
	}
}

func newScreensSetChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-channel [screen-id] <channel-id>",
		Short: "Switch a screen (or multiple) to a channel",
		Example: `  pintomind screens set-channel 42 7
  pintomind screens set-channel --ids 1,2,3 7
  pintomind screens set-channel --all 7`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, _ := cmd.Flags().GetString("ids")
			all, _ := cmd.Flags().GetBool("all")

			var singleScreenID, channelID string
			if ids != "" || all {
				if len(args) != 1 {
					return fmt.Errorf("expected <channel-id> when using --ids or --all")
				}
				channelID = args[0]
			} else {
				if len(args) != 2 {
					return fmt.Errorf("expected <screen-id> <channel-id>")
				}
				singleScreenID = args[0]
				channelID = args[1]
			}

			chID, err := strconv.Atoi(channelID)
			if err != nil {
				return fmt.Errorf("channel-id must be an integer")
			}

			targetIDs, bulk, err := resolveScreenIDs(cmd, singleScreenID, ids, all)
			if err != nil {
				return err
			}

			body := map[string]any{"screen": map[string]any{"channel_id": chID}}
			return sendScreenPatch(cmd, targetIDs, bulk, body,
				fmt.Sprintf("Switched to channel %s", channelID))
		},
	}
	addTargetFlags(cmd)
	return cmd
}

func newScreensTempChannelCmd() *cobra.Command {
	var duration int
	var until string
	var toggle bool

	cmd := &cobra.Command{
		Use:   "temp-channel [screen-id] <channel-id>",
		Short: "Set a temporary channel override on a screen",
		Example: `  pintomind screens temp-channel 42 7 --duration 3600
  pintomind screens temp-channel 42 7 --until 2025-12-31T23:59:00Z
  pintomind screens temp-channel --all 7 --duration 1800
  pintomind screens temp-channel 42 7 --toggle`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, _ := cmd.Flags().GetString("ids")
			all, _ := cmd.Flags().GetBool("all")

			var singleScreenID, channelID string
			if ids != "" || all {
				if len(args) != 1 {
					return fmt.Errorf("expected <channel-id> when using --ids or --all")
				}
				channelID = args[0]
			} else {
				if len(args) != 2 {
					return fmt.Errorf("expected <screen-id> <channel-id>")
				}
				singleScreenID = args[0]
				channelID = args[1]
			}

			chID, err := strconv.Atoi(channelID)
			if err != nil {
				return fmt.Errorf("channel-id must be an integer")
			}

			screen := map[string]any{"temporary_channel_id": chID}
			if toggle {
				screen["temporary_channel_active"] = "toggle"
			} else {
				screen["temporary_channel_active"] = true
			}
			if duration > 0 {
				screen["temporary_channel_duration"] = duration
			}
			if until != "" {
				screen["temporary_channel_until"] = until
			}

			targetIDs, bulk, err := resolveScreenIDs(cmd, singleScreenID, ids, all)
			if err != nil {
				return err
			}

			body := map[string]any{"screen": screen}
			return sendScreenPatch(cmd, targetIDs, bulk, body,
				fmt.Sprintf("Set temporary channel %s", channelID))
		},
	}
	addTargetFlags(cmd)
	cmd.Flags().IntVar(&duration, "duration", 0, "Duration in seconds")
	cmd.Flags().StringVar(&until, "until", "", "ISO8601 timestamp until which override is active")
	cmd.Flags().BoolVar(&toggle, "toggle", false, "Toggle temporary channel active state")
	return cmd
}

func newScreensWatchCmd() *cobra.Command {
	var interval int

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Poll and display screen status, refreshing every N seconds",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			tick := time.NewTicker(time.Duration(interval) * time.Second)
			defer tick.Stop()

			printScreens := func() error {
				var resp ScreensResponse
				if err := a.Client.Get("/screens", url.Values{"per_page": {"1000"}}, &resp); err != nil {
					return err
				}
				// Clear screen
				fmt.Print("\033[H\033[2J")
				fmt.Printf("Screens — %s  (refreshing every %ds, Ctrl+C to quit)\n\n",
					time.Now().Format("15:04:05"), interval)
				printTable(cmd, screenTableHeaders, screenTableRows(resp.Items))
				return nil
			}

			if err := printScreens(); err != nil {
				return err
			}
			for range tick.C {
				if err := printScreens(); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&interval, "interval", 5, "Refresh interval in seconds")
	return cmd
}

func newScreensWaitOnlineCmd() *cobra.Command {
	var timeout int

	return &cobra.Command{
		Use:   "wait-online <id>",
		Short: "Block until a screen comes online (useful in scripts)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			deadline := time.Now().Add(time.Duration(timeout) * time.Second)
			fmt.Printf("Waiting for screen %s to come online...\n", args[0])

			for {
				var resp map[string]any
				if err := a.Client.Get("/screens/"+args[0], nil, &resp); err != nil {
					return err
				}
				if screen, ok := resp["screen"].(map[string]any); ok {
					if online, _ := screen["online"].(bool); online {
						fmt.Printf("Screen %s is online.\n", args[0])
						return nil
					}
				}
				if timeout > 0 && time.Now().After(deadline) {
					return fmt.Errorf("timed out waiting for screen %s to come online", args[0])
				}
				time.Sleep(3 * time.Second)
				fmt.Print(".")
			}
		},
	}
}
