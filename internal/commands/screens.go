package commands

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

type Screen struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Online    bool   `json:"online"`
	ChannelID any    `json:"channel_id"`
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
		newScreenActionCmd("toggle-night-mode", "Toggle night mode on the screen", "toggle_night_mode", ""),
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
			rows := make([][]string, len(resp.Items))
			for i, s := range resp.Items {
				status := "offline"
				if s.Online {
					status = "online"
				}
				ch := "-"
				if s.ChannelID != nil {
					ch = fmt.Sprint(s.ChannelID)
				}
				rows[i] = []string{strconv.Itoa(s.ID), s.Name, status, ch}
			}
			printTable(cmd, []string{"ID", "NAME", "STATUS", "CHANNEL"}, rows)
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
				rows := make([][]string, len(resp.Items))
				for i, s := range resp.Items {
					status := "offline"
					if s.Online {
						status = "online"
					}
					ch := "-"
					if s.ChannelID != nil {
						ch = fmt.Sprint(s.ChannelID)
					}
					rows[i] = []string{strconv.Itoa(s.ID), s.Name, status, ch}
				}
				printTable(cmd, []string{"ID", "NAME", "STATUS", "CHANNEL"}, rows)
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
