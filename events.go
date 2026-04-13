package discord

// events.go — typed event registration and dispatch.
//
// Each Discord Gateway event has a strongly-typed handler signature so callers
// get compile-time safety rather than interface{} type assertions.

import (
	"encoding/json"
	"sync"
)

// ---------------------------------------------------------------------------
// Handler types — one per supported event
// ---------------------------------------------------------------------------

type (
	// ReadyHandler is called when the bot is connected and identified.
	ReadyHandler func(*Bot, *ReadyEvent)

	// MessageCreateHandler is called when a message is created in any channel
	// the bot can see (subject to Intents).
	MessageCreateHandler func(*Bot, *Message)

	// MessageUpdateHandler is called when a message is edited.
	MessageUpdateHandler func(*Bot, *Message)

	// MessageDeleteHandler is called when a message is deleted.
	MessageDeleteHandler func(*Bot, *MessageDeleteEvent)

	// GuildCreateHandler is called when the bot joins a guild or when a guild
	// becomes available on startup.
	GuildCreateHandler func(*Bot, *Guild)

	// GuildDeleteHandler is called when the bot is removed from a guild or the
	// guild becomes unavailable.
	GuildDeleteHandler func(*Bot, *GuildUnavailable)

	// ReactionAddHandler is called when a user adds a reaction to a message.
	ReactionAddHandler func(*Bot, *MessageReactionAddEvent)

	// ReactionRemoveHandler is called when a reaction is removed from a message.
	ReactionRemoveHandler func(*Bot, *MessageReactionRemoveEvent)

	// InteractionCreateHandler is called for every INTERACTION_CREATE event
	// (slash commands, button clicks, select menus, etc.).
	InteractionCreateHandler func(*Bot, *Interaction)

	// GuildMemberAddHandler is called when a user joins a guild.
	GuildMemberAddHandler func(*Bot, *GuildMemberAddEvent)

	// GuildMemberRemoveHandler is called when a user leaves or is removed from a guild.
	GuildMemberRemoveHandler func(*Bot, *GuildMemberRemoveEvent)

	// GuildMemberUpdateHandler is called when a guild member's state changes.
	GuildMemberUpdateHandler func(*Bot, *GuildMemberUpdateEvent)

	// GuildBanAddHandler is called when a user is banned from a guild.
	GuildBanAddHandler func(*Bot, *GuildBanAddEvent)

	// GuildBanRemoveHandler is called when a user is unbanned from a guild.
	GuildBanRemoveHandler func(*Bot, *GuildBanRemoveEvent)
)

// ---------------------------------------------------------------------------
// Dispatcher
// ---------------------------------------------------------------------------

// eventDispatcher holds all registered handlers and dispatches incoming events.
type eventDispatcher struct {
	mu sync.RWMutex

	onReady             []ReadyHandler
	onMessageCreate     []MessageCreateHandler
	onMessageUpdate     []MessageUpdateHandler
	onMessageDelete     []MessageDeleteHandler
	onGuildCreate       []GuildCreateHandler
	onGuildDelete       []GuildDeleteHandler
	onReactionAdd       []ReactionAddHandler
	onReactionRemove    []ReactionRemoveHandler
	onInteractionCreate []InteractionCreateHandler
	onGuildMemberAdd    []GuildMemberAddHandler
	onGuildMemberRemove []GuildMemberRemoveHandler
	onGuildMemberUpdate []GuildMemberUpdateHandler
	onGuildBanAdd       []GuildBanAddHandler
	onGuildBanRemove    []GuildBanRemoveHandler
}

func newEventDispatcher() *eventDispatcher {
	return &eventDispatcher{}
}

// ---------------------------------------------------------------------------
// Registration helpers (called by Bot.On* methods)
// ---------------------------------------------------------------------------

