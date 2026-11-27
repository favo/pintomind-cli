package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/assets"
)

func NewSetupCmd(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up integrations for pintomind",
		Long:  "Run all setup steps or individual ones. Installs AI-agent skills and shell completions.",
	}
	cmd.AddCommand(newSetupInitCmd(root))
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
	return &cobra.Command{
		Use:   "all",
		Short: "Run all setup steps (Claude/Codex skills + shell completion)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := installClaudeSkill(); err != nil {
				return err
			}
			fmt.Println()
			if err := installCodexSkill(); err != nil {
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
}

func newSetupClaudeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "claude",
		Short: "Install the pintomind skill for Claude Code",
		Long: `Installs the pintomind skill to ~/.claude/skills/pintomind: SKILL.md and the
references/ it loads when needed. Re-running it upgrades an earlier install.

Once installed, Claude Code can use the /pintomind slash command to interact
with your screens on your behalf from any project.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := installClaudeSkill(); err != nil {
				return err
			}
			fmt.Println()
			fmt.Println("You can now use /pintomind in any Claude Code session.")
			return nil
		},
	}
}

func newSetupCodexCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "codex",
		Aliases: []string{"openai", "chatgpt"},
		Short:   "Install the pintomind skill for Codex and ChatGPT/OpenAI agents",
		Long: `Installs the pintomind skill to $CODEX_HOME/skills/pintomind, or
~/.codex/skills/pintomind when CODEX_HOME is unset.

The install includes SKILL.md, references/ and agents/openai.yaml metadata for
OpenAI skill UIs such as Codex and ChatGPT-compatible agents. Re-running it
upgrades an earlier install.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := installCodexSkill(); err != nil {
				return err
			}
			fmt.Println()
			fmt.Println("You can now invoke the skill as $pintomind in Codex/OpenAI agents.")
			return nil
		},
	}
}

func installClaudeSkill() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home directory: %w", err)
	}

	skillDir := filepath.Join(home, ".claude", "skills", "pintomind")
	if err := installSkill(skillDir, false); err != nil {
		return fmt.Errorf("installing Claude skill: %w", err)
	}

	stale := filepath.Join(home, ".claude", "skills", "pintomind.md")
	if _, err := os.Stat(stale); err == nil {
		_ = os.Remove(stale)
	}

	fmt.Printf("Installed Claude skill to %s\n", skillDir)
	return nil
}

func installCodexSkill() error {
	root, err := codexHome()
	if err != nil {
		return err
	}

	skillDir := filepath.Join(root, "skills", "pintomind")
	if err := installSkill(skillDir, true); err != nil {
		return fmt.Errorf("installing Codex skill: %w", err)
	}

	fmt.Printf("Installed Codex/OpenAI skill to %s\n", skillDir)
	return nil
}

// installSkill writes the embedded skill folder to dir. references/ is ours alone, so it is
// replaced as a whole and a reference dropped from the binary doesn't linger after an upgrade.
func installSkill(dir string, withAgents bool) error {
	if err := os.RemoveAll(filepath.Join(dir, "references")); err != nil {
		return fmt.Errorf("removing old references: %w", err)
	}
	return fs.WalkDir(assets.Skill, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if !withAgents && strings.HasPrefix(path, "agents/") {
			return nil
		}
		content, err := fs.ReadFile(assets.Skill, path)
		if err != nil {
			return err
		}
		return writeSkillFile(filepath.Join(dir, filepath.FromSlash(path)), content)
	})
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

func writeSkillFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating skills directory: %w", err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("writing skill file: %w", err)
	}
	return nil
}
