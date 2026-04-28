package commands

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

type Member struct {
	ID            int    `json:"id"`
	Status        string `json:"status"`
	Role          string `json:"role"`
	UserID        int    `json:"user_id"`
	UserName      string `json:"user_name"`
	UserEmail     string `json:"user_email"`
	InvitedByID   any    `json:"invited_by_id"`
	InvitedByName string `json:"invited_by_name"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type MembersResponse struct {
	Total int      `json:"total"`
	Items []Member `json:"items"`
}

func NewMembersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "members",
		Short: "Manage account members",
	}
	cmd.AddCommand(newMembersListCmd())
	cmd.AddCommand(newMembersShowCmd())
	cmd.AddCommand(newMembersInviteCmd())
	cmd.AddCommand(newMembersUpdateCmd())
	cmd.AddCommand(newMembersRemoveCmd())
	cmd.AddCommand(newMembersResendInviteCmd())
	return cmd
}

func newMembersListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List account members",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			applyPagination(cmd, q)

			var resp MembersResponse
			if err := a.Client.Get("/members", q, &resp); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, m := range resp.Items {
				rows[i] = []string{
					strconv.Itoa(m.ID),
					m.Status,
					m.Role,
					m.UserName,
					m.UserEmail,
				}
			}
			printTable(cmd, []string{"ID", "STATUS", "ROLE", "NAME", "EMAIL"}, rows)
			return nil
		},
	}
	addPaginationFlags(cmd)
	return cmd
}

func newMembersShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a member",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/members/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newMembersInviteCmd() *cobra.Command {
	var email, role, firstname, lastname string

	cmd := &cobra.Command{
		Use:   "invite --email <email> --role <role>",
		Short: "Invite a user to the account",
		Example: `  pintomind members invite --email carol@example.com --role editor --firstname Carol --lastname Christensen`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			body := map[string]any{
				"member": map[string]any{
					"email":     email,
					"role":      role,
					"firstname": firstname,
					"lastname":  lastname,
				},
			}
			var resp map[string]any
			if err := a.Client.Post("/members", body, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
			} else {
				status, _ := resp["status"].(string)
				fmt.Printf("Invited %s as %s (status: %s)\n", email, role, status)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "Email address of the user to invite (required)")
	cmd.Flags().StringVar(&role, "role", "", "Role to assign: account_owner, admin, editor (required)")
	cmd.Flags().StringVar(&firstname, "firstname", "", "First name of the user (required)")
	cmd.Flags().StringVar(&lastname, "lastname", "", "Last name of the user (required)")
	_ = cmd.MarkFlagRequired("email")
	_ = cmd.MarkFlagRequired("role")
	_ = cmd.MarkFlagRequired("firstname")
	_ = cmd.MarkFlagRequired("lastname")
	return cmd
}

func newMembersUpdateCmd() *cobra.Command {
	var role string

	cmd := &cobra.Command{
		Use:   "update <id> --role <role>",
		Short: "Update a member's role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			body := map[string]any{"member": map[string]any{"role": role}}
			var resp map[string]any
			if err := a.Client.Patch("/members/"+args[0], body, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
			} else {
				fmt.Printf("Updated member %s role to %s\n", args[0], role)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&role, "role", "", "New role to assign: account_owner, admin, editor (required)")
	_ = cmd.MarkFlagRequired("role")
	return cmd
}

func newMembersRemoveCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a member from the account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if !force {
				fmt.Printf("Remove member %s from the account? Pass --force to skip this prompt.\n", args[0])
				var confirm string
				fmt.Print("Type 'yes' to confirm: ")
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if err := a.Client.Delete("/members/" + args[0]); err != nil {
				return err
			}
			fmt.Printf("Removed member %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newMembersResendInviteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resend-invite <id>",
		Short: "Resend invitation email to a pending member",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Post("/members/"+args[0]+"/resend_invite", nil, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
			} else {
				fmt.Printf("Invitation resent to member %s\n", args[0])
			}
			return nil
		},
	}
}
