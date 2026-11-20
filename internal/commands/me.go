package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func NewMeCmd() *cobra.Command {
	var showScopes bool

	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show who the connection is logged in as and what it acts for",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var result map[string]any
			if err := a.Client.Get("/me", nil, &result); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(result)
				return nil
			}
			fmt.Printf("Connection: %s\n", a.ActiveConnection)
			if env, ok := result["env"].(string); ok && env != "" {
				fmt.Printf("Env:     %s\n", env)
			}
			if user, ok := result["user"].(map[string]any); ok {
				if name, ok := user["name"].(string); ok && name != "" {
					fmt.Printf("User:    %s\n", name)
				}
			}
			if network, ok := result["network"].(map[string]any); ok {
				if name, ok := network["name"].(string); ok && name != "" {
					fmt.Printf("Network: %s (%v", name, network["id"])
					if role, ok := network["role"].(string); ok && role != "" {
						fmt.Printf(", %s", role)
					}
					fmt.Println(")")
				}
			}
			if acc, ok := result["account"].(map[string]any); ok {
				if name, ok := acc["name"].(string); ok && name != "" {
					fmt.Printf("Account: %s (%v)\n", name, acc["id"])
				}
			}
			if oauth, ok := result["oauth"].(map[string]any); ok {
				if client, ok := oauth["client"].(string); ok && client != "" {
					fmt.Printf("OAuth:   %s\n", client)
				}
				scopes, _ := oauth["scopes"].([]any)
				if showScopes {
					printOAuthScopes(scopes)
				} else if len(scopes) > 0 {
					fmt.Printf("Scopes:  %d (use --scopes to list)\n", len(scopes))
				}
			}
			tok, _ := result["api_key"].(map[string]any)
			if tok != nil {
				if name, ok := tok["name"].(string); ok && name != "" {
					fmt.Printf("API key: %s\n", name)
				}
			}
			if showScopes && tok != nil {
				printScopes(tok["scopes"])
			} else if tok != nil {
				if scopes, ok := tok["scopes"].(map[string]any); ok && len(scopes) > 0 {
					fmt.Printf("Scopes:  %d resources (use --scopes to list)\n", len(scopes))
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&showScopes, "scopes", false, "List scopes per resource")
	return cmd
}

// printOAuthScopes groups "resource:action" scopes the way an API key's scopes are shown.
func printOAuthScopes(scopes []any) {
	grouped := map[string]any{}
	for _, raw := range scopes {
		scope, ok := raw.(string)
		if !ok {
			continue
		}
		resource, action, found := strings.Cut(scope, ":")
		if !found {
			action = ""
		}
		actions, _ := grouped[resource].([]any)
		grouped[resource] = append(actions, action)
	}
	printScopes(grouped)
}

func printScopes(raw any) {
	scopes, ok := raw.(map[string]any)
	if !ok || len(scopes) == 0 {
		return
	}
	keys := make([]string, 0, len(scopes))
	width := 0
	for k := range scopes {
		keys = append(keys, k)
		if len(k) > width {
			width = len(k)
		}
	}
	sort.Strings(keys)

	fmt.Println("Scopes:")
	for _, k := range keys {
		actions, ok := scopes[k].([]any)
		if !ok {
			continue
		}
		parts := make([]string, 0, len(actions))
		for _, a := range actions {
			if s, ok := a.(string); ok {
				parts = append(parts, s)
			}
		}
		fmt.Printf("  %-*s  %s\n", width, k, strings.Join(parts, ", "))
	}
}

func NewNetworkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "network [network-id]",
		Short: "Show a network's identity and member/screen stats",
		Long: `Shows the network a network API key belongs to, or for an OAuth login: the given network, the
one the connection acts for, or the login's only network.`,
		Example: `  pintomind network
  pintomind network 14`,
		Args:        cobra.MaximumNArgs(1),
		Annotations: map[string]string{NoAccountAnnotation: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			client := *a.Client
			if d, ok := a.Config.Domains[a.ActiveConnection]; ok && d.IsOAuth() {
				id, err := networkFor(cmd, args, client.NetworkID)
				if err != nil {
					return err
				}
				client.AccountID, client.NetworkID = 0, id
			} else if len(args) == 1 {
				return fmt.Errorf("an API key connection shows its own network only; leave out the ID, or log in with: pintomind login")
			}

			var result map[string]any
			if err := client.Get("/network", nil, &result); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(result)
				return nil
			}
			printNetworkSummary(result)
			return nil
		},
	}
}

// networkFor picks the network an OAuth login asks about: the argument, the connection's
// network, or the login's only network.
func networkFor(cmd *cobra.Command, args []string, current int) (int, error) {
	if len(args) == 1 {
		return atoiArg(args[0], "network ID")
	}
	if current != 0 {
		return current, nil
	}
	resp, err := fetchLoginChoices(cmd)
	if err != nil {
		return 0, err
	}
	switch len(resp.Networks) {
	case 0:
		return 0, fmt.Errorf("this login may act for no network")
	case 1:
		return resp.Networks[0].ID, nil
	}
	printNetworks(cmd, resp.Networks, 0)
	return 0, fmt.Errorf("this login may act for several networks; pass one: pintomind network <id>")
}

func printNetworkSummary(result map[string]any) {
	network, ok := result["network"].(map[string]any)
	if !ok {
		printJSON(result)
		return
	}
	fmt.Printf("Network: %v (%v)\n", network["name"], network["id"])
	stats, _ := network["stats"].(map[string]any)
	if members, ok := stats["members"]; ok {
		fmt.Printf("Members: %v\n", members)
	}
	if screens, ok := stats["screens"].(map[string]any); ok {
		fmt.Printf("Screens: %v online, %v offline\n", screens["online"], screens["offline"])
	}
}
