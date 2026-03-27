package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// NewCompletionInstallCmd returns an "install" subcommand to be added to
// Cobra's auto-generated "completion" command.
func NewCompletionInstallCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "install [bash|zsh|fish]",
		Short: "Install shell completion for the current user (no root required)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := detectShell()
			if len(args) == 1 {
				shell = args[0]
			}
			if shell == "" {
				return fmt.Errorf("could not detect shell — pass it explicitly: pintomind completion install [bash|zsh|fish]")
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

func detectShell() string {
	shell := os.Getenv("SHELL")
	return filepath.Base(shell)
}

func installBash(root *cobra.Command) error {
	dir := filepath.Join(os.Getenv("HOME"), ".local", "share", "bash-completion", "completions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	dest := filepath.Join(dir, "pintomind")

	var buf bytes.Buffer
	if err := root.GenBashCompletion(&buf); err != nil {
		return err
	}
	if err := os.WriteFile(dest, buf.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Printf("Installed bash completion to %s\n\n", dest)
	fmt.Println("This location is auto-loaded by bash-completion 2.x on most distros.")
	fmt.Println("If tab completion is not active yet, add this to your ~/.bashrc:")
	fmt.Printf("  source %s\n", dest)
	return nil
}

func installZsh(root *cobra.Command) error {
	dir := filepath.Join(os.Getenv("HOME"), ".zfunc")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	dest := filepath.Join(dir, "_pintomind")

	var buf bytes.Buffer
	if err := root.GenZshCompletion(&buf); err != nil {
		return err
	}
	if err := os.WriteFile(dest, buf.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Printf("Installed zsh completion to %s\n\n", dest)
	fmt.Println("Make sure your ~/.zshrc contains:")
	fmt.Println("  fpath=(~/.zfunc $fpath)")
	fmt.Println("  autoload -U compinit && compinit")
	fmt.Println("\nThen reload your shell: exec zsh")
	return nil
}

func installFish(root *cobra.Command) error {
	dir := filepath.Join(os.Getenv("HOME"), ".config", "fish", "completions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	dest := filepath.Join(dir, "pintomind.fish")

	var buf bytes.Buffer
	if err := root.GenFishCompletion(&buf, true); err != nil {
		return err
	}
	if err := os.WriteFile(dest, buf.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Printf("Installed fish completion to %s\n\n", dest)
	fmt.Println("Fish auto-loads completions from this directory — no further setup needed.")
	fmt.Println("Reload completions in a running session with: exec fish")
	return nil
}
