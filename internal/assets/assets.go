// Package assets embeds static files that are distributed with the binary.
// Keep SKILL.md usable across Claude Code and Codex/OpenAI skill installs.
package assets

import _ "embed"

//go:embed SKILL.md
var PintomindSkill []byte

//go:embed agents/openai.yaml
var OpenAIMetadata []byte
