package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// ScheduleBlock is one block in a channel schedule plan.
// type "channel" uses channel_id/from_time/to_time, "night" uses from_time/to_time/device_off/animation_type,
// "reboot" uses hour. Blocks in a week plan also have wday (0 = Sunday).
type ScheduleBlock struct {
	Type          string  `json:"type"`
	Wday          *int    `json:"wday,omitempty"`
	ChannelID     *int    `json:"channel_id,omitempty"`
	FromTime      string  `json:"from_time,omitempty"`
	ToTime        string  `json:"to_time,omitempty"`
	Hour          *int    `json:"hour,omitempty"`
	DeviceOff     *bool   `json:"device_off,omitempty"`
	AnimationType *string `json:"animation_type,omitempty"`
}

type ChannelSchedule struct {
	Mode         string          `json:"mode"`
	TimeZone     string          `json:"time_zone"`
	Template     *IDName         `json:"template"`
	DayPlan      []ScheduleBlock `json:"day_plan"`
	WeekPlan     []ScheduleBlock `json:"week_plan"`
	NextChangeAt *string         `json:"next_change_at"`
	NextRebootAt *string         `json:"next_reboot_at"`
}

type IDName struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ChannelScheduleResponse struct {
	Success         bool            `json:"success"`
	ChannelSchedule ChannelSchedule `json:"channel_schedule"`
}

type ScheduleTemplate struct {
	ID        int             `json:"id"`
	Name      string          `json:"name"`
	Mode      string          `json:"mode"`
	DayPlan   []ScheduleBlock `json:"day_plan"`
	WeekPlan  []ScheduleBlock `json:"week_plan"`
	ScreenIDs []int           `json:"screen_ids"`
}

type ScheduleTemplatesResponse struct {
	Total int                `json:"total"`
	Items []ScheduleTemplate `json:"items"`
}

var weekdayNames = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

func newScreensScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schedule",
		Aliases: []string{"channel-schedule"},
		Short:   "Show and change a screen's channel schedule (channel by time of day, night mode, reboot)",
		Long: `A screen has a day plan (used every day) and a week plan (blocks per weekday). The mode picks the
active one; both are kept. Time not covered by a block shows the screen's standard channel.

Blocks:
  {"type":"channel","channel_id":12,"from_time":"07:00","to_time":"09:30"}
  {"type":"night","from_time":"23:00","to_time":"06:00","device_off":false,"animation_type":"sleepy"}
  {"type":"reboot","hour":2}
Week plan blocks also have "wday" (0 = Sunday). A to_time before the from_time runs past midnight.
Times are in the time zone of the screen's standard channel.`,
	}
	cmd.AddCommand(newScreensScheduleShowCmd())
	cmd.AddCommand(newScreensScheduleSetCmd())
	cmd.AddCommand(newScreensScheduleAddCmd())
	cmd.AddCommand(newScreensScheduleApplyTemplateCmd())
	cmd.AddCommand(newScreensScheduleClearCmd())
	return cmd
}

func schedulePath(screenID string) string {
	return "/screens/" + screenID + "/channel_schedule"
}

func newScreensScheduleShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <screen-id>",
		Short: "Show a screen's channel schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp ChannelScheduleResponse
			if err := a.Client.Get(schedulePath(args[0]), nil, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}
			printSchedule(cmd, resp.ChannelSchedule)
			return nil
		},
	}
}

