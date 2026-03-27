---
name: pintomind
description: Interact with Pintomind / Infoskjermen screens, channels, resources, and themes via the pintomind CLI. Use for ANY Pintomind question or action.
---

You are an expert at using the `pintomind` CLI to interact with the Pintomind / Infoskjermen API.

## Setup

Config is stored in `~/.config/pintomind/config.json`. Multiple domains are supported.

```bash
pintomind config add infoskjermen.no --api-key sk-xxx
pintomind config add develop --api-key sk-dev --url https://develop.infoskjermen.no
pintomind config use develop
pintomind config list
```

## Global Flags

- `--domain <name>` — override active domain for a single command
- `--json` — output raw JSON

## Screens

```bash
pintomind screens list
pintomind screens list --online
pintomind screens list --offline
pintomind screens show <id>
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
pintomind resources show <id>
pintomind resources stats
pintomind resources refresh <id>

pintomind resources create --type text_slide --data '{"title":"Hello","body":"World"}'
pintomind resources update <id> --data '{"title":"Updated"}'
pintomind resources append <id> --items '[{"text":"item 1"}]'
pintomind resources delete <id>
pintomind resources delete <id> --force
```

## Themes

```bash
pintomind themes list
pintomind themes show <id>
pintomind themes stats
```

## Identity

```bash
pintomind me
pintomind network
```

## Tips

- Find screen IDs: `pintomind screens list --json | jq '.items[] | {id, name}'`
- Reload all screens at once: `pintomind screens reload --all`
- Apply a theme to every channel: `pintomind channels set-theme --all <theme-id>`
- Use `--domain develop` to target the dev environment without changing your default.
- Token scopes matter: `channels:read:stats` is required for stats endpoints; account tokens (not network tokens) are required for channel posts.
