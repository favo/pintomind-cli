package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// parseJSONFlag decodes a flag holding JSON, @path to a JSON file, or - for stdin.
func parseJSONFlag(flag, value string) (any, error) {
	var data []byte
	var err error
	switch {
	case value == "-":
		data, err = io.ReadAll(os.Stdin)
	case strings.HasPrefix(value, "@"):
		data, err = os.ReadFile(value[1:])
	default:
		data = []byte(value)
	}
	if err != nil {
		return nil, fmt.Errorf("--%s: %w", flag, err)
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("invalid JSON for --%s: %w", flag, err)
	}
	return out, nil
}

type ScheduleBlock struct {
	Type          string  `json:"type"`
	Wday          *int    `json:"wday"`
	ChannelID     *int    `json:"channel_id"`
	FromTime      string  `json:"from_time"`
	ToTime        string  `json:"to_time"`
	Hour          *int    `json:"hour"`
	DeviceOff     bool    `json:"device_off"`
	AnimationType *string `json:"animation_type"`
}

type ChannelSchedule struct {
	Mode     string `json:"mode"`
	TimeZone string `json:"time_zone"`
	Template *struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"template"`
	DayPlan      []ScheduleBlock `json:"day_plan"`
	WeekPlan     []ScheduleBlock `json:"week_plan"`
	NextChangeAt *string         `json:"next_change_at"`
	NextRebootAt *string         `json:"next_reboot_at"`
}

const planHelp = `A plan is a JSON array of blocks, given inline, as @file.json or - for stdin:
  {"type":"channel","channel_id":12,"from_time":"08:00","to_time":"16:00"}
  {"type":"night","from_time":"22:00","to_time":"06:00","device_off":false,"animation_type":"sleepy"}
  {"type":"reboot","hour":4}
Week plan blocks also have "wday" (0 = Sunday … 6 = Saturday). A plan replaces the whole plan;
gaps show the screen's standard channel. Times are in the time zone of the standard channel
(templates: each screen's own).`

func newScreensScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Show or change a screen's channel schedule (day/week plan, night mode, reboot)",
	}
	cmd.AddCommand(newScheduleShowCmd(), newScheduleSetCmd(), newScheduleClearCmd(), newScheduleTemplatesCmd())
	return cmd
}

func schedulePath(screenID string) string {
	return "/screens/" + screenID + "/channel_schedule"
}

func newScheduleShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <screen-id>",
		Short: "Show a screen's channel schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if a.JSONOutput {
				return getJSON(cmd, schedulePath(args[0]), nil)
			}
			var resp struct {
				Schedule ChannelSchedule `json:"channel_schedule"`
			}
			if err := a.Client.Get(schedulePath(args[0]), nil, &resp); err != nil {
				return err
			}
			s := resp.Schedule
			fmt.Printf("Mode:        %s\n", s.Mode)
			fmt.Printf("Time zone:   %s\n", s.TimeZone)
			if s.Template != nil {
				fmt.Printf("Template:    %s (%d)\n", s.Template.Name, s.Template.ID)
			}
			if s.NextChangeAt != nil {
				fmt.Printf("Next change: %s\n", *s.NextChangeAt)
			}
			if s.NextRebootAt != nil {
				fmt.Printf("Next reboot: %s\n", *s.NextRebootAt)
			}
			printPlan(cmd, "Day plan", s.DayPlan, false)
			printPlan(cmd, "Week plan", s.WeekPlan, true)
			return nil
		},
	}
}

var weekdays = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

func printPlan(cmd *cobra.Command, title string, plan []ScheduleBlock, week bool) {
	fmt.Printf("\n%s:\n", title)
	if len(plan) == 0 {
		fmt.Println("  (empty: the standard channel all day)")
		return
	}
	headers := []string{"TYPE", "FROM", "TO", "DETAILS"}
	if week {
		headers = append([]string{"DAY"}, headers...)
	}
	rows := make([][]string, len(plan))
	for i, b := range plan {
		var details string
		switch b.Type {
		case "channel":
			if b.ChannelID != nil {
				details = "channel " + strconv.Itoa(*b.ChannelID)
			}
		case "night":
			var parts []string
			if b.DeviceOff {
				parts = append(parts, "display off")
			}
			if b.AnimationType != nil {
				parts = append(parts, "animation "+*b.AnimationType)
			}
			details = strings.Join(parts, ", ")
		case "reboot":
			if b.Hour != nil {
				details = fmt.Sprintf("in hour %02d", *b.Hour)
			}
		}
		row := []string{b.Type, b.FromTime, b.ToTime, details}
		if week {
			day := ""
			if b.Wday != nil && *b.Wday >= 0 && *b.Wday < len(weekdays) {
				day = weekdays[*b.Wday]
			}
			row = append([]string{day}, row...)
		}
		rows[i] = row
	}
	printTable(cmd, headers, rows)
}

// planBody collects --mode, --day-plan and --week-plan into a request body.
func planBody(cmd *cobra.Command, mode, dayPlan, weekPlan string) (map[string]any, error) {
	body := map[string]any{}
	setIfChanged(cmd, body, "mode", "mode", mode)
	for flag, value := range map[string]string{"day-plan": dayPlan, "week-plan": weekPlan} {
		if !cmd.Flags().Changed(flag) {
			continue
		}
		plan, err := parseJSONFlag(flag, value)
		if err != nil {
			return nil, err
		}
		if _, ok := plan.([]any); !ok {
			return nil, fmt.Errorf("--%s must be a JSON array of blocks", flag)
		}
		body[strings.ReplaceAll(flag, "-", "_")] = plan
	}
	return body, nil
}