func newScreensScheduleSetCmd() *cobra.Command {
	var mode, data, file string

	cmd := &cobra.Command{
		Use:   "set [screen-id] [--mode daily|weekly] [--data '<json>' | --file <path>]",
		Short: "Change a screen's channel schedule; omitted fields are left unchanged",
		Long: `--data/--file take the channel_schedule object: mode, day_plan, week_plan. day_plan and week_plan
replace the whole plan. The output of "schedule show --json" (its channel_schedule) can be sent back.
Use --file - to read from stdin.`,
		Example: `  pintomind screens schedule set 42 --mode weekly
  pintomind screens schedule set 42 --data '{"day_plan":[{"type":"channel","channel_id":12,"from_time":"07:00","to_time":"09:30"},{"type":"night","from_time":"23:00","to_time":"06:00"},{"type":"reboot","hour":2}]}'
  pintomind screens schedule set --ids 1,2,3 --file schedule.json
  pintomind screens schedule show 42 --json | jq .channel_schedule | pintomind screens schedule set 43 --file -`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			schedule, err := readJSONObject(data, file)
			if err != nil {
				return err
			}
			if schedule == nil {
				schedule = map[string]any{}
			}
			// The output of "show --json" also has read-only fields; send only the writable ones
			for key := range schedule {
				if key != "mode" && key != "day_plan" && key != "week_plan" && key != "template_id" {
					delete(schedule, key)
				}
			}
			if mode != "" {
				schedule["mode"] = mode
			}
			if len(schedule) == 0 {
				return fmt.Errorf("nothing to change: pass --mode, --data or --file")
			}

			ids, err := scheduleTargets(cmd, args)
			if err != nil {
				return err
			}
			return putSchedules(cmd, ids, map[string]any{"channel_schedule": schedule}, "Updated the channel schedule")
		},
	}
	addTargetFlags(cmd)
	cmd.Flags().StringVar(&mode, "mode", "", "Active plan: daily or weekly")
	cmd.Flags().StringVar(&data, "data", "", "channel_schedule fields as JSON")
	cmd.Flags().StringVar(&file, "file", "", "Read the channel_schedule JSON from a file (- for stdin)")
	return cmd
}

func newScreensScheduleAddCmd() *cobra.Command {
	var channelID, wday, rebootHour int
	var from, to, animation string
	var night, deviceOff bool

	cmd := &cobra.Command{
		Use:   "add <screen-id> (--channel <id> | --night | --reboot <hour>) [--from HH:MM --to HH:MM] [--wday 0-6]",
		Short: "Add one block to a screen's schedule (day plan, or week plan with --wday)",
		Long: `Adds a block to the day plan, or to the week plan when --wday is given. The server rejects a block
that overlaps another one; remove or change that one first with "schedule set".`,
		Example: `  pintomind screens schedule add 42 --channel 12 --from 07:00 --to 09:30
  pintomind screens schedule add 42 --night --from 23:00 --to 06:00 --animation sleepy
  pintomind screens schedule add 42 --reboot 2
  pintomind screens schedule add 42 --channel 14 --wday 5 --from 15:00 --to 17:00`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)

			block := ScheduleBlock{}
			kinds := 0
			if cmd.Flags().Changed("channel") {
				kinds++
				block.Type = "channel"
				block.ChannelID = &channelID
			}
			if night {
				kinds++
				block.Type = "night"
				block.DeviceOff = &deviceOff
				if animation != "" {
					block.AnimationType = &animation
				}
			}
			if cmd.Flags().Changed("reboot") {
				kinds++
				block.Type = "reboot"
				block.Hour = &rebootHour
			}
			if kinds != 1 {
				return fmt.Errorf("pass exactly one of --channel, --night or --reboot")
			}
			if block.Type != "reboot" {
				if from == "" || to == "" {
					return fmt.Errorf("--from and --to are required for a %s block", block.Type)
				}
				block.FromTime, block.ToTime = from, to
			}

			plan := "day_plan"
			if cmd.Flags().Changed("wday") {
				plan = "week_plan"
				block.Wday = &wday
			}

			var current ChannelScheduleResponse
			if err := a.Client.Get(schedulePath(args[0]), nil, &current); err != nil {
				return err
			}
			blocks := current.ChannelSchedule.DayPlan
			if plan == "week_plan" {
				blocks = current.ChannelSchedule.WeekPlan
			}
			blocks = append(blocks, block)

			return putSchedules(cmd, []string{args[0]}, map[string]any{"channel_schedule": map[string]any{plan: blocks}},
				fmt.Sprintf("Added a %s block to the %s", block.Type, strings.Replace(plan, "_", " ", 1)))
		},
	}
	cmd.Flags().IntVar(&channelID, "channel", 0, "Channel ID to show")
	cmd.Flags().BoolVar(&night, "night", false, "Night mode block")
	cmd.Flags().IntVar(&rebootHour, "reboot", 0, "Reboot within this hour (0-23)")
	cmd.Flags().StringVar(&from, "from", "", "Start time HH:MM")
	cmd.Flags().StringVar(&to, "to", "", "End time HH:MM (24:00 for end of day; before --from runs past midnight)")
	cmd.Flags().IntVar(&wday, "wday", 0, "Weekday for the week plan, 0 = Sunday")
	cmd.Flags().BoolVar(&deviceOff, "device-off", false, "Night mode: turn the display off (PieOS screens)")
	cmd.Flags().StringVar(&animation, "animation", "", "Night mode animation: sleepy or sleepy_rooster")
	return cmd
}

func newScreensScheduleApplyTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply-template [screen-id] <template-id>",
		Short: "Give a screen (or several) a schedule template and link them to it",
		Example: `  pintomind screens schedule apply-template 42 4
  pintomind screens schedule apply-template --ids 1,2,3 4`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, _ := cmd.Flags().GetString("ids")
			all, _ := cmd.Flags().GetBool("all")

			screenArgs := args
			if ids != "" || all {
				if len(args) != 1 {
					return fmt.Errorf("expected <template-id> when using --ids or --all")
				}
				screenArgs = nil
			} else if len(args) != 2 {
				return fmt.Errorf("expected <screen-id> <template-id>")
			} else {
				screenArgs = args[:1]
			}

			templateID, err := strconv.Atoi(args[len(args)-1])
			if err != nil {
				return fmt.Errorf("template-id must be an integer")
			}

			targets, err := scheduleTargets(cmd, screenArgs)
			if err != nil {
				return err
			}
			body := map[string]any{"channel_schedule": map[string]any{"template_id": templateID}}
			return putSchedules(cmd, targets, body, fmt.Sprintf("Applied template %d", templateID))
		},
	}
	addTargetFlags(cmd)
	return cmd
}

func newScreensScheduleClearCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear [screen-id]",
		Short: "Remove a screen's schedule: standard channel all day, no night mode or reboot",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			targets, err := scheduleTargets(cmd, args)
			if err != nil {
				return err
			}
			for _, id := range targets {
				if err := a.Client.Delete(schedulePath(id)); err != nil {
					return fmt.Errorf("screen %s: %w", id, err)
				}
				fmt.Printf("Removed the channel schedule on screen %s\n", id)
			}
			return nil
		},
	}
	addTargetFlags(cmd)
	return cmd
}

// scheduleTargets resolves [screen-id] / --ids / --all to a list of screen IDs.
// The schedule endpoints take one screen at a time, so bulk targets are sent one by one.
func scheduleTargets(cmd *cobra.Command, args []string) ([]string, error) {
	ids, _ := cmd.Flags().GetString("ids")
	all, _ := cmd.Flags().GetBool("all")
	single := ""
	if len(args) > 0 {
		single = args[0]
	}
	targets, _, err := resolveScreenIDs(cmd, single, ids, all)
	if err != nil {
		return nil, err
	}
	return strings.Split(targets, ","), nil
}

func putSchedules(cmd *cobra.Command, screenIDs []string, body map[string]any, successMsg string) error {
	a := app(cmd)
	for _, id := range screenIDs {
		var resp ChannelScheduleResponse
		if err := a.Client.Put(schedulePath(strings.TrimSpace(id)), body, &resp); err != nil {
			return fmt.Errorf("screen %s: %w", id, err)
		}
		if a.JSONOutput {
			printJSON(resp)
		} else if len(screenIDs) == 1 {
			fmt.Println(successMsg)
			fmt.Println()
			printSchedule(cmd, resp.ChannelSchedule)
		} else {
			fmt.Printf("%s on screen %s\n", successMsg, id)
		}
	}
	return nil
}

