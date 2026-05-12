---
name: pintomind
description: Interact with Pintomind / Infoskjermen screens, channels, resources, media, themes, color palettes, and font families via the pintomind CLI. Use for ANY Pintomind question or action.
---

You are an expert at using the `pintomind` CLI to interact with the Pintomind / Infoskjermen API. You help users manage screens, channels, resources, media, posts, themes, color palettes, and font families.

## Setup

Config is stored in `~/.config/pintomind/config.json`. Multiple connections are supported.

```bash
pintomind connection add app.infoskjermen.no sk-xxx
pintomind connection add develop sk-dev --url https://develop.infoskjermen.no
pintomind connection use develop
pintomind connection list
pintomind connection show   # show active connection and masked API key
```

## Global Flags

- `--connection <name>` — override active connection for a single command
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
pintomind resources list --type text
pintomind resources list --type text,calendar_events   # comma-separated
pintomind resources list --page 2 --per-page 50
pintomind resources show <id>
pintomind resources stats
pintomind resources refresh <id>

pintomind resources update <id> --data '{"label":"Updated"}'
pintomind resources append <id> --items '[{"text":"item 1"}]'
pintomind resources delete <id>
pintomind resources delete <id> --force
```

### Creating resources — helper subcommands

Type-specific subcommands replace the need for raw JSON.

```bash
# Text resource
pintomind resources create text --label "Greeting" --text "Hello world"

# Feed resource (RSS/Atom)
pintomind resources create feed --url https://example.com/rss.xml --title "News"

# Calendar resource (iCal/WebCal URL)
pintomind resources create calendar --url https://cal.example.com/feed.ics --title "Events"
pintomind resources create calendar --url webcal://cal.example.com/feed.ics --color "#FF5733"

# External webpage (iframe, https only)
pintomind resources create external-webpage --url https://dashboard.example.com --title "Dashboard"

# External image (from URL)
pintomind resources create external-image --url https://example.com/logo.png --title "Logo"

# QR code
pintomind resources create qr-code --url https://example.com --title "Website"
pintomind resources create qr-code --url https://example.com --label-text "Scan me"

# HTML resource
pintomind resources create html --html-code "<h1>Hello</h1>" --title "Heading"

# YouTube resource
pintomind resources create youtube --url "https://www.youtube.com/watch?v=dQw4w9WgXcQ" --title "Video"

# Location resource (used for weather/forecast posts)
pintomind resources create location --name "Oslo" --lat 59.9139 --lon 10.7522 --country-code NO --timezone Europe/Oslo
pintomind resources create location --name "Oslo" --lat 59.9139 --lon 10.7522 --country-code NO --timezone Europe/Oslo --altitude 23 --country "Norway"
```

### Creating resources — raw JSON

```bash
pintomind resources create --type text --data '{"label":"Hello","text":"World"}'
```

## Schemas

Inspect valid fields for resource and post types:

```bash
pintomind schemas list
pintomind schemas show <id>           # e.g. pintomind schemas show text
pintomind schemas show post_<alias>   # e.g. pintomind schemas show post_image
pintomind schemas show grid_templates # named grid layouts for plain post variations
```

Resource schema keys: `calendar`, `calendar_events`, `entur`, `external_image`, `external_webpage`, `feed`, `feed_items`, `html`, `location`, `qr_code`, `text`, `youtube`, `poster_page`.

Post schema keys: `post_plain`, `post_image`, `post_video`, `post_youtube`, `post_iframe`, `post_calendar`, `post_clock`, `post_world_clock`, `post_counter`, `post_forecast`, `post_feed`, `post_entur`, `post_power_price`, `post_poster`, `post_time_schedule`.

Media box schema keys: `media_box_image`, `media_box_icon`, `media_box_emoji`, `media_box_gif`, `media_box_unsplash`, `media_box_qr_code`.

Theme/palette/font schema keys: `theme`, `color_palette`, `font_family_remote_css`, `font_family_uploaded`.

## Media Collections

```bash
pintomind media-collections list
pintomind media-collections list --sort-by title
pintomind media-collections show <id>
pintomind media-collections create --title "Campaign photos" --category image
pintomind media-collections create --title "Logos" --category logo --icon "star"
pintomind media-collections update <id> --title "Updated title"
pintomind media-collections update <id> --icon "heart"
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
pintomind media upload <collection-id> https://example.com/photo.jpg --name "Lobby photo"
pintomind media upload <collection-id> ./deck.pdf --extract-pages
pintomind media update <id> --name "Updated name"
pintomind media update <id> --collection-id <target-collection-id>
pintomind media delete <id>
pintomind media delete <id> --force
```

Use `media upload` for normal local file or URL uploads. It handles download, checksum, the direct upload flow, and media record creation.

## Posts

Posts are content cards (image, plain text, video, calendar, clock, etc.) that get published to channels. Account token required (network tokens cannot manage posts).

```bash
pintomind posts list
pintomind posts list --type image
pintomind posts list --type image,plain                  # comma-separated
pintomind posts list --archived                          # archived posts only
pintomind posts list --deleted                           # soft-deleted posts only
pintomind posts list --sort-by created_at:desc
pintomind posts list --page 2 --per-page 50
pintomind posts show <id>
pintomind posts delete <id>
pintomind posts delete <id> --force
```

### Creating posts — helper subcommands

Type-specific subcommands handle uploads, resource creation, and publishing in one step. The raw `--type / --data` form still works for all other post types.

```bash
# Image post — upload files and/or reference existing media IDs
pintomind posts create image --name "Gallery" --image ./a.jpg --image ./b.jpg --channel-id 7
pintomind posts create image --name "Promo" --media 42 --media 43 --duration 10
pintomind posts create image --name "Mix" --image ./photo.jpg --media 55 --channel-id 7 --channel-id 8
pintomind posts create image --name "X" --image ./photo.jpg --media-collection 12  # override collection

