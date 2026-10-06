package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestElementTarget(t *testing.T) {
	if got := elementTarget("@1/0/2"); got["path"] != "@1/0/2" || got["selector"] != nil {
		t.Errorf("paths should be sent as path, got %v", got)
	}
	if got := elementTarget(".post.visible"); got["selector"] != ".post.visible" || got["path"] != nil {
		t.Errorf("anything else should be sent as selector, got %v", got)
	}
}

func TestPostState(t *testing.T) {
	current := 2
	left := 4
	area := debugArea{Name: "A", Current: &current}

	cases := []struct {
		post debugPost
		want string
	}{
		{debugPost{ID: 1, Ready: true, Scheduled: true}, "-"},
		{debugPost{ID: 2, Ready: true, Scheduled: true, Remaining: &left}, "showing, 4s left"},
		{debugPost{ID: 3, Ready: false, Scheduled: false}, "not ready, hidden by schedule"},
		{debugPost{ID: 4, Ready: true, Scheduled: true, Hidden: true}, "hidden from console"},
	}
	for _, c := range cases {
		if got := postState(area, c.post); got != c.want {
			t.Errorf("postState(%d) = %q, want %q", c.post.ID, got, c.want)
		}
	}
}

func TestEvalCode(t *testing.T) {
	if code, _ := evalCode([]string{"42", "document.title"}, ""); code != "document.title" {
		t.Errorf("code from argument, got %q", code)
	}

	file := filepath.Join(t.TempDir(), "probe.js")
	if err := os.WriteFile(file, []byte("return 1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _ := evalCode([]string{"42"}, file); code != "return 1" {
		t.Errorf("code from file, got %q", code)
	}

	if _, err := evalCode([]string{"42"}, ""); err == nil {
		t.Error("missing code should be an error")
	}
}

func TestHumanBytesAndTruncate(t *testing.T) {
	if got := humanBytes(730675); got != "713.5 kB" {
		t.Errorf("humanBytes = %q", got)
	}
	if got := truncate("Stoltenberg før budsjettet", 12); got != "Stoltenberg…" {
		t.Errorf("truncate = %q", got)
	}
}
