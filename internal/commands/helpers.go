package commands

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// resolveScreenIDs returns the target IDs string and whether it is a bulk call.
// Priority: --all > --ids > single positional ID.
func resolveScreenIDs(cmd *cobra.Command, singleID, ids string, all bool) (targetIDs string, bulk bool, err error) {
	if all {
		a := app(cmd)
		var resp ScreensResponse
		if err := a.Client.Get("/screens", url.Values{"per_page": {"1000"}}, &resp); err != nil {
			return "", false, fmt.Errorf("fetching all screens: %w", err)
		}
		if len(resp.Items) == 0 {
			return "", false, fmt.Errorf("no screens found")
		}
		parts := make([]string, len(resp.Items))
		for i, s := range resp.Items {
			parts[i] = strconv.Itoa(s.ID)
		}
		return strings.Join(parts, ","), true, nil
	}
	if ids != "" {
		return ids, true, nil
	}
	if singleID != "" {
		return singleID, false, nil
	}
	return "", false, fmt.Errorf("provide a screen ID, --ids <id1,id2>, or --all")
}

// resolveChannelIDs returns all channel IDs as a slice when --all is set,
// or a single-element slice for a single ID argument.
// Note: the channel API has no bulk endpoint, so callers must loop.
func resolveChannelIDs(cmd *cobra.Command, singleID string, all bool) ([]string, error) {
	if all {
		a := app(cmd)
		var resp ChannelsResponse
		if err := a.Client.Get("/channels", url.Values{"per_page": {"1000"}}, &resp); err != nil {
			return nil, fmt.Errorf("fetching all channels: %w", err)
		}
		if len(resp.Items) == 0 {
			return nil, fmt.Errorf("no channels found")
		}
		ids := make([]string, len(resp.Items))
		for i, c := range resp.Items {
			ids[i] = strconv.Itoa(c.ID)
		}
		return ids, nil
	}
	if singleID == "" {
		return nil, fmt.Errorf("provide a channel ID or --all")
	}
	return []string{singleID}, nil
}

// sendScreenPatch sends a PATCH to a single screen or to screens/bulk.
func sendScreenPatch(cmd *cobra.Command, targetIDs string, bulk bool, body map[string]any, successMsg string) error {
	a := app(cmd)
	var resp map[string]any
	var err error
	if bulk {
		q := url.Values{"ids": {targetIDs}}
		err = a.Client.Patch("/screens/bulk?"+q.Encode(), body, &resp)
	} else {
		err = a.Client.Patch("/screens/"+targetIDs, body, &resp)
	}
	if err != nil {
		return err
	}
	if a.JSONOutput {
		printJSON(resp)
	} else {
		fmt.Println(successMsg)
	}
	return nil
}

// addTargetFlags attaches --ids and --all to a command and returns pointers to them.
func addTargetFlags(cmd *cobra.Command) (ids *string, all *bool) {
	ids = new(string)
	all = new(bool)
	cmd.Flags().StringVar(ids, "ids", "", "Comma-separated screen IDs")
	cmd.Flags().BoolVar(all, "all", false, "Apply to all screens")
	return
}

// addPaginationFlags attaches --page and --per-page to a list command.
func addPaginationFlags(cmd *cobra.Command) {
	cmd.Flags().Int("page", 0, "Page number")
	cmd.Flags().Int("per-page", 0, "Results per page (API default: 200)")
}

// applyPagination adds page/per-page to a query if set.
func applyPagination(cmd *cobra.Command, q url.Values) {
	if p, _ := cmd.Flags().GetInt("page"); p > 0 {
		q.Set("page", strconv.Itoa(p))
	}
	if pp, _ := cmd.Flags().GetInt("per-page"); pp > 0 {
		q.Set("per_page", strconv.Itoa(pp))
	}
}
