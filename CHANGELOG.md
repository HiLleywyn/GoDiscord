# Changelog

All notable changes to GoDiscord are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

### Added

- **`commands.go`** — `CommandContext` now exposes `GuildID`, `ChannelID`,
  `AuthorID`, and `Member` fields, populated from the triggering message.
  Previously callers had to extract these manually from `ctx.Message`.

- **`commands.go`** — Permission gate on `Command`: `RequiredPermissions`
  (Discord bitfield, all bits must be present) and `PermCheck` (custom
  `func(*CommandContext) bool`). Either gate failing invokes the bot's
  denied-callback (if registered) and skips the handler.

- **`bot.go`** — `Bot.SetCommandDenied(fn func(*CommandContext, string))`:
  registers a callback called when a permission gate blocks a command.

- **`bot.go`** — 18 new event registration methods: `OnChannelCreate`,
  `OnChannelUpdate`, `OnChannelDelete`, `OnGuildUpdate`, `OnGuildRoleCreate`,
  `OnGuildRoleUpdate`, `OnGuildRoleDelete`, `OnThreadCreate`, `OnThreadUpdate`,
  `OnThreadDelete`, `OnInviteCreate`, `OnInviteDelete`, `OnWebhooksUpdate`,
  `OnVoiceStateUpdate`, `OnTypingStart`, `OnMessageDeleteBulk`,
  `OnReactionRemoveAll`, `OnReactionRemoveEmoji`.

- **`events.go`** — Gateway dispatch for all 18 new event types above.
  `CHANNEL_CREATE/UPDATE/DELETE`, `GUILD_UPDATE`, and `THREAD_CREATE/UPDATE/DELETE`
  unmarshal directly into `Channel` / `Guild` aliases for zero-overhead field access.

- **`rest.go`** — 44 new endpoints:
  - **User**: `GetCurrentUser`, `ModifyCurrentUser`, `GetCurrentUserGuilds`, `LeaveGuild`
  - **Guild**: `ModifyGuild`, `SearchGuildMembers`, `GetGuildInvites`, `GetGuildEmojis`,
    `CreateEmoji`, `ModifyEmoji`, `DeleteEmoji`, `GetGuildBansPaginated`, `GetGuildAuditLog`
  - **Roles**: `CreateRole`, `ModifyRole`, `DeleteRole`, `ModifyRolePositions`
  - **Channels**: `CreateChannel`, `DeleteChannel`, `GetChannelMessages`, `GetPinnedMessages`,
    `CreateChannelInvite`, `GetChannelInvites`, `TriggerTypingIndicator`
  - **Invites**: `GetInvite`, `DeleteInvite`
  - **Threads**: `CreateThreadFromMessage`, `CreateThreadWithoutMessage`, `JoinThread`,
    `LeaveThread`, `AddThreadMember`, `RemoveThreadMember`, `GetThreadMembers`, `GetActiveThreads`
  - **Reactions**: `GetReactions`, `DeleteAllReactions`, `DeleteAllReactionsForEmoji`
  - **Voice**: `GetVoiceRegions`
  - **Audit log**: `GetGuildAuditLog`

- **`types.go`** — Expanded `Guild` with: `Banner`, `Splash`, `AFKChannelID`, `AFKTimeout`,
  `VerificationLevel`, `MFALevel`, `ExplicitContentFilter`, `DefaultMessageNotifications`,
  `NSFWLevel`, `PublicUpdatesChannelID`.

- **`types.go`** — Expanded `Channel` with: `Bitrate`, `UserLimit`, `RateLimitPerUser`,
  `LastMessageID`, `DefaultAutoArchiveDuration`, `PermissionOverwrites`, `ThreadMetadata`, `Member`.

- **`types.go`** — Expanded `Message` with: `Pinned`, `MentionEveryone`, `MentionRoles`,
  `Mentions`, `WebhookID`, `Type`, `Flags`, `ReferencedMessage`, `Thread`, `Components`.

