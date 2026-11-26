package cli

import (
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/assets"
)

func TestIsUpdateCmdOnlyMatchesTopLevelUpdate(t *testing.T) {
	root := NewRootCmd()

	topLevelUpdate, _, err := root.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	if !isUpdateCmd(topLevelUpdate) {
		t.Fatal("expected top-level update command to match")
	}

	postsUpdate, _, err := root.Find([]string{"posts", "update"})
	if err != nil {
		t.Fatal(err)
	}
	if isUpdateCmd(postsUpdate) {
		t.Fatal("posts update must not be treated as the top-level update command")
	}
}

func TestNewCommandsAreRegistered(t *testing.T) {
	root := NewRootCmd()
	for _, path := range []string{
		"login", "logout", "connection use-account", "connection use-network", "accounts", "networks", "network",
		"channels create", "channels update", "channels delete", "channels move-post", "channels preview",
		"channels search", "channels layouts", "channels sharing show", "channels sharing start",
		"channels sharing update", "channels sharing stop", "channels sharing codes add",
		"channels sharing invites list", "channels sharing invites resend",
		"screens delete", "screens schedule show", "screens schedule set", "screens schedule clear",
		"screens schedule templates list", "screens schedule templates create",
		"screens provisioning create", "screens provisioning revoke", "screens rain", "screens stop-effects",
		"posts archive", "posts restore", "posts preview", "posts plain-layouts",
		"resources teams", "emojis", "unsplash", "giphy", "icons",
		"themes areas", "themes set-areas", "themes set-effect", "themes preview-url",
	} {
		args := strings.Fields(path)
		cmd, _, err := root.Find(args)
		if err != nil || cmd.Name() != args[len(args)-1] {
			t.Errorf("command %q is not registered", path)
		}
	}
}

func TestEveryCommandIsInTheSkill(t *testing.T) {
	var text strings.Builder
	fs.WalkDir(assets.Skill, ".", func(path string, entry fs.DirEntry, err error) error {
		if err == nil && strings.HasSuffix(path, ".md") {
			content, _ := fs.ReadFile(assets.Skill, path)
			text.Write(content)
		}
		return nil
	})
	skill := text.String()

	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			if child.Hidden || child.Name() == "help" || child.Name() == "completion" {
				continue
			}
			path := strings.TrimPrefix(child.CommandPath(), "pintomind ")
			if !mentioned(skill, path) {
				t.Errorf("%q is not mentioned in the skill", path)
			}
			walk(child)
		}
	}
	walk(NewRootCmd())
}

// mentioned finds "pintomind <path>", "`<path>" or an alternation such as "screens snow|rain".
func mentioned(skill, path string) bool {
	if strings.Contains(skill, "pintomind "+path) || strings.Contains(skill, "`"+path) {
		return true
	}
	parent, name := path[:max(strings.LastIndex(path, " "), 0)], path[strings.LastIndex(path, " ")+1:]
	pattern := regexp.MustCompile(regexp.QuoteMeta("pintomind "+parent) + ` ([\w|-]+)`)
	for _, match := range pattern.FindAllStringSubmatch(skill, -1) {
		if slices.Contains(strings.Split(match[1], "|"), name) {
			return true
		}
	}
	return false
}
