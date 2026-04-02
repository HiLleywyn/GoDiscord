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
)

// ---------------------------------------------------------------------------
// Dispatcher
// ---------------------------------------------------------------------------

// eventDispatcher holds all registered handlers and dispatches incoming events.
type eventDispatcher struct {
	mu sync.RWMutex

	onReady          []ReadyHandler
	onMessageCreate  []MessageCreateHandler
	onMessageUpdate  []MessageUpdateHandler
	onMessageDelete  []MessageDeleteHandler
	onGuildCreate    []GuildCreateHandler
	onGuildDelete    []GuildDeleteHandler
	onReactionAdd    []ReactionAddHandler
	onReactionRemove []ReactionRemoveHandler
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
	}
}