func printSchedule(cmd *cobra.Command, s ChannelSchedule) {
	active := "day plan"
	if s.Mode == "weekly" {
		active = "week plan"
	}
	template := "-"
	if s.Template != nil {
		template = fmt.Sprintf("%s (%d)", s.Template.Name, s.Template.ID)
	}
	fmt.Printf("Mode:        %s (%s active)\n", s.Mode, active)
	fmt.Printf("Time zone:   %s\n", s.TimeZone)
	fmt.Printf("Template:    %s\n", template)
	fmt.Printf("Next change: %s\n", stringOrDash(s.NextChangeAt))
	fmt.Printf("Next reboot: %s\n", stringOrDash(s.NextRebootAt))

	printPlan(cmd, "Day plan", s.DayPlan)
	printPlan(cmd, "Week plan", s.WeekPlan)
}

func printPlan(cmd *cobra.Command, title string, blocks []ScheduleBlock) {
	fmt.Printf("\n%s\n", title)
	if len(blocks) == 0 {
		fmt.Println("  (empty — the standard channel all day)")
		return
	}
	rows := make([][]string, len(blocks))
	for i, b := range blocks {
		day := "-"
		if b.Wday != nil && *b.Wday >= 0 && *b.Wday < len(weekdayNames) {
			day = weekdayNames[*b.Wday]
		}
		rows[i] = append([]string{day}, blockColumns(b)...)
	}
	printTable(cmd, []string{"DAY", "TYPE", "FROM", "TO", "DETAILS"}, rows)
}

func blockColumns(b ScheduleBlock) []string {
	switch b.Type {
	case "reboot":
		hour := 0
		if b.Hour != nil {
			hour = *b.Hour
		}
		return []string{"reboot", fmt.Sprintf("%02d:00", hour), fmt.Sprintf("%02d:00", hour+1), "random minute in the hour"}
	case "night":
		details := []string{}
		if b.AnimationType != nil && *b.AnimationType != "" {
			details = append(details, "animation "+*b.AnimationType)
		}
		if b.DeviceOff != nil && *b.DeviceOff {
			details = append(details, "display off")
		}
		return []string{"night", b.FromTime, b.ToTime, strings.Join(details, ", ")}
	default:
		channel := "-"
		if b.ChannelID != nil {
			channel = "channel " + strconv.Itoa(*b.ChannelID)
		}
		return []string{b.Type, b.FromTime, b.ToTime, channel}
	}
}

func stringOrDash(s *string) string {
	if s == nil || *s == "" {
		return "-"
	}
	return *s
}

// readJSONObject reads a JSON object from --data or --file (- for stdin). Returns nil when neither is set.
func readJSONObject(data, file string) (map[string]any, error) {
	if data != "" && file != "" {
		return nil, fmt.Errorf("use --data or --file, not both")
	}

	var raw []byte
	switch {
	case data != "":
		raw = []byte(data)
	case file == "-":
		var err error
		if raw, err = io.ReadAll(os.Stdin); err != nil {
			return nil, err
		}
	case file != "":
		var err error
		if raw, err = os.ReadFile(file); err != nil {
			return nil, err
		}
	default:
		return nil, nil
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return out, nil
}

// --- templates ---------------------------------------------------------------

func NewScheduleTemplatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schedule-templates",
		Aliases: []string{"channel-schedule-templates"},
		Short:   "Manage channel schedule templates (saved schedules screens can use)",
		Long: `A template is a saved channel schedule. Give it to a screen with
"pintomind screens schedule apply-template". Changing a template's plans or mode changes every linked
screen. Deleting a template keeps the screens' schedules. Needs an account API key.`,
	}
	cmd.AddCommand(newScheduleTemplatesListCmd())
	cmd.AddCommand(newScheduleTemplatesShowCmd())
	cmd.AddCommand(newScheduleTemplatesCreateCmd())
	cmd.AddCommand(newScheduleTemplatesUpdateCmd())
	cmd.AddCommand(newScheduleTemplatesDeleteCmd())
	return cmd
}