# Plain text post
pintomind posts create plain --name "Welcome" --heading "<p>Hello</p>" --channel-id 7
pintomind posts create plain --name "Msg" --heading "<p>Hi</p>" --heading-alignment center --heading-fontsize 150 --body "<p>Body</p>" --body-alignment left --body-fontsize 80

# Feed post — creates a feed resource then the post
pintomind posts create feed --name "News" --url https://example.com/rss.xml --channel-id 7

# Calendar post — creates a calendar resource from an iCal/WebCal URL
pintomind posts create calendar --name "Events" --url https://cal.example.com/feed.ics --channel-id 7

# Iframe post — create from a URL, HTML content, or image URL (exactly one required)
pintomind posts create iframe --name "Dashboard" --url https://dashboard.example.com --channel-id 7
pintomind posts create iframe --name "Announcement" --html "<h1>Hello</h1>" --channel-id 7
pintomind posts create iframe --name "Banner" --image https://example.com/banner.png --channel-id 7

# HTML post — shortcut for iframe with HTML content (creates html resource + iframe post)
pintomind posts create html --name "Announcement" --html "<h1>Hello world</h1>" --channel-id 7

# Poster post — from a network template
pintomind posts create poster --name "Spring campaign" --source-id 42 --channel-id 7
pintomind posts create poster --name "Brand poster" --source-id 42 --color-palette-id 5 --channel-id 7
```

All `posts create <type>` helpers accept:
- `--channel-id N` (repeatable) — publish to one or more channels immediately
- `--area F11` — channel area for all publications
- `--full-screen` — publish fullscreen to all channels
- `--until <YYYY-MM-DD[THH:MM]>` — auto-expire the post at this date/time (attaches a one-period schedule that unpublishes on expiry). Time defaults to `23:59` when omitted.

`--until` is evaluated in each channel's local time zone. For richer schedules (recurring weekly windows, multiple periods, or other expiry actions) use `pintomind posts schedule set` after creating the post.

### Top-level `publish` shorthand

One-liner for the most common flow: upload a file and publish to a channel.

```bash
pintomind publish ./photo.jpg --channel-id 7 --name "Lobby photo"
pintomind publish https://example.com/photo.jpg --channel-id 7 --channel-id 8
pintomind publish ./photo.jpg --channel-id 7 --media-collection 12 --duration 10
pintomind publish ./flyer.jpg --channel-id 7 --until 2026-06-30T17:00
pintomind publish ./flyer.jpg --channel-id 7 --until 2026-06-30
```

Auto-detects the default image collection. Use `--collection-id` to override. `--until` behaves the same as on `posts create *`.

### Time schedule

Each post can have at most one time schedule that controls when it is visible. A schedule combines:

- `recurring_rules` — weekly weekday/time windows (e.g. Mon–Fri 08:00–18:00). Optional `weeks` (`all` / `odd` / `even`) for bi-weekly cadence.
- `time_periods` — concrete date ranges (e.g. 1 Jun – 30 Jun, 09:00–17:00).
- `schedule_expiry_action` — what happens when the entire schedule has fully expired: `delete`, `archive`, or `unpublish`.

```bash
# Read the post's current schedule (404 when no schedule is set)
pintomind posts schedule get <post-id>
pintomind posts schedule get <post-id> --json

