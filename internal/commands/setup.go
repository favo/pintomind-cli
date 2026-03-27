package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/assets"
)

func NewSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up integrations for pintomind",
	}
	cmd.AddCommand(newSetupClaudeCmd())
	return cmd
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
