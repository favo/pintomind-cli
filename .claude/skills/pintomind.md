---
name: pintomind
description: Interact with Pintomind / Infoskjermen screens, channels, resources, and themes via the pintomind CLI. Use for ANY Pintomind question or action.
---

You are an expert at using the `pintomind` CLI to interact with the Pintomind / Infoskjermen API.

## Setup

Config is stored in `~/.config/pintomind/config.json`. Multiple domains are supported (e.g. production and develop environments).

```bash
# Add a domain
pintomind config add infoskjermen.no --api-key sk-xxx

# Add a dev domain with custom URL
pintomind config add develop --api-key sk-dev --url https://develop.infoskjermen.no

# Switch default domain
pintomind config use develop

# List configured domains
pintomind config list
```

## Global Flags

- `--domain <name>` — override active domain for a single command
- `--json` — output raw JSON (for piping or scripting)

## Screens

```bash
pintomind screens list                          # list all screens
pintomind screens list --online                 # only online
pintomind screens list --offline                # only offline
pintomind screens show <id>                     # show screen details (JSON)

# Commands
pintomind screens command <id> reload
pintomind screens command <id> reboot
pintomind screens command <id> clear_cache
pintomind screens command <id> upgrade_firmware
pintomind screens command <id> identify
pintomind screens command <id> toggle_night_mode
pintomind screens command --ids 1,2,3 reload    # bulk

# Remote control signals
pintomind screens signal <id> next
pintomind screens signal <id> previous
pintomind screens signal <id> play
pintomind screens signal <id> pause
pintomind screens signal <id> toggle_play
pintomind screens signal --ids 1,2 next         # bulk

# Effects
pintomind screens signal <id> confetti_fire --effect
pintomind screens signal <id> confetti_fireworks --effect
pintomind screens signal <id> snow --effect

# Switch channel
pintomind screens set-channel <screen-id> <channel-id>
pintomind screens set-channel --ids 1,2,3 <channel-id>  # bulk

# Temporary channel override
pintomind screens temp-channel <screen-id> <channel-id> --duration 3600
pintomind screens temp-channel <screen-id> <channel-id> --until 2025-12-31T23:59:00Z
pintomind screens temp-channel <screen-id> <channel-id> --toggle
```

## Channels

```bash
pintomind channels list
pintomind channels list --sort-by name
pintomind channels show <id>
pintomind channels posts <id>                   # list posts (account token required)
pintomind channels stats                        # requires channels:read:stats scope
pintomind channels stats <id>
pintomind channels set-theme <channel-id> <theme-id>
```

## Resources

```bash
pintomind resources list
pintomind resources list --type text_slide
pintomind resources show <id>
pintomind resources stats
pintomind resources refresh <id>                # for external resources

# Create (type and JSON data required)
pintomind resources create --type text_slide --data '{"title":"Hello","body":"World"}'

# Update
pintomind resources update <id> --data '{"title":"Updated"}'

# Append items (for resource types that support it)
pintomind resources append <id> --items '[{"text":"item 1"}]'

# Delete (soft-delete first, hard-delete on second call)
pintomind resources delete <id>
pintomind resources delete <id> --force         # skip confirmation
```

## Themes

```bash
pintomind themes list
pintomind themes show <id>
pintomind themes stats
```

## Identity

```bash
pintomind me                    # show token/user/account info
pintomind network               # show network identity (network token required)
```

## Tips

- To find a screen ID: `pintomind screens list --json | jq '.items[] | {id, name}'`
- To send a command to all online screens: `pintomind screens list --online --json | jq '[.items[].id] | join(",")' -r | xargs -I{} pintomind screens command --ids {} reload`
- Use `--domain develop` to run a command against the dev environment without changing your default.
- Token scopes affect what's accessible: `channels:read:stats` is required for stats endpoints; account tokens (not network tokens) are required for channel posts.