# Set or replace a schedule (PUT — omitted sections are left unchanged)
pintomind posts schedule set <post-id> \
  --recurring-rule 'from_wday=1,to_wday=5,from_time=08:00,to_time=18:00' \
  --recurring-rule 'from_wday=6,to_wday=6,from_time=10:00,to_time=14:00,weeks=odd' \
  --period 'from_date=2026-06-01,to_date=2026-06-30,from_time=09:00,to_time=17:00' \
  --expiry-action archive

# Replace just one section with an empty list
pintomind posts schedule set <post-id> --clear-periods
pintomind posts schedule set <post-id> --clear-rules

# Delete the schedule (post becomes always-visible)
pintomind posts schedule clear <post-id>
pintomind posts schedule clear <post-id> --force
```

Flag syntax for `set`:

- `--period`: comma-separated `key=value`. Required: `from_date` (`YYYY-MM-DD`), `to_date`. Optional: `from_time` (`HH:MM`), `to_time`. Repeatable.
- `--recurring-rule`: required: `from_wday` (0=Sun–6=Sat), `to_wday`, `from_time`, `to_time`. Optional: `weeks` (`all`, `odd`, `even`). Repeatable.
- `--expiry-action`: `delete`, `archive`, or `unpublish`.

Inspect the JSON schema with `pintomind posts schema time-schedule`.

Time zones: weekdays and times are evaluated in the time zone of the channel the post is published to, not in your local zone. A post published to channels in different time zones turns on/off at each channel's local wall-clock time.

### Creating posts — raw JSON (all post types)

`posts create --type <alias> --data '<json>'` accepts any post type. The CLI wraps it in `{"type": ..., "post": {...}}` for you.

**Important:** all post fields — including type-specific ones — are flat inside the `post` object. Do not wrap them in an `options` sub-object.

Always inspect the schema for the post type first to learn what fields are accepted:

```bash
pintomind schemas show post_image
pintomind schemas show post_plain
pintomind schemas show post_calendar
```

Common shared fields: `name`, `title`, `duration`, `show_title`, `area`.

```bash
# Image post — use images[] with media_id values from `pintomind media upload`
pintomind posts create --type image --data '{
  "name":"Spring campaign",
  "title":"New collection",
  "duration_per_item":7,
  "images":[{"media_id":42},{"media_id":43}],
  "aspect_ratio":"fill",
  "grid_layout":1
}'

# Video post — video_media_id is a single integer
pintomind posts create --type video --data '{
  "name":"Promo",
  "video_media_id":101,
  "sound":true
}'

# Clock post (no resources)
pintomind posts create --type clock --data '{
  "name":"Lobby clock",
  "clock_variant":"analog",
  "show_date":true
}'

# Calendar post (raw)
pintomind posts create --type calendar --data '{
  "name":"Today",
  "template":"list",
  "max_items":5,
  "resource_ids":[55],
  "limit_days":1,
  "show_location":true
}'

# Poster from a network template (see poster-templates section)
pintomind posts create --type poster --source-id <template-id> --data '{"name":"My Poster"}'

