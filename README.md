# pintomind-cli

Command-line interface for the [Pintomind / Infoskjermen](https://infoskjermen.no) API. Control your screens, channels, resources, media, and themes from the terminal or from Claude Code skills.

## Installation

### One-line install (Linux and macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/favo-no/pintomind-cli/main/install.sh | sh
```

Downloads the pre-built binary for your OS and architecture to `~/.local/bin/pintomind`. Override the install directory with `INSTALL_DIR`:

```bash
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/favo-no/pintomind-cli/main/install.sh | sh
```

### Build from source

Requires [Go](https://go.dev/) 1.22+ (or `mise install go`):

```bash
git clone https://github.com/favo-no/pintomind-cli
cd pintomind-cli
make install
```

### Verify

```bash
pintomind version
pintomind version --check   # check for a newer release on GitHub
```

---

## First-time setup

Run all setup steps in one command:

```bash
pintomind setup all
```

Or run them individually:

```bash
pintomind setup claude       # install the Claude Code skill
pintomind setup completion   # install shell tab-completion
```

---

## Configuration

Config is stored in `~/.config/pintomind/config.json`. You can add multiple accounts with separate API keys — useful for keeping production and development environments apart.

### Add an account

```bash
pintomind config add app.infoskjermen.no sk-your-key
```

The first account added becomes the default. The base URL defaults to `https://<name>`. Override it with `--url`:

```bash
pintomind config add develop sk-dev-key --url https://develop.infoskjermen.no
```

### Show active account

```bash
pintomind config show
```

### List configured accounts

```bash
pintomind config list
```

The active default is marked with `*`.

### Switch default account

```bash
pintomind config use develop
```

### Remove an account

```bash
pintomind config remove develop
```

### Override account per command

Use `--account` to target a specific account without changing the default:

```bash
pintomind --account develop screens list
```

---

## Global flags

| Flag | Description |
|------|-------------|
| `--account <name>` | Override active account for this command |
| `--json` | Output raw JSON (useful for scripting and piping to `jq`) |
| `--verbose` / `-v` | Print HTTP request URL and response status to stderr |

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
pintomind screens list --page 2 --per-page 50
pintomind screens show <id>
pintomind screens stats                  # Online/offline counts
pintomind screens watch                  # Refresh screen list every N seconds
pintomind screens watch --interval 10
pintomind screens wait-online <id>       # Block until screen comes online
pintomind screens wait-online <id> --timeout 120
```

**Targeting flags** — all screen action commands accept:

| Flag | Description |
|------|-------------|
| `<id>` | Single screen by ID |
| `--ids 1,2,3` | Comma-separated screen IDs (bulk) |
| `--all` | All screens (fetches IDs automatically, uses bulk endpoint) |

**Commands:**

```bash
pintomind screens reload            [id|--ids ...|--all]
pintomind screens reboot            [id|--ids ...|--all]
pintomind screens clear-cache       [id|--ids ...|--all]
pintomind screens upgrade-firmware  [id|--ids ...|--all]
pintomind screens identify          [id|--ids ...|--all]
pintomind screens toggle-night-mode [id|--ids ...|--all]
```

**Remote control signals:**

```bash
pintomind screens next          [id|--ids ...|--all]
pintomind screens previous      [id|--ids ...|--all]
pintomind screens play          [id|--ids ...|--all]
pintomind screens pause         [id|--ids ...|--all]
pintomind screens toggle-play   [id|--ids ...|--all]
pintomind screens forwards      [id|--ids ...|--all]
pintomind screens backwards     [id|--ids ...|--all]
```

**Effects:**

```bash
pintomind screens confetti-fire      [id|--ids ...|--all]
pintomind screens confetti-fireworks [id|--ids ...|--all]
pintomind screens school-parade      [id|--ids ...|--all]
pintomind screens snow               [id|--ids ...|--all]
```

**Switch channel:**

```bash
pintomind screens set-channel <screen-id> <channel-id>
pintomind screens set-channel --ids 1,2,3 <channel-id>
pintomind screens set-channel --all <channel-id>
```

**Temporary channel override:**

```bash
pintomind screens temp-channel <screen-id> <channel-id> --duration 3600
pintomind screens temp-channel <screen-id> <channel-id> --until 2025-12-31T23:59:00Z
pintomind screens temp-channel --all <channel-id> --duration 1800
pintomind screens temp-channel <screen-id> <channel-id> --toggle
```

### Channels

```bash
pintomind channels list
pintomind channels list --sort-by name
pintomind channels list --page 2 --per-page 50
pintomind channels show <id>
pintomind channels posts <id>            # Requires account token
pintomind channels stats                 # Requires channels:read:stats scope
pintomind channels stats <id>
pintomind channels set-theme <channel-id> <theme-id>
pintomind channels set-theme --all <theme-id>   # Apply to all channels
```

### Resources

```bash
pintomind resources list
pintomind resources list --type text_slide
pintomind resources list --type text_slide,calendar_events   # comma-separated types
pintomind resources list --page 2 --per-page 50
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

### Media Collections

```bash
pintomind media-collections list
pintomind media-collections list --sort-by title
pintomind media-collections list --page 2 --per-page 50
pintomind media-collections show <id>
pintomind media-collections create --title "Campaign photos" --category image
pintomind media-collections update <id> --title "Updated title"
pintomind media-collections delete <id>
pintomind media-collections delete <id> --force
```

Categories are `background`, `logo`, `image`, `video`, and `document`.

### Media

```bash
pintomind media list <collection-id>
pintomind media list <collection-id> --sort-by created_at:desc
pintomind media show <id>
pintomind media update <id> --name "Lobby photo"
pintomind media update <id> --collection-id <target-collection-id>
pintomind media delete <id>
pintomind media delete <id> --force
```

**Upload a file** (the CLI handles checksum, direct upload, and media creation):

```bash
pintomind media upload <collection-id> ./photo.jpg --name "Lobby photo"
pintomind media upload <collection-id> ./deck.pdf --extract-pages
```

For advanced flows, claim an already-created direct upload signed ID:

```bash
pintomind media create <collection-id> --source <signed-id> --name "Uploaded file"
```

### Schemas

Inspect the API schema for resource types (useful for knowing which fields are valid in `--data`):

```bash
pintomind schemas list
pintomind schemas show <id>
```

### Themes

```bash
pintomind themes list
pintomind themes list --page 2
pintomind themes show <id>
pintomind themes stats
```

### Raw API access

Send any API request directly — useful for endpoints not yet covered by a dedicated command:

```bash
pintomind api /screens
pintomind api /screens/42
pintomind api GET /channels?sort_by=name
echo '{"screen":{"command":"reload"}}' | pintomind api PATCH /screens/42
```

`METHOD` defaults to `GET`. For `POST`, `PATCH`, and `PUT`, the JSON body is read from stdin.

---

## Shell completion

Install completions for your user without needing root:

```bash
pintomind setup completion        # auto-detects your shell
pintomind setup completion bash
pintomind setup completion zsh
pintomind setup completion fish
```

**Bash** — writes to `~/.local/share/bash-completion/completions/pintomind`, auto-loaded by bash-completion 2.x. If it doesn't activate immediately, add to `~/.bashrc`:
```bash
source ~/.local/share/bash-completion/completions/pintomind
```

**Zsh** — writes to `~/.zfunc/_pintomind`. Make sure your `~/.zshrc` contains:
```zsh
fpath=(~/.zfunc $fpath)
autoload -U compinit && compinit
```
Then reload: `exec zsh`

**Fish** — writes to `~/.config/fish/completions/pintomind.fish`, auto-loaded immediately.

To just print the completion script (e.g. for manual setup), use `pintomind completion bash|zsh|fish` without `install`.

---

## Scripting examples

Get all screen IDs:

```bash
pintomind screens list --json | jq '[.items[].id]'
```

Reload all online screens:

```bash
pintomind screens reload --all
```

Wait for a screen to come back online after a reboot:

```bash
pintomind screens reboot 42 && pintomind screens wait-online 42
```

Find all channels and pick one by name:

```bash
pintomind channels list --json | jq '.items[] | select(.name | test("lobby"; "i"))'
```

Set a theme on every channel:

```bash
pintomind channels set-theme --all 5
```

Inspect valid fields for `text_slide` resources:

```bash
pintomind schemas show text_slide
```

---

## Claude Code skill

The skill is embedded in the binary. Install it with:

```bash
pintomind setup claude
```

This writes `~/.claude/skills/pintomind/SKILL.md`, making `/pintomind` available as a slash command in any Claude Code session. Use `--force` to overwrite an existing installation.

Once installed, Claude can interact with your screens on your behalf — listing screens, sending commands, switching channels, and more.