func addPlanFlags(cmd *cobra.Command, mode, dayPlan, weekPlan *string) {
	cmd.Flags().StringVar(mode, "mode", "", "daily (uses the day plan) or weekly (uses the week plan)")
	cmd.Flags().StringVar(dayPlan, "day-plan", "", "Day plan blocks as JSON, @file or -")
	cmd.Flags().StringVar(weekPlan, "week-plan", "", "Week plan blocks as JSON, @file or -")
}

func newScheduleSetCmd() *cobra.Command {
	var mode, dayPlan, weekPlan string
	var templateID int

	cmd := &cobra.Command{
		Use:   "set <screen-id>",
		Short: "Change a screen's channel schedule, or apply a template",
		Long:  "Omitted fields are left unchanged. --template-id applies a template and links the screen to it.\n\n" + planHelp,
		Example: `  pintomind screens schedule set 42 --mode daily --day-plan '[{"type":"channel","channel_id":12,"from_time":"08:00","to_time":"16:00"},{"type":"reboot","hour":4}]'
  pintomind screens schedule set 42 --mode weekly --week-plan @week.json
  pintomind screens schedule set 42 --template-id 3`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := planBody(cmd, mode, dayPlan, weekPlan)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("template-id") {
				if len(body) > 0 {
					return errors.New("--template-id can't be combined with --mode or a plan")
				}
				body["template_id"] = templateID
			}
			if len(body) == 0 {
				return errors.New("provide --mode, --day-plan, --week-plan or --template-id")
			}
			return sendJSON(cmd, "PATCH", schedulePath(args[0]), map[string]any{"channel_schedule": body})
		},
	}
	addPlanFlags(cmd, &mode, &dayPlan, &weekPlan)
	cmd.Flags().IntVar(&templateID, "template-id", 0, "Apply this channel schedule template")
	return cmd
}

func newScheduleClearCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "clear <screen-id>",
		Short: "Remove a screen's channel schedule (standard channel all day, no night mode or reboot)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmAction(fmt.Sprintf("Remove the channel schedule of screen %s?", args[0]), force) {
				return nil
			}
			return sendJSON(cmd, "DELETE", schedulePath(args[0]), nil)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

type ScheduleTemplate struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Mode      string `json:"mode"`
	ScreenIDs []int  `json:"screen_ids"`
	UpdatedAt string `json:"updated_at"`
}

func newScheduleTemplatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Manage channel schedule templates that screens can share",
	}

	var sortBy, fields string
	list := &cobra.Command{
		Use:   "list",
		Short: "List channel schedule templates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			setStringIf(q, "sort_by", sortBy)
			setStringIf(q, "fields", fields)
			applyPagination(cmd, q)
			if a.JSONOutput || fields != "" {
				return getJSON(cmd, "/channel_schedule_templates", q)
			}
			var resp struct {
				Total      int                `json:"total"`
				Pagination *Pagination        `json:"pagination"`
				Items      []ScheduleTemplate `json:"items"`
			}
			if err := a.Client.Get("/channel_schedule_templates", q, &resp); err != nil {
				return err
			}
			printTotal(resp.Total, resp.Pagination)
			rows := make([][]string, len(resp.Items))
			for i, t := range resp.Items {
				screens := make([]string, len(t.ScreenIDs))
				for j, id := range t.ScreenIDs {
					screens[j] = strconv.Itoa(id)
				}
				rows[i] = []string{strconv.Itoa(t.ID), t.Name, t.Mode, strings.Join(screens, ","), t.UpdatedAt}
			}
			printTable(cmd, []string{"ID", "NAME", "MODE", "SCREENS", "UPDATED AT"}, rows)
			return nil
		},
	}
	list.Flags().StringVar(&sortBy, "sort-by", "", "Sort field, e.g. name or updated_at:desc")
	addFieldsFlag(list, &fields)
	addPaginationFlags(list, 200)

	show := &cobra.Command{
		Use:   "show <id>",
		Short: "Show a channel schedule template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getJSON(cmd, "/channel_schedule_templates/"+args[0], nil)
		},
	}

	cmd.AddCommand(list, show, newScheduleTemplateWriteCmd(true), newScheduleTemplateWriteCmd(false))

	var force bool
	del := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a template (linked screens keep their schedules)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmAction(fmt.Sprintf("Delete channel schedule template %s?", args[0]), force) {
				return nil
			}
			return runDelete(cmd, "/channel_schedule_templates/"+args[0], "template "+args[0])
		},
	}
	del.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	cmd.AddCommand(del)
	return cmd
}

func newScheduleTemplateWriteCmd(create bool) *cobra.Command {
	var name, mode, dayPlan, weekPlan string
	use, short, nargs := "update <id>", "Change a template (linked screens get the new schedule)", 1
	if create {
		use, short, nargs = "create --name <name>", "Create a channel schedule template", 0
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  short + ".\n\n" + planHelp,
		Args:  cobra.ExactArgs(nargs),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := planBody(cmd, mode, dayPlan, weekPlan)
			if err != nil {
				return err
			}
			setIfChanged(cmd, body, "name", "name", name)
			if len(body) == 0 {
				return errors.New("provide at least one field")
			}
			wrapped := map[string]any{"channel_schedule_template": body}
			if create {
				return sendJSON(cmd, "POST", "/channel_schedule_templates", wrapped)
			}
			return sendJSON(cmd, "PATCH", "/channel_schedule_templates/"+args[0], wrapped)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Template name, unique per account")
	addPlanFlags(cmd, &mode, &dayPlan, &weekPlan)
	if create {
		_ = cmd.MarkFlagRequired("name")
	}
	return cmd
}
