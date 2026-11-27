# Media, media boxes and image lookups

Media collections and uploads, media boxes (how images attach to posts, posters and themes), and icon/emoji/Unsplash/Giphy lookups.

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

### Upload tasks and direct uploads

`media upload` processes files asynchronously and returns task ids:

```bash
pintomind tasks show <task-id>
pintomind tasks wait <task-id>                          # poll until done, print the media ids
pintomind media create <collection-id> --source <signed-id> --wait   # from a POST /direct_uploads signed_id
```

## Media Boxes

Media boxes are visual containers used by posts, poster pages, and themes. A box is always created on its own first (`pintomind media-boxes create ...`), then attached where it should show up:

| Target | Field | Shape |
|---|---|---|
| Plain post media slots | `post.media_box_ids` | array of ids, slot order |
| Poster post | `resource.media_box_ids` | hash of node position → id |
| Any post background | `post.background_id` | single id |
| Theme background image | `theme.background_media_id` | single id |
| Theme logo | `theme.logo_id` | single id |

Every single-id field takes `null` to detach. Attaching a box owned by another account returns `404`.

**Plain posts** use `media_box_ids` as an **array** of IDs in slot order. The server writes each box UUID into the selected grid and sets `post_id` on the attached boxes. Max 4 boxes. The grid variation is selected automatically from body presence and length, media count, and media shape (wide/portrait). Pass `variation` (`--variation` on `posts create plain`) with a named layout to override, or `auto` to re-run the selection; omitting `variation` on update preserves the current layout.

To pick a layout yourself instead of relying on auto, run `pintomind schemas show post_plain`: the `variation` property's `x-variation-details` object describes every layout — media slot count, whether it has a body slot, the composition, what it's best for, and when auto selects it. Choose the entry matching your media count and body needs.

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
pintomind media-boxes create icon --icon-name rocket-launch --icon-type duotone --relative-size 0.8
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

# Replace attached boxes on an existing plain post (layout preserved when variation is omitted)
pintomind posts update <post-id> --data '{"media_box_ids":[203]}'

# Force a named layout, or re-run automatic layout selection
pintomind posts update <post-id> --data '{"variation":"R121-HIIT"}'
pintomind posts update <post-id> --data '{"variation":"auto"}'

# Attach boxes to a poster post (hash of node-position → id, via poster_page resource)
pintomind resources update <resource-id> --data '{"media_box_ids": {"hero-image": 201, "avatar": 202}}'

# Remove all poster media boxes
pintomind resources update <resource-id> --data '{"media_box_ids": {}}'
```

Removing a box ID from a plain post update deletes the orphaned box. For poster posts, each `media_box_ids` hash submission is a full replacement.

### Media box types

| Type | Required fields | Key optional fields |
|---|---|---|
| `media` | `media_id` (integer — ID from media library) | `background_size` (`"cover"`/`"contain"`, default `"cover"`), `x`, `y` (0–1 focal point, default `0.5`), `background_fill_type`, `relative_size` |
| `icon` | `icon_name` (string) | `icon_type`, `relative_size` (default `0.9`) |
| `emoji` | `emoji` (Unicode character, e.g. `"🎂"`) | `relative_size` (default `0.9`) |
| `gif` | `gif_id` (URL and metadata auto-fetched from Giphy) | `background_size`, `x`, `y`, `background_fill_type` |
| `unsplash` | `photo_id` (URL and author data auto-fetched from Unsplash) | `background_size` (default `"cover"`), `x`, `y`, `background_fill_type` |
| `qr_code` | `url` (URL to encode) | `relative_size` (default `0.9`) |

Inspect exact fields with `pintomind schemas show media_box_image`, `media_box_icon`, `media_box_emoji`, `media_box_gif`, `media_box_unsplash`, or `media_box_qr_code`.

## Icons, emojis, Unsplash and Giphy

```bash
pintomind icons                         # icon names, types, and categories for icon media boxes
pintomind icons arrow --category ui     # only matching names
pintomind emojis heart                  # `emoji` value for media_box.emoji (preset without a query)
pintomind unsplash forest [--page 2]    # photo IDs for media_box.photo_id; credit the photographer
pintomind giphy party [--offset 25]     # GIF IDs for media_box.gif_id; show "Powered by GIPHY"
```

Scope: `media_boxes:read`. Returns `{ names: [...], types: [...], categories: [...] }`. Use `icon_name` values here with `pintomind media-boxes create icon --icon-name <name> --icon-type <type>`.

## Tips

- Upload a file: `pintomind media upload <collection-id> ./photo.jpg --name "Lobby photo"`
