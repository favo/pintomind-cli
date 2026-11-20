package cli

import (
	"strings"
	"testing"
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
	} {
		args := strings.Fields(path)
		cmd, _, err := root.Find(args)
		if err != nil || cmd.Name() != args[len(args)-1] {
			t.Errorf("command %q is not registered", path)
		}
	}
}
