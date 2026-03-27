package commands

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

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

type ScreenResponse struct {
	Screen map[string]any `json:"screen"`
}

func NewScreensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "screens",
		Short: "Manage screens",
	}
	cmd.AddCommand(newScreensListCmd())
	cmd.AddCommand(newScreensShowCmd())
	cmd.AddCommand(newScreensCommandCmd())
	cmd.AddCommand(newScreensSignalCmd())
	cmd.AddCommand(newScreensSetChannelCmd())
	cmd.AddCommand(newScreensTempChannelCmd())
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

var validCommands = []string{
	"reload", "reboot", "clear_cache", "upgrade_firmware",
	"identify", "trigger_effect", "remote_control", "toggle_night_mode",
}

func newScreensCommandCmd() *cobra.Command {
	var ids string

	cmd := &cobra.Command{
		Use:   "command <id|--ids id1,id2> <command>",
		Short: "Send a command to a screen or multiple screens",
		Long: fmt.Sprintf("Valid commands: %s", strings.Join(validCommands, ", ")),
		Example: `  pintomind screens command 42 reload
  pintomind screens command 42 identify
  pintomind screens command --ids 1,2,3 reboot`,
		Args: func(cmd *cobra.Command, args []string) error {
			if ids != "" {
				if len(args) != 1 {
					return fmt.Errorf("expected <command> when using --ids")
				}
				return nil
			}
			if len(args) != 2 {
				return fmt.Errorf("expected <id> <command>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var screenID, command string
			if ids != "" {
				command = args[0]
			} else {
				screenID = args[0]
				command = args[1]
			}

			body := map[string]any{"screen": map[string]any{"command": command}}

			if ids != "" {
				q := url.Values{"ids": {ids}}
				var resp map[string]any
				if err := a.Client.Patch("/screens/bulk?"+q.Encode(), body, &resp); err != nil {
					return err
				}
				if a.JSONOutput {
					printJSON(resp)
				} else {
					fmt.Printf("Sent command %q to screens %s\n", command, ids)
				}
			} else {
				var resp map[string]any
				if err := a.Client.Patch("/screens/"+screenID, body, &resp); err != nil {
					return err
				}
				if a.JSONOutput {
					printJSON(resp)
				} else {
					fmt.Printf("Sent command %q to screen %s\n", command, screenID)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&ids, "ids", "", "Comma-separated screen IDs for bulk operation")
	return cmd
}

var validSignals = []string{"next", "previous", "play", "pause", "toggle_play", "forwards", "backwards"}
var validEffects = []string{"confetti_fire", "confetti_fireworks", "confetti_school_parade", "snow"}

func newScreensSignalCmd() *cobra.Command {
	var isEffect bool
	var ids string

	cmd := &cobra.Command{
		Use:   "signal <id> <signal>",
		Short: "Send a remote_control signal or trigger an effect on a screen",
		Long: fmt.Sprintf("Remote control signals: %s\nEffects (use --effect): %s",
			strings.Join(validSignals, ", "),
			strings.Join(validEffects, ", ")),
		Example: `  pintomind screens signal 42 next
  pintomind screens signal 42 confetti_fire --effect`,
		Args: func(cmd *cobra.Command, args []string) error {
			if ids != "" {
				if len(args) != 1 {
					return fmt.Errorf("expected <signal> when using --ids")
				}
				return nil
			}
			if len(args) != 2 {
				return fmt.Errorf("expected <id> <signal>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var screenID, signal string
			if ids != "" {
				signal = args[0]
			} else {
				screenID = args[0]
				signal = args[1]
			}

			command := "remote_control"
			if isEffect {
				command = "trigger_effect"
			}

			body := map[string]any{
				"screen": map[string]any{
					"command": command,
					"signal":  signal,
				},
			}

			if ids != "" {
				q := url.Values{"ids": {ids}}
				var resp map[string]any
				if err := a.Client.Patch("/screens/bulk?"+q.Encode(), body, &resp); err != nil {
					return err
				}
				if a.JSONOutput {
					printJSON(resp)
				} else {
					fmt.Printf("Sent signal %q to screens %s\n", signal, ids)
				}
			} else {
				var resp map[string]any
				if err := a.Client.Patch("/screens/"+screenID, body, &resp); err != nil {
					return err
				}
				if a.JSONOutput {
					printJSON(resp)
				} else {
					fmt.Printf("Sent signal %q to screen %s\n", signal, screenID)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&isEffect, "effect", false, "Treat signal as a trigger_effect signal")
	cmd.Flags().StringVar(&ids, "ids", "", "Comma-separated screen IDs for bulk operation")
	return cmd
}

func newScreensSetChannelCmd() *cobra.Command {
	var ids string

	cmd := &cobra.Command{
		Use:   "set-channel <screen-id> <channel-id>",
		Short: "Switch a screen (or multiple) to a channel",
		Example: `  pintomind screens set-channel 42 7
  pintomind screens set-channel --ids 1,2,3 7`,
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)

			var screenID, channelID string
			if ids != "" {
				if len(args) != 1 {
					return fmt.Errorf("expected <channel-id> when using --ids")
				}
				channelID = args[0]
			} else {
				if len(args) != 2 {
					return fmt.Errorf("expected <screen-id> <channel-id>")
				}
				screenID = args[0]
				channelID = args[1]
			}

			chID, err := strconv.Atoi(channelID)
			if err != nil {
				return fmt.Errorf("channel-id must be an integer")
			}

			body := map[string]any{"screen": map[string]any{"channel_id": chID}}

			if ids != "" {
				q := url.Values{"ids": {ids}}
				var resp map[string]any
				if err := a.Client.Patch("/screens/bulk?"+q.Encode(), body, &resp); err != nil {
					return err
				}
				if a.JSONOutput {
					printJSON(resp)
				} else {
					fmt.Printf("Switched screens %s to channel %s\n", ids, channelID)
				}
			} else {
				var resp map[string]any
				if err := a.Client.Patch("/screens/"+screenID, body, &resp); err != nil {
					return err
				}
				if a.JSONOutput {
					printJSON(resp)
				} else {
					fmt.Printf("Switched screen %s to channel %s\n", screenID, channelID)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&ids, "ids", "", "Comma-separated screen IDs for bulk operation")
	return cmd
}

func newScreensTempChannelCmd() *cobra.Command {
	var duration int
	var until string
	var toggle bool
	var ids string

	cmd := &cobra.Command{
		Use:   "temp-channel <screen-id> <channel-id>",
		Short: "Set a temporary channel override on a screen",
		Example: `  pintomind screens temp-channel 42 7 --duration 3600
  pintomind screens temp-channel 42 7 --until 2025-12-31T23:59:00Z
  pintomind screens temp-channel 42 7 --toggle`,
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)

			var screenID, channelID string
			if ids != "" {
				if len(args) != 1 {
					return fmt.Errorf("expected <channel-id> when using --ids")
				}
				channelID = args[0]
			} else {
				if len(args) != 2 {
					return fmt.Errorf("expected <screen-id> <channel-id>")
				}
				screenID = args[0]
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

			body := map[string]any{"screen": screen}

			if ids != "" {
				q := url.Values{"ids": {ids}}
				var resp map[string]any
				if err := a.Client.Patch("/screens/bulk?"+q.Encode(), body, &resp); err != nil {
					return err
				}
				if a.JSONOutput {
					printJSON(resp)
				} else {
					fmt.Printf("Set temporary channel %s on screens %s\n", channelID, ids)
				}
			} else {
				var resp map[string]any
				if err := a.Client.Patch("/screens/"+screenID, body, &resp); err != nil {
					return err
				}
				if a.JSONOutput {
					printJSON(resp)
				} else {
					fmt.Printf("Set temporary channel %s on screen %s\n", channelID, screenID)
				}
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&duration, "duration", 0, "Duration in seconds")
	cmd.Flags().StringVar(&until, "until", "", "ISO8601 timestamp until which override is active")
	cmd.Flags().BoolVar(&toggle, "toggle", false, "Toggle temporary channel active state")
	cmd.Flags().StringVar(&ids, "ids", "", "Comma-separated screen IDs for bulk operation")
	return cmd
}