# Poster from scratch
pintomind posts create --type poster --data '{
  "name":"Lobby poster",
  "resources":[{
    "title":"Lobby poster",
    "poster_data":"{\"backgroundFill\":\"#ffffff\",\"templateDimensions\":{},\"templates\":{},\"aspectRatio\":\"16/9\"}",
    "color_palette_id":123
  }]
}'
```

### Updating posts

The post `type` cannot be changed after creation. Pass only the fields you want to change.

```bash
pintomind posts update 123 --data '{"title":"Updated title"}'
pintomind posts update 123 --data '{"sound":false}'
```

### Publishing posts to channels

Creating or updating a post does **not** put it on a channel — that is a separate step.

```bash
pintomind posts publications <post-id>                              # list channel placements
pintomind posts publish <post-id> --channel-id 7                    # publish to a channel
pintomind posts publish <post-id> --channel-id 7 --channel-id 12    # multiple channels in one call
pintomind posts publish <post-id> --channel-id 7 --area F11         # specific area
pintomind posts publish <post-id> --channel-id 7 --full-screen      # fullscreen
pintomind posts unpublish <post-id>                                 # unpublish from ALL channels
pintomind posts unpublish <post-id> --channel-id 7                  # unpublish from a specific channel
pintomind posts unpublish <post-id> --channel-id 7 --channel-id 12 --force
```

### Typical end-to-end flow

```bash
# 1. Upload an image to a media collection
pintomind media upload 42 ./photo.jpg --name "Lobby photo" --json | jq '.media.id'

# 2. Create an image post using that media ID
pintomind posts create --type image --data '{"name":"Lobby","images":[{"media_id":<id>}],"duration_per_item":7}'

# 3. Publish it to a channel
pintomind posts publish <post-id> --channel-id <channel-id>
```

## Poster Templates

Poster templates are poster posts shared via the account's content networks that can be duplicated into new account-owned poster posts.

```bash
pintomind poster-templates list
pintomind poster-templates list --sort-by name
pintomind poster-templates list --page 2 --per-page 50
pintomind poster-templates show <id>   # includes raw_poster_data
```

Fields per template: `id`, `name`, `aspect_ratios`, `resource_id`, `raw_poster_data`.

- `aspect_ratios`: array of supported aspect ratio strings (e.g. `["16/9","9/16"]`)
- `resource_id`: ID of the backing `PosterPageResource`
- `raw_poster_data`: JSON string for use as `resource.poster_data` when updating a poster resource

## Poster Posts

Poster posts are backed by a `PosterPageResource` (type alias `poster_page`) that holds the design data.

### Creating a poster from a network template

```bash
# 1. List available templates
pintomind poster-templates list --json

# 2. Duplicate a template into a new post owned by the account
pintomind posts create --type poster --source-id <template-id> --data '{"name":"My Poster"}'

# 3. Update the poster content via the backing resource
pintomind resources update <resource-id> --data '{"poster_data": "{\"backgroundFill\":\"#fff\",\"aspectRatio\":\"16/9\",\"templates\":{},\"templateDimensions\":{}}"}'

# 4. Attach media boxes to poster nodes (hash of node-position → media_box_id)
pintomind resources update <resource-id> --data '{"media_box_ids": {"hero-image": 201}}'

# 5. Publish to a channel
pintomind posts publish <post-id> --channel-id <channel-id>
```

### Creating a poster from scratch

```bash
# Create the post, then update the backing resource
pintomind posts create --type poster --data '{"name": "Lobby poster"}'
pintomind resources update <resource-id> --data '{
  "poster_data": "{\"backgroundFill\":\"#ffffff\",\"templateDimensions\":{},\"templates\":{},\"aspectRatio\":\"16/9\",\"metaDescription\":\"Minimalist poster.\"}",
  "color_palette_id": 123
}'
```

The post response includes `resource_ids`; use the first ID as `<resource-id>`.

### Updating poster content

```bash
pintomind resources update <resource-id> --data '{"poster_data": "<json-string>"}'
pintomind resources update <resource-id> --data '{"color_palette_id": 5}'
```

`poster_data` is a JSON string with top-level keys: `backgroundFill`, `templateDimensions`, `templates`, `aspectRatio`, `metaDescription`.

`poster_page` resources also accept `media_box_ids` (hash) and `color_palette_id`. Update them via `resources update`.

**Note:** `poster_page` resources are auto-created by poster posts. Do not create them manually via `resources create`.

### Media Boxes

Media boxes are visual containers used by both plain posts and poster posts.

**Plain posts** use `media_box_ids` as an **array** of IDs in slot order. The server writes each box UUID into the selected grid and sets `post_id` on the attached boxes. Max 4 boxes; grid variation selected by count: 0 → `R12-HT`, 1 → `R12-HIT`, 2 → `R121-HIIT`, 3 → `R131-HIIIT`, 4 → `R14-HIIIIT`.

**Poster posts** use `media_box_ids` as a **hash** of `{ "node-position": box_id }` on the backing `PosterPageResource`. The key is the `id` of the target node inside the poster template. Each submission is a full replacement; omitted positions are removed. Pass `{}` to remove all.

```bash
pintomind media-boxes list
pintomind media-boxes list --type media,emoji
pintomind media-boxes list --post-id <post-id>
pintomind media-boxes show <id>

