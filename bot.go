// Package discord is a custom Discord bot framework written entirely in Go
// with zero external dependencies.
//
// Quick start:
//
//	bot := discord.New("Bot TOKEN", discord.IntentsDefault)
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
package discord

import (
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

	gateway  *gateway
	events   *eventDispatcher
	commands *commandHandler

	self *User
	mu   sync.RWMutex
}

// New creates a new Bot with the given token and gateway intents.
//
// The token should NOT include the "Bot " prefix — the framework adds it.
func New(token string, intents Intents) *Bot {
	b := &Bot{
		token:   token,
		intents: intents,
		events:  newEventDispatcher(),
	}
	b.Rest = newRestClient(token)
	b.gateway = newGateway(b)
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

// OnGuildMemberRemove registers a handler called when a user leaves or is removed from a guild.
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

// ---------------------------------------------------------------------------
// Command framework
// ---------------------------------------------------------------------------

// SetPrefix enables the built-in command handler with the given prefix string
// (e.g. "!" or ">>"). Must be called before Run().
func (b *Bot) SetPrefix(prefix string) *Bot {
	b.mu.Lock()
	b.commands = newCommandHandler(prefix)
	b.mu.Unlock()
	return b
}

// AddCommand registers a command. SetPrefix must be called first.
func (b *Bot) AddCommand(cmd *Command) *Bot {
	b.mu.RLock()
	ch := b.commands
	b.mu.RUnlock()

	if ch == nil {
		// Auto-create handler with empty prefix so callers don't have to
		// call SetPrefix if they just want AddCommand.
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

// Commands returns a slice of all registered commands.
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
// Presence
// ---------------------------------------------------------------------------

// SetActivity sets the bot's activity (e.g. "Playing Chess").
// activityType is one of the Activity* constants (ActivityPlaying, etc.).
// Must be called before Run() to take effect on startup; can also be called
// after Run() to update live.
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

	// If already connected, push the update immediately.
	if b.gateway != nil && b.gateway.conn != nil {
		_ = b.gateway.updatePresence(p)
	}
	return b
}

// SetStatus sets the bot's online status ("online", "idle", "dnd", "invisible").
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

// Self returns the bot's own User object, available after the Ready event.
// Returns nil if called before the bot has connected.
func (b *Bot) Self() *User {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.self
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Run connects to the Discord Gateway and blocks until Stop() is called or
// an unrecoverable error occurs.
func (b *Bot) Run() error {
	if err := b.gateway.start(); err != nil {
		return err
	}
	// Block until the gateway shuts down.
	<-b.gateway.doneCh
	return nil
}

// Stop gracefully disconnects from the Discord Gateway.
func (b *Bot) Stop() {
	b.gateway.stop()
}
