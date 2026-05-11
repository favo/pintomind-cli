package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type TimeSchedule struct {
	ID                   int              `json:"id"`
	RecurringRules       []map[string]any `json:"recurring_rules"`
	TimePeriods          []map[string]any `json:"time_periods"`
	ScheduleExpiryAction string           `json:"schedule_expiry_action"`
	PostState            string           `json:"post_state"`
}

func newPostsScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage a post's time schedule (when the post is visible)",
	}
	cmd.AddCommand(newPostsScheduleGetCmd())
	cmd.AddCommand(newPostsScheduleSetCmd())
	cmd.AddCommand(newPostsScheduleClearCmd())
	return cmd
}

func newPostsScheduleGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <post-id>",
		Short: "Show the post's current time schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/posts/"+args[0]+"/time_schedule", nil, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}
			ts, _ := resp["time_schedule"].(map[string]any)
			if ts == nil {
				printJSON(resp)
				return nil
			}
			fmt.Printf("Schedule ID:     %v\n", ts["id"])
			fmt.Printf("Post state:      %v\n", ts["post_state"])
			fmt.Printf("On expiry:       %v\n\n", ts["schedule_expiry_action"])

			if rules, _ := ts["recurring_rules"].([]any); len(rules) > 0 {
				fmt.Println("Recurring rules:")
				rows := make([][]string, len(rules))
				for i, r := range rules {
					m, _ := r.(map[string]any)
					rows[i] = []string{
						fmt.Sprintf("%v", m["from_wday"]),
						fmt.Sprintf("%v", m["to_wday"]),
						fmt.Sprintf("%v", m["from_time"]),
						fmt.Sprintf("%v", m["to_time"]),
						fmt.Sprintf("%v", m["weeks"]),
					}
				}
				printTable(cmd, []string{"FROM_WDAY", "TO_WDAY", "FROM_TIME", "TO_TIME", "WEEKS"}, rows)
				fmt.Println()
			} else {
				fmt.Println("Recurring rules: (none)")
			}

			if periods, _ := ts["time_periods"].([]any); len(periods) > 0 {
				fmt.Println("Time periods:")
				rows := make([][]string, len(periods))
				for i, p := range periods {
					m, _ := p.(map[string]any)
					rows[i] = []string{
						fmt.Sprintf("%v", m["from_date"]),
						fmt.Sprintf("%v", m["to_date"]),
						fmt.Sprintf("%v", m["from_time"]),
						fmt.Sprintf("%v", m["to_time"]),
					}
				}
				printTable(cmd, []string{"FROM_DATE", "TO_DATE", "FROM_TIME", "TO_TIME"}, rows)
			} else {
				fmt.Println("Time periods: (none)")
			}
			return nil
		},
	}
}

func newPostsScheduleSetCmd() *cobra.Command {
	var (
		periods       []string
		rules         []string
		expiryAction  string
		clearPeriods  bool
		clearRules    bool
	)
	cmd := &cobra.Command{
		Use:   "set <post-id>",
		Short: "Set the post's time schedule (PUT /time_schedule, partial update)",
		Example: `  pintomind posts schedule set 123 \
    --recurring-rule 'from_wday=1,to_wday=5,from_time=08:00,to_time=18:00' \
    --period 'from_date=2026-06-01,to_date=2026-06-30,from_time=09:00,to_time=17:00' \
    --expiry-action archive`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			ts := map[string]any{}

			if clearPeriods {
				if len(periods) > 0 {
					return fmt.Errorf("cannot combine --clear-periods with --period")
				}
				ts["time_periods"] = []map[string]any{}
			} else if len(periods) > 0 {
				parsed := make([]map[string]any, len(periods))
				for i, p := range periods {
					m, err := parsePeriodFlag(p)
					if err != nil {
						return err
					}
					parsed[i] = m
				}
				ts["time_periods"] = parsed
			}

			if clearRules {
				if len(rules) > 0 {
					return fmt.Errorf("cannot combine --clear-rules with --recurring-rule")
				}
				ts["recurring_rules"] = []map[string]any{}
			} else if len(rules) > 0 {
				parsed := make([]map[string]any, len(rules))
				for i, r := range rules {
					m, err := parseRecurringRuleFlag(r)
					if err != nil {
						return err
					}
					parsed[i] = m
				}
				ts["recurring_rules"] = parsed
			}

			if expiryAction != "" {
				if !validExpiryActions[expiryAction] {
					return fmt.Errorf("--expiry-action must be one of delete, archive, unpublish")
				}
				ts["schedule_expiry_action"] = expiryAction
			}

			if len(ts) == 0 {
				return fmt.Errorf("no schedule changes provided; pass --period, --recurring-rule, --expiry-action, --clear-periods, or --clear-rules")
			}

			var resp map[string]any
			if err := a.Client.Put("/posts/"+args[0]+"/time_schedule", map[string]any{"time_schedule": ts}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&periods, "period", nil,
		"Date-range window, key=val,key=val. Required: from_date, to_date. Optional: from_time, to_time. Repeatable.")
	cmd.Flags().StringArrayVar(&rules, "recurring-rule", nil,
		"Weekly window, key=val,key=val. Required: from_wday (0-6), to_wday, from_time (HH:MM), to_time. Optional: weeks (all|odd|even). Repeatable.")
	cmd.Flags().StringVar(&expiryAction, "expiry-action", "",
		"Action when schedule fully expires: delete, archive, unpublish")
	cmd.Flags().BoolVar(&clearPeriods, "clear-periods", false, "Replace all time_periods with an empty list")
	cmd.Flags().BoolVar(&clearRules, "clear-rules", false, "Replace all recurring_rules with an empty list")
	return cmd
}

func newPostsScheduleClearCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "clear <post-id>",
		Short: "Delete the post's time schedule (post becomes always-visible)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if !force {
				fmt.Printf("Clear time schedule for post %s? Type 'yes' to confirm: ", args[0])
				var confirm string
				fmt.Scanln(&confirm)
				if strings.TrimSpace(confirm) != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if err := a.Client.Delete("/posts/" + args[0] + "/time_schedule"); err != nil {
				return err
			}
			fmt.Printf("Cleared schedule for post %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

