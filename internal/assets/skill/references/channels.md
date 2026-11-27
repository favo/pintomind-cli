# Channels

Channels (what screens show), layouts and sharing in the Go app.

## Channels

```bash
pintomind channels list
pintomind channels list --sort-by name
pintomind channels list --page 2 --per-page 50
pintomind channels show <id>
pintomind channels posts <id>           # account token required
pintomind channels posts <id> --visible                 # only currently visible publications (--hidden for the inverse)
pintomind channels posts <id> --sort-by position        # play order (also: area, created_at, updated_at; append :desc; comma-separate)
pintomind channels posts <id> --fields id,title,position,will_be_visible_at,will_be_hidden_at --json
pintomind channels stats                # requires channels:read:stats scope
pintomind channels stats <id>
pintomind channels set-theme <channel-id> <theme-id>
pintomind channels set-theme --all <theme-id>

pintomind channels create --name Lobby                                   # screen channel, template AB
pintomind channels create --name Lobby --template ABC --orientation portrait --extension ticker
pintomind channels create --name "Our app" --type go
pintomind channels update <id> --name "New name" --theme-id 3
pintomind channels update <id> --template AB --orientation landscape   # screen channels
pintomind channels update <id> --board-label News --board-background-color primary-600   # Go channels
pintomind channels delete <id>                       # soft-delete (off screens and schedules); again to hard-delete
pintomind channels layouts [--template AB] [--orientation portrait] [--favorites]   # valid template/orientation/extension
pintomind channels move-post <channel-id> <post-id> --area B --position 0           # F11 = full screen
pintomind channels preview <id>                      # signed link, valid an hour, no login
pintomind channels search kirke                      # public Go channels (channels:search scope)
pintomind channels search --lat 60.39 --lng 5.32
```

Creating and deleting channels needs a role that manages channels.

### Sharing a channel in the Go app

```bash
pintomind channels sharing show <id>
pintomind channels sharing start <id> --code menighet         # everyone with a share code (made up without --code)
pintomind channels sharing start <id> --invites [--web]       # only invited email addresses
pintomind channels sharing update <id> --web=false            # also open in a browser or not
pintomind channels sharing stop <id>
pintomind channels sharing codes add <id> [--code x]          # at most four codes
pintomind channels sharing codes update <id> <code-id> <new-code>
pintomind channels sharing codes remove <id> <code-id>
pintomind channels sharing invites list <id> [--q kari]
pintomind channels sharing invites add <id> kari@example.com ola@example.com
pintomind channels sharing invites rename|resend|remove <id> <invite-id> ...
```

The way of sharing can't be changed: stop and share again. Starting to share needs a login or an
API key that belongs to a user.

## Tips

- Apply a theme to every channel: `pintomind channels set-theme --all <theme-id>` (confirm first: every screen changes)
