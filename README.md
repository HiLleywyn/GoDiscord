# GoDiscord

> A zero-dependency Discord bot framework written in 100% pure Go.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

GoDiscord implements [Discord Gateway v10](https://discord.com/developers/docs/topics/gateway) and the Discord REST API v10 using only the Go standard library — no `github.com/gorilla/websocket`, no `github.com/bwmarrin/discordgo`, no external packages at all.

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

GoDiscord requires **Go 1.21 or later**.

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
// Check whether a member has both KickMembers and BanMembers.
perms := discord.Permission(member.Permissions)
if perms.Has(discord.PermKickMembers, discord.PermBanMembers) {
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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

---

## License

MIT — see [LICENSE](LICENSE).
