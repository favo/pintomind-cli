package commands

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/api"
	"favo/pintomind-cli/internal/config"
)

const defaultSetupURL = "https://app.infoskjermen.no"

func newSetupInitCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:     "init",
		Aliases: []string{"first-time"},
		Short:   "Guided first-time setup (connection + skills + completion)",
		Long: `Walks through first-time configuration:

  1. Add a connection (name + URL + API key)
  2. Verify the API key by calling /me
  3. Optionally install Claude / Codex skills and shell completion`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSetupInit(root)
		},
	}
}

func runSetupInit(root *cobra.Command) error {
	fmt.Println()
	fmt.Println("  ┌─────────────────────────────────────────────────┐")
	fmt.Println("  │  Welcome to the Pintomind CLI!                  │")
	fmt.Println("  └─────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("  Let's get you set up. This will only take a minute.")
	fmt.Println()
	fmt.Println("  We'll:")
	fmt.Println("    1. Connect your account with an API key")
	fmt.Println("    2. Verify it works")
	fmt.Println("    3. (Optional) Install AI skills and shell completion")
	fmt.Println()
	fmt.Println("  Don't have an API key yet? Generate one from your")
	fmt.Println("  Pintomind dashboard at https://app.infoskjermen.no")
	fmt.Println()
	fmt.Println("  Press Ctrl+C at any time to cancel.")
	fmt.Println()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	name, baseURL, apiKey, _, err := promptAndVerifyConnection(cfg)
	if err != nil {
		if isAborted(err) {
			fmt.Println("Aborted.")
			return nil
		}
		return err
	}

	cfg.Domains[name] = config.Domain{APIKey: apiKey, BaseURL: baseURL}
	cfg.DefaultDomain = name
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("Saved connection %q (%s) and set as default.\n\n", name, baseURL)

	installClaude, err := confirmStep("Install Claude Code skill?", true)
	if err != nil {
		return err
	}
	if installClaude {
		if err := installClaudeSkill(); err != nil {
			fmt.Printf("Claude skill install failed: %v\n", err)
		}
		fmt.Println()
	}

	installCodex, err := confirmStep("Install Codex / OpenAI skill?", true)
	if err != nil {
		return err
	}
	if installCodex {
		if err := installCodexSkill(); err != nil {
			fmt.Printf("Codex skill install failed: %v\n", err)
		}
		fmt.Println()
	}

	installCompletion, err := confirmStep("Install shell completion?", true)
	if err != nil {
		return err
	}
	if installCompletion {
		if err := installCompletionForCurrentShell(root); err != nil {
			fmt.Printf("Completion install failed: %v\n", err)
		}
	}

	fmt.Println()
	fmt.Println("Setup done. Try: pintomind me")
	return nil
}

// promptAndVerifyConnection prompts the user for connection details and verifies
// the API key. Loops until verification succeeds or the user aborts.
func promptAndVerifyConnection(cfg *config.Config) (name, baseURL, apiKey, identity string, err error) {
	name = "default"
	if _, exists := cfg.Domains["default"]; exists {
		name = "default-2"
	}

	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Connection name").Value(&name).Validate(huh.ValidateNotEmpty()),
		),
	).Run(); err != nil {
		return "", "", "", "", err
	}

	baseURL = defaultSetupURL
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("API URL").
				Description("Press Enter to continue, or edit to use a custom URL.").
				Value(&baseURL).
				Validate(huh.ValidateNotEmpty()),
		),
	).Run(); err != nil {
		return "", "", "", "", err
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")

	for {
		apiKey = ""
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("API key").
					EchoMode(huh.EchoModePassword).
					Value(&apiKey).
					Validate(huh.ValidateNotEmpty()),
			),
		).Run(); err != nil {
			return "", "", "", "", err
		}

		fmt.Printf("\nVerifying %s ...\n", baseURL)
		ident, vErr := verifyAPIKey(baseURL, apiKey)
		if vErr == nil {
			printVerifySuccess(ident)
			return name, baseURL, apiKey, formatIdentity(ident), nil
		}
		fmt.Printf("Verification failed: %v\n\n", vErr)

		var retry bool
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().Title("Try again with a different key?").Value(&retry),
			),
		).Run(); err != nil {
			return "", "", "", "", err
		}
		if !retry {
			return "", "", "", "", huh.ErrUserAborted
		}
	}
}

type tokenIdentity struct {
	User    string
	Account string
	Token   string
}

func verifyAPIKey(baseURL, apiKey string) (tokenIdentity, error) {
	client := api.New(baseURL, apiKey)
	var result map[string]any
	if err := client.Get("/me", nil, &result); err != nil {
		return tokenIdentity{}, err
	}

	var ident tokenIdentity
	if user, ok := result["user"].(map[string]any); ok {
		ident.User, _ = user["name"].(string)
	}
	if acc, ok := result["account"].(map[string]any); ok {
		ident.Account, _ = acc["name"].(string)
	}
	if tok, ok := result["api_key"].(map[string]any); ok {
		ident.Token, _ = tok["name"].(string)
	}
	return ident, nil
}

func formatIdentity(ident tokenIdentity) string {
	var parts []string
	if ident.User != "" {
		parts = append(parts, "User: "+ident.User)
	}
	if ident.Account != "" {
		parts = append(parts, "Account: "+ident.Account)
	}
	if ident.Token != "" {
		parts = append(parts, "Token: "+ident.Token)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

func printVerifySuccess(ident tokenIdentity) {
	greeting := "Hello!"
	if first := firstName(ident.User); first != "" {
		greeting = "Hello, " + first + "!"
	}

	fmt.Println()
	fmt.Println("  ✨  Success! Your API key works.")
	fmt.Println()
	fmt.Println("  " + greeting + " You're connected as:")
	if ident.User != "" {
		fmt.Printf("    • User:    %s\n", ident.User)
	}
	if ident.Account != "" {
		fmt.Printf("    • Account: %s\n", ident.Account)
	}
	if ident.Token != "" {
		fmt.Printf("    • Token:   %s\n", ident.Token)
	}
	fmt.Println()
}

func firstName(fullName string) string {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return ""
	}
	if i := strings.IndexByte(fullName, ' '); i > 0 {
		return fullName[:i]
	}
	return fullName
}

func confirmStep(title string, defaultValue bool) (bool, error) {
	value := defaultValue
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Title(title).Value(&value),
		),
	).Run(); err != nil {
		if isAborted(err) {
			return false, nil
		}
		return false, err
	}
	return value, nil
}

func installCompletionForCurrentShell(root *cobra.Command) error {
	shell := detectShell()
	if shell == "" {
		fmt.Println("Could not detect shell — skipping completion install.")
		fmt.Println("Run later: pintomind setup completion [bash|zsh|fish]")
		return nil
	}
	switch shell {
	case "bash":
		return installBash(root)
	case "zsh":
		return installZsh(root)
	case "fish":
		return installFish(root)
	default:
		fmt.Printf("Shell %q not supported for completion — skipping.\n", shell)
		return nil
	}
}
