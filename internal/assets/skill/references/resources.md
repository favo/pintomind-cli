# Resources and schemas

Data sources posts pull from (text, feeds, calendars, web pages, locations …) and the JSON schemas for resource and post fields.

## Resources

```bash
pintomind resources list
pintomind resources list --type text
pintomind resources list --type text,calendar_events   # comma-separated
pintomind resources list --page 2 --per-page 50
pintomind resources list --include-shared               # also what the account's networks share (shared: true), e.g. leagues
pintomind resources show <id>
pintomind resources teams <league-id>                   # teams for a scores_and_fixtures post's follow_team_id
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

# Hand-written items instead of a URL (or --interactive)
pintomind resources create feed-items --items '[{"title":"Breaking news","content":"Details here","url":"https://example.com"}]'
pintomind resources create calendar-events --items '[{"start_at":"2026-06-01T09:00:00Z","end_at":"2026-06-01T10:00:00Z","description":"Meeting"}]'
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
pintomind schemas show post_plain     # includes x-variation-details: every grid layout for plain posts
```

Resource schema keys: `calendar`, `calendar_events`, `entur`, `external_image`, `external_webpage`, `feed`, `feed_items`, `html`, `location`, `qr_code`, `text`, `youtube`, `poster_page`.

Post schema keys: `post_plain`, `post_image`, `post_video`, `post_youtube`, `post_iframe`, `post_calendar`, `post_clock`, `post_world_clock`, `post_counter`, `post_forecast`, `post_feed`, `post_entur`, `post_power_price`, `post_poster`, `post_time_schedule`.

Media box schema keys: `media_box_image`, `media_box_icon`, `media_box_emoji`, `media_box_gif`, `media_box_unsplash`, `media_box_qr_code`.

Theme/palette/font schema keys: `theme`, `color_palette`, `font_family_remote_css`, `font_family_uploaded`.

Most content commands also have a `schema` subcommand for their own fields:
`pintomind resources schema [type]`, `pintomind color-palettes schema`, `pintomind font-families schema`,
`pintomind media-boxes schema`, `pintomind themes schema`.
