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
		Short: "Show current token identity and connection context",
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
			if acc, ok := result["account"].(map[string]any); ok {
				if name, ok := acc["name"].(string); ok && name != "" {
					fmt.Printf("Account: %s\n", name)
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
		Use:   "network",
		Short: "Show network identity and stats (network token required)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var result map[string]any
			if err := a.Client.Get("/network", nil, &result); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(result)
				return nil
			}
			printJSON(result)
			return nil
		},
	}
}
