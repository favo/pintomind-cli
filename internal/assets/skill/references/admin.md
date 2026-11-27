# Connections, webhooks, integrations, identity and raw API

## Connections and login

Config is stored in `~/.config/pintomind/config.json`. Multiple connections are supported. A
connection either logs in with the user's own account (OAuth, `pintomind login`) or uses an API key.

```bash
# Log in with the user's account in the browser (recommended). Tokens go in the OS keychain.
pintomind login                                       # https://app.infoskjermen.no
pintomind login --url https://app.pintomind.com
pintomind login --name develop --url https://develop.infoskjermen.no
pintomind logout                                      # revoke the login, keep the connection

# API keys (scripts, CI, machines without a browser)
pintomind connection add app.infoskjermen.no sk-xxx
pintomind connection add develop sk-dev --url https://develop.infoskjermen.no

pintomind connection use develop
pintomind connection remove develop                   # logs out first on an OAuth connection
pintomind connection list                             # auth type and what each connection acts for
pintomind connection show                             # active connection details
```

`pintomind login` needs a browser on the same machine (it listens on 127.0.0.1 for the redirect);
over SSH, use an API key. After logging in, the user confirms the Pintomind CLI in the browser
("Logg inn"); "Avbryt" cancels and the CLI reports the login as cancelled, saving nothing. If the CLI knows no OAuth client for a host it asks for `--client-id`.

### Accounts and networks (OAuth connections)

An OAuth login belongs to the user, not to one account. Each OAuth connection acts for one account
(requests send `Account-Id`) or one network (`Network-Id`), chosen after login:

```bash
pintomind accounts                          # accounts the login may act for
pintomind networks                          # networks the login may act for
pintomind connection use-account <id>       # act for an account (no ID: pick interactively)
pintomind connection use-network <id>       # act for a network
pintomind --network-id 7 webhooks list      # override for one command (also --account-id)
```

Acting for a network reaches what a network API key reaches: screens of members that let the
network manage them (all screens for some network roles), and network-wide webhooks. Changing
anything needs a role that may manage the network ("CAN WRITE" in `networks`). Endpoints that
need an account (posts, media, …) answer 403 for a network connection; switch with `use-account`.
If an account needs SSO, `accounts` / `networks` say so: log in again through that SSO.

## Webhooks

```bash
pintomind webhooks events                  # event names; NETWORK-WIDE marks those a network may use
pintomind webhooks list                    # SCOPE: account or network
pintomind webhooks create --event screen.offline --url https://example.com/hook [--screen-id 42] [--disabled]
pintomind webhooks show <id>
pintomind webhooks update <id> --url https://… --enabled=false
pintomind webhooks test <id>               # synthetic delivery (enabled webhooks only)
pintomind webhooks deliveries <id>
pintomind webhooks delete <id>
```

The secret is in the create response only. Events: `screen.online`, `screen.offline`,
`screen.paired`, `screen.unpaired`, `resource.refresh_failed`, and `network.*` (accounts that own a
network). Acting for a network (`connection use-network` or `--network-id`), webhooks are the
network's network-wide ones: a `screen.*` webhook fires for every screen the network may manage,
and `--screen-id` must be one of those screens.

## Identity

```bash
pintomind me                 # user, account or network, OAuth client or API key
pintomind me --scopes
pintomind accounts           # OAuth: accounts the login may act for
pintomind networks           # OAuth: networks the login may act for
pintomind network [id]       # network identity and stats; OAuth: the given network, the connection's, or the only one
```

## Raw API access

For endpoints not yet covered by a dedicated command:

```bash
pintomind api /screens
pintomind api /screens/42
pintomind api GET /channels?sort_by=name
echo '{"screen":{"command":"reload"}}' | pintomind api PATCH /screens/42
```

## Integrations

Integrations (calendars, social media, Microsoft 365 …) and the resources made from them. Connecting a new one goes through the provider's login in the web app or the Go app.

```bash
pintomind integrations list
pintomind integrations types                    # what the account may connect
pintomind integrations show <id>                # with the resources made from it
pintomind integrations available <id>           # what the provider offers, numbered
pintomind integrations available <id> --group-id <workspace>   # Power BI: one workspace at a time
pintomind integrations add-resource <id> 3      # make a resource from offer number 3
pintomind integrations remove-resource <id> <resource-id>
pintomind integrations delete <id> [--with-resources]
```

A resource used in posts or shared with the network is not deleted; the error says why. An integration with resources is only deleted with `--with-resources`, which deletes them too.

## Installing and updating

```bash
pintomind setup init          # guided: connection, skills, shell completion
pintomind setup claude        # install or upgrade this skill for Claude Code (~/.claude/skills/pintomind)
pintomind setup codex         # the same for Codex / OpenAI agents
pintomind setup all           # both skills and shell completion
pintomind version
pintomind update              # update the CLI itself
```

After `pintomind update`, run `setup claude` / `setup codex` again so the installed skill matches the CLI.

## Tips

- Network webhooks: `pintomind --network-id <id> webhooks create --event screen.offline --url https://…` creates a network-wide webhook firing for every screen the network manages (deliveries are batched over 30 s with a `screens` list instead of `screen`, at most 500 per delivery; each entry has `account_id` and `occurred_at`).
- A 401 on an OAuth connection is retried after a token refresh; "login has expired or was revoked" means: `pintomind login --name <connection>`.
