# pintomind-cli

Command-line interface for the [Pintomind / Infoskjermen](https://infoskjermen.no) API. Control your screens, channels, resources, and themes from the terminal or from Claude Code skills.

## Installation

### Prerequisites

- [Go](https://go.dev/) 1.22+ (or use [mise](https://mise.jdx.dev/): `mise install go`)

### Install from source

```bash
git clone https://github.com/jonh/pintomind-cli
cd pintomind-cli
make install
```

This builds the binary and installs it to `~/.local/bin/pintomind`. Make sure that directory is on your `$PATH`.

### Verify

```bash
pintomind --version
```

---

## Configuration

Config is stored in `~/.config/pintomind/config.json`. You can add multiple domains with separate API keys — useful for keeping production and development environments apart.

### Add a domain

```bash
pintomind config add infoskjermen.no --api-key sk-your-key
```

The first domain added becomes the default. The base URL defaults to `https://<domain>`. Override it with `--url`:

```bash
pintomind config add develop --api-key sk-dev-key --url https://develop.infoskjermen.no
```

### List configured domains

```bash
pintomind config list
```

The active default is marked with `*`.

### Switch default domain

```bash
pintomind config use develop
```

### Remove a domain

```bash
pintomind config remove develop
```

### Override domain per command

Use `--domain` to target a specific domain without changing the default:

```bash
pintomind --domain develop screens list
```

---

## Global flags

| Flag | Description |
|------|-------------|
| `--domain <name>` | Override active domain for this command |
| `--json` | Output raw JSON (useful for scripting and piping to `jq`) |

---

## Commands

### Identity

```bash
pintomind me           # Show token, user, and account info
pintomind network      # Show network identity and stats (requires network token)
```

### Screens

```bash
pintomind screens list
pintomind screens list --online          # Only online screens
pintomind screens list --offline         # Only offline screens
pintomind screens show <id>
```

**Send a command to a screen:**

```bash
pintomind screens command <id> reload
pintomind screens command <id> reboot
pintomind screens command <id> clear_cache
pintomind screens command <id> upgrade_firmware
pintomind screens command <id> identify
pintomind screens command <id> toggle_night_mode
```

**Bulk commands** (use `--ids`):

```bash
pintomind screens command --ids 1,2,3 reload
```

**Remote control signals:**

```bash
pintomind screens signal <id> next
pintomind screens signal <id> previous
pintomind screens signal <id> play
pintomind screens signal <id> pause
pintomind screens signal <id> toggle_play
```

**Effects** (pass `--effect`):

```bash
pintomind screens signal <id> confetti_fire --effect
pintomind screens signal <id> confetti_fireworks --effect
pintomind screens signal <id> confetti_school_parade --effect
pintomind screens signal <id> snow --effect
```

**Switch channel:**

```bash
pintomind screens set-channel <screen-id> <channel-id>
pintomind screens set-channel --ids 1,2,3 <channel-id>
```

**Temporary channel override:**

```bash
pintomind screens temp-channel <screen-id> <channel-id> --duration 3600
pintomind screens temp-channel <screen-id> <channel-id> --until 2025-12-31T23:59:00Z
pintomind screens temp-channel <screen-id> <channel-id> --toggle
```

### Channels

```bash
pintomind channels list
pintomind channels list --sort-by name
pintomind channels show <id>
pintomind channels posts <id>            # Requires account token
pintomind channels stats                 # Requires channels:read:stats scope
pintomind channels stats <id>
pintomind channels set-theme <channel-id> <theme-id>
```

### Resources

```bash
pintomind resources list
pintomind resources list --type text_slide
pintomind resources show <id>
pintomind resources stats
pintomind resources refresh <id>         # For external resources only
```

**Create a resource:**

```bash
pintomind resources create --type text_slide --data '{"title":"Hello","body":"World"}'
```

**Update a resource:**

```bash
pintomind resources update <id> --data '{"title":"Updated title"}'
```

**Append items** (for resource types that support it):

```bash
pintomind resources append <id> --items '[{"text":"New item"}]'
```

**Delete a resource** (soft-delete on first call, hard-delete on second):

```bash
pintomind resources delete <id>
pintomind resources delete <id> --force  # Skip confirmation prompt
```

### Themes

```bash
pintomind themes list
pintomind themes show <id>
pintomind themes stats
```

---

## Shell completion

```bash
pintomind completion bash   # Bash
pintomind completion zsh    # Zsh
pintomind completion fish   # Fish
```

Follow the instructions printed by each command to enable completion in your shell.

---

## Scripting examples

Get all screen IDs as a list:

```bash
pintomind screens list --json | jq '[.items[].id]'
```

Reload all online screens:

```bash
pintomind screens list --online --json \
  | jq '[.items[].id] | join(",")' -r \
  | xargs -I{} pintomind screens command --ids {} reload
```

Find all channels and pick one by name:

```bash
pintomind channels list --json | jq '.items[] | select(.name | test("lobby"; "i"))'
```

---

## Claude Code skill

A skill file is included at `.claude/skills/pintomind.md`. To make it available as a `/pintomind` slash command in any project, install it globally:

```bash
mkdir -p ~/.claude/skills
cp .claude/skills/pintomind.md ~/.claude/skills/pintomind.md
```

Claude will then know how to use the CLI to interact with your screens on your behalf.
