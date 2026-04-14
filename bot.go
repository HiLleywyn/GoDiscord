// Package discord is a zero-dependency Discord bot framework written in pure Go.
//
// It implements Discord Gateway v10 and provides a typed event system,
// prefix-based command framework, slash command / interaction support, and a
// complete REST client — all without any external dependencies.
//
// # Quick Start
//
//	bot := discord.New("YOUR_TOKEN", discord.IntentGuilds|discord.IntentGuildMessages)
//
//	bot.OnReady(func(b *discord.Bot, e *discord.ReadyEvent) {
//	    fmt.Println("Logged in as", e.User.Tag())
//	})
//
//	bot.OnMessageCreate(func(b *discord.Bot, m *discord.Message) {
//	    if m.Content == "!ping" {
//	        b.Rest.SendMessage(m.ChannelID, "Pong!")
//	    }
//	})
//
//	log.Fatal(bot.Run())
//
// # Functional Options
//
// Pass Option values to New() to customise the bot at construction time:
//
//	bot := discord.New(token, intents,
//	    discord.WithLogger(myZapLogger),
//	)
package discord

import (
	"strings"
	"sync"
)

// Bot is the central object of the framework. Create one with New().
type Bot struct {
	// Rest exposes the Discord REST API. Use it to send messages, manage
	// channels, ban members, etc.
	Rest *RestClient

	// --- private ---
	token           string
	intents         Intents
	initialPresence *presence
	log             Logger

	gateway  *gateway
	events   *eventDispatcher
	commands *commandHandler

	self *User
	mu   sync.RWMutex
}