# Create boxes
pintomind media-boxes create media --media-id 42
pintomind media-boxes create media --media-id 42 --background-size cover --x 0.5 --y 0.5
pintomind media-boxes create icon --icon-name rocket-launch --icon-type regular
pintomind media-boxes create icon --icon-name rocket-launch --icon-type solid --relative-size 0.8
pintomind media-boxes create emoji --emoji '✨'
pintomind media-boxes create gif --gif-id xT9IgG50Fb7Mi0only
pintomind media-boxes create unsplash --photo-id abc123 --background-size cover
pintomind media-boxes create qr-code --url https://example.com

# Update/delete boxes
pintomind media-boxes update <id> --data '{"icon_name":"fire"}'
pintomind media-boxes delete <id> --force

# Attach boxes to a plain post (array, slot order)
pintomind posts create --type plain --data '{
  "name":"Welcome",
  "heading":"<p>Hello</p>",
  "body":"<p>Welcome to the office</p>",
  "media_box_ids":[201,202]
}'

# Replace attached boxes on an existing plain post
pintomind posts update <post-id> --data '{"media_box_ids":[203]}'

# Attach boxes to a poster post (hash of node-position → id, via poster_page resource)
pintomind resources update <resource-id> --data '{"media_box_ids": {"hero-image": 201, "avatar": 202}}'

# Remove all poster media boxes
pintomind resources update <resource-id> --data '{"media_box_ids": {}}'
```

Removing a box ID from a plain post update deletes the orphaned box. For poster posts, each `media_box_ids` hash submission is a full replacement.

#### Media box types

| Type | Required fields | Key optional fields |
|---|---|---|
| `media` | `media_id` (integer — ID from media library) | `background_size` (`"cover"`/`"contain"`, default `"cover"`), `x`, `y` (0–1 focal point, default `0.5`), `background_fill_type`, `relative_size` |
| `icon` | `icon_name` (string) | `icon_type`, `relative_size` (default `0.9`) |
| `emoji` | `emoji` (Unicode character, e.g. `"🎂"`) | `relative_size` (default `0.9`) |
| `gif` | `gif_id` (URL and metadata auto-fetched from Giphy) | `background_size`, `x`, `y`, `background_fill_type` |
| `unsplash` | `photo_id` (URL and author data auto-fetched from Unsplash) | `background_size` (default `"cover"`), `x`, `y`, `background_fill_type` |
| `qr_code` | `url` (URL to encode) | `relative_size` (default `0.9`) |

Inspect exact fields with `pintomind schemas show media_box_image`, `media_box_icon`, `media_box_emoji`, `media_box_gif`, `media_box_unsplash`, or `media_box_qr_code`.

## Icons

```bash
pintomind icons   # lists available icon names, types, and categories for icon media boxes
```

Scope: `media_boxes:read`. Returns `{ names: [...], types: [...], categories: [...] }`. Use `icon_name` values here with `pintomind media-boxes create icon --icon-name <name> --icon-type <type>`.

## Themes

```bash
pintomind themes list
pintomind themes list --sort-by name
pintomind themes list --page 2 --per-page 50
pintomind themes show <id>
pintomind themes stats
pintomind themes create --data '{"name":"My theme","font_family_header_id":1,"font_family_body_id":2,"background_color":"#1a1a2e","text_color":"#ffffff"}'
pintomind themes create --data '{"name":"Brand theme","font_family_header_id":1,"font_family_body_id":2,"background_color":"#1a1a2e","text_color":"#ffffff","color_palette_id":3}'
pintomind themes update <id> --data '{"name":"Updated name"}'
pintomind themes update <id> --data '{"color_palette_id":5}'
pintomind themes delete <id>
pintomind themes delete <id> --force
```

Required fields for `create`: `name`, `font_family_header_id`, `font_family_body_id`, `background_color`, `text_color`.
Optional: `color_palette_id`, `variation`, `areas`, and any flat layout/background/text properties.
Deleting a theme resets channels using it to the account's standard theme.

## Color Palettes

```bash
pintomind color-palettes list
pintomind color-palettes list --sort-by name
pintomind color-palettes list --page 2 --per-page 50
pintomind color-palettes show <id>
pintomind color-palettes stats
pintomind color-palettes create --name "Brand" --primary-color "#1F8A8A"
pintomind color-palettes create --name "Brand" --primary-color "#1F8A8A" --secondary-color "#F5A623" --tertiary-color "#2C3E50"
pintomind color-palettes update <id> --name "Updated name"
pintomind color-palettes update <id> --primary-color "#FF0000"
pintomind color-palettes delete <id>
pintomind color-palettes delete <id> --force
```

Color values must be hex (e.g. `#1F8A8A`). The API auto-generates the full 12-step color ramps from the base colors.
Deleting a palette repoints themes using it to another available palette; returns an error if none exists.
Palettes shared with network members cannot be deleted.

