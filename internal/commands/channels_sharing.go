package commands

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newChannelsSharingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sharing",
		Short: "Share a channel in the Go app with share codes or email invites",
	}
	cmd.AddCommand(newSharingShowCmd())
	cmd.AddCommand(newSharingStartCmd())
	cmd.AddCommand(newSharingUpdateCmd())
	cmd.AddCommand(newSharingStopCmd())
	cmd.AddCommand(newSharingCodesCmd())
	cmd.AddCommand(newSharingInvitesCmd())
	return cmd
}

type ChannelSharing struct {
	Availability   *string `json:"availability"`
	AvailableOnWeb bool    `json:"available_on_web"`
	Codes          []struct {
		ID          int    `json:"id"`
		Code        string `json:"code"`
		URL         string `json:"url"`
		Subscribers int    `json:"subscribers"`
		Editable    bool   `json:"editable"`
	} `json:"codes"`
	MaxCodes     int `json:"max_codes"`
	InvitesCount int `json:"invites_count"`
	ActiveUsers  *struct {
		LastWeek     int `json:"last_week"`
		LastMonth    int `json:"last_month"`
		LastHalfYear int `json:"last_half_year"`
	} `json:"active_users"`
	CanShare bool `json:"can_share"`
}

func sharingPath(channelID string) string {
	return "/channels/" + channelID + "/sharing"
}

func newSharingShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <channel-id>",
		Short: "Show how a channel is shared: codes, invites and active users",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if a.JSONOutput {
				return getJSON(cmd, sharingPath(args[0]), nil)
			}
			var resp struct {
				Sharing ChannelSharing `json:"sharing"`
			}
			if err := a.Client.Get(sharingPath(args[0]), nil, &resp); err != nil {
				return err
			}
			s := resp.Sharing
			if s.Availability == nil {
				fmt.Println("Not shared.")
				if !s.CanShare {
					fmt.Println("The account can't share more channels.")
				}
				return nil
			}
			fmt.Printf("Shared with:    %s\n", *s.Availability)
			fmt.Printf("Opens on web:   %s\n", map[bool]string{true: "yes", false: "no"}[s.AvailableOnWeb])
			if s.ActiveUsers != nil {
				fmt.Printf("Active users:   %d last week, %d last month, %d last half year\n",
					s.ActiveUsers.LastWeek, s.ActiveUsers.LastMonth, s.ActiveUsers.LastHalfYear)
			}
			if *s.Availability == "invites" {
				fmt.Printf("Invites:        %d (list with: pintomind channels sharing invites list %s)\n", s.InvitesCount, args[0])
				return nil
			}
			fmt.Printf("Codes (%d of at most %d):\n\n", len(s.Codes), s.MaxCodes)
			rows := make([][]string, len(s.Codes))
			for i, c := range s.Codes {
				rows[i] = []string{strconv.Itoa(c.ID), c.Code, strconv.Itoa(c.Subscribers), check(c.Editable), c.URL}
			}
			printTable(cmd, []string{"ID", "CODE", "SUBSCRIBERS", "EDITABLE", "URL"}, rows)
			return nil
		},
	}
}

func newSharingStartCmd() *cobra.Command {
	var code string
	var invites, onWeb bool

	cmd := &cobra.Command{
		Use:   "start <channel-id>",
		Short: "Share a channel in Go with everyone (a share code) or by email invites",
		Long: `Shares the channel with everyone who has a share code (the given --code, or a made-up one),
or with --invites only with invited email addresses. The way of sharing can't be changed later:
stop sharing and share again. Needs a login or an API key that belongs to a user.`,
		Example: `  pintomind channels sharing start 9 --code menighet
  pintomind channels sharing start 9 --invites --web`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sharing := map[string]any{"availability": "everyone"}
			if invites {
				if code != "" {
					return errors.New("--code only applies when sharing with everyone")
				}
				sharing["availability"] = "invites"
			}
			setStringIfNotEmpty(sharing, "code", code)
			setIfChanged(cmd, sharing, "web", "available_on_web", onWeb)
			return sendJSON(cmd, "POST", sharingPath(args[0]), map[string]any{"sharing": sharing})
		},
	}
	cmd.Flags().StringVar(&code, "code", "", "Share code (at least 3 letters, digits, - or _; default: made up)")
	cmd.Flags().BoolVar(&invites, "invites", false, "Share only with invited email addresses")
	cmd.Flags().BoolVar(&onWeb, "web", false, "Let the channel also open in a browser")
	return cmd
}

func setStringIfNotEmpty(body map[string]any, key, value string) {
	if value != "" {
		body[key] = value
	}
}

func newSharingUpdateCmd() *cobra.Command {
	var onWeb bool
	cmd := &cobra.Command{
		Use:     "update <channel-id> --web=<true|false>",
		Short:   "Change whether a shared channel also opens in a browser",
		Example: `  pintomind channels sharing update 9 --web=false`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("web") {
				return errors.New("pass --web=true or --web=false")
			}
			return sendJSON(cmd, "PATCH", sharingPath(args[0]), map[string]any{"sharing": map[string]any{"available_on_web": onWeb}})
		},
	}
	cmd.Flags().BoolVar(&onWeb, "web", false, "Whether the channel also opens in a browser")
	return cmd
}

func newSharingStopCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "stop <channel-id>",
		Short: "Stop sharing a channel in Go (codes and invites in use are deactivated)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmAction(fmt.Sprintf("Stop sharing channel %s? Its Go users lose it.", args[0]), force) {
				return nil
			}
			return sendJSON(cmd, "DELETE", sharingPath(args[0]), nil)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newSharingCodesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "codes",
		Short: "Manage the share codes of a channel shared with everyone (list them with 'sharing show')",
	}

	var addCode string
	add := &cobra.Command{
		Use:   "add <channel-id>",
		Short: "Add a share code (at most four; made up unless --code is given)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var body any
			if addCode != "" {
				body = map[string]any{"share_code": map[string]any{"code": addCode}}
			}
			return sendJSON(cmd, "POST", sharingPath(args[0])+"/codes", body)
		},
	}
	add.Flags().StringVar(&addCode, "code", "", "The code")

	update := &cobra.Command{
		Use:   "update <channel-id> <code-id> <new-code>",
		Short: "Change a share code (not one someone follows)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return sendJSON(cmd, "PATCH", sharingPath(args[0])+"/codes/"+args[1],
				map[string]any{"share_code": map[string]any{"code": args[2]}})
		},
	}

	var force bool
	remove := &cobra.Command{
		Use:   "remove <channel-id> <code-id>",
		Short: "Remove a share code (not the last one; stop sharing instead)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmAction(fmt.Sprintf("Remove share code %s from channel %s?", args[1], args[0]), force) {
				return nil
			}
			return sendJSON(cmd, "DELETE", sharingPath(args[0])+"/codes/"+args[1], nil)
		},
	}
	remove.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	cmd.AddCommand(add, update, remove)
	return cmd
}

type ChannelInvite struct {
	ID        int     `json:"id"`
	Email     string  `json:"email"`
	Name      *string `json:"name"`
	Status    string  `json:"status"`
	Devices   int     `json:"devices"`
	InvitedAt *string `json:"invited_at"`
}

func newSharingInvitesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invites",
		Short: "Manage the email invites of a channel shared by invitation",
	}

	var query, sortBy, fields string
	list := &cobra.Command{
		Use:   "list <channel-id>",
		Short: "List invites, newest first",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			q := url.Values{}
			setStringIf(q, "q", query)
			setStringIf(q, "sort_by", sortBy)
			setStringIf(q, "fields", fields)
			applyPagination(cmd, q)
			if a.JSONOutput || fields != "" {
				return getJSON(cmd, sharingPath(args[0])+"/invites", q)
			}
			var resp struct {
				Total      int             `json:"total"`
				Pagination *Pagination     `json:"pagination"`
				Items      []ChannelInvite `json:"items"`
			}
			if err := a.Client.Get(sharingPath(args[0])+"/invites", q, &resp); err != nil {
				return err
			}
			printTotal(resp.Total, resp.Pagination)
			rows := make([][]string, len(resp.Items))
			for i, inv := range resp.Items {
				rows[i] = []string{strconv.Itoa(inv.ID), inv.Email, deref(inv.Name), inv.Status, strconv.Itoa(inv.Devices), deref(inv.InvitedAt)}
			}
			printTable(cmd, []string{"ID", "EMAIL", "NAME", "STATUS", "DEVICES", "INVITED AT"}, rows)
			return nil
		},
	}
	list.Flags().StringVar(&query, "q", "", "Match email or name")
	list.Flags().StringVar(&sortBy, "sort-by", "", "Sort field: email, name, invited_at, created_at (append :desc)")
	addFieldsFlag(list, &fields)
	addPaginationFlags(list, 200)

	add := &cobra.Command{
		Use:     "add <channel-id> <email>...",
		Short:   "Invite email addresses (at most 100 at a time; each gets its own code by email)",
		Example: `  pintomind channels sharing invites add 9 kari@example.com ola@example.com`,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var emails []string
			for _, arg := range args[1:] {
				for _, e := range strings.Split(arg, ",") {
					if e = strings.TrimSpace(e); e != "" {
						emails = append(emails, e)
					}
				}
			}
			return sendJSON(cmd, "POST", sharingPath(args[0])+"/invites", map[string]any{"invite": map[string]any{"emails": emails}})
		},
	}

	rename := &cobra.Command{
		Use:   "rename <channel-id> <invite-id> <name>",
		Short: "Name an invite",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return sendJSON(cmd, "PATCH", sharingPath(args[0])+"/invites/"+args[1], map[string]any{"invite": map[string]any{"name": args[2]}})
		},
	}

	resend := &cobra.Command{
		Use:   "resend <channel-id> <invite-id>",
		Short: "Email an invite again",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return sendJSON(cmd, "POST", sharingPath(args[0])+"/invites/"+args[1]+"/resend", nil)
		},
	}

	var force bool
	remove := &cobra.Command{
		Use:   "remove <channel-id> <invite-id>",
		Short: "Remove an invite (one in use is deactivated and its device loses the channel)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmAction(fmt.Sprintf("Remove invite %s from channel %s?", args[1], args[0]), force) {
				return nil
			}
			return sendJSON(cmd, "DELETE", sharingPath(args[0])+"/invites/"+args[1], nil)
		},
	}
	remove.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	cmd.AddCommand(list, add, rename, resend, remove)
	return cmd
}
