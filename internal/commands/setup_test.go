package commands

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"favo/pintomind-cli/internal/assets"
)

func TestInstallClaudeSkillUpgradesASingleFileInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	skillDir := filepath.Join(home, ".claude", "skills", "pintomind")
	writeTestFile(t, filepath.Join(skillDir, "SKILL.md"), "old single-file skill")
	writeTestFile(t, filepath.Join(skillDir, "references", "dropped.md"), "a reference no longer shipped")
	writeTestFile(t, filepath.Join(skillDir, "notes.txt"), "the user's own file")
	writeTestFile(t, filepath.Join(home, ".claude", "skills", "pintomind.md"), "older still")

	if err := installClaudeSkill(); err != nil {
		t.Fatal(err)
	}

	assertMatchesEmbedded(t, skillDir, false)
	assertMissing(t, filepath.Join(skillDir, "references", "dropped.md"))
	assertMissing(t, filepath.Join(home, ".claude", "skills", "pintomind.md"))
	assertMissing(t, filepath.Join(skillDir, "agents"))
	if _, err := os.Stat(filepath.Join(skillDir, "notes.txt")); err != nil {
		t.Errorf("a file the CLI doesn't manage was removed: %v", err)
	}
}

func TestInstallCodexSkillIncludesTheAgentsMetadata(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)

	if err := installCodexSkill(); err != nil {
		t.Fatal(err)
	}
	assertMatchesEmbedded(t, filepath.Join(root, "skills", "pintomind"), true)
}

func assertMatchesEmbedded(t *testing.T, dir string, withAgents bool) {
	t.Helper()
	count := 0
	err := fs.WalkDir(assets.Skill, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || (!withAgents && filepath.Dir(path) == "agents") {
			return err
		}
		want, _ := fs.ReadFile(assets.Skill, path)
		got, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil || string(got) != string(want) {
			t.Errorf("%s is not the embedded file (err %v)", path, err)
		}
		count++
		return nil
	})
	if err != nil || count < 2 {
		t.Fatalf("walked %d embedded files: %v", count, err)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("%s should not exist (err %v)", path, err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
