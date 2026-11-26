# Themes, colour palettes and font families: commands

The commands and the API shape. For designing a theme (what looks good, the workflow, how to check
it), read `theme-design.md` as well.

A theme sets how a channel looks on every screen showing it: colours, fonts, background, gaps,
signal colours, named **theme areas** (styles a layout area can use), **extensions** (extra styling
for cards, clocks, calendars, glass …) and an optional ambient **screen effect**. A channel always
has one theme; screens show their channel's theme.

## Themes

```bash
pintomind themes list
pintomind themes list --sort-by name
pintomind themes list --page 2 --per-page 50
pintomind themes show <id>                 # full JSON: attributes, areas, theme_areas, theme_extensions (with ids), preview_url
pintomind themes areas <id>                # theme areas, extensions and the area mapping as tables, with ids
pintomind themes preview-url <id>          # signed preview link, valid one hour, no login needed
pintomind themes stats
pintomind themes schema                    # JSON schema of the theme fields
pintomind themes create --data '{"name":"My theme","font_family_header_id":1,"font_family_body_id":2,"background_color":"#1a1a2e","text_color":"#ffffff"}'
pintomind themes create --data @theme.json # from a file; --data - reads stdin
pintomind themes update <id> --data '{"name":"Updated name"}'
pintomind themes update <id> --data '{"color_palette_id":5}'
pintomind themes set-areas <id> --data '{"*:*:*":{"B":{"style":"accent"}}}'
pintomind themes set-effect <id> snow      # snow, rain, thunderstorm, leaves, hearts, sparkles or none
pintomind themes delete <id>
pintomind themes delete <id> --force
pintomind channels set-theme <channel-id> <theme-id>
```

`create` needs `name`, `font_family_header_id`, `font_family_body_id`, `background_color` and
`text_color`. Unknown fields are refused with `422` and the list of permitted ones. `update` changes
only the fields given; every update makes the screens using the theme reload it. Only themes the
account owns can be changed; `list` also shows system and network themes, which can be used but not
edited. Deleting a theme resets channels using it to the account's standard theme. Every `--data`
flag in the CLI takes inline JSON, `@path/to/file.json` or `-` for stdin.

### Theme fields

All flat inside the `theme` object (`pintomind themes schema` has types, enums and ranges):

| Group | Fields |
|---|---|
| Identity | `name`, `color_palette_id`, `font_family_header_id`, `font_family_body_id` |
| Background | `background_color`, `background_opacity` (0–100), `background_pattern`, `background_noise` (a texture), `background_repeat`, `background_size`, `background_position_x/y`, `background_media_id` (image, see below) |
| Cards | `background_contrast_strategy`: `text`, `text-inverted`, `black`, `white`, `primary`, `primary-inverted` (the fill behind cards and boxes in posts) |
| Text | `text_color`, `text_shadow_color`, `text_shadow_opacity`, `text_transform_header`, `text_transform_body` (`none uppercase lowercase capitalize`) |
| Layout | `composition_gap`, `composition_margin` (0–20), `composition_gap_color`, `composition_margin_color`, `combined_composition_frame_color`, `logo_id`, `logo_max_width` (10–150), `logo_max_height` (30–150), `logo_swap` |
| Signal colours | `mark_color` (highlights), `chart_line_color`, `second_hand_color`, `counter_color`, `calendar_dot_color`, `forecast_icons_color`, `plain_icons_color`, `contrast_color`, `poll_bar_color` |
| Effect | `screen_effect`: `snow rain thunderstorm leaves hearts sparkles`, `""` for none |
| Nested | `theme_extensions_attributes`, `theme_areas_attributes`, `areas` (below) |

`text_weight_*` and `icon_name` are stored but change nothing on screen. There is no `variation`
field; the API refuses it.

**Colour values** are hex (`#1F8A8A`, `#FFFFFF66` with alpha) or a palette reference
`<primary|secondary|tertiary>-<weight>`, weights `50 100 150 200 300 400 500 600 700 800 900 950`,
which follow the theme's `color_palette_id`. A reference to a colour the palette lacks (no
secondary, say) falls back to the same primary weight. Empty means "not set".

