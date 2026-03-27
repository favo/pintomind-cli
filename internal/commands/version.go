package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

func NewVersionCmd(version string) *cobra.Command {
	var check bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the pintomind CLI version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Printf("pintomind version %s\n", version)
			if !check {
				return nil
			}
			latest, err := latestGitHubRelease("favo-no", "pintomind-cli")
			if err != nil {
				return fmt.Errorf("checking for updates: %w", err)
			}
			if latest == "" {
				fmt.Println("Could not determine latest release.")
				return nil
			}
			current := "v" + version
			if version == "dev" {
				current = "dev"
			}
			if latest == current {
				fmt.Printf("You are up to date (%s).\n", latest)
			} else {
				fmt.Printf("A new version is available: %s (you have %s)\n", latest, current)
				fmt.Println("To upgrade, re-run the install script:")
				fmt.Println("  curl -fsSL https://raw.githubusercontent.com/favo-no/pintomind-cli/main/install.sh | sh")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Check for a newer release on GitHub")
	return cmd
}

func latestGitHubRelease(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.TagName, nil
}
