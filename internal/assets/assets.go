// Package assets embeds static files that are distributed with the binary.
// Keep pintomind-skill.md in sync with .claude/skills/pintomind.md.
package assets

import _ "embed"

//go:embed pintomind-skill.md
var ClaudeSkill []byte
