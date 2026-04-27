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
		Long:  "Run all setup steps or individual ones. Installs AI-agent skills and shell completions.",
	}
	cmd.AddCommand(newSetupClaudeCmd())
	cmd.AddCommand(newSetupCodexCmd())
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

	cmd := &cobra.Command{
		Use:   "all",
		Short: "Run all setup steps (Claude/Codex skills + shell completion)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := installClaudeSkill(force); err != nil {
				return err
			}
			fmt.Println()
			if err := installCodexSkill(force); err != nil {
				return err
			}
			fmt.Println()

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
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite installed skills if present")
	return cmd
}

func newSetupClaudeCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Install the pintomind skill for Claude Code",
		Long: `Installs the pintomind skill file to ~/.claude/skills/pintomind/SKILL.md.

Once installed, Claude Code can use the /pintomind slash command to interact
with your screens on your behalf from any project.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := installClaudeSkill(force); err != nil {
				return err
			}
			fmt.Println()
			fmt.Println("You can now use /pintomind in any Claude Code session.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite if already installed")
	return cmd
}

func newSetupCodexCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "codex",
		Aliases: []string{"openai", "chatgpt"},
		Short:   "Install the pintomind skill for Codex and ChatGPT/OpenAI agents",
		Long: `Installs the pintomind skill to $CODEX_HOME/skills/pintomind, or
~/.codex/skills/pintomind when CODEX_HOME is unset.

The install includes SKILL.md plus agents/openai.yaml metadata for OpenAI
skill UIs such as Codex and ChatGPT-compatible agents.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := installCodexSkill(force); err != nil {
				return err
			}
			fmt.Println()
			fmt.Println("You can now invoke the skill as $pintomind in Codex/OpenAI agents.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite if already installed")
	return cmd
}

func installClaudeSkill(force bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home directory: %w", err)
	}

	skillDir := filepath.Join(home, ".claude", "skills", "pintomind")
	dest := filepath.Join(skillDir, "SKILL.md")

	wrote, err := writeSkillFile(dest, assets.PintomindSkill, force)
	if err != nil {
		return fmt.Errorf("installing Claude skill: %w", err)
	}

	stale := filepath.Join(home, ".claude", "skills", "pintomind.md")
	if _, err := os.Stat(stale); err == nil {
		_ = os.Remove(stale)
	}

	if wrote {
		fmt.Printf("Installed Claude skill to %s\n", dest)
	} else {
		fmt.Printf("Claude skill already installed at %s\n", dest)
		fmt.Println("Use --force to overwrite.")
	}
	return nil
}

func installCodexSkill(force bool) error {
	root, err := codexHome()
	if err != nil {
		return err
	}

	skillDir := filepath.Join(root, "skills", "pintomind")
	skillDest := filepath.Join(skillDir, "SKILL.md")
	metadataDest := filepath.Join(skillDir, "agents", "openai.yaml")

	wroteSkill, err := writeSkillFile(skillDest, assets.PintomindSkill, force)
	if err != nil {
		return fmt.Errorf("installing Codex skill: %w", err)
	}
	wroteMetadata, err := writeSkillFile(metadataDest, assets.OpenAIMetadata, force)
	if err != nil {
		return fmt.Errorf("installing OpenAI skill metadata: %w", err)
	}

	if wroteSkill || wroteMetadata {
		fmt.Printf("Installed Codex/OpenAI skill to %s\n", skillDir)
	} else {
		fmt.Printf("Codex/OpenAI skill already installed at %s\n", skillDir)
		fmt.Println("Use --force to overwrite.")
	}
	return nil
}

func codexHome() (string, error) {
	if root := os.Getenv("CODEX_HOME"); root != "" {
		return root, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".codex"), nil
}

func writeSkillFile(path string, content []byte, force bool) (bool, error) {
	if _, err := os.Stat(path); err == nil && !force {
		return false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, fmt.Errorf("creating skills directory: %w", err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return false, fmt.Errorf("writing skill file: %w", err)
	}
	return true, nil
}
