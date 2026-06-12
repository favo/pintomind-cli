package commands

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type PostNotification struct {
	ID                int    `json:"id"`
	SendAt            string `json:"send_at"`
	Title             string `json:"title"`
	Body              string `json:"body"`
	Status            string `json:"status"`
	Thumbnail         string `json:"thumbnail"`
	NotificationsSent int    `json:"notifications_sent"`
	NotificationsRead int    `json:"notifications_read"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

type PostNotificationsResponse struct {
	Total int                `json:"total"`
	Items []PostNotification `json:"items"`
}

func newPostsNotificationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notifications",
		Short: "Manage push notifications for a post (Go mobile app)",
		Long: `Manage push notifications sent to the Go mobile app for a post.

A notification is delivered to the notification-enabled devices of every channel
the post is published to. Title and body are optional — when omitted they are
generated from the post at send time.`,
	}
	cmd.AddCommand(newPostsNotificationsListCmd())
	cmd.AddCommand(newPostsNotificationsCreateCmd())
	cmd.AddCommand(newPostsNotificationsCancelCmd())
	return cmd
}

func newPostsNotificationsListCmd() *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "list <post-id>",
		Short: "List a post's push notifications (most recent first)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			q := url.Values{}
			if status != "" {
				if status != "queued" && status != "sent" {
					return fmt.Errorf("--status must be queued or sent")
				}
				q.Set("status", status)
			}
			applyPagination(cmd, q)

			var resp PostNotificationsResponse
			if err := a.Client.Get("/posts/"+args[0]+"/notifications", q, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, n := range resp.Items {
				rows[i] = []string{
					strconv.Itoa(n.ID),
					n.Status,
					n.SendAt,
					firstNonEmpty(n.Title, "(auto)"),
					strconv.Itoa(n.NotificationsSent),
					strconv.Itoa(n.NotificationsRead),
				}
			}
			printTable(cmd, []string{"ID", "STATUS", "SEND_AT", "TITLE", "SENT", "READ"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Filter by status: queued or sent")
	addPaginationFlags(cmd)
	return cmd
}

func newPostsNotificationsCreateCmd() *cobra.Command {
	var sendAt, title, body string
	cmd := &cobra.Command{
		Use:   "create <post-id> --send-at <iso8601>",
		Short: "Schedule a push notification for a post",
		Long: `Schedule a push notification for a post.

--send-at must be a future ISO 8601 datetime. Title (max 50) and body (max 150)
are optional; leave them blank to auto-fill from the post at send time.
A post may have at most 12 pending notifications.`,
		Example: `  pintomind posts notifications create 123 --send-at 2026-07-01T15:30:00Z
  pintomind posts notifications create 123 --send-at 2026-07-01T15:30:00Z \
    --title "Breaking news" --body "New content has just been published"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			notification := map[string]any{"send_at": sendAt}
			if title != "" {
				notification["title"] = title
			}
			if body != "" {
				notification["body"] = body
			}
			reqBody := map[string]any{"notification": notification}
			var resp map[string]any
			if err := a.Client.Post("/posts/"+args[0]+"/notifications", reqBody, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&sendAt, "send-at", "", "When to send, ISO 8601 datetime in the future (required)")
	cmd.Flags().StringVar(&title, "title", "", "Notification title (max 50; auto-filled from post if blank)")
	cmd.Flags().StringVar(&body, "body", "", "Notification body (max 150; auto-filled from post if blank)")
	_ = cmd.MarkFlagRequired("send-at")
	return cmd
}

func newPostsNotificationsCancelCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "cancel <post-id> <notification-id>",
		Short: "Cancel a pending (queued) push notification",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if !force {
				fmt.Printf("Cancel notification %s for post %s? Type 'yes' to confirm: ", args[1], args[0])
				var confirm string
				fmt.Scanln(&confirm)
				if strings.TrimSpace(confirm) != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if err := a.Client.Delete("/posts/" + args[0] + "/notifications/" + args[1]); err != nil {
				return err
			}
			fmt.Printf("Cancelled notification %s for post %s\n", args[1], args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
