package commands

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

type Webhook struct {
	ID              int    `json:"id"`
	Event           string `json:"event"`
	URL             string `json:"url"`
	ScreenID        *int   `json:"screen_id"`
	Enabled         bool   `json:"enabled"`
	LastDeliveredAt string `json:"last_delivered_at"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type WebhooksResponse struct {
	Total int       `json:"total"`
	Items []Webhook `json:"items"`
}

type WebhookEvent struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Family      string `json:"family"`
}

type WebhookEventsResponse struct {
	Events []WebhookEvent `json:"events"`
}

func NewWebhooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhooks",
		Short: "Manage outbound webhooks",
	}
	cmd.AddCommand(newWebhooksListCmd())
	cmd.AddCommand(newWebhooksEventsCmd())
	cmd.AddCommand(newWebhooksShowCmd())
	cmd.AddCommand(newWebhooksCreateCmd())
	cmd.AddCommand(newWebhooksUpdateCmd())
	cmd.AddCommand(newWebhooksDeleteCmd())
	cmd.AddCommand(newWebhooksTestCmd())
	cmd.AddCommand(newWebhooksDeliveriesCmd())
	return cmd
}

type WebhookDelivery struct {
	ID           int    `json:"id"`
	WebhookID    int    `json:"webhook_id"`
	ScreenID     *int   `json:"screen_id"`
	ResourceID   *int   `json:"resource_id"`
	Attempt      int    `json:"attempt"`
	HTTPStatus   *int   `json:"http_status"`
	Successful   bool   `json:"successful"`
	ErrorMessage string `json:"error_message"`
	DeliveredAt  string `json:"delivered_at"`
	CreatedAt    string `json:"created_at"`
	Payload      any    `json:"payload"`
}

type WebhookDeliveriesResponse struct {
	Total int               `json:"total"`
	Items []WebhookDelivery `json:"items"`
}

func newWebhooksDeliveriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deliveries <webhook-id>",
		Short: "List past deliveries for a webhook (newest first)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			q := url.Values{}
			applyPagination(cmd, q)

			var resp WebhookDeliveriesResponse
			if err := a.Client.Get("/webhooks/"+args[0]+"/deliveries", q, &resp); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, d := range resp.Items {
				status := "-"
				if d.HTTPStatus != nil {
					status = strconv.Itoa(*d.HTTPStatus)
				}
				rows[i] = []string{
					strconv.Itoa(d.ID),
					strconv.Itoa(d.Attempt),
					status,
					strconv.FormatBool(d.Successful),
					d.DeliveredAt,
					d.ErrorMessage,
				}
			}
			printTable(cmd, []string{"ID", "ATTEMPT", "HTTP", "OK", "DELIVERED AT", "ERROR"}, rows)
			return nil
		},
	}
	addPaginationFlags(cmd)
	return cmd
}

func newWebhooksListCmd() *cobra.Command {
	var sortBy string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List webhooks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			if sortBy != "" {
				q.Set("sort_by", sortBy)
			}
			applyPagination(cmd, q)

			var resp WebhooksResponse
			if err := a.Client.Get("/webhooks", q, &resp); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, w := range resp.Items {
				screen := "all"
				if w.ScreenID != nil {
					screen = strconv.Itoa(*w.ScreenID)
				}
				rows[i] = []string{
					strconv.Itoa(w.ID),
					w.Event,
					w.URL,
					screen,
					strconv.FormatBool(w.Enabled),
					w.LastDeliveredAt,
				}
			}
			printTable(cmd, []string{"ID", "EVENT", "URL", "SCREEN", "ENABLED", "LAST DELIVERED"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field (created_at|updated_at, e.g. created_at:desc)")
	addPaginationFlags(cmd)
	return cmd
}

func newWebhooksEventsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "events",
		Short: "List available webhook event names",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var resp WebhookEventsResponse
			if err := a.Client.Get("/webhooks/events", nil, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}
			rows := make([][]string, len(resp.Events))
			for i, e := range resp.Events {
				rows[i] = []string{e.Name, e.Family, e.Description}
			}
			printTable(cmd, []string{"NAME", "FAMILY", "DESCRIPTION"}, rows)
			return nil
		},
	}
}

func newWebhooksShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a webhook (secret not included)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/webhooks/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newWebhooksCreateCmd() *cobra.Command {
	var (
		event      string
		webhookURL string
		screenID   int
		disabled   bool
	)

	cmd := &cobra.Command{
		Use:   "create --event <name> --url <https-url>",
		Short: "Create a webhook (response includes the secret — store it)",
		Example: `  pintomind webhooks create \
    --event screen.offline \
    --url "https://example.com/hooks/pintomind"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			body := map[string]any{
				"event": event,
				"url":   webhookURL,
			}
			if cmd.Flags().Changed("screen-id") {
				body["screen_id"] = screenID
			}
			if disabled {
				body["enabled"] = false
			}
			var resp map[string]any
			if err := a.Client.Post("/webhooks", map[string]any{"webhook": body}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&event, "event", "", "Event name (see 'webhooks events') (required)")
	cmd.Flags().StringVar(&webhookURL, "url", "", "Destination URL (required)")
	cmd.Flags().IntVar(&screenID, "screen-id", 0, "Scope to one screen (only for screen.online/screen.offline)")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "Create webhook disabled (default enabled)")
	_ = cmd.MarkFlagRequired("event")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func newWebhooksUpdateCmd() *cobra.Command {
	var (
		webhookURL string
		enabled    bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a webhook (url, enabled)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			body := map[string]any{}
			if cmd.Flags().Changed("url") {
				body["url"] = webhookURL
			}
			if cmd.Flags().Changed("enabled") {
				body["enabled"] = enabled
			}
			if len(body) == 0 {
				return fmt.Errorf("provide at least one field to update")
			}
			var resp map[string]any
			if err := a.Client.Patch("/webhooks/"+args[0], map[string]any{"webhook": body}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&webhookURL, "url", "", "Destination URL")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable or disable delivery")
	return cmd
}

func newWebhooksDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if !force {
				fmt.Printf("Delete webhook %s? Pass --force to skip this prompt.\n", args[0])
				var confirm string
				fmt.Print("Type 'yes' to confirm: ")
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if err := a.Client.Delete("/webhooks/" + args[0]); err != nil {
				return err
			}
			fmt.Printf("Deleted webhook %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newWebhooksTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test <id>",
		Short: "Enqueue a synthetic delivery against the webhook URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Post("/webhooks/"+args[0]+"/test", nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}
