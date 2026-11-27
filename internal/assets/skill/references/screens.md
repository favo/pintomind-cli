# Screens

Screens (players), remote control, effects, schedules, provisioning and remote debugging.

## Screens

```bash
pintomind screens connect <code>                  # connect a new screen using the code it displays
pintomind screens connect <code> --channel-id 17  # ...and show a channel immediately
pintomind screens connect <code> --name "Lobby" --notes "Reception desk"  # ...with name and notes
pintomind screens update <id> --name "Lobby" --notes "Reception desk"     # rename / set notes (empty --notes clears)
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
pintomind screens toggle-night-mode [id|--ids ...|--all]   # --on / --off forces a state instead of toggling
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
pintomind screens snow|rain|thunderstorm|leaves|hearts|sparkles [id|--ids ...|--all]
pintomind screens snow 42 --duration 60 --intensity high    # duration in seconds (0 = until stopped); intensity low|medium|high
pintomind screens stop-effects       [id|--ids ...|--all]
```

Weather-like effects (snow, rain, thunderstorm, leaves, hearts, sparkles) replace each other;
confetti stacks on top. `confetti-fire` is one burst and has no `--duration`. A reload ends effects.

### Disconnect, schedule and provisioning

```bash
pintomind screens delete <id>                 # disconnect from the account (screen shows a new code)

# Channel schedule: day/week plan, night mode and reboot (times in the standard channel's time zone)
pintomind screens schedule show <screen-id>
pintomind screens schedule set <screen-id> --mode daily \
  --day-plan '[{"type":"channel","channel_id":12,"from_time":"08:00","to_time":"16:00"},{"type":"night","from_time":"22:00","to_time":"06:00"},{"type":"reboot","hour":4}]'
pintomind screens schedule set <screen-id> --mode weekly --week-plan @week.json   # blocks need "wday" 0 (Sun)..6 (Sat)
pintomind screens schedule set <screen-id> --template-id 3                        # apply and link a template
pintomind screens schedule clear <screen-id>
pintomind screens schedule templates list|show|create|update|delete               # --name --mode --day-plan --week-plan

# One-time provisioning tokens a player redeems to become a screen (secret shown once)
pintomind screens provisioning create --screen-id 42
pintomind screens provisioning create --new-screen --name Lobby --channel-id 7 --expires-in 86400
pintomind screens provisioning list
pintomind screens provisioning revoke <token-id>
```

A plan replaces the whole plan; gaps show the standard channel. Night blocks take `device_off`
(PieOS only) and `animation_type` (`sleepy`, `sleepy_rooster` or null).

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

### Remote debugging

`screens debug` runs remote console commands on a single screen. It needs an API key with the
`screens:debug` scope (an OAuth login never gets it; use `--connection <api-key-connection>`), and the screen must be online and run butler 0.17.0 or newer (reload the
screen to update it). Elements are given as a CSS selector or as a path such as `@1/0/2`, which
`dom` and `elements` print. Add `--json` for the screen's raw answer.

```bash
# What is playing
pintomind screens debug status <screen-id>          # butler version, loop state, current post per area
pintomind screens debug posts <screen-id>           # every post in every area: ready, scheduled, hidden, remaining
pintomind screens debug post <screen-id> <post-id>  # the data the screen has for a post

# Playback
pintomind screens debug loop <screen-id> [status|pause|resume|toggle|next|previous|next-part|previous-part]
pintomind screens debug show-post <screen-id> <post-id>    # jump to a post
pintomind screens debug hide-post <screen-id> <post-id>    # hide until the screen reloads
pintomind screens debug unhide-post <screen-id> <post-id>

# Health, logs and network
pintomind screens debug health <screen-id> [--fps]  # memory, connection; --fps measures for ~2s
pintomind screens debug logs <screen-id> [--limit 100] [--level error|warn|info|log]
pintomind screens debug logs <screen-id> -f [--interval 3s]  # follow; with --json streams one entry per line
pintomind screens debug network <screen-id> [--errors] [--category data|script|style|image|media|font|other] \
  [--query text] [--sort started|size|time|name|status] [--desc] [--limit 500]

# DOM inspection
pintomind screens debug dom <screen-id> [selector|@path]       # an element and its children, to walk the tree
pintomind screens debug elements <screen-id> <selector|@path>  # size, position, visibility of matches
pintomind screens debug elements <screen-id> <selector|@path> --full  # computed styles, box model, attributes (JSON)
pintomind screens debug highlight <screen-id> <selector|@path> # outline on the physical screen for 5 seconds

# Run JavaScript in the screen view
pintomind screens debug eval <screen-id> 'document.title'
pintomind screens debug eval <screen-id> --path @1/1/2 '$0.getBoundingClientRect()'  # $0 = element at path
pintomind screens debug eval <screen-id> --file snippet.js
echo 'return App.loop.isRunning' | pintomind screens debug eval <screen-id> -
```

Typical flow for "why is this post not showing": `status` → `posts` (check ready/scheduled/hidden)
→ `post <id>` → `logs --level error` → `network --errors`.

## Tips

- Reload all screens at once: `pintomind screens reload --all`
- Wait for a screen after reboot: `pintomind screens reboot 42 && pintomind screens wait-online 42`
