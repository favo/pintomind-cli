package commands

import (
	"fmt"
	"net/url"
	"strconv"

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

	// Read
	cmd.AddCommand(newScreensListCmd())
	cmd.AddCommand(newScreensShowCmd())

	// Channel assignment
	cmd.AddCommand(newScreensSetChannelCmd())
	cmd.AddCommand(newScreensTempChannelCmd())

	// Direct commands
	cmd.AddCommand(newScreenActionCmd("reload", "Reload the screen", "reload", ""))
	cmd.AddCommand(newScreenActionCmd("reboot", "Reboot the screen", "reboot", ""))
	cmd.AddCommand(newScreenActionCmd("clear-cache", "Clear the screen cache", "clear_cache", ""))
	cmd.AddCommand(newScreenActionCmd("upgrade-firmware", "Upgrade screen firmware", "upgrade_firmware", ""))
	cmd.AddCommand(newScreenActionCmd("identify", "Identify the screen (shows overlay)", "identify", ""))
	cmd.AddCommand(newScreenActionCmd("toggle-night-mode", "Toggle night mode on the screen", "toggle_night_mode", ""))

	// Remote control signals
	cmd.AddCommand(newScreenActionCmd("next", "Skip to next content", "remote_control", "next"))
	cmd.AddCommand(newScreenActionCmd("previous", "Go to previous content", "remote_control", "previous"))
	cmd.AddCommand(newScreenActionCmd("play", "Play content", "remote_control", "play"))
	cmd.AddCommand(newScreenActionCmd("pause", "Pause content", "remote_control", "pause"))
	cmd.AddCommand(newScreenActionCmd("toggle-play", "Toggle play/pause", "remote_control", "toggle_play"))
	cmd.AddCommand(newScreenActionCmd("forwards", "Skip forwards", "remote_control", "forwards"))
	cmd.AddCommand(newScreenActionCmd("backwards", "Skip backwards", "remote_control", "backwards"))

	// Effects
	cmd.AddCommand(newScreenActionCmd("confetti-fire", "Trigger confetti fire effect", "trigger_effect", "confetti_fire"))
	cmd.AddCommand(newScreenActionCmd("confetti-fireworks", "Trigger confetti fireworks effect", "trigger_effect", "confetti_fireworks"))
	cmd.AddCommand(newScreenActionCmd("school-parade", "Trigger school parade effect", "trigger_effect", "confetti_school_parade"))
	cmd.AddCommand(newScreenActionCmd("snow", "Trigger snow effect", "trigger_effect", "snow"))

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
