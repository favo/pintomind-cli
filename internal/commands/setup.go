package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/assets"
)

func NewSetupCmd(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up integrations for pintomind",
		Long:  "Run all setup steps or individual ones. Installs the Claude Code skill and shell completions.",
	}
	cmd.AddCommand(newSetupClaudeCmd())
	cmd.AddCommand(newSetupCompletionCmd(root))
	cmd.AddCommand(newSetupAllCmd(root))
	return cmd
}

func newSetupCompletionCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "Install shell completion for the current user",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := detectShell()
			if len(args) == 1 {
				shell = args[0]
			}
			if shell == "" {
				return fmt.Errorf("could not detect shell — pass it explicitly: pintomind setup completion [bash|zsh|fish]")
			}
			switch shell {
			case "bash":
				return installBash(root)
			case "zsh":
				return installZsh(root)
			case "fish":
				return installFish(root)
			default:
				return fmt.Errorf("unsupported shell %q — supported: bash, zsh, fish", shell)
			}
		},
	}
}

func newSetupAllCmd(root *cobra.Command) *cobra.Command {
	var force bool

	return &cobra.Command{
		Use:   "all",
		Short: "Run all setup steps (Claude skill + shell completion)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Install Claude skill
			skillDir := filepath.Join(os.Getenv("HOME"), ".claude", "skills", "pintomind")
			dest := filepath.Join(skillDir, "SKILL.md")

			if _, err := os.Stat(dest); err == nil && !force {
				fmt.Printf("Claude skill already installed at %s\n", dest)
			} else {
				if err := os.MkdirAll(skillDir, 0755); err != nil {
					return fmt.Errorf("creating skills directory: %w", err)
				}
				if err := os.WriteFile(dest, assets.ClaudeSkill, 0644); err != nil {
					return fmt.Errorf("writing skill file: %w", err)
				}
				stale := filepath.Join(os.Getenv("HOME"), ".claude", "skills", "pintomind.md")
				if _, err := os.Stat(stale); err == nil {
					_ = os.Remove(stale)
				}
				fmt.Printf("Installed pintomind skill to %s\n", dest)
			}

			fmt.Println()

			// Install shell completion
			shell := detectShell()
			if shell == "" {
				fmt.Println("Could not detect shell — skipping completion install.")
				fmt.Println("Run: pintomind setup completion [bash|zsh|fish]")
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
				fmt.Println("Run: pintomind setup completion [bash|zsh|fish]")
			}
			return nil
		},
	}
}

func newSetupClaudeCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Install the pintomind skill for Claude Code",
		Long: `Installs the pintomind skill file to ~/.claude/skills/pintomind.md.

Once installed, Claude Code can use the /pintomind slash command to interact
with your screens on your behalf from any project.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			skillDir := filepath.Join(os.Getenv("HOME"), ".claude", "skills", "pintomind")
			dest := filepath.Join(skillDir, "SKILL.md")

			if _, err := os.Stat(dest); err == nil && !force {
				fmt.Printf("Skill already installed at %s\n", dest)
				fmt.Println("Use --force to overwrite.")
				return nil
			}

			if err := os.MkdirAll(skillDir, 0755); err != nil {
				return fmt.Errorf("creating skills directory: %w", err)
			}
			if err := os.WriteFile(dest, assets.ClaudeSkill, 0644); err != nil {
				return fmt.Errorf("writing skill file: %w", err)
			}

			// Remove stale flat file from older installs
			stale := filepath.Join(os.Getenv("HOME"), ".claude", "skills", "pintomind.md")
			if _, err := os.Stat(stale); err == nil {
				_ = os.Remove(stale)
			}

			fmt.Printf("Installed pintomind skill to %s\n\n", dest)
			fmt.Println("You can now use /pintomind in any Claude Code session.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite if already installed")
	return cmd
}