- **`types.go`** — New structs: `PermissionOverwrite`, `VoiceState`, `BanEntry`, `Invite`,
  `ThreadMetadata`, `ThreadMember`, `AuditLog`, `AuditLogEntry`, `AuditLogOptions`, `AuditLogChange`.

- **`types.go`** — `ChannelCreateEvent`, `ChannelUpdateEvent`, `ChannelDeleteEvent`, and
  `GuildUpdateEvent` are now type aliases for `Channel` / `Guild` respectively. Handlers
  receive the full object directly with no wrapper struct to unwrap.

- **`types.go`** — `ThreadCreateEvent`, `ThreadUpdateEvent`, `ThreadDeleteEvent` are type
  aliases for `Channel` — consistent with `CHANNEL_*` events since threads are channels.

- **`types.go`** — `GuildRoleCreateEvent.Role` and `GuildRoleUpdateEvent.Role` are now
  `Role` (value) instead of `*Role`, eliminating a double-pointer when taking the address.

### Changed

- **`README.md`** — Updated events table to list all 32 event types with payload types.
  Added `CommandContext` field reference and permission gate examples.
  Updated feature table to reflect 32 events and expanded REST surface.
- **`README.md`** — Expanded the rate-limiting section to name
  `maxRateLimitRetries` explicitly, reworded the opening paragraph to
  call out the zero-dependency WebSocket client and panic-recovered
  dispatcher, added a dedicated "Vendored builds" subsection under
  Installation, and extended the Security section with the
  `maxFramePayload` guard and the `url.PathEscape` reaction-path fix.
- **`rest.go`** — Fixed the stale file-header comment that claimed 429
  responses were "retried once"; the actual behaviour (3 retries,
  bounded by `maxRateLimitRetries`) is now documented in the header.

---

## [1.0.0] — 2026-04-13

### Security

- **`websocket.go`** — Added `maxFramePayload` constant (64 MiB). `readFrame`
  now rejects frames that report a payload length greater than `maxFramePayload`
  before allocating memory. Previously, a malformed or adversarial server frame
  could trigger an out-of-memory crash by claiming an enormous payload length.
  Also guards against the 8-byte length field producing a negative `int64`
  (high-bit overflow), which would have caused an immediate panic in
  `make([]byte, plen)`.

- **`rest.go`** — `AddReaction` and `RemoveReaction` now pass the `emoji`
  parameter through `url.PathEscape` before interpolating it into the request
  path. Previously a crafted emoji string containing `/` or `..` could
  silently target a different Discord API endpoint (path injection).

- **`rest.go`** — Rate-limit retry in `do()` is now bounded by
  `maxRateLimitRetries` (3). Previously the recursive retry had no depth
  limit, so a server that continuously returned 429 could cause unbounded
  recursion and a stack overflow. Callers now receive an `*APIError` with
  status 429 once the budget is exhausted.

- **`rest.go`** — Renamed inner `body` variable in error-parsing block to
  `errBody` to eliminate shadowing of the function parameter `body`. The
  previous shadowing was not exploitable but was a latent correctness hazard
  for future refactoring.

### Added

- **`permissions.go`** — `ParsePermission(s string) (Permission, error)` parses
  the decimal permission string Discord sends for members and roles (e.g.
  `"2147483651"`) into a `Permission` value. Uses `strconv.ParseUint`
  internally, providing a clear error on malformed input. Replaces the fragile
  `fmt.Sscanf` pattern used in the examples.

- **`permissions.go`** — `MustParsePermission(s string) Permission` — panics on
  error; intended for compile-time constant initialisation only.

- **`rest.go`** — `BulkDeleteMessages` now validates that the caller provides
  between 2 and 100 message IDs before making any HTTP request, matching
  Discord's documented constraint and providing a clear error message.

- **`rest.go`** — `GetMessages` clamps `limit` to `[1, 100]`. Previously
  passing `0` or a negative value would have produced a malformed request.

