# Posts and posters

Creating, updating, scheduling and publishing posts, attachments, poster templates and poster posts.

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
pintomind posts archive <id>                             # unpublishes from every channel
pintomind posts restore <id>                             # archived or soft-deleted; not republished
pintomind posts preview <id> [--data '{"title":"x"}'] [--channel-id 7] [--area F11]   # unsaved changes
pintomind posts preview --type plain --data '{"heading":"<p>Hi</p>"}'                 # a new post
pintomind posts preview --type poster --source-id 42                                  # a poster template
pintomind posts plain-layouts                            # valid `variation` values for plain posts
```

Previews answer a signed link, valid for an hour, that needs no login. Archiving and restoring need
the right to delete the post.

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
pintomind posts create plain --name "Duo" --heading "<p>Hi</p>" --body "<p>Text</p>" --media 42 --media 43 --variation R13-HTII  # named grid layout (or 'auto'); omit for auto. Layouts: schemas show post_plain

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
pintomind posts create counter --name "New Year" --countdown-type to_date --target-date 2027-01-01 --target-time 00:00 --channel-id 7
pintomind posts create counter --name "Since launch" --countdown-type from_date --from-date 2024-01-01 --template rings
pintomind posts create poster --name "Spring campaign" --source-id 42 --channel-id 7
pintomind posts create poster --name "Brand poster" --source-id 42 --color-palette-id 5 --channel-id 7
```

All `posts create <type>` helpers accept:
- `--channel-id N` (repeatable) — publish to one or more channels immediately
- `--area F11` — channel area for all publications
- `--position N` — 0-based play-order position within the channel area (0 = first; omit for channel default)
- `--document <file-or-url>` (repeatable) — upload into the document media collection and attach as a Go document attachment
- `--full-screen` — publish fullscreen to all channels
- `--priority <value>` — playback priority: `once_per_round` (default), `twice_per_round`, `between_each_post`, `only_post_in_area`, `only_post_in_channel`, `every_other_round`, `once_per_hour`. Any post schema (`pintomind schemas show post_<alias>`) describes each value's behavior.
- `--until <YYYY-MM-DD[THH:MM]>` — auto-expire the post at this date/time (attaches a one-period schedule that unpublishes on expiry). Time defaults to `23:59` when omitted.

`--until` is evaluated in each channel's local time zone. For richer schedules (recurring weekly windows, multiple periods, or other expiry actions) use `pintomind posts schedule set` after creating the post.

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

### Post attachments (Go mobile app)

Attachments are extra content shown with a post in the branded Go mobile app. Types: `text` (max one per post — edit the existing one instead of adding another), `link` (a tappable button; `link_type`: `link`, `mail`, `sms`, `tel`, `go_channel`; `url` is the email address or phone number for `mail`/`sms`/`tel`), `document` (references media in a document media collection). Schema: `pintomind schemas show post_attachments`.

```bash
pintomind posts attachments list <post-id>
pintomind posts attachments add text <post-id> --text "<p>More info</p>" --alignment center
pintomind posts attachments add link <post-id> --url https://example.com --label "Sign up"
pintomind posts attachments add link <post-id> --url hi@example.com --link-type mail
pintomind posts attachments add link <post-id> --url +4712345678 --link-type tel
pintomind posts attachments add document <post-id> --media 42          # existing document media
pintomind posts attachments add document <post-id> --file ./menu.pdf   # upload to document collection, then attach
pintomind posts attachments update <post-id> <attachment-id> --text "<p>Updated</p>"   # pass only fields to change
pintomind posts attachments delete <post-id> <attachment-id> --force
```

### Push notifications (Go mobile app)

A notification goes to the notification-enabled devices of every channel the post is published to.
Title (max 50) and body (max 150) are filled in from the post when left out; at most 12 pending per post.

```bash
pintomind posts notifications list <post-id>
pintomind posts notifications create <post-id> --send-at 2026-07-01T15:30:00Z [--title "…"] [--body "…"]
pintomind posts notifications cancel <post-id> <notification-id>
```

Shortcut at creation time: every `posts create <type>` helper accepts `--document <file-or-url>` (repeatable) — uploads into the default document media collection and attaches after the post is created.

### Creating posts — raw JSON (all post types)

`posts create --type <alias> --data '<json>'` accepts any post type. The CLI wraps it in `{"type": ..., "post": {...}}` for you.

**Important:** all post fields — including type-specific ones — are flat inside the `post` object. Do not wrap them in an `options` sub-object.

Always inspect the schema for the post type first to learn what fields are accepted:

```bash
pintomind schemas show post_image
pintomind schemas show post_plain
pintomind schemas show post_calendar
```

Common shared fields: `name`, `title`, `duration`, `show_title`, `area`, `priority` (playback frequency/takeover policy — enum and per-value behavior in every post schema), `background_id` (a MediaBox id used as post background — create one with `pintomind media-boxes create ...`; pass `null` on update to clear).

Typed `posts create` subcommands also accept `--background-media-box <id>` and the `image` subcommand accepts `--template center|maxpane|fullpane|image_grid`.

```bash
# Image post — use images[] with media_id values from `pintomind media upload`
# template: center | maxpane | fullpane | image_grid (legacy aspect_ratio/grid_layout/variation removed)
pintomind posts create --type image --data '{
  "name":"Spring campaign",
  "title":"New collection",
  "duration_per_item":7,
  "images":[{"media_id":42},{"media_id":43}],
  "template":"fullpane"
}'

# Video post — video_media_id for one video, video_media_ids for several played in order
pintomind posts create --type video --data '{
  "name":"Promo",
  "video_media_id":101,
  "sound":true
}'
pintomind posts create --type video --data '{"name":"Promos","video_media_ids":[101,102]}'

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
pintomind posts publish <post-id> --channel-id 7 --position 0       # 0-based play-order position in the area (omit for channel default)
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

## Tips

- Image posts use `images:[{"media_id":...}]`; video posts use `video_media_id:<id>` for one video or `video_media_ids:[...]` for several; neither uses `resource_ids`.
- Post fields are always flat inside `post`, no `options` sub-object.
