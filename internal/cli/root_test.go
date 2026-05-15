package cli

import "testing"

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
