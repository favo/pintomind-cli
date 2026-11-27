// Package assets embeds static files that are distributed with the binary.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed all:skill
var skillFiles embed.FS

// Skill is the pintomind skill folder: SKILL.md, references/ and agents/openai.yaml.
// Keep it usable across Claude Code and Codex/OpenAI skill installs.
var Skill, _ = fs.Sub(skillFiles, "skill")
