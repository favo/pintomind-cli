package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func NewUpdateCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update pintomind CLI to the latest release",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return selfUpdate(version)
		},
	}
}

func selfUpdate(currentVersion string) error {
	latest, err := latestGitHubRelease("favo", "pintomind-cli")
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	current := "v" + currentVersion
	if currentVersion == "dev" {
		current = "dev"
	}

	if latest == current {
		fmt.Printf("Already up to date (%s).\n", current)
		return nil
	}

	fmt.Printf("Updating %s → %s...\n", current, latest)

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		realPath = execPath
	}

	url := releaseDownloadURL(latest)
	fmt.Printf("Downloading %s...\n", url)

	// Download to temp file in same dir to avoid cross-device rename
	tmpFile, err := os.CreateTemp(filepath.Dir(realPath), ".pintomind-update-*")
	if err != nil {
		// Fallback to system temp dir
		tmpFile, err = os.CreateTemp("", "pintomind-update-*")
		if err != nil {
			return fmt.Errorf("creating temp file: %w", err)
		}
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("downloading update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		tmpFile.Close()
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing update: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	if err := os.Rename(tmpPath, realPath); err != nil {
		return fmt.Errorf("replacing binary at %s: %w", realPath, err)
	}

	// Invalidate update-check cache so next run sees the new version
	invalidateUpdateCache()

	fmt.Printf("Updated to %s.\n", latest)
	return nil
}

// CheckForUpdate prints a notice to stderr if a newer version is available.
// At most one GitHub request per 24 hours; result is cached locally.
func CheckForUpdate(version string) {
	if version == "dev" {
		return
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return
	}
	cacheFile := filepath.Join(cacheDir, "pintomind", "update-check")

	var latest string

	if info, err := os.Stat(cacheFile); err == nil && time.Since(info.ModTime()) < 24*time.Hour {
		data, err := os.ReadFile(cacheFile)
		if err != nil || len(data) == 0 {
			return
		}
		latest = strings.TrimSpace(string(data))
	} else {
		client := &http.Client{Timeout: 3 * time.Second}
		latest, err = latestGitHubReleaseWithClient("favo", "pintomind-cli", client)
		if err != nil {
			return
		}
		_ = os.MkdirAll(filepath.Dir(cacheFile), 0755)
		_ = os.WriteFile(cacheFile, []byte(latest), 0644)
	}

	current := "v" + version
	if latest != "" && latest != current {
		fmt.Fprintf(os.Stderr, "Update available: %s → %s  (run: pintomind update)\n", current, latest)
	}
}

func invalidateUpdateCache() {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return
	}
	_ = os.Remove(filepath.Join(cacheDir, "pintomind", "update-check"))
}

func releaseDownloadURL(tag string) string {
	return fmt.Sprintf(
		"https://github.com/favo/pintomind-cli/releases/download/%s/pintomind_%s_%s",
		tag, runtime.GOOS, runtime.GOARCH,
	)
}
