package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/appctx"
)

var validExpiryActions = map[string]bool{"delete": true, "archive": true, "unpublish": true}

// untilOpts holds the --until flag value used by post create commands.
type untilOpts struct {
	until string
}

func addUntilFlag(cmd *cobra.Command, o *untilOpts) {
	cmd.Flags().StringVar(&o.until, "until", "",
		"Auto-expire the post at this date/time (YYYY-MM-DD or YYYY-MM-DDTHH:MM). Evaluated in each channel's local time zone.")
}

// parseKVPairs splits "k=v,k=v" into a map and trims whitespace.
func parseKVPairs(s string) (map[string]string, error) {
	out := map[string]string{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		eq := strings.Index(part, "=")
		if eq < 0 {
			return nil, fmt.Errorf("expected key=value in %q", part)
		}
		k := strings.TrimSpace(part[:eq])
		v := strings.TrimSpace(part[eq+1:])
		if k == "" {
			return nil, fmt.Errorf("empty key in %q", part)
		}
		out[k] = v
	}
	return out, nil
}

func validateDate(s string) error {
	_, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("invalid date %q (want YYYY-MM-DD)", s)
	}
	return nil
}

func validateClock(s string) error {
	_, err := time.Parse("15:04", s)
	if err != nil {
		return fmt.Errorf("invalid time %q (want HH:MM)", s)
	}
	return nil
}

// parsePeriodFlag parses --period 'from_date=...,to_date=...,from_time=...,to_time=...'.
func parsePeriodFlag(s string) (map[string]any, error) {
	kv, err := parseKVPairs(s)
	if err != nil {
		return nil, fmt.Errorf("--period: %w", err)
	}
	allowed := map[string]bool{"from_date": true, "to_date": true, "from_time": true, "to_time": true}
	for k := range kv {
		if !allowed[k] {
			return nil, fmt.Errorf("--period: unknown key %q (allowed: from_date, to_date, from_time, to_time)", k)
		}
	}
	if kv["from_date"] == "" || kv["to_date"] == "" {
		return nil, fmt.Errorf("--period: from_date and to_date are required")
	}
	if err := validateDate(kv["from_date"]); err != nil {
		return nil, fmt.Errorf("--period: %w", err)
	}
	if err := validateDate(kv["to_date"]); err != nil {
		return nil, fmt.Errorf("--period: %w", err)
	}
	out := map[string]any{"from_date": kv["from_date"], "to_date": kv["to_date"]}
	if v, ok := kv["from_time"]; ok && v != "" {
		if err := validateClock(v); err != nil {
			return nil, fmt.Errorf("--period: %w", err)
		}
		out["from_time"] = v
	}
	if v, ok := kv["to_time"]; ok && v != "" {
		if err := validateClock(v); err != nil {
			return nil, fmt.Errorf("--period: %w", err)
		}
		out["to_time"] = v
	}
	return out, nil
}

// parseRecurringRuleFlag parses --recurring-rule 'from_wday=N,to_wday=N,from_time=HH:MM,to_time=HH:MM,weeks=all|odd|even'.
func parseRecurringRuleFlag(s string) (map[string]any, error) {
	kv, err := parseKVPairs(s)
	if err != nil {
		return nil, fmt.Errorf("--recurring-rule: %w", err)
	}
	allowed := map[string]bool{"from_wday": true, "to_wday": true, "from_time": true, "to_time": true, "weeks": true}
	for k := range kv {
		if !allowed[k] {
			return nil, fmt.Errorf("--recurring-rule: unknown key %q (allowed: from_wday, to_wday, from_time, to_time, weeks)", k)
		}
	}
	if kv["from_wday"] == "" || kv["to_wday"] == "" || kv["from_time"] == "" || kv["to_time"] == "" {
		return nil, fmt.Errorf("--recurring-rule: from_wday, to_wday, from_time, to_time are required")
	}
	fromWd, err := strconv.Atoi(kv["from_wday"])
	if err != nil || fromWd < 0 || fromWd > 6 {
		return nil, fmt.Errorf("--recurring-rule: from_wday must be 0-6 (Sun-Sat)")
	}
	toWd, err := strconv.Atoi(kv["to_wday"])
	if err != nil || toWd < 0 || toWd > 6 {
		return nil, fmt.Errorf("--recurring-rule: to_wday must be 0-6 (Sun-Sat)")
	}
	if err := validateClock(kv["from_time"]); err != nil {
		return nil, fmt.Errorf("--recurring-rule: %w", err)
	}
	if err := validateClock(kv["to_time"]); err != nil {
		return nil, fmt.Errorf("--recurring-rule: %w", err)
	}
	out := map[string]any{
		"from_wday": fromWd,
		"to_wday":   toWd,
		"from_time": kv["from_time"],
		"to_time":   kv["to_time"],
	}
	if w, ok := kv["weeks"]; ok && w != "" {
		if w != "all" && w != "odd" && w != "even" {
			return nil, fmt.Errorf("--recurring-rule: weeks must be all, odd, or even")
		}
		out["weeks"] = w
	}
	return out, nil
}

// parseUntilArg accepts YYYY-MM-DD, YYYY-MM-DDTHH:MM, or "YYYY-MM-DD HH:MM".
// Returns the to_date and to_time strings (to_time defaults to "23:59" when only a date is supplied).
func parseUntilArg(s string) (string, string, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse("2006-01-02T15:04", s); err == nil {
		return t.Format("2006-01-02"), t.Format("15:04"), nil
	}
	if t, err := time.Parse("2006-01-02 15:04", s); err == nil {
		return t.Format("2006-01-02"), t.Format("15:04"), nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Format("2006-01-02"), "23:59", nil
	}
	return "", "", fmt.Errorf("invalid --until value %q (want YYYY-MM-DD or YYYY-MM-DDTHH:MM)", s)
}

// applyUntilSchedule PUTs a one-period schedule that runs from today until the parsed --until value.
func applyUntilSchedule(a *appctx.App, postID int, o untilOpts) error {
	if o.until == "" {
		return nil
	}
	toDate, toTime, err := parseUntilArg(o.until)
	if err != nil {
		return err
	}
	body := map[string]any{
		"time_schedule": map[string]any{
			"time_periods": []map[string]any{{
				"from_date": time.Now().Format("2006-01-02"),
				"from_time": "00:00",
				"to_date":   toDate,
				"to_time":   toTime,
			}},
			"schedule_expiry_action": "unpublish",
		},
	}
	var resp map[string]any
	if err := a.Client.Put("/posts/"+strconv.Itoa(postID)+"/time_schedule", body, &resp); err != nil {
		return fmt.Errorf("attaching --until schedule: %w", err)
	}
	if !a.JSONOutput {
		fmt.Printf("Scheduled post %d until %s %s (unpublish on expiry)\n", postID, toDate, toTime)
	}
	return nil
}