func newScheduleTemplatesListCmd() *cobra.Command {
	var sortBy string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List schedule templates",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			if sortBy != "" {
				q.Set("sort_by", sortBy)
			}
			applyPagination(cmd, q)

			var resp ScheduleTemplatesResponse
			if err := a.Client.Get("/channel_schedule_templates", q, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, t := range resp.Items {
				screens := make([]string, len(t.ScreenIDs))
				for j, id := range t.ScreenIDs {
					screens[j] = strconv.Itoa(id)
				}
				rows[i] = []string{strconv.Itoa(t.ID), t.Name, t.Mode, strconv.Itoa(len(t.DayPlan)), strconv.Itoa(len(t.WeekPlan)), strings.Join(screens, ",")}
			}
			printTable(cmd, []string{"ID", "NAME", "MODE", "DAY BLOCKS", "WEEK BLOCKS", "SCREENS"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field: name, created_at, updated_at (e.g. name:asc)")
	addPaginationFlags(cmd)
	return cmd
}

func newScheduleTemplatesShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a schedule template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp struct {
				Template ScheduleTemplate `json:"channel_schedule_template"`
			}
			if err := a.Client.Get("/channel_schedule_templates/"+args[0], nil, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}
			t := resp.Template
			fmt.Printf("Template: %s (%d)\n", t.Name, t.ID)
			fmt.Printf("Mode:     %s\n", t.Mode)
			fmt.Printf("Screens:  %v\n", t.ScreenIDs)
			printPlan(cmd, "Day plan", t.DayPlan)
			printPlan(cmd, "Week plan", t.WeekPlan)
			return nil
		},
	}
}

func templateBody(name, mode, data, file string) (map[string]any, error) {
	template, err := readJSONObject(data, file)
	if err != nil {
		return nil, err
	}
	if template == nil {
		template = map[string]any{}
	}
	for key := range template {
		if key != "name" && key != "mode" && key != "day_plan" && key != "week_plan" {
			delete(template, key)
		}
	}
	if name != "" {
		template["name"] = name
	}
	if mode != "" {
		template["mode"] = mode
	}
	return map[string]any{"channel_schedule_template": template}, nil
}

func newScheduleTemplatesCreateCmd() *cobra.Command {
	var name, mode, data, file string

	cmd := &cobra.Command{
		Use:   "create --name <name> (--data '<json>' | --file <path>)",
		Short: "Create a schedule template",
		Long:  `--data/--file take day_plan and week_plan (and optionally mode). A screen's schedule can be copied: "screens schedule show 42 --json | jq .channel_schedule".`,
		Example: `  pintomind schedule-templates create --name Kantine --data '{"day_plan":[{"type":"channel","channel_id":12,"from_time":"10:30","to_time":"13:00"},{"type":"reboot","hour":2}]}'
  pintomind screens schedule show 42 --json | jq .channel_schedule | pintomind schedule-templates create --name "Like 42" --file -`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			body, err := templateBody(name, mode, data, file)
			if err != nil {
				return err
			}
			var resp map[string]any
			if err := a.Client.Post("/channel_schedule_templates", body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Template name, unique per account (required)")
	cmd.Flags().StringVar(&mode, "mode", "", "daily (default) or weekly")
	cmd.Flags().StringVar(&data, "data", "", "Template fields as JSON")
	cmd.Flags().StringVar(&file, "file", "", "Read the template JSON from a file (- for stdin)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newScheduleTemplatesUpdateCmd() *cobra.Command {
	var name, mode, data, file string

	cmd := &cobra.Command{
		Use:   "update <id> [--name <name>] [--mode daily|weekly] [--data '<json>' | --file <path>]",
		Short: "Change a schedule template; every linked screen gets the new schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			body, err := templateBody(name, mode, data, file)
			if err != nil {
				return err
			}
			if len(body["channel_schedule_template"].(map[string]any)) == 0 {
				return fmt.Errorf("nothing to change: pass --name, --mode, --data or --file")
			}
			var resp map[string]any
			if err := a.Client.Patch("/channel_schedule_templates/"+args[0], body, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New name")
	cmd.Flags().StringVar(&mode, "mode", "", "daily or weekly")
	cmd.Flags().StringVar(&data, "data", "", "Template fields as JSON")
	cmd.Flags().StringVar(&file, "file", "", "Read the template JSON from a file (- for stdin)")
	return cmd
}

func newScheduleTemplatesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a schedule template (linked screens keep their schedules)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if err := a.Client.Delete("/channel_schedule_templates/" + args[0]); err != nil {
				return err
			}
			fmt.Printf("Deleted schedule template %s\n", args[0])
			return nil
		},
	}
}
