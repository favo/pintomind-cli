# Designing a theme

How to make a theme that reads well on a screen across a room, and how to check it before handing it
over. The commands are in `themes.md`.

## Workflow

1. **Ask what it is for**: brand colours and logo, fonts, light or dark, which channels and screens
   (landscape, portrait or both), a season or occasion.
2. **Fonts**: `pintomind font-families list`. Pick a readable body font (`suitable_for_body`) and,
   if wanted, a display font for headers. Add a Google Font with `font-families create remote-css`.
3. **Palette**: one brand colour is enough; `pintomind color-palettes create --name … --primary-color
   …` generates the weights. Add a secondary colour only for one accent.
4. **Start from a known-good shape** rather than inventing values: the example below, or
   `themes show` of a system theme you like (`themes list`), copied into a file and adjusted.
5. **Create** with `pintomind themes create --data @theme.json`. Map every theme area you add in
   `areas`; an unmapped area shows nowhere.
6. **Check it** (below), fix, and only then put it on a channel with
   `pintomind channels set-theme <channel> <theme>`: the change shows on every screen at once, so
   confirm with the user first. The same goes for updating a theme or palette that channels use.

## Design rules

- **Contrast first.** At least 7:1 between `text_color` and what is behind it (the background, an
  area's fill, a card). A screen is read from metres away, at an angle, in daylight. Muted text and
  card fills are fractions of the text colour, so a weak base contrast gets weaker in captions and
  cards.
- **Pick a side.** Dark backgrounds (near-black, `primary-900/950`) with white or `primary-50/100`
  text suit most screens: less glare, and saturated accents pop. Light themes (white, `primary-50`)
  need dark text (`primary-900/950` or black) and a gap colour such as `#CCCCCC`.
- **Palette weights**: solid fills `primary-600/700` with white text, soft fills `primary-200` with
  `primary-900` text, highlights (`mark_color`) `primary-300`, chart lines `primary-500`, the second
  hand `primary-600`. Use `secondary-*` for one accent at most.
- **One accent area.** An accent style on one layout area (B, or the clock) frames the layout; on
  every area it fights the content. Text on an accent area must clear the same contrast bar against
  the accent fill.
- **Images need help.** Over a background image use a text shadow (`text_shadow_color` `#000000`,
  `text_shadow_opacity` 30–50) and a frame area with a translucent fill (or a
  `content_frame_extension`), so text sits on a card. Pick calm, dark or blurred images.
- **Glass** (`glass_extension` on a theme area) suits a background image or a strong colour behind
  it: a light tint (`#FFFFFF` at opacity 10–20), a thin rim (`border_color` `#FFFFFF66`,
  `border_width` 1) and rounded corners. Over a flat colour there is nothing to blur. Older screens
  show it without blur, so the tint alone must still give enough contrast.
- **Patterns and textures** are decoration: keep them low-contrast against the background colour.
  Most textures darken (multiply), so they vanish on black and dull light colours.
- **Fonts.** The body font must stay readable at small sizes; display fonts are for headers only.
  Uppercase headers (`text_transform_header`) read well when short. Weight and text size are not
  theme settings: text scales with the screen and the post.
- **Gaps.** `composition_gap` 4–10 with a colour slightly off the background separates areas. An
  empty gap colour lets the background show through, which suits rounded or glass areas.
- **Screen effects** (`themes set-effect`) draw over the whole screen, posts included. Keep them to
  themes with a calm layout, for a season or an occasion, and check that text stays readable while
  particles pass over it.
- **Signal colours and extensions** are for the posts the channel actually shows (clock, calendar,
  weather, counters). Don't spend effort on ones nothing on the channel uses.

## Checking a theme

Look before you call it done, in landscape and portrait.

1. `pintomind themes preview-url <id>` gives a link to a preview of the theme on sample content that
   works without logging in for an hour. Open it (or show it to the user) at 16:9 and 9:16.
2. Check every text against what is behind it: the background, each mapped theme area, cards.
   Aim for 7:1.
3. Check each theme area shows where you meant: `pintomind themes areas <id>` lists the mapping;
   every area should be in it.
4. On a real channel: `pintomind channels preview <channel-id>`, or a test channel with the theme
   before the real one.

## Example

A dark brand theme with a card fill, an accent area on B and a glass area in portrait:

```json
{
  "name": "Acme dark",
  "font_family_header_id": 12, "font_family_body_id": 14, "color_palette_id": 7,
  "background_color": "primary-950", "text_color": "#FFFFFF",
  "background_contrast_strategy": "white",
  "composition_gap": 6, "composition_margin": 6, "composition_gap_color": "#000000",
  "mark_color": "primary-300", "chart_line_color": "primary-400", "second_hand_color": "secondary-500",
  "theme_extensions_attributes": [
    { "type": "contrast_extension", "contrast_color": "#FFFFFF", "contrast_opacity": 12, "border_radius": 8 }
  ],
  "theme_areas_attributes": [
    { "name": "Accent", "identifier": "accent",
      "background_color": "primary-600", "text_color": "#FFFFFF", "border_radius": 12 },
    { "name": "Glass", "identifier": "glass",
      "background_color": "#FFFFFF", "background_opacity": 15,
      "border_color": "#FFFFFF66", "border_width": 1, "border_radius": 16,
      "theme_extensions_attributes": [ { "type": "glass_extension", "glass_blur": 20 } ] }
  ],
  "areas": {
    "*:*:*": { "B": { "style": "accent" } },
    "*:portrait:*": { "A": { "style": "glass" } }
  }
}
```
