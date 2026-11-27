package assets

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

func TestSkillLinksEveryReferenceAndEveryLinkResolves(t *testing.T) {
	skill, err := fs.ReadFile(Skill, "SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(skill), "---\nname: pintomind\ndescription: ") {
		t.Error("SKILL.md must start with the name and description frontmatter")
	}

	linked := map[string]bool{}
	for _, match := range regexp.MustCompile(`\]\((references/[^)]+)\)`).FindAllStringSubmatch(string(skill), -1) {
		linked[match[1]] = true
		if _, err := fs.Stat(Skill, match[1]); err != nil {
			t.Errorf("SKILL.md links %s, which is not embedded", match[1])
		}
	}

	references, _ := fs.Glob(Skill, "references/*.md")
	for _, reference := range references {
		if !linked[reference] {
			t.Errorf("%s is embedded but SKILL.md never links it, so no agent will load it", reference)
		}
	}
	if len(references) == 0 {
		t.Fatal("no references embedded")
	}
}
