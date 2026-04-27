---
name: pintomind
description: Interact with Pintomind / Infoskjermen screens, channels, resources, media, and themes via the pintomind CLI. Use for ANY Pintomind question or action.
---

You are an expert at using the `pintomind` CLI to interact with the Pintomind / Infoskjermen API.

## Setup

Config is stored in `~/.config/pintomind/config.json`. Multiple accounts are supported.

```bash
pintomind config add app.infoskjermen.no sk-xxx
pintomind config add develop sk-dev --url https://develop.infoskjermen.no
pintomind config use develop
pintomind config list
pintomind config show   # show active account and masked API key
```

## Global Flags

- `--account <name>` — override active account for a single command
- `--json` — output raw JSON
- `--verbose` / `-v` — print HTTP request URL and response status to stderr

## Screens

```bash
pintomind screens list
pintomind screens list --online
pintomind screens list --offline
pintomind screens list --page 2 --per-page 50
pintomind screens show <id>
pintomind screens stats                  # online/offline counts
pintomind screens watch                  # live-refresh every 5s (--interval N)
pintomind screens wait-online <id>       # block until online (--timeout N)
```

### Targeting

All screen action commands accept one of:
- `<id>` — single screen
- `--ids 1,2,3` — comma-separated IDs (bulk endpoint)
- `--all` — all screens (auto-fetches IDs, uses bulk endpoint)

### Commands

```bash
pintomind screens reload            [id|--ids ...|--all]
pintomind screens reboot            [id|--ids ...|--all]
pintomind screens clear-cache       [id|--ids ...|--all]
pintomind screens upgrade-firmware  [id|--ids ...|--all]
pintomind screens identify          [id|--ids ...|--all]
pintomind screens toggle-night-mode [id|--ids ...|--all]
```

### Remote control signals

```bash
pintomind screens next          [id|--ids ...|--all]
pintomind screens previous      [id|--ids ...|--all]
pintomind screens play          [id|--ids ...|--all]
pintomind screens pause         [id|--ids ...|--all]
pintomind screens toggle-play   [id|--ids ...|--all]
pintomind screens forwards      [id|--ids ...|--all]
pintomind screens backwards     [id|--ids ...|--all]
```

### Effects

```bash
pintomind screens confetti-fire      [id|--ids ...|--all]
pintomind screens confetti-fireworks [id|--ids ...|--all]
pintomind screens school-parade      [id|--ids ...|--all]
pintomind screens snow               [id|--ids ...|--all]
```

### Channel assignment

```bash
# Permanent channel switch
pintomind screens set-channel <screen-id> <channel-id>
pintomind screens set-channel --ids 1,2,3 <channel-id>
pintomind screens set-channel --all <channel-id>

# Temporary channel override
pintomind screens temp-channel <screen-id> <channel-id> --duration 3600
pintomind screens temp-channel <screen-id> <channel-id> --until 2025-12-31T23:59:00Z
pintomind screens temp-channel --all <channel-id> --duration 1800
pintomind screens temp-channel <screen-id> <channel-id> --toggle
```

## Channels

```bash
pintomind channels list
pintomind channels list --sort-by name
pintomind channels list --page 2 --per-page 50
pintomind channels show <id>
pintomind channels posts <id>           # account token required
pintomind channels stats                # requires channels:read:stats scope
pintomind channels stats <id>
pintomind channels set-theme <channel-id> <theme-id>
pintomind channels set-theme --all <theme-id>
```

## Resources

```bash
pintomind resources list
pintomind resources list --type text_slide
pintomind resources list --type text_slide,calendar_events   # comma-separated
pintomind resources list --page 2 --per-page 50
pintomind resources show <id>
pintomind resources stats
pintomind resources refresh <id>

pintomind resources create --type text_slide --data '{"title":"Hello","body":"World"}'
pintomind resources update <id> --data '{"title":"Updated"}'
pintomind resources append <id> --items '[{"text":"item 1"}]'
pintomind resources delete <id>
pintomind resources delete <id> --force
```

## Schemas

Inspect valid fields for resource types:

```bash
pintomind schemas list
pintomind schemas show <id>   # e.g. pintomind schemas show text_slide
```

## Media Collections

```bash
pintomind media-collections list
pintomind media-collections list --sort-by title
pintomind media-collections show <id>
pintomind media-collections create --title "Campaign photos" --category image
pintomind media-collections update <id> --title "Updated title"
pintomind media-collections delete <id>
pintomind media-collections delete <id> --force
```

Categories are `background`, `logo`, `image`, `video`, and `document`.

## Media

```bash
pintomind media list <collection-id>
pintomind media list <collection-id> --sort-by created_at:desc
pintomind media show <id>
pintomind media upload <collection-id> ./photo.jpg --name "Lobby photo"
pintomind media upload <collection-id> ./deck.pdf --extract-pages
pintomind media update <id> --name "Updated name"
pintomind media update <id> --collection-id <target-collection-id>
pintomind media delete <id>
pintomind media delete <id> --force
```

Use `media upload` for normal file uploads. It handles the direct upload flow and creates the media record.

## Themes

```bash
pintomind themes list
pintomind themes list --page 2
pintomind themes show <id>
pintomind themes stats
```

## Identity

```bash
pintomind me
pintomind network
```

## Raw API access

For endpoints not yet covered by a dedicated command:

```bash
pintomind api /screens
pintomind api /screens/42
pintomind api GET /channels?sort_by=name
echo '{"screen":{"command":"reload"}}' | pintomind api PATCH /screens/42
```

## Tips

- Find screen IDs: `pintomind screens list --json | jq '.items[] | {id, name}'`
- Reload all screens at once: `pintomind screens reload --all`
- Wait for a screen after reboot: `pintomind screens reboot 42 && pintomind screens wait-online 42`
- Apply a theme to every channel: `pintomind channels set-theme --all <theme-id>`
- Upload a file to media: `pintomind media upload <collection-id> ./photo.jpg --name "Lobby photo"`
- Inspect valid resource fields: `pintomind schemas show text_slide`
- Use `--account develop` to target the dev environment without changing your default.
- Token scopes matter: `channels:read:stats` is required for stats endpoints; account tokens (not network tokens) are required for channel posts.