### Theme areas and the area mapping

A **theme area** is a named style: `name`, `identifier` (the key the mapping uses), background
fields, `text_color` and the other text and signal colours, border fields (`border_color`,
`border_style` `solid dashed dotted` (anything else is refused), `border_width` 0–10, `border_radius` or the four corner radii
0–20) and its own `theme_extensions_attributes`.

A theme area shows only where **`areas`** maps it onto a layout area:

```json
{
  "*:*:*":           { "B": { "style": "accent" } },
  "*:portrait:*":    { "A": { "style": "glass" } },
  "ABC:landscape:*": { "CLOCK": { "style": "none" } }
}
```

- Keys select layouts, more specific winning: `*:*:*` (all), `*:portrait:*` / `*:landscape:*`
  (every layout in one orientation), `<template>:*:*`, `<template>:<orientation>:*`,
  `<template>:<orientation>:<extension>`. Other key shapes are refused.
- Layout areas: `A`, `B`, `C`, `D`, `CLOCK`, `TICKER`, `LOGO`; a channel's layout decides which it
  has (`pintomind channels layouts`). `none` is the plain theme.
- `areas` **replaces the whole mapping**; send every key you want to keep. A style that isn't one
  of the theme's identifiers is refused, and so is renaming or removing an area the mapping uses
  unless the same request sends a new `areas`.
- A theme area left out of the mapping shows nowhere. `set-areas` sends just the mapping.

### Changing nested areas and extensions

An extension on a theme area overrides the same extension on the theme inside that area only; values
it leaves blank fall back to the theme's.

`theme_areas_attributes` and `theme_extensions_attributes` are arrays. An entry without `id` is
added, one with an `id` is changed, and `id` plus `"_destroy": true` removes it. Get the ids from
`themes areas <id>` or `themes show <id>`.

```bash
# Add an area with glass and map it in portrait
pintomind themes update 12 --data '{
  "theme_areas_attributes": [
    { "name": "Glass", "identifier": "glass", "background_color": "#FFFFFF", "background_opacity": 15,
      "border_color": "#FFFFFF66", "border_width": 1, "border_radius": 16,
      "theme_extensions_attributes": [ { "type": "glass_extension", "glass_blur": 20 } ] }
  ],
  "areas": { "*:*:*": { "B": { "style": "accent" } }, "*:portrait:*": { "A": { "style": "glass" } } }
}'

# Change area 5's fill, remove extension 9 from it, remove theme-level extension 3
pintomind themes update 12 --data '{
  "theme_areas_attributes": [ { "id": 5, "background_color": "primary-700",
                                "theme_extensions_attributes": [ { "id": 9, "_destroy": true } ] } ],
  "theme_extensions_attributes": [ { "id": 3, "_destroy": true } ]
}'
```

### Extension types

`type` is the alias below. Theme-level extensions apply to the whole theme; area-level ones only
inside that theme area.

