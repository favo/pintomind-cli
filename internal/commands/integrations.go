package commands

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

type Integration struct {
	ID                  int     `json:"id"`
	Type                string  `json:"type"`
	Label               string  `json:"label"`
	OK                  bool    `json:"ok"`
	Error               *string `json:"error"`
	Beta                bool    `json:"beta"`
	ConnectedAs         *string `json:"connected_as"`
	CreatedBy           *string `json:"created_by"`
	CreatedAt           string  `json:"created_at"`
	ResourceCount       int     `json:"resource_count"`
	SharedResourceCount int     `json:"shared_resource_count"`
	CanDelete           bool    `json:"can_delete"`
}

type IntegrationResource struct {
	ID        int     `json:"id"`
	UUID      string  `json:"uuid"`
	Title     *string `json:"title"`
	Type      string  `json:"type"`
	OK        bool    `json:"ok"`
	Error     *string `json:"error"`
	PostCount int     `json:"post_count"`
	Shared    bool    `json:"shared"`
}

type IntegrationDetail struct {
	Integration
	Resources []IntegrationResource `json:"resources"`
}

type IntegrationsResponse struct {
	Integrations []Integration `json:"integrations"`
}

type IntegrationType struct {
	Type                   string  `json:"type"`
	Label                  string  `json:"label"`
	Description            *string `json:"description"`
	Beta                   bool    `json:"beta"`
	ConnectedIntegrationID *int    `json:"connected_integration_id"`
}

type IntegrationTypesResponse struct {
	Types []IntegrationType `json:"types"`
}

type AvailableResource struct {
	Title      *string `json:"title"`
	Type       string  `json:"type"`
	TypeLabel  *string `json:"type_label"`
	Color      *string `json:"color"`
	ResourceID *int    `json:"resource_id"`
	OK         *bool   `json:"ok"`
	Token      *string `json:"token"`
}

type AvailableGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AvailableResourcesResponse struct {
	Groups    []AvailableGroup    `json:"groups"`
	Resources []AvailableResource `json:"resources"`
}

func NewIntegrationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integrations",
		Short: "Manage integrations (calendars, social media, Microsoft 365 …) and the resources made from them",
		Long: `Manage the account's integrations and the resources made from them.

They use the resources scopes. Connecting a new integration goes through the provider's login in
the web app or the Go app, so the CLI lists, inspects and deletes integrations and makes resources
from connected ones.`,
	}
	cmd.AddCommand(newIntegrationsListCmd())
	cmd.AddCommand(newIntegrationsTypesCmd())
	cmd.AddCommand(newIntegrationsShowCmd())
	cmd.AddCommand(newIntegrationsAvailableCmd())
	cmd.AddCommand(newIntegrationsAddResourceCmd())
	cmd.AddCommand(newIntegrationsRemoveResourceCmd())
	cmd.AddCommand(newIntegrationsDeleteCmd())
	return cmd
}

func integrationStatus(ok bool, errMessage *string) string {
	if ok {
		return "ok"
	}
	if errMessage != nil && *errMessage != "" {
		return "failed: " + *errMessage
	}
	return "failed"
}

func newIntegrationsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the account's integrations with status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var resp IntegrationsResponse
			if err := a.Client.Get("/integrations", nil, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printResponseJSON(a, resp)
				return nil
			}
			rows := make([][]string, len(resp.Integrations))
			for i, it := range resp.Integrations {
				rows[i] = []string{
					strconv.Itoa(it.ID),
					it.Type,
					it.Label,
					deref(it.ConnectedAs),
					strconv.Itoa(it.ResourceCount),
					strconv.Itoa(it.SharedResourceCount),
					integrationStatus(it.OK, it.Error),
				}
			}
			printTable(cmd, []string{"ID", "TYPE", "LABEL", "CONNECTED AS", "RESOURCES", "SHARED", "STATUS"}, rows)
			return nil
		},
	}
}

func newIntegrationsTypesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "types",
		Short: "List the integration types the account may connect",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var resp IntegrationTypesResponse
			if err := a.Client.Get("/integrations/types", nil, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printResponseJSON(a, resp)
				return nil
			}
			rows := make([][]string, len(resp.Types))
			for i, t := range resp.Types {
				connected := ""
				if t.ConnectedIntegrationID != nil {
					connected = strconv.Itoa(*t.ConnectedIntegrationID)
				}
				rows[i] = []string{t.Type, t.Label, check(t.Beta), connected, deref(t.Description)}
			}
			printTable(cmd, []string{"TYPE", "LABEL", "BETA", "CONNECTED", "DESCRIPTION"}, rows)
			return nil
		},
	}
}

func newIntegrationsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show an integration and the resources made from it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp IntegrationDetail
			if err := a.Client.Get("/integrations/"+url.PathEscape(args[0]), nil, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printResponseJSON(a, resp)
				return nil
			}
			fmt.Printf("%d  %s (%s)\n", resp.ID, resp.Label, resp.Type)
			if resp.ConnectedAs != nil {
				fmt.Printf("Connected as: %s\n", *resp.ConnectedAs)
			}
			if resp.CreatedBy != nil {
				fmt.Printf("Created by:   %s, %s\n", *resp.CreatedBy, resp.CreatedAt)
			}
			fmt.Printf("Status:       %s\n\n", integrationStatus(resp.OK, resp.Error))
			rows := make([][]string, len(resp.Resources))
			for i, r := range resp.Resources {
				rows[i] = []string{
					strconv.Itoa(r.ID),
					deref(r.Title),
					r.Type,
					strconv.Itoa(r.PostCount),
					check(r.Shared),
					integrationStatus(r.OK, r.Error),
				}
			}
			printTable(cmd, []string{"RESOURCE", "TITLE", "TYPE", "POSTS", "SHARED", "STATUS"}, rows)
			return nil
		},
	}
}

func fetchAvailableResources(cmd *cobra.Command, id, groupID string) (AvailableResourcesResponse, error) {
	q := url.Values{}
	setStringIf(q, "group_id", groupID)
	var resp AvailableResourcesResponse
	err := app(cmd).Client.Get("/integrations/"+url.PathEscape(id)+"/available_resources", q, &resp)
	return resp, err
}

func newIntegrationsAvailableCmd() *cobra.Command {
	var groupID string

	cmd := &cobra.Command{
		Use:   "available <id>",
		Short: "List what the provider offers to make resources from",
		Long: `List what the provider offers to make resources from: calendars, pages, reports …

Rows with a resource id are made already. Make one of the others with
'integrations add-resource <id> <number>', using the number in the first column. Power BI lists
one workspace at a time: pick it with --group-id from the workspaces listed.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			resp, err := fetchAvailableResources(cmd, args[0], groupID)
			if err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}
			if len(resp.Groups) > 0 {
				groups := make([][]string, len(resp.Groups))
				for i, g := range resp.Groups {
					groups[i] = []string{g.ID, g.Name}
				}
				printTable(cmd, []string{"GROUP ID", "WORKSPACE"}, groups)
				fmt.Println()
			}
			rows := make([][]string, len(resp.Resources))
			for i, r := range resp.Resources {
				made := ""
				if r.ResourceID != nil {
					made = strconv.Itoa(*r.ResourceID)
				}
				rows[i] = []string{strconv.Itoa(i + 1), deref(r.Title), deref(r.TypeLabel), made}
			}
			printTable(cmd, []string{"#", "TITLE", "TYPE", "RESOURCE"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&groupID, "group-id", "", "Power BI workspace to list")
	return cmd
}

func newIntegrationsAddResourceCmd() *cobra.Command {
	var (
		groupID string
		token   string
	)

	cmd := &cobra.Command{
		Use:   "add-resource <id> [<number>]",
		Short: "Make a resource from one of the provider's offers",
		Long: `Make a resource from one of the provider's offers, by its number in
'integrations available <id>', or by the token from 'integrations available <id> --json'.
An offer that is made already returns the existing resource.`,
		Example: `  pintomind integrations available 12
  pintomind integrations add-resource 12 3`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if (len(args) == 2) == (token != "") {
				return fmt.Errorf("give either the offer's number or --token")
			}
			if len(args) == 2 {
				number, err := atoiArg(args[1], "number")
				if err != nil {
					return err
				}
				offers, err := fetchAvailableResources(cmd, args[0], groupID)
				if err != nil {
					return err
				}
				if number < 1 || number > len(offers.Resources) {
					return fmt.Errorf("no offer number %d: the provider offers %d", number, len(offers.Resources))
				}
				offer := offers.Resources[number-1]
				if offer.Token == nil {
					return fmt.Errorf("%s is made already, as resource %d", deref(offer.Title), deref0(offer.ResourceID))
				}
				token = *offer.Token
			}
			var resp map[string]any
			if err := a.Client.Post("/integrations/"+url.PathEscape(args[0])+"/resources", map[string]any{"token": token}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&groupID, "group-id", "", "Power BI workspace the number refers to")
	cmd.Flags().StringVar(&token, "token", "", "An offer's token from 'integrations available <id> --json'")
	return cmd
}

func deref0(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func newIntegrationsRemoveResourceCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove-resource <id> <resource-id>",
		Short: "Delete a resource made from the integration",
		Long: `Delete a resource made from the integration. A resource shared with the network, or used in
posts, is not deleted: the error names the posts to remove it from first.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmAction(fmt.Sprintf("Delete resource %s of integration %s?", args[1], args[0]), force) {
				return nil
			}
			return runDelete(cmd, "/integrations/"+url.PathEscape(args[0])+"/resources/"+url.PathEscape(args[1]), "resource "+args[1])
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newIntegrationsDeleteCmd() *cobra.Command {
	var (
		force         bool
		withResources bool
	)

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an integration",
		Long: `Delete an integration. One with resources is only deleted with --with-resources, which deletes
them too; one with resources shared with the network is not deleted.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			question := fmt.Sprintf("Delete integration %s?", args[0])
			if withResources {
				question = fmt.Sprintf("Delete integration %s and every resource made from it?", args[0])
			}
			if !confirmAction(question, force) {
				return nil
			}
			path := "/integrations/" + url.PathEscape(args[0])
			if withResources {
				path += "?force=true"
			}
			return runDelete(cmd, path, "integration "+args[0])
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&withResources, "with-resources", false, "Also delete the resources made from it")
	return cmd
}
