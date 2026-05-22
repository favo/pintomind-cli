# pintomind-cli

Command-line interface for the [Pintomind / Infoskjermen](https://infoskjermen.no) API. Control your screens, channels, resources, media, and themes from the terminal or from AI-agent skills.

## Installation

### One-line install (Linux and macOS)

```bash
curl -fsSL https://dl.pintomind.com/pintomind-cli/install.sh | sh
```

Downloads the pre-built binary for your OS and architecture to `~/.local/bin/pintomind`. Override the install directory with `INSTALL_DIR`:

```bash
INSTALL_DIR=/usr/local/bin curl -fsSL https://dl.pintomind.com/pintomind-cli/install.sh | sh
```

### Build from source

Requires [Go](https://go.dev/) 1.22+ (or `mise install go`):

```bash
git clone https://github.com/favo/pintomind-cli
cd pintomind-cli
make install
```

### Verify and update

```bash
pintomind version
pintomind version --check   # check for a newer release on GitHub
pintomind update            # self-update to the latest release
```

---

## First-time setup

The fastest way to get started is the guided wizard — it walks you through adding a connection, verifies your API key, and (optionally) installs the AI skills and shell completion:

```bash
pintomind setup init
```

That's it. You can re-run any individual step later:

```bash
pintomind setup claude       # install the Claude Code skill
pintomind setup codex        # install the Codex / ChatGPT-compatible skill
pintomind setup completion   # install shell tab-completion
pintomind setup all          # all three at once (no connection prompt)
```

If you'd rather add the connection by hand, see [Connections](#connections) below.

---

## Connections

Config is stored in `~/.config/pintomind/config.json`. You can add multiple connections with separate API keys — useful for keeping production and development environments apart.

> Tip: `pintomind setup init` is the easiest way to add your first connection. The commands below are for managing connections after that.

### Add a connection

```bash
pintomind connection add main sk-your-key
```

The first connection added becomes the default. You can add more connections if you have multiple api keys

```bash
pintomind connection add alternative-account sk-seondary-key
```

### Show active connection

```bash
pintomind connection show
```

### List connections

```bash
pintomind connection list
```

The active default is marked with `*`.

### Switch default connection

```bash
pintomind connection use alternative-account
```

### Remove a connection

```bash
pintomind connection remove alternative-account
```

### Override connection per command

Use `--connection` to target a specific connection without changing the default:

```bash
pintomind --connection alternative-account screens list
```

---

## Global flags

| Flag | Description |
|------|-------------|
| `--connection <name>` | Override active connection for this command |
| `--json` | Output raw JSON (useful for scripting and piping to `jq`) |
| `--verbose` / `-v` | Print HTTP request URL and response status to stderr |

---

## Commands

### Quick publish

Upload a file or URL and publish it as an image post in one command — use `posts create image` with `--channel-id`:

```bash
pintomind posts create image --image ./photo.jpg --channel-id 7 --name "Lobby photo"
pintomind posts create image --image https://example.com/banner.jpg --channel-id 7 --channel-id 8
pintomind posts create image --image ./photo.jpg --channel-id 7 --duration 10 --full-screen
pintomind posts create image --image ./flyer.jpg --channel-id 7 --until 2026-06-30T17:00
```

Uploads the file, waits for processing, creates the post, and publishes it. Use `--until <YYYY-MM-DD[THH:MM]>` to auto-expire (unpublishes when the time is reached; for other expiry actions use `posts schedule set` after creation). See the [Posts](#posts) section for full flag reference.

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

### Posts

```bash
pintomind posts list
pintomind posts list --type image
pintomind posts list --type feed,calendar   # comma-separated types
pintomind posts list --archived
pintomind posts list --sort-by created_at:desc
pintomind posts show <id>
pintomind posts update <id> --data '{"name":"Updated name"}'
pintomind posts delete <id>
pintomind posts delete <id> --force
```

**Create posts with typed subcommands:**

Each subcommand accepts `--channel-id` (repeatable), `--area`, `--full-screen`, `--background-media-box <id>` (MediaBox id used as the post background — create one first with `pintomind media-boxes create ...`), plus `--until <YYYY-MM-DD[THH:MM]>` to auto-expire (unpublish) the post after creation.

```bash
# Image post — upload files and/or reference existing media IDs
pintomind posts create image --name "Gallery" --image ./a.jpg --image ./b.jpg --channel-id 7
pintomind posts create image --name "Promo" --media 42 --media 43 --duration 10
pintomind posts create image --name "Mix" --image ./photo.jpg --media 55 --channel-id 7
pintomind posts create image --image ./deck.pdf  # PDF is auto-extracted into pages
pintomind posts create image --image ./photo.jpg --template fullpane  # center | maxpane | fullpane | image_grid

# Plain text post
pintomind posts create plain --name "Welcome" --heading "<p>Hello</p>" --channel-id 7
pintomind posts create plain --name "Msg" --body "<p>Content</p>" --body-alignment center --body-fontsize 150

# Feed post (creates an RSS/Atom resource automatically)
pintomind posts create feed --name "News" --url https://rss.example.com --channel-id 7

# Calendar post (creates a calendar resource from an iCal/WebCal URL automatically)
pintomind posts create calendar --name "Events" --url https://calendar.example.com/feed.ics --channel-id 7

# Iframe post — embed a webpage, raw HTML, or a remote image
pintomind posts create iframe --name "Dashboard" --url https://dashboard.example.com --channel-id 7
pintomind posts create iframe --name "Announcement" --html "<h1>Hello world</h1>" --channel-id 7
pintomind posts create iframe --name "Banner" --image https://example.com/banner.png --channel-id 7

# HTML shortcut — iframe post displaying inline HTML (same as iframe --html)
pintomind posts create html --name "Notice" --html "<h1>Closed today</h1>" --channel-id 7

# Poster post
pintomind posts create poster --name "Spring campaign" --source-id <template-id> --channel-id 7
pintomind posts create poster --name "Brand poster" --source-id <template-id> --color-palette-id 5
```

**Raw create (any type, full JSON control):**

```bash
pintomind posts create --type image --data '{"name":"Spring","duration_per_item":7,"images":[{"media_id":42}],"template":"fullpane"}'
pintomind posts create --type plain --data '{"heading":"<p>Hello</p>","body":"<p>World</p>","media_box_ids":[201,202],"background_id":301}'
pintomind posts create --type poster --source-id <template-id> --data '{"name":"My poster"}'
```

Image templates: `center` (centered, full image), `maxpane` (fill, full image), `fullpane` (fill, cropped), `image_grid` (mosaic). The legacy `aspect_ratio` / `grid_layout` / `variation` fields are no longer accepted.

Any post type accepts `background_id` (a MediaBox id) as a post background. Pass `null` in `posts update --data` to clear.

**Publications — manage where a post is displayed:**

```bash
pintomind posts publications <post-id>                          # List a post's channel publications
pintomind posts publish <post-id> --channel-id 7                # Publish to one channel
pintomind posts publish <post-id> --channel-id 7 --channel-id 12  # Publish to multiple channels in one call
pintomind posts publish <post-id> --channel-id 7 --area F11
pintomind posts publish <post-id> --channel-id 7 --full-screen
pintomind posts unpublish <post-id>                             # Unpublish from ALL channels
pintomind posts unpublish <post-id> --channel-id 7              # Unpublish from specific channel(s)
pintomind posts unpublish <post-id> --channel-id 7 --channel-id 12 --force
```

**Time schedule — control when a post is visible:**

Each post can carry one schedule combining recurring weekly windows and/or concrete date ranges. Times are evaluated in each channel's local time zone.

```bash
pintomind posts schedule get <post-id>
pintomind posts schedule set <post-id> \
  --recurring-rule 'from_wday=1,to_wday=5,from_time=08:00,to_time=18:00' \
  --period 'from_date=2026-06-01,to_date=2026-06-30,from_time=09:00,to_time=17:00' \
  --expiry-action archive
pintomind posts schedule set <post-id> --clear-periods   # empty just that section
pintomind posts schedule clear <post-id>                 # remove the schedule entirely
```

Flag syntax for `set`:

- `--period 'from_date=YYYY-MM-DD,to_date=YYYY-MM-DD[,from_time=HH:MM,to_time=HH:MM]'` (repeatable)
- `--recurring-rule 'from_wday=N,to_wday=N,from_time=HH:MM,to_time=HH:MM[,weeks=all|odd|even]'` (repeatable; `wday` 0=Sun–6=Sat)
- `--expiry-action delete|archive|unpublish` — behavior when the schedule has fully expired

For a quick "expire at X" use `--until` on `posts create *` / `publish` instead. See `pintomind posts schema time-schedule` for the JSON schema.

### Resources

```bash
pintomind resources list
pintomind resources list --type feed
pintomind resources list --type feed,calendar   # comma-separated types
pintomind resources list --page 2 --per-page 50
pintomind resources show <id>
pintomind resources stats
pintomind resources refresh <id>         # For external resources only
```

**Create resources with typed subcommands:**

```bash
pintomind resources create text --label "Greeting" --text "Hello world"
pintomind resources create feed --url https://rss.example.com --title "News"
pintomind resources create calendar --url https://calendar.example.com/feed.ics --color "#FF5733"
pintomind resources create external-webpage --url https://example.com
pintomind resources create external-image --url https://example.com/banner.png
pintomind resources create qr-code --url https://example.com --label-text "Scan me"
pintomind resources create html --html-code "<h1>Hello</h1>" --title "Notice"
pintomind resources create location --name "Oslo" --lat 59.9139 --lon 10.7522 --country-code NO --timezone Europe/Oslo
pintomind resources create youtube --url "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
```

**Raw create (any type, full JSON control):**

```bash
pintomind resources create --type text --data '{"label":"Hello","text":"World"}'
```

**Update a resource:**

```bash
pintomind resources update <id> --data '{"label":"Updated label"}'
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

**Upload a file** (the CLI handles checksum, direct upload, and media record creation):

```bash
pintomind media upload <collection-id> ./photo.jpg --name "Lobby photo"
pintomind media upload <collection-id> https://example.com/photo.jpg --name "Lobby photo"
pintomind media upload <collection-id> ./deck.pdf --extract-pages
```

Uploads return a task immediately (`202 Accepted`). Use `--wait` to block until processing finishes and print the resulting media IDs:

```bash
pintomind media upload <collection-id> ./deck.pdf --extract-pages --wait
```

For advanced flows, claim an already-created direct upload signed ID:

```bash
pintomind media create <collection-id> --source <signed-id> --name "Uploaded file"
pintomind media create <collection-id> --source <signed-id> --extract-pages --wait
```

### Tasks

Media uploads are processed asynchronously. `posts create image` waits automatically. For raw uploads (`media upload` / `media create` without `--wait`), inspect or wait for the task manually:

```bash
pintomind tasks show <id>    # Current state: pending / processing / completed / failed
pintomind tasks wait <id>    # Block until done, print resulting media IDs
```

Task responses include `id`, `status`, `progress` (0–100), `result.media_ids`, and `error`.

### Poster Templates

Poster templates are poster posts shared via the account's content networks that can be duplicated into new account-owned poster posts.

```bash
pintomind poster-templates list
pintomind poster-templates list --sort-by name
pintomind poster-templates show <id>   # includes raw_poster_data
```

### Poster Posts

Poster posts are backed by a `PosterPageResource` that holds the design data.

**Create from a network template (easiest):**

```bash
pintomind posts create poster --name "Spring campaign" --source-id <template-id> --channel-id 7
pintomind posts create poster --name "Brand poster" --source-id <template-id> --color-palette-id 5
```

**Create from scratch:**

```bash
# Create the post, then update the backing resource with poster_data
pintomind posts create --type poster --data '{"name": "Lobby poster"}'
pintomind resources update <resource-id> --data '{"poster_data": "{\"backgroundFill\":\"#ffffff\",\"templateDimensions\":{},\"templates\":{},\"aspectRatio\":\"16/9\"}","color_palette_id": 123}'
```

**Update poster content:**

```bash
pintomind resources update <resource-id> --data '{"poster_data": "<json-string>"}'
```

### Media Boxes

Media boxes are reusable visual slots for plain posts. Create boxes first, then pass their IDs to a plain post in `media_box_ids` slot order. The API picks the plain-post grid variation from the number of boxes.

```bash
pintomind media-boxes list
pintomind media-boxes list --type media,emoji
pintomind media-boxes list --post-id <post-id>
pintomind media-boxes show <id>

# Create boxes
pintomind media-boxes create media --media-id 42
pintomind media-boxes create media --media-id 42 --background-size cover --x 0.5 --y 0.5
pintomind media-boxes create icon --icon-name rocket-launch --icon-type regular
pintomind media-boxes create emoji --emoji '✨'
pintomind media-boxes create gif --gif-id xT9IgG50Fb7Mi0only
pintomind media-boxes create unsplash --photo-id abc123
pintomind media-boxes create qr-code --url https://example.com

# Update or delete boxes
pintomind media-boxes update <id> --data '{"icon_name":"fire"}'
pintomind media-boxes delete <id>
pintomind media-boxes delete <id> --force

# Attach boxes to a plain post
pintomind posts create --type plain --data '{
  "name":"Welcome",
  "heading":"<p>Hello</p>",
  "body":"<p>Welcome to the office</p>",
  "media_box_ids":[201,202]
}'
```

Media box types: `media` (requires `media_id`), `icon` (requires `icon_name`, usually `icon_type`), `emoji` (requires `emoji`), `gif` (requires `gif_id`, `gif_url`), `unsplash` (requires `photo_url`). Inspect exact fields with `pintomind schemas show media_box_image`, `media_box_icon`, `media_box_emoji`, `media_box_gif`, or `media_box_unsplash`.

Browse available icon names:

```bash
pintomind icons
pintomind icons --json | jq '[.[] | select(.category == "communication")]'
```

### Schemas

Inspect the API schema for resource types (useful for knowing which fields are valid in `--data`):

```bash
pintomind schemas list
pintomind schemas show <id>
pintomind schemas show text
pintomind schemas show feed
pintomind schemas show calendar
```

### Themes

```bash
pintomind themes list
pintomind themes list --page 2
pintomind themes show <id>
pintomind themes stats
```

### Color Palettes

```bash
pintomind color-palettes list   # alias: pintomind palettes list
pintomind color-palettes show <id>
pintomind color-palettes create --data '{"name":"Brand","colors":["#FF0000","#00FF00"]}'
pintomind color-palettes update <id> --data '{"name":"Brand v2"}'
pintomind color-palettes delete <id>
pintomind color-palettes stats
```

### Font Families

```bash
pintomind font-families list   # alias: pintomind fonts list
pintomind font-families list --type remote_css
pintomind font-families show <id>
pintomind font-families stats

# Remote CSS font (e.g. Google Fonts)
pintomind font-families create remote-css --name "Inter" --url "https://fonts.googleapis.com/css2?family=Inter:wght@400;700"

# Upload font files (.ttf/.otf/.woff/.woff2)
pintomind font-families create uploaded --name "MyFont" --font-normal ./MyFont-Regular.ttf --font-bold ./MyFont-Bold.ttf

pintomind font-families update <id> --name "Renamed"
pintomind font-families delete <id>
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

Upload a PDF, wait for page extraction, then create an image post from all pages:

```bash
pintomind posts create image --image ./brochure.pdf --name "Brochure" --channel-id 7
```

Inspect valid fields for a resource type:

```bash
pintomind schemas show text
pintomind schemas show location
```

---

## AI Agent Skills

The Pintomind skill is embedded in the binary.

For Claude Code:

```bash
pintomind setup claude
```

This writes `~/.claude/skills/pintomind/SKILL.md`, making `/pintomind` available as a slash command in any Claude Code session. Use `--force` to overwrite an existing installation.

For Codex and ChatGPT/OpenAI-compatible agents:

```bash
pintomind setup codex
```

This writes `SKILL.md` and `agents/openai.yaml` to `$CODEX_HOME/skills/pintomind`, or `~/.codex/skills/pintomind` when `CODEX_HOME` is unset. The command also works as `pintomind setup openai` or `pintomind setup chatgpt`.

Once installed, agents can interact with your screens on your behalf — listing screens, sending commands, switching channels, uploading media, creating posts, and more.
