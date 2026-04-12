# Changelog

All notable changes to GoDiscord are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

### Added

**New files**
- `logger.go` — `Logger` interface, `WithLogger` functional option, `NoopLogger` for silencing output in tests.
- `permissions.go` — `Permission` bitflag type with all 53 Discord permission constants (`PermKickMembers`, `PermModerateMembers`, etc.), composite sets (`PermModerator`, `PermDefaultText`), and `Has()`, `Any()`, `Add()`, `Remove()`, `Toggle()`, `String()` methods.
- `errors.go` — Typed `APIError` with HTTP status, Discord JSON error code, and convenience predicates (`IsNotFound()`, `IsForbidden()`, `IsRateLimit()`, `IsServerError()`). Includes all common Discord JSON error code constants (`ErrCodeMissingPermissions`, `ErrCodeUnknownMember`, etc.).

**`bot.go`**
- `New()` now accepts variadic `Option` values: `discord.New(token, intents, discord.WithLogger(l))`.
- Panics with a clear message if token is empty.
- `Bot.Use(...MiddlewareFunc)` to register command middleware chains.
- `Bot.Log()` to access the active logger from outside the package.

**`commands.go`**
- Quoted-string argument parsing: `!ban @user "repeated rule violations"` → `Args: ["@user", "repeated rule violations"]`.
- `HandlerFunc` and `MiddlewareFunc` types exported for type-safe middleware.
- `Bot.Use()` for registering global middleware that wraps every command handler.
- `Command.Usage` field for documenting expected argument syntax in help embeds.

**`interactions.go`**
- `Button()` and `LinkButton()` component builder helpers.
- `GetGlobalCommands()`, `CreateGlobalCommand()`, `BulkOverwriteGlobalCommands()` REST methods.
- `CreateFollowupMessage()`, `EditFollowupMessage()`, `DeleteFollowupMessage()` — send follow-up messages up to 15 minutes after an interaction.

**`rest.go`**
- `AddMemberRole()` / `RemoveMemberRole()` — add or remove a role from a guild member.
- `SendEmbedDM()` — send an embed via direct message.
- `EditMessageComplex()` — edit with full `MessageEdit` payload (content + embeds + components).
- Webhook support: `CreateWebhook()`, `GetWebhook()`, `GetChannelWebhooks()`, `DeleteWebhook()`, `ExecuteWebhook()`.
- REST errors now return `*APIError` instead of `fmt.Errorf` strings, enabling `errors.As` inspection.

**`types.go`**
- `Member` gains `PremiumSince`, `Pending`, `Permissions`, `CommunicationDisabledUntil` fields.
- `MessageSend` and `MessageEdit` gain `Components []Component` for attaching buttons and select menus.

### Changed

**`gateway.go`**
- Reconnect back-off changed from fixed 5 s to exponential back-off: starts at 1 s, doubles each failed attempt, caps at 5 min, with ±20 % random jitter.
- `sessionID` and `resumeURL` are now protected by `sessionMu sync.RWMutex` — eliminates a data race under concurrent reconnects.
- Heartbeat loop now detects zombie connections: if a heartbeat ACK is not received before the next beat, the connection is closed to force a Resume.
- `lastACK` is set to `true` before the first heartbeat to prevent false zombie detection on startup.
- Resume URL is cleared on dial failure so the next attempt uses the primary gateway.
- `InvalidSession` jitter is now random 1–5 s (per Discord recommendation) rather than fixed 1 s.

**`events.go`**
- All handler goroutines are now wrapped in `safeGo()` — panics inside handlers are caught, logged with a full stack trace, and the bot continues running.
- Unmarshal errors in each event case now log the error and event type for easier debugging.
- Unknown event types are silently discarded (forward-compatible with new Discord events).

**`rest.go`**
- `newRestClient` now accepts the `*Bot` for logger access (used for rate-limit log messages).
- Rate-limit sleeps now log the endpoint and retry delay via `b.log`.
- Response body is read into memory before decoding so `io.ReadAll` errors don't silently hide API error bodies.

### Fixed

- `Bot.SetActivity()` / `Bot.SetStatus()` could race on `gateway.conn` — now guarded via `sessionMu`.
- `gateway.stop()` now reads `conn` under `sessionMu` to avoid a nil-pointer race.
- `heartbeatLoop` no longer starts the first tick without waiting for jitter, preventing a duplicate heartbeat on reconnect.

---

## [0.1.0] — Initial release (feat/modbot-extensions)

### Added

- Core Gateway v10 implementation (`gateway.go`, `websocket.go`).
- Prefix command framework (`commands.go`).
- Event dispatcher with 8 event types (`events.go`).
- REST client with messages, channels, guilds, members, reactions (`rest.go`).
- Discord type definitions (`types.go`).
- Interactions v2: slash commands, select menus, interaction callbacks (`interactions.go`).
- 6 additional event types: `GUILD_MEMBER_ADD/REMOVE/UPDATE`, `GUILD_BAN_ADD/REMOVE`, `INTERACTION_CREATE`.
- REST methods for moderation: `ModifyGuildMember`, `TimeoutMember`, `GetGuildBan(s)`, `BulkDeleteMessages`, `ModifyChannel`, `EditChannelPermissions`, `DeleteChannelPermission`.