## Font Families

```bash
pintomind font-families list
pintomind font-families list --type remote_css
pintomind font-families list --type remote_css,uploaded,standard
pintomind font-families list --sort-by name
pintomind font-families show <id>
pintomind font-families stats

# Remote CSS font (e.g. Google Fonts)
pintomind font-families create remote-css --name "Inter" \
  --url "https://fonts.googleapis.com/css2?family=Inter:wght@400;700"

# Uploaded font — pass local file paths (.ttf/.otf/.woff/.woff2)
pintomind font-families create uploaded --name "MyFont" \
  --font-normal ./MyFont-Regular.ttf --font-bold ./MyFont-Bold.ttf
pintomind font-families create uploaded --name "MyFont" \
  --font-normal ./MyFont-Regular.ttf --font-bold ./MyFont-Bold.ttf \
  --font-italic ./MyFont-Italic.ttf --font-bold-italic ./MyFont-BoldItalic.ttf

pintomind font-families update <id> --name "Renamed"
pintomind font-families update <id> --suitable-for-body
pintomind font-families delete <id>
pintomind font-families delete <id> --force
```

Font types:
- `remote_css` — references a hosted CSS file. Required: `--url`. Optional: `--font-name` (picked from CSS if omitted).
- `uploaded` — uploads `.ttf`/`.otf`/`.woff`/`.woff2` files as multipart/form-data. Required: `--font-normal`, `--font-bold`. Optional: `--font-italic`, `--font-bold-italic`.
- `standard` — system fonts, read-only (cannot be created or deleted).

Deleting a font resets themes using it to the system default (`figtree`).
Fonts shared with network members cannot be deleted.

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
- List all color palettes: `pintomind color-palettes list --json | jq '.items[] | {id, name, primary_color}'`
- Create a palette and use it in a theme: `pintomind color-palettes create --name "Brand" --primary-color "#1F8A8A"` then `pintomind themes update <theme-id> --data '{"color_palette_id":<id>}'`
- Add a Google Font: `pintomind font-families create remote-css --name "Inter" --url "https://fonts.googleapis.com/css2?family=Inter:wght@400;700"`
- Upload a file to media: `pintomind media upload <collection-id> ./photo.jpg --name "Lobby photo"`
- Publish a photo to a channel in one step: `pintomind publish ./photo.jpg --channel-id 7`
- Inspect valid resource fields: `pintomind schemas show text`
- Inspect valid post fields: `pintomind schemas show post_image` (prefix `post_`)
- Publishing is separate from raw `posts create`: use `posts publish <post-id> --channel-id <channel-id>` or pass `--channel-id` to helper subcommands
- Image posts use `images:[{"media_id":...}]`; video posts use `video_media_id:<id>` (single integer) — neither uses `resource_ids`
- Post fields are always flat inside `post` — no `options` sub-object
- Use `--connection develop` to target the dev environment without changing your default.
- Token scopes matter: `channels:read:stats` is required for stats endpoints; account tokens (not network tokens) are required for posts and channel posts.
