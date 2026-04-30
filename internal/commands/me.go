package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewMeCmd() *cobra.Command {
	return &cobra.Command{
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
			if user, ok := result["user"].(map[string]any); ok {
				fmt.Printf("User:    %v\n", user["email"])
			}
			if acc, ok := result["account"].(map[string]any); ok {
				fmt.Printf("Account: %v\n", acc["name"])
			}
			if tok, ok := result["token"].(map[string]any); ok {
				fmt.Printf("Token:   %v\n", tok["name"])
				if scopes, ok := tok["scopes"]; ok {
					fmt.Printf("Scopes:  %v\n", scopes)
				}
			}
			return nil
		},
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