// New creates a new Bot with the given token and gateway intents.
//
// The token should NOT include the "Bot " prefix — the framework adds it
// automatically wherever required.
//
// Pass zero or more Option values to customise the bot:
//
//	bot := discord.New(token, intents, discord.WithLogger(myLogger))
//
// New panics if token is empty.
func New(token string, intents Intents, opts ...Option) *Bot {
	token = strings.TrimSpace(token)
	if token == "" {
		panic("discord: token must not be empty")
	}

	b := &Bot{
		token:   token,
		intents: intents,
		events:  newEventDispatcher(),
		log:     defaultLogger,
	}
	b.Rest = newRestClient(token, b)
	b.gateway = newGateway(b)

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// ---------------------------------------------------------------------------
// Event registration — returns *Bot for method chaining
// ---------------------------------------------------------------------------

// OnReady registers a handler called once after a successful Identify.
func (b *Bot) OnReady(h ReadyHandler) *Bot {
	b.events.addReady(h)
	return b
}

// OnMessageCreate registers a handler for new messages.
func (b *Bot) OnMessageCreate(h MessageCreateHandler) *Bot {
	b.events.addMessageCreate(h)
	return b
}

// OnMessageUpdate registers a handler for edited messages.
func (b *Bot) OnMessageUpdate(h MessageUpdateHandler) *Bot {
	b.events.addMessageUpdate(h)
	return b
}

// OnMessageDelete registers a handler for deleted messages.
func (b *Bot) OnMessageDelete(h MessageDeleteHandler) *Bot {
	b.events.addMessageDelete(h)
	return b
}

// OnGuildCreate registers a handler called when the bot joins a guild or a
// guild becomes available on startup.
func (b *Bot) OnGuildCreate(h GuildCreateHandler) *Bot {
	b.events.addGuildCreate(h)
	return b
}

// OnGuildDelete registers a handler called when the bot is removed from a
// guild or the guild goes unavailable.
func (b *Bot) OnGuildDelete(h GuildDeleteHandler) *Bot {
	b.events.addGuildDelete(h)
	return b
}

// OnReactionAdd registers a handler for reaction-add events.
func (b *Bot) OnReactionAdd(h ReactionAddHandler) *Bot {
	b.events.addReactionAdd(h)
	return b
}

// OnReactionRemove registers a handler for reaction-remove events.
func (b *Bot) OnReactionRemove(h ReactionRemoveHandler) *Bot {
	b.events.addReactionRemove(h)
	return b
}

// OnInteractionCreate registers a handler for all interaction events
// (slash commands, button clicks, select menu choices, etc.).
func (b *Bot) OnInteractionCreate(h InteractionCreateHandler) *Bot {
	b.events.addInteractionCreate(h)
	return b
}

// OnGuildMemberAdd registers a handler called when a user joins a guild.
func (b *Bot) OnGuildMemberAdd(h GuildMemberAddHandler) *Bot {
	b.events.addGuildMemberAdd(h)
	return b
}

// OnGuildMemberRemove registers a handler called when a user leaves or is
// removed from a guild.
func (b *Bot) OnGuildMemberRemove(h GuildMemberRemoveHandler) *Bot {
	b.events.addGuildMemberRemove(h)
	return b
}

// OnGuildMemberUpdate registers a handler for guild member state changes
// (role changes, nickname updates, timeout applied/removed, etc.).
func (b *Bot) OnGuildMemberUpdate(h GuildMemberUpdateHandler) *Bot {
	b.events.addGuildMemberUpdate(h)
	return b
}

// OnGuildBanAdd registers a handler called when a user is banned from a guild.
func (b *Bot) OnGuildBanAdd(h GuildBanAddHandler) *Bot {
	b.events.addGuildBanAdd(h)
	return b
}

// OnGuildBanRemove registers a handler called when a ban is lifted.
func (b *Bot) OnGuildBanRemove(h GuildBanRemoveHandler) *Bot {
	b.events.addGuildBanRemove(h)
	return b
}

// OnChannelCreate registers a handler called when a channel is created.
func (b *Bot) OnChannelCreate(h ChannelCreateHandler) *Bot {
	b.events.addChannelCreate(h)
	return b
}

// OnChannelUpdate registers a handler called when a channel is updated.
func (b *Bot) OnChannelUpdate(h ChannelUpdateHandler) *Bot {
	b.events.addChannelUpdate(h)
	return b
}

// OnChannelDelete registers a handler called when a channel is deleted.
func (b *Bot) OnChannelDelete(h ChannelDeleteHandler) *Bot {
	b.events.addChannelDelete(h)
	return b
}

// OnGuildUpdate registers a handler called when a guild's settings change.
func (b *Bot) OnGuildUpdate(h GuildUpdateHandler) *Bot {
	b.events.addGuildUpdate(h)
	return b
}

// OnGuildRoleCreate registers a handler called when a role is created in a guild.
func (b *Bot) OnGuildRoleCreate(h GuildRoleCreateHandler) *Bot {
	b.events.addGuildRoleCreate(h)
	return b
}

// OnGuildRoleUpdate registers a handler called when a role is updated in a guild.
func (b *Bot) OnGuildRoleUpdate(h GuildRoleUpdateHandler) *Bot {
	b.events.addGuildRoleUpdate(h)
	return b
}

// OnGuildRoleDelete registers a handler called when a role is deleted from a guild.
func (b *Bot) OnGuildRoleDelete(h GuildRoleDeleteHandler) *Bot {
	b.events.addGuildRoleDelete(h)
	return b
}

// OnThreadCreate registers a handler called when a thread is created.
func (b *Bot) OnThreadCreate(h ThreadCreateHandler) *Bot {
	b.events.addThreadCreate(h)
	return b
}

// OnThreadUpdate registers a handler called when a thread is updated.
func (b *Bot) OnThreadUpdate(h ThreadUpdateHandler) *Bot {
	b.events.addThreadUpdate(h)
	return b
}

// OnThreadDelete registers a handler called when a thread is deleted.
func (b *Bot) OnThreadDelete(h ThreadDeleteHandler) *Bot {
	b.events.addThreadDelete(h)
	return b
}

// OnInviteCreate registers a handler called when an invite is created.
func (b *Bot) OnInviteCreate(h InviteCreateHandler) *Bot {
	b.events.addInviteCreate(h)
	return b
}

// OnInviteDelete registers a handler called when an invite is deleted.
func (b *Bot) OnInviteDelete(h InviteDeleteHandler) *Bot {
	b.events.addInviteDelete(h)
	return b
}

// OnWebhooksUpdate registers a handler called when a channel's webhooks change.
func (b *Bot) OnWebhooksUpdate(h WebhooksUpdateHandler) *Bot {
	b.events.addWebhooksUpdate(h)
	return b
}

// OnVoiceStateUpdate registers a handler called when a user's voice state changes.
func (b *Bot) OnVoiceStateUpdate(h VoiceStateUpdateHandler) *Bot {
	b.events.addVoiceStateUpdate(h)
	return b
}

// OnTypingStart registers a handler called when a user starts typing.
func (b *Bot) OnTypingStart(h TypingStartHandler) *Bot {
	b.events.addTypingStart(h)
	return b
}

// OnMessageDeleteBulk registers a handler called when multiple messages are deleted at once.
func (b *Bot) OnMessageDeleteBulk(h MessageDeleteBulkHandler) *Bot {
	b.events.addMessageDeleteBulk(h)
	return b
}

// OnReactionRemoveAll registers a handler called when all reactions are removed from a message.
func (b *Bot) OnReactionRemoveAll(h ReactionRemoveAllHandler) *Bot {
	b.events.addReactionRemoveAll(h)
	return b
}

// OnReactionRemoveEmoji registers a handler called when all reactions for a specific emoji are removed.
func (b *Bot) OnReactionRemoveEmoji(h ReactionRemoveEmojiHandler) *Bot {
	b.events.addReactionRemoveEmoji(h)
	return b
}

// OnUserUpdate registers a handler called when the current user's properties change.
func (b *Bot) OnUserUpdate(h UserUpdateHandler) *Bot {
	b.events.addUserUpdate(h)
	return b
}

// OnIntegrationCreate registers a handler called when an integration is created in a guild.
func (b *Bot) OnIntegrationCreate(h IntegrationCreateHandler) *Bot {
	b.events.addIntegrationCreate(h)
	return b
}

// OnIntegrationUpdate registers a handler called when an integration is updated in a guild.
func (b *Bot) OnIntegrationUpdate(h IntegrationUpdateHandler) *Bot {
	b.events.addIntegrationUpdate(h)
	return b
}

// OnIntegrationDelete registers a handler called when an integration is deleted from a guild.
func (b *Bot) OnIntegrationDelete(h IntegrationDeleteHandler) *Bot {
	b.events.addIntegrationDelete(h)
	return b
}

// ---------------------------------------------------------------------------
// Command framework
// ---------------------------------------------------------------------------

// SetPrefix enables the built-in prefix command handler with the given prefix
// string (e.g. "!" or ">>"). Must be called before Run().
func (b *Bot) SetPrefix(prefix string) *Bot {
	b.mu.Lock()
	b.commands = newCommandHandler(prefix)
	b.mu.Unlock()
	return b
}

// AddCommand registers a command. If SetPrefix was never called the prefix
// defaults to "!".
func (b *Bot) AddCommand(cmd *Command) *Bot {
	b.mu.RLock()
	ch := b.commands
	b.mu.RUnlock()

	if ch == nil {
		b.mu.Lock()
		if b.commands == nil {
			b.commands = newCommandHandler("!")
		}
		ch = b.commands
		b.mu.Unlock()
	}
	ch.register(cmd)
	return b
}

// RemoveCommand deregisters a command and all its aliases by name.
func (b *Bot) RemoveCommand(name string) *Bot {
	b.mu.RLock()
	ch := b.commands
	b.mu.RUnlock()
	if ch != nil {
		ch.unregister(name)
	}
	return b
}

// Commands returns a slice of all registered commands (no alias duplicates).
func (b *Bot) Commands() []*Command {
	b.mu.RLock()
	ch := b.commands
	b.mu.RUnlock()
	if ch == nil {
		return nil
	}
	return ch.list()
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

// SetCommandDenied registers a callback invoked when a command's
// RequiredPermissions or PermCheck gate blocks an invocation. fn receives
// the failing CommandContext and a short reason string. Replaces any prior
// callback. Only useful after SetPrefix or AddCommand has been called.
func (b *Bot) SetCommandDenied(fn func(*CommandContext, string)) *Bot {
	b.mu.RLock()
	ch := b.commands
	b.mu.RUnlock()

	if ch == nil {
		b.mu.Lock()
		if b.commands == nil {
			b.commands = newCommandHandler("!")
		}
		ch = b.commands
		b.mu.Unlock()
	}
	ch.mu.Lock()
	ch.onDenied = fn
	ch.mu.Unlock()
	return b
}

// Use registers one or more middleware functions that wrap every command
// handler. Middleware is applied in registration order, so the first Use()
// call wraps the outermost layer.
//
//	bot.Use(func(next discord.HandlerFunc) discord.HandlerFunc {
//	    return func(ctx *discord.CommandContext) {
//	        log.Printf("[cmd] %s by %s", ctx.Command.Name, ctx.Message.Author.Username)
//	        next(ctx)
//	    }
//	})
func (b *Bot) Use(mw ...MiddlewareFunc) *Bot {
	b.mu.RLock()
	ch := b.commands
	b.mu.RUnlock()

	if ch == nil {
		b.mu.Lock()
		if b.commands == nil {
			b.commands = newCommandHandler("!")
		}
		ch = b.commands
		b.mu.Unlock()
	}
	ch.use(mw...)
	return b
}

// ---------------------------------------------------------------------------
// Presence
// ---------------------------------------------------------------------------

// SetActivity sets the bot's displayed activity (e.g. "Watching over the server").
// activityType is one of the Activity* constants.
// Call before Run() to set the startup presence; or after Run() to update live.
func (b *Bot) SetActivity(name string, activityType int) *Bot {
	p := presence{
		Status: "online",
		Activities: []activity{
			{Name: name, Type: activityType},
		},
	}
	b.mu.Lock()
	b.initialPresence = &p
	b.mu.Unlock()

	if b.gateway != nil && b.gateway.conn != nil {
		_ = b.gateway.updatePresence(p)
	}
	return b
}

// SetStatus sets the bot's online status: "online", "idle", "dnd", or "invisible".
func (b *Bot) SetStatus(status string) *Bot {
	b.mu.Lock()
	if b.initialPresence == nil {
		b.initialPresence = &presence{Status: status}
	} else {
		b.initialPresence.Status = status
	}
	p := *b.initialPresence
	b.mu.Unlock()

	if b.gateway != nil && b.gateway.conn != nil {
		_ = b.gateway.updatePresence(p)
	}
	return b
}

// ---------------------------------------------------------------------------
// Self
// ---------------------------------------------------------------------------

// Self returns the bot's own User object, populated after the Ready event.
// Returns nil if called before the bot has connected.
func (b *Bot) Self() *User {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.self
}

// ---------------------------------------------------------------------------
// Logging
// ---------------------------------------------------------------------------

// Log returns the active logger. Always non-nil.
func (b *Bot) Log() Logger {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.log
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Run connects to the Discord Gateway and blocks until Stop() is called or an
// unrecoverable error occurs. It is equivalent to calling gateway.start() and
// then waiting on the done channel.
func (b *Bot) Run() error {
	if err := b.gateway.start(); err != nil {
		return err
	}
	<-b.gateway.doneCh
	return nil
}

// Stop gracefully disconnects from the Discord Gateway. Run() returns after
// Stop() completes.
func (b *Bot) Stop() {
	b.gateway.stop()
}