| `type` | Where | Properties | What it does |
|---|---|---|---|
| `contrast_extension` | theme, area | `contrast_color`, `contrast_opacity` (30), `contrast_text_color`, `contrast_current_color`, border fields | card/box fill behind text in most posts (calendar, departures, scores, grids) |
| `content_frame_extension` | area only | `content_frame_color` (#000), `content_frame_opacity` (30), `content_frame_padding` (20), border fields | a translucent card with padding inside the post |
| `calendar_extension` | theme, area | `header_background_color`, `header_text_color`, `contrast_color`, `contrast_opacity`, `contrast_text_color` | calendar header and events |
| `clock_extension` | theme, area | `time_color`, `date_color`, `delimiter_color`, `second_hand_color`, `widget_*_color` | clocks, world clock, clock widgets |
| `counter_extension` | theme, area | `counter_color`, `text_color`, `marker_color` | countdowns and counters |
| `forecast_extension` | theme, area | `icon_type` (`duotone` or `themed`), `icon_color`, `inner_sun_color`, `outer_sun_color`, `raindrop_color`, `moon_color` | weather icons |
| `poster_extension` | theme, area | `background_color`, `text_color` | colours of posters that follow the theme palette |
| `glass_extension` | area only | `glass_blur` 0–40 (16), `glass_saturation` 100–200 (160), `glass_highlight_opacity` 0–100 (50), `glass_shadow_opacity` 0–100 (25) | frosted glass: blurs what is behind the area, adds a sheen, rim highlight and shadow. The tint and rim are the area's own `background_color`/`background_opacity` and border. Not in the web editor (API only); screens without backdrop blur show it unblurred |

### Theme background image and logo

Both are media boxes attached by id (`background_media_id`, `logo_id`). Use the `set-background` / `set-logo` helpers — they create the box and attach it in one step:

```bash
# From a local file or URL: uploads to the default background/logo media collection,
# wraps it in a media box, attaches it. Override the collection with --media-collection.
pintomind themes set-background <theme-id> --file ./background.jpg
pintomind themes set-background <theme-id> --file https://example.com/bg.jpg --background-size cover
pintomind themes set-logo <theme-id> --file ./logo.png

# From the media library (creates a `media` box, then attaches it)
pintomind themes set-background <theme-id> --media-id 42
pintomind themes set-background <theme-id> --media-id 42 --background-size cover --x 0.4 --y 0.5
pintomind themes set-logo <theme-id> --media-id 77

# From Unsplash or Giphy
pintomind themes set-background <theme-id> --unsplash-photo-id abc123 --background-size cover
pintomind themes set-background <theme-id> --gif-id xT9IgG50Fb7Mi0only

# Attach a box that already exists
pintomind themes set-background <theme-id> --media-box 4711

# Remove
pintomind themes set-background <theme-id> --clear
pintomind themes set-logo <theme-id> --clear
```

Exactly one of `--file`, `--media-box`, `--media-id`, `--unsplash-photo-id`, `--gif-id`, `--clear` per call. Add `--json` to get the full theme response instead of the summary lines. `--file` accepts the same inputs as `media upload` (local path or URL) and waits for the upload task to finish.

The raw fields also work through `themes create` / `themes update`:

```bash
pintomind themes update <theme-id> --data '{"background_media_id":4711,"logo_id":4712}'
pintomind themes update <theme-id> --data '{"background_media_id":null}'
```

`media`, `unsplash`, and `gif` boxes make sense as a background; a logo is normally a `media` box. A box belonging to another account gives `404`. Replacing a box leaves the previous one in the account — delete it with `pintomind media-boxes delete <id>` if it is no longer used. An image background hides the pattern.

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

Colours are hex (e.g. `#1F8A8A`). On create the API generates twelve weights (50 … 950) per base
colour. On update, a base colour that changes regenerates its weights; one left out or sent
unchanged keeps them. **Every theme using the palette changes with it**, and the screens showing
those themes reload: check `color-palettes show <id>` (its `themes`) before changing a palette in
use. Deleting a palette repoints themes using it to another available palette; returns an error if
none exists. Palettes shared with network members cannot be deleted.

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
pintomind font-families update <id> --force-text-transform uppercase
pintomind font-families delete <id>
pintomind font-families delete <id> --force
```

Font types:
- `remote_css` — references a hosted CSS file. Required: `--url`. Optional: `--font-name` (picked from CSS if omitted). The CSS must contain `@font-face` rules.
- `uploaded` — uploads `.ttf`/`.otf`/`.woff`/`.woff2` files as multipart/form-data. Required: `--font-normal`, `--font-bold`. Optional: `--font-italic`, `--font-bold-italic`.
- `standard` — built-in fonts, read-only (cannot be created or deleted).

`force_text_transform` makes the font always upper/lower/capitalised; a theme's own
`text_transform_*` wins over it. `suitable_for_body` marks fonts readable at small sizes.
Deleting a font resets themes using it to the system default (`figtree`).
Fonts shared with network members cannot be deleted.

## Tips

- List palettes: `pintomind color-palettes list --json | jq '.items[] | {id, name, primary_color}'`
- Create a palette and use it: `pintomind color-palettes create --name "Brand" --primary-color "#1F8A8A"`, then `pintomind themes update <theme-id> --data '{"color_palette_id":<id>}'`