- **`rest.go`** — `BanMember` clamps `deleteMessageDays` to `[0, 7]` (Discord's
  enforced range). Out-of-range values are silently clamped rather than causing
  an API error.

- **`.gitignore`** — Standard Go `.gitignore` covering binaries, test output,
  vendor directory, and common editor/OS artefacts.

- **`.github/workflows/ci.yml`** — GitHub Actions CI pipeline that runs on
  every push to `main` and on pull requests. Matrix-tests against Go 1.21,
  1.22, and 1.23. Steps: `go build`, `go vet`, `go test -race`, and `gofmt`
  format check.

- **`commands_test.go`** — Unit tests for `parseArgs` (15 cases including
  quoted strings, escape sequences, and edge cases) and `buildChain` (no-op
  chain, multi-middleware execution order).

- **`gateway_test.go`** — Unit tests for `backoffDelay`: first-attempt range,
  mean-value growth across attempts, and cap-at-maximum assertion.

- **`permissions_test.go`** — Unit tests for `ParsePermission`,
  `MustParsePermission`, `Has`, `Any`, `Add`, `Remove`, `Toggle`, `IsAdmin`,
  and `String` (including unknown-bit labelling).

- **`errors_test.go`** — Unit tests for `APIError.Error()` formatting (with and
  without Discord code), all five predicate methods, and `errors.As`
  compatibility.

- **`rest_test.go`** — Unit tests for in-process input-validation logic in
  `BulkDeleteMessages`, `GetMessages`, and `BanMember`.

- **`websocket_test.go`** — Unit tests for `maxFramePayload` value, WebSocket
  frame header construction, and the RFC 6455 §1.3 `wsComputeAccept` test
  vector.

### Changed

- **`example/slash/main.go`** — Fixed integer overflow in the `/roll` command:
  `byte(max)` was truncating `max` to 8 bits for values > 255, producing a
  wrong modulus. The result is now computed as `int64(lastDigit) % max + 1`.

- **`example/slash/main.go`** — Replaced deprecated `strings.Title` (removed
  from idiomatic Go since 1.18) with an inline `strings.ToUpper(s[:1]) +
  s[1:]` capitalisation.

- **`example/basic/main.go`** — Replaced `fmt.Sscanf(member.Permissions, "%d",
  (*uint64)(&perms))` with `discord.ParsePermission(member.Permissions)`.

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

### Added (polish pass — now part of 1.0.0)

**New files**
- `logger.go` — `Logger` interface, `WithLogger` functional option, `NoopLogger` for silencing output in tests.
- `permissions.go` — `Permission` bitflag type with all 53 Discord permission constants, composite sets, and utility methods.
- `errors.go` — Typed `APIError` with HTTP status, Discord JSON error code, and convenience predicates. Includes all common Discord JSON error code constants.

**`bot.go`**
- `New()` accepts variadic `Option` values.
- Panics with a clear message if token is empty.
- `Bot.Use(...MiddlewareFunc)` for command middleware chains.
- `Bot.Log()` to access the active logger.

**`commands.go`**
- Quoted-string argument parsing.
- `HandlerFunc` and `MiddlewareFunc` types exported for type-safe middleware.
- `Command.Usage` field.

**`interactions.go`**
- `Button()` and `LinkButton()` component builder helpers.
- Global and guild command REST methods.
- Follow-up message REST methods.

**`rest.go`**
- `AddMemberRole()` / `RemoveMemberRole()`.
- `SendEmbedDM()`, `EditMessageComplex()`.
- Full webhook support.
- REST errors return `*APIError`.

**`gateway.go`**
- Exponential back-off reconnect (1 s → 5 min, ±20 % jitter).
- `sessionMu` protecting session ID and resume URL.
- Zombie connection detection via heartbeat ACK tracking.
- Resume URL cleared on dial failure.
- `InvalidSession` jitter (random 1–5 s per Discord recommendation).

**`events.go`**
- All handler goroutines wrapped in `safeGo()` with panic recovery.
- Unknown event types silently discarded (forward-compatible).
