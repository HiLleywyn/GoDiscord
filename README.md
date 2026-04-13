# GoDiscord

> A zero-dependency Discord bot framework written in 100% pure Go.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![CI](https://github.com/hilleywyn/godiscord/actions/workflows/ci.yml/badge.svg)](https://github.com/hilleywyn/godiscord/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

GoDiscord implements [Discord Gateway v10](https://discord.com/developers/docs/topics/gateway) and the Discord REST API v10 using only the Go standard library — no `github.com/gorilla/websocket`, no `github.com/bwmarrin/discordgo`, no external packages at all. It ships its own RFC 6455 WebSocket client and a typed event dispatcher, and every handler runs under panic recovery so one misbehaving callback can't take the bot down.

---

## Features

| Area | What's included |
|------|----------------|
| **Gateway** | WebSocket connection, Identify/Resume, heartbeat, exponential back-off reconnect, zombie detection |
| **Events** | Typed handlers, panic-recovery goroutines, 14 event types |
| **Commands** | Prefix-based routing, quoted-string args, middleware chain |
| **Interactions** | Slash commands, select menus, buttons, ephemeral responses, follow-up messages |
| **REST** | Messages, reactions, guilds, members, roles, channels, webhooks, bans |
| **Utilities** | `Permission` bitflag type (53 constants), pluggable `Logger` interface, structured `APIError` |

---

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    discord "github.com/hilleywyn/godiscord"
)

func main() {
    bot := discord.New("YOUR_BOT_TOKEN", discord.IntentGuilds|discord.IntentGuildMessages|discord.IntentMessageContent)

    bot.OnReady(func(b *discord.Bot, e *discord.ReadyEvent) {
        fmt.Println("Logged in as", e.User.Tag())
    })

    bot.AddCommand(&discord.Command{
        Name:        "ping",
        Description: "Check that the bot is alive",
        Handler: func(ctx *discord.CommandContext) {
            ctx.Reply("Pong! 🏓")
        },
    })

    log.Fatal(bot.Run())
}
```

See [`example/basic/`](example/basic/) for a runnable starter bot and [`example/slash/`](example/slash/) for slash-command usage.

---

## Installation

```bash
go get github.com/hilleywyn/godiscord
```

GoDiscord requires **Go 1.21 or later**. There are no transitive
dependencies: after `go get`, `go.sum` lists only GoDiscord itself.

### Vendored builds

For self-contained deployments (e.g. scratch Docker images), vendor
the module and build with `-mod=vendor`:

```bash
GOWORK=off go mod tidy
GOWORK=off go mod vendor
go build -mod=vendor ./...
```

The `GOWORK=off` disables workspace mode so a sibling `go.work` file
doesn't pull in live-dev paths during vendoring.

---

## Configuration

### Intents

Discord requires bots to declare which events they wish to receive. Combine intent flags with `|`:

```go
intents := discord.IntentGuilds |
    discord.IntentGuildMembers |        // privileged
    discord.IntentGuildMessages |
    discord.IntentMessageContent        // privileged
```

Privileged intents (`GuildMembers`, `GuildPresences`, `MessageContent`) must also be enabled in the [Discord Developer Portal](https://discord.com/developers/applications) → your app → **Bot** → **Privileged Gateway Intents**.

### Functional options

Pass `Option` values to `New()` to customise the bot:

```go
bot := discord.New(token, intents,
    discord.WithLogger(myZapLogger), // replace the default log.Printf logger
)
```

---

## Events

Register event handlers before calling `bot.Run()`:

```go
bot.OnMessageCreate(func(b *discord.Bot, m *discord.Message) {
    // ...
})

bot.OnGuildMemberAdd(func(b *discord.Bot, e *discord.GuildMemberAddEvent) {
    // ...
})

bot.OnInteractionCreate(func(b *discord.Bot, i *discord.Interaction) {
    // ...
})
```

All handlers run in separate goroutines. Panics are caught and logged — a bad handler will never crash the bot process.

**Supported events:**

| Event | Handler type |
|-------|-------------|
| `READY` | `ReadyHandler` |
| `MESSAGE_CREATE` | `MessageCreateHandler` |
| `MESSAGE_UPDATE` | `MessageUpdateHandler` |
| `MESSAGE_DELETE` | `MessageDeleteHandler` |
| `GUILD_CREATE` | `GuildCreateHandler` |
| `GUILD_DELETE` | `GuildDeleteHandler` |
| `GUILD_MEMBER_ADD` | `GuildMemberAddHandler` |
| `GUILD_MEMBER_REMOVE` | `GuildMemberRemoveHandler` |
| `GUILD_MEMBER_UPDATE` | `GuildMemberUpdateHandler` |
| `GUILD_BAN_ADD` | `GuildBanAddHandler` |
| `GUILD_BAN_REMOVE` | `GuildBanRemoveHandler` |
| `MESSAGE_REACTION_ADD` | `ReactionAddHandler` |
| `MESSAGE_REACTION_REMOVE` | `ReactionRemoveHandler` |
| `INTERACTION_CREATE` | `InteractionCreateHandler` |

---

## Prefix Commands

```go
bot.SetPrefix("!")

bot.AddCommand(&discord.Command{
    Name:        "ban",
    Description: "Ban a member",
    Usage:       "@user [days] [reason]",
    Handler: func(ctx *discord.CommandContext) {
        // ctx.Args is quoted-string aware:
        // !ban @user 7 "posting malware links" → ["@user", "7", "posting malware links"]
        if len(ctx.Args) == 0 {
            ctx.Reply("Usage: !ban @user [days] [reason]")
            return
        }
        // ...
    },
})
```

### Middleware

```go
bot.Use(func(next discord.HandlerFunc) discord.HandlerFunc {
    return func(ctx *discord.CommandContext) {
        log.Printf("[cmd] %s invoked by %s", ctx.Command.Name, ctx.Message.Author.Username)
        next(ctx)
    }
})
```

---

## Slash Commands & Interactions

```go
// Register a guild command on GUILD_CREATE.
bot.OnGuildCreate(func(b *discord.Bot, g *discord.Guild) {
    b.Rest.CreateGuildCommand(b.Self().ID, g.ID, discord.ApplicationCommand{
        Name:        "hello",
        Description: "Say hello",
    })
})

// Handle it.
bot.OnInteractionCreate(func(b *discord.Bot, i *discord.Interaction) {
    if i.Type != discord.InteractionTypeApplicationCommand {
        return
    }
    if i.Data.Name == "hello" {
        b.Rest.CreateInteractionResponse(i.ID, i.Token, discord.InteractionResponse{
            Type: discord.InteractionCallbackTypeChannelMessage,
            Data: &discord.InteractionResponseData{
                Content: "Hello, " + i.Author().Username + "!",
                Flags:   discord.MessageFlagEphemeral,
            },
        })
    }
})
```

### Select menus

```go
components := []discord.Component{
    discord.ActionRow(
        discord.StringSelect("menu:main", "Choose an option…", []discord.SelectMenuOption{
            {Label: "Option A", Value: "a"},
            {Label: "Option B", Value: "b"},
        }),
    ),
}
```

### Buttons

```go
components := []discord.Component{
    discord.ActionRow(
        discord.Button("Confirm", "confirm:yes", discord.ButtonStyleSuccess, false),
        discord.Button("Cancel",  "confirm:no",  discord.ButtonStyleDanger,  false),
    ),
}
```

---

## Permissions

```go
// Parse the decimal string Discord sends for members and roles.
// An empty member.Permissions returns (0, nil) - only a non-numeric
// or out-of-range string produces a non-nil error.
perms, err := discord.ParsePermission(member.Permissions)
if err != nil {
    // member.Permissions was malformed (not a base-10 uint64).
}

// Check whether a member has both KickMembers and BanMembers.
if perms.Has(discord.PermKickMembers, discord.PermBanMembers) {
    // ...
}

// Check whether a member has at least one of several flags.
if perms.Any(discord.PermManageMessages, discord.PermAdministrator) {
    // ...
}

// Build a permission set.
modPerms := discord.Permission(0).Add(
    discord.PermManageMessages,
    discord.PermModerateMembers,
    discord.PermViewAuditLog,
)
```

---

## Rate Limiting

GoDiscord handles `429 Too Many Requests` responses automatically. When Discord
returns a rate-limit response the client reads the `Retry-After` header, sleeps
for the indicated duration, and retries the request. Retries are capped at
**3 attempts per call** (`maxRateLimitRetries`); if the budget is exhausted a
`*APIError` with `StatusCode == 429` is returned so you can decide how to
proceed.

```go
var apiErr *discord.APIError
if errors.As(err, &apiErr) && apiErr.IsRateLimit() {
    // retry budget exhausted — back off at a higher level
}
```

The retry budget applies per-request, not per-process, so an occasional
429 on one endpoint doesn't starve the next call.

## Error Handling

REST calls return `*APIError` on failure, which can be inspected with `errors.As`:

```go
_, err := bot.Rest.GetGuildMember(guildID, userID)

var apiErr *discord.APIError
if errors.As(err, &apiErr) {
    switch {
    case apiErr.IsNotFound():
        // user not in guild
    case apiErr.IsForbidden():
        // missing permissions
    case apiErr.IsServerError():
        // Discord-side 5xx — retry after a back-off
    case apiErr.Code == discord.ErrCodeMissingPermissions:
        // specific Discord error code
    }
}
```

---

## Custom Logger

Implement the `Logger` interface to route GoDiscord's log output to any sink:

```go
type zapAdapter struct{ log *zap.SugaredLogger }

func (a zapAdapter) Printf(f string, args ...interface{}) { a.log.Infof(f, args...) }
func (a zapAdapter) Println(args ...interface{})          { a.log.Info(args...) }

bot := discord.New(token, intents, discord.WithLogger(zapAdapter{sugar}))
```

To silence all logging (e.g. in tests):

```go
bot := discord.New(token, intents, discord.WithLogger(discord.NoopLogger{}))
```

---

## File Structure

```
godiscord/
├── bot.go           Entry point — Bot struct, New(), lifecycle, event/command registration
├── gateway.go       Gateway v10 — WebSocket connection, heartbeat, reconnect loop
├── websocket.go     Zero-dependency RFC 6455 WebSocket client
├── events.go        Typed event dispatcher with panic-recovery goroutines
├── commands.go      Prefix command routing, quoted-arg parser, middleware chain
├── interactions.go  Slash commands, buttons, select menus, interaction callbacks
├── rest.go          Discord REST API v10 client
├── types.go         All Discord data types and Gateway payload structs
├── permissions.go   Permission bitflag type + all 53 Discord permission constants
├── errors.go        APIError type + Discord JSON error code constants
├── logger.go        Logger interface + default/noop implementations
└── example/
    ├── basic/       Ping-pong prefix bot starter
    └── slash/       Slash command + ephemeral response starter
```

---

## Vendoring

If you're using GoDiscord as a vendored dependency (e.g. in a self-contained Docker build):

```bash
GOWORK=off go mod tidy
GOWORK=off go mod vendor
go build -mod=vendor ./...
```

---

## Troubleshooting

### Bot comes online then immediately disconnects

Discord closes the WebSocket with a `4014` close code when a **privileged
intent** (`GuildMembers`, `GuildPresences`, `MessageContent`) is declared in
code but not enabled in the Developer Portal. Go to
[Discord Developer Portal](https://discord.com/developers/applications) →
your application → **Bot** → **Privileged Gateway Intents** and enable the
intents your bot requests. GoDiscord surfaces the 4014 close code in the
gateway log so it's straightforward to recognise in traces.

### Messages are received but `m.Content` is always empty

`MessageContent` is a privileged intent (see above). You must both request
`discord.IntentMessageContent` in `discord.New()` **and** enable it in the
Developer Portal.

### Bot reconnects repeatedly with "session not resumable"

This is normal after a long outage or when the session sequence number falls
too far behind. GoDiscord automatically clears the session and re-identifies
with a fresh `Identify` payload. No action is required.

### `*APIError` with status 429 is returned

GoDiscord automatically retries rate-limited requests up to 3 times. Receiving
a `429` error means the budget was exhausted. Add a backoff at the call site or
reduce the frequency of the operation.

### `discord: token must not be empty` panic at startup

`discord.New()` panics if the token string is empty or whitespace-only.
Ensure `DISCORD_TOKEN` (or however you supply the token) is set in the
process environment before calling `New()`.

---

## Security

GoDiscord is a framework library. Its security posture:

- **No SQL, no shell execution** — there is no injection surface beyond what
  bot code introduces itself.
- **TLS only** — the Gateway and REST client connect exclusively over TLS.
- **Bounded allocation** — WebSocket frame payloads are capped at 64 MiB
  (`maxFramePayload`) to prevent memory-exhaustion attacks from a
  compromised gateway connection. Negative payload lengths (8-byte length
  field with its high bit set) are rejected before allocation.
- **Bounded retries** — Rate-limit retries are capped at
  `maxRateLimitRetries` to prevent infinite recursion from a non-compliant
  server.
- **Path-safe REST** — `AddReaction` and `RemoveReaction` pass the emoji
  parameter through `url.PathEscape`, blocking path-injection via a
  crafted emoji string.
- **Token isolation** — The bot token is stored in an unexported field and
  never logged; it appears only in `Authorization` headers.

Please report security issues privately via GitHub's
[Security Advisories](https://github.com/hilleywyn/godiscord/security/advisories/new)
rather than opening a public issue.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

---

## License

MIT — see [LICENSE](LICENSE).