func (d *eventDispatcher) addReady(h ReadyHandler) {
	d.mu.Lock()
	d.onReady = append(d.onReady, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addMessageCreate(h MessageCreateHandler) {
	d.mu.Lock()
	d.onMessageCreate = append(d.onMessageCreate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addMessageUpdate(h MessageUpdateHandler) {
	d.mu.Lock()
	d.onMessageUpdate = append(d.onMessageUpdate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addMessageDelete(h MessageDeleteHandler) {
	d.mu.Lock()
	d.onMessageDelete = append(d.onMessageDelete, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addGuildCreate(h GuildCreateHandler) {
	d.mu.Lock()
	d.onGuildCreate = append(d.onGuildCreate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addGuildDelete(h GuildDeleteHandler) {
	d.mu.Lock()
	d.onGuildDelete = append(d.onGuildDelete, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addReactionAdd(h ReactionAddHandler) {
	d.mu.Lock()
	d.onReactionAdd = append(d.onReactionAdd, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addReactionRemove(h ReactionRemoveHandler) {
	d.mu.Lock()
	d.onReactionRemove = append(d.onReactionRemove, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addInteractionCreate(h InteractionCreateHandler) {
	d.mu.Lock()
	d.onInteractionCreate = append(d.onInteractionCreate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addGuildMemberAdd(h GuildMemberAddHandler) {
	d.mu.Lock()
	d.onGuildMemberAdd = append(d.onGuildMemberAdd, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addGuildMemberRemove(h GuildMemberRemoveHandler) {
	d.mu.Lock()
	d.onGuildMemberRemove = append(d.onGuildMemberRemove, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addGuildMemberUpdate(h GuildMemberUpdateHandler) {
	d.mu.Lock()
	d.onGuildMemberUpdate = append(d.onGuildMemberUpdate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addGuildBanAdd(h GuildBanAddHandler) {
	d.mu.Lock()
	d.onGuildBanAdd = append(d.onGuildBanAdd, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addGuildBanRemove(h GuildBanRemoveHandler) {
	d.mu.Lock()
	d.onGuildBanRemove = append(d.onGuildBanRemove, h)
	d.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Dispatch — called by the gateway read-loop
// ---------------------------------------------------------------------------

// dispatch unmarshals the raw JSON data and calls all matching handlers.
// Handlers run in their own goroutines so a slow handler cannot block the
// gateway read-loop.
func (d *eventDispatcher) dispatch(b *Bot, eventType string, data json.RawMessage) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	switch eventType {
	case "READY":
		if len(d.onReady) == 0 {
			return
		}
		var e ReadyEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		for _, h := range d.onReady {
			h := h
			go h(b, &e)
		}

	case "MESSAGE_CREATE":
		if len(d.onMessageCreate) == 0 && b.commands == nil {
			return
		}
		var e Message
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		// Let the command handler run first (synchronously, still fast).
		if b.commands != nil {
			go b.commands.handle(b, &e)
		}
		for _, h := range d.onMessageCreate {
			h := h
			go h(b, &e)
		}

	case "MESSAGE_UPDATE":
		if len(d.onMessageUpdate) == 0 {
			return
		}
		var e Message
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		for _, h := range d.onMessageUpdate {
			h := h
			go h(b, &e)
		}

	case "MESSAGE_DELETE":
		if len(d.onMessageDelete) == 0 {
			return
		}
		var e MessageDeleteEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		for _, h := range d.onMessageDelete {
			h := h
			go h(b, &e)
		}

	case "GUILD_CREATE":
		if len(d.onGuildCreate) == 0 {
			return
		}
		var e Guild
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		for _, h := range d.onGuildCreate {
			h := h
			go h(b, &e)
		}

	case "GUILD_DELETE":
		if len(d.onGuildDelete) == 0 {
			return
		}
		var e GuildUnavailable
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		for _, h := range d.onGuildDelete {
			h := h
			go h(b, &e)
		}

	case "MESSAGE_REACTION_ADD":
		if len(d.onReactionAdd) == 0 {
			return
		}
		var e MessageReactionAddEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		for _, h := range d.onReactionAdd {
			h := h
			go h(b, &e)
		}

	case "MESSAGE_REACTION_REMOVE":
		if len(d.onReactionRemove) == 0 {
			return
		}
		var e MessageReactionRemoveEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		for _, h := range d.onReactionRemove {
			h := h
			go h(b, &e)
		}

	case "INTERACTION_CREATE":
		if len(d.onInteractionCreate) == 0 {
			return
		}
		var e Interaction
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		for _, h := range d.onInteractionCreate {
			h := h
			go h(b, &e)
		}

	case "GUILD_MEMBER_ADD":
		if len(d.onGuildMemberAdd) == 0 {
			return
		}
		var e GuildMemberAddEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		for _, h := range d.onGuildMemberAdd {
			h := h
			go h(b, &e)
		}

	case "GUILD_MEMBER_REMOVE":
		if len(d.onGuildMemberRemove) == 0 {
			return
		}
		var e GuildMemberRemoveEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		for _, h := range d.onGuildMemberRemove {
			h := h
			go h(b, &e)
		}

	case "GUILD_MEMBER_UPDATE":
		if len(d.onGuildMemberUpdate) == 0 {
			return
		}
		var e GuildMemberUpdateEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		for _, h := range d.onGuildMemberUpdate {
			h := h
			go h(b, &e)
		}

	case "GUILD_BAN_ADD":
		if len(d.onGuildBanAdd) == 0 {
			return
		}
		var e GuildBanAddEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		for _, h := range d.onGuildBanAdd {
			h := h
			go h(b, &e)
		}

	case "GUILD_BAN_REMOVE":
		if len(d.onGuildBanRemove) == 0 {
			return
		}
		var e GuildBanRemoveEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return
		}
		for _, h := range d.onGuildBanRemove {
			h := h
			go h(b, &e)
		}
	}
}
