package discord

// events.go — typed event registration and dispatch.
//
// Each Discord Gateway event has a strongly-typed handler signature so callers
// get compile-time safety rather than interface{} type assertions.
//
// All handlers run in their own goroutines so a slow handler cannot block the
// gateway read-loop. Each goroutine is wrapped in a panic-recovery shim that
// logs the stack trace and continues rather than crashing the process.

import (
	"encoding/json"
	"runtime/debug"
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

	// GuildMemberRemoveHandler is called when a user leaves or is removed from
	// a guild.
	GuildMemberRemoveHandler func(*Bot, *GuildMemberRemoveEvent)

	// GuildMemberUpdateHandler is called when a guild member's state changes
	// (role changes, nickname updates, timeout applied/removed, etc.).
	GuildMemberUpdateHandler func(*Bot, *GuildMemberUpdateEvent)

	// GuildBanAddHandler is called when a user is banned from a guild.
	GuildBanAddHandler func(*Bot, *GuildBanAddEvent)

	// GuildBanRemoveHandler is called when a user is unbanned from a guild.
	GuildBanRemoveHandler func(*Bot, *GuildBanRemoveEvent)

	// ChannelCreateHandler is called when a channel is created.
	ChannelCreateHandler func(*Bot, *ChannelCreateEvent)

	// ChannelUpdateHandler is called when a channel is updated.
	ChannelUpdateHandler func(*Bot, *ChannelUpdateEvent)

	// ChannelDeleteHandler is called when a channel is deleted.
	ChannelDeleteHandler func(*Bot, *ChannelDeleteEvent)

	// GuildUpdateHandler is called when a guild's settings change.
	GuildUpdateHandler func(*Bot, *GuildUpdateEvent)

	// GuildRoleCreateHandler is called when a role is created in a guild.
	GuildRoleCreateHandler func(*Bot, *GuildRoleCreateEvent)

	// GuildRoleUpdateHandler is called when a role is updated in a guild.
	GuildRoleUpdateHandler func(*Bot, *GuildRoleUpdateEvent)

	// GuildRoleDeleteHandler is called when a role is deleted from a guild.
	GuildRoleDeleteHandler func(*Bot, *GuildRoleDeleteEvent)

	// ThreadCreateHandler is called when a thread is created.
	ThreadCreateHandler func(*Bot, *ThreadCreateEvent)

	// ThreadUpdateHandler is called when a thread is updated.
	ThreadUpdateHandler func(*Bot, *ThreadUpdateEvent)

	// ThreadDeleteHandler is called when a thread is deleted.
	ThreadDeleteHandler func(*Bot, *ThreadDeleteEvent)

	// InviteCreateHandler is called when an invite is created.
	InviteCreateHandler func(*Bot, *InviteCreateEvent)

	// InviteDeleteHandler is called when an invite is deleted.
	InviteDeleteHandler func(*Bot, *InviteDeleteEvent)

	// WebhooksUpdateHandler is called when a channel's webhooks change.
	WebhooksUpdateHandler func(*Bot, *WebhooksUpdateEvent)

	// VoiceStateUpdateHandler is called when a user's voice state changes.
	VoiceStateUpdateHandler func(*Bot, *VoiceStateUpdateEvent)

	// TypingStartHandler is called when a user starts typing.
	TypingStartHandler func(*Bot, *TypingStartEvent)

	// MessageDeleteBulkHandler is called when multiple messages are deleted at once.
	MessageDeleteBulkHandler func(*Bot, *MessageDeleteBulkEvent)

	// ReactionRemoveAllHandler is called when all reactions are removed from a message.
	ReactionRemoveAllHandler func(*Bot, *ReactionRemoveAllEvent)

	// ReactionRemoveEmojiHandler is called when all reactions for a specific emoji are removed.
	ReactionRemoveEmojiHandler func(*Bot, *ReactionRemoveEmojiEvent)

	// UserUpdateHandler is called when the current user's properties change.
	UserUpdateHandler func(*Bot, *UserUpdateEvent)

	// IntegrationCreateHandler is called when an integration is created in a guild.
	IntegrationCreateHandler func(*Bot, *IntegrationCreateEvent)

	// IntegrationUpdateHandler is called when an integration is updated in a guild.
	IntegrationUpdateHandler func(*Bot, *IntegrationUpdateEvent)

	// IntegrationDeleteHandler is called when an integration is deleted from a guild.
	IntegrationDeleteHandler func(*Bot, *IntegrationDeleteEvent)
)

// ---------------------------------------------------------------------------
// Dispatcher
// ---------------------------------------------------------------------------

// eventDispatcher holds all registered handlers and dispatches incoming events.
type eventDispatcher struct {
	mu sync.RWMutex

	onReady               []ReadyHandler
	onMessageCreate       []MessageCreateHandler
	onMessageUpdate       []MessageUpdateHandler
	onMessageDelete       []MessageDeleteHandler
	onGuildCreate         []GuildCreateHandler
	onGuildDelete         []GuildDeleteHandler
	onReactionAdd         []ReactionAddHandler
	onReactionRemove      []ReactionRemoveHandler
	onInteractionCreate   []InteractionCreateHandler
	onGuildMemberAdd      []GuildMemberAddHandler
	onGuildMemberRemove   []GuildMemberRemoveHandler
	onGuildMemberUpdate   []GuildMemberUpdateHandler
	onGuildBanAdd         []GuildBanAddHandler
	onGuildBanRemove      []GuildBanRemoveHandler
	onChannelCreate       []ChannelCreateHandler
	onChannelUpdate       []ChannelUpdateHandler
	onChannelDelete       []ChannelDeleteHandler
	onGuildUpdate         []GuildUpdateHandler
	onGuildRoleCreate     []GuildRoleCreateHandler
	onGuildRoleUpdate     []GuildRoleUpdateHandler
	onGuildRoleDelete     []GuildRoleDeleteHandler
	onThreadCreate        []ThreadCreateHandler
	onThreadUpdate        []ThreadUpdateHandler
	onThreadDelete        []ThreadDeleteHandler
	onInviteCreate        []InviteCreateHandler
	onInviteDelete        []InviteDeleteHandler
	onWebhooksUpdate      []WebhooksUpdateHandler
	onVoiceStateUpdate    []VoiceStateUpdateHandler
	onTypingStart         []TypingStartHandler
	onMessageDeleteBulk   []MessageDeleteBulkHandler
	onReactionRemoveAll   []ReactionRemoveAllHandler
	onReactionRemoveEmoji []ReactionRemoveEmojiHandler
	onUserUpdate          []UserUpdateHandler
	onIntegrationCreate   []IntegrationCreateHandler
	onIntegrationUpdate   []IntegrationUpdateHandler
	onIntegrationDelete   []IntegrationDeleteHandler
}

func newEventDispatcher() *eventDispatcher {
	return &eventDispatcher{}
}

// ---------------------------------------------------------------------------
// Panic recovery
// ---------------------------------------------------------------------------

// safeGo launches fn in a new goroutine. If fn panics, the panic is recovered,
// logged via b.log, and the goroutine exits cleanly — preventing a single
// bad handler from crashing the entire bot process.
func safeGo(b *Bot, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				b.log.Printf("[events] handler panic: %v\n%s", r, debug.Stack())
			}
		}()
		fn()
	}()
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

func (d *eventDispatcher) addChannelCreate(h ChannelCreateHandler) {
	d.mu.Lock()
	d.onChannelCreate = append(d.onChannelCreate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addChannelUpdate(h ChannelUpdateHandler) {
	d.mu.Lock()
	d.onChannelUpdate = append(d.onChannelUpdate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addChannelDelete(h ChannelDeleteHandler) {
	d.mu.Lock()
	d.onChannelDelete = append(d.onChannelDelete, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addGuildUpdate(h GuildUpdateHandler) {
	d.mu.Lock()
	d.onGuildUpdate = append(d.onGuildUpdate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addGuildRoleCreate(h GuildRoleCreateHandler) {
	d.mu.Lock()
	d.onGuildRoleCreate = append(d.onGuildRoleCreate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addGuildRoleUpdate(h GuildRoleUpdateHandler) {
	d.mu.Lock()
	d.onGuildRoleUpdate = append(d.onGuildRoleUpdate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addGuildRoleDelete(h GuildRoleDeleteHandler) {
	d.mu.Lock()
	d.onGuildRoleDelete = append(d.onGuildRoleDelete, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addThreadCreate(h ThreadCreateHandler) {
	d.mu.Lock()
	d.onThreadCreate = append(d.onThreadCreate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addThreadUpdate(h ThreadUpdateHandler) {
	d.mu.Lock()
	d.onThreadUpdate = append(d.onThreadUpdate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addThreadDelete(h ThreadDeleteHandler) {
	d.mu.Lock()
	d.onThreadDelete = append(d.onThreadDelete, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addInviteCreate(h InviteCreateHandler) {
	d.mu.Lock()
	d.onInviteCreate = append(d.onInviteCreate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addInviteDelete(h InviteDeleteHandler) {
	d.mu.Lock()
	d.onInviteDelete = append(d.onInviteDelete, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addWebhooksUpdate(h WebhooksUpdateHandler) {
	d.mu.Lock()
	d.onWebhooksUpdate = append(d.onWebhooksUpdate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addVoiceStateUpdate(h VoiceStateUpdateHandler) {
	d.mu.Lock()
	d.onVoiceStateUpdate = append(d.onVoiceStateUpdate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addTypingStart(h TypingStartHandler) {
	d.mu.Lock()
	d.onTypingStart = append(d.onTypingStart, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addMessageDeleteBulk(h MessageDeleteBulkHandler) {
	d.mu.Lock()
	d.onMessageDeleteBulk = append(d.onMessageDeleteBulk, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addReactionRemoveAll(h ReactionRemoveAllHandler) {
	d.mu.Lock()
	d.onReactionRemoveAll = append(d.onReactionRemoveAll, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addReactionRemoveEmoji(h ReactionRemoveEmojiHandler) {
	d.mu.Lock()
	d.onReactionRemoveEmoji = append(d.onReactionRemoveEmoji, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addUserUpdate(h UserUpdateHandler) {
	d.mu.Lock()
	d.onUserUpdate = append(d.onUserUpdate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addIntegrationCreate(h IntegrationCreateHandler) {
	d.mu.Lock()
	d.onIntegrationCreate = append(d.onIntegrationCreate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addIntegrationUpdate(h IntegrationUpdateHandler) {
	d.mu.Lock()
	d.onIntegrationUpdate = append(d.onIntegrationUpdate, h)
	d.mu.Unlock()
}

func (d *eventDispatcher) addIntegrationDelete(h IntegrationDeleteHandler) {
	d.mu.Lock()
	d.onIntegrationDelete = append(d.onIntegrationDelete, h)
	d.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Dispatch — called by the gateway read-loop
// ---------------------------------------------------------------------------

// dispatch unmarshals the raw JSON data and calls all matching handlers.
// Every handler runs inside safeGo so panics are contained and logged rather
// than crashing the process.
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
			b.log.Printf("[events] READY unmarshal error: %v", err)
			return
		}
		for _, h := range d.onReady {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "MESSAGE_CREATE":
		if len(d.onMessageCreate) == 0 && b.commands == nil {
			return
		}
		var e Message
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] MESSAGE_CREATE unmarshal error: %v", err)
			return
		}
		if b.commands != nil {
			safeGo(b, func() { b.commands.handle(b, &e) })
		}
		for _, h := range d.onMessageCreate {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "MESSAGE_UPDATE":
		if len(d.onMessageUpdate) == 0 {
			return
		}
		var e Message
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] MESSAGE_UPDATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onMessageUpdate {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "MESSAGE_DELETE":
		if len(d.onMessageDelete) == 0 {
			return
		}
		var e MessageDeleteEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] MESSAGE_DELETE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onMessageDelete {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "GUILD_CREATE":
		if len(d.onGuildCreate) == 0 {
			return
		}
		var e Guild
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] GUILD_CREATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onGuildCreate {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "GUILD_DELETE":
		if len(d.onGuildDelete) == 0 {
			return
		}
		var e GuildUnavailable
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] GUILD_DELETE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onGuildDelete {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "MESSAGE_REACTION_ADD":
		if len(d.onReactionAdd) == 0 {
			return
		}
		var e MessageReactionAddEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] MESSAGE_REACTION_ADD unmarshal error: %v", err)
			return
		}
		for _, h := range d.onReactionAdd {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "MESSAGE_REACTION_REMOVE":
		if len(d.onReactionRemove) == 0 {
			return
		}
		var e MessageReactionRemoveEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] MESSAGE_REACTION_REMOVE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onReactionRemove {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "INTERACTION_CREATE":
		if len(d.onInteractionCreate) == 0 {
			return
		}
		var e Interaction
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] INTERACTION_CREATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onInteractionCreate {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "GUILD_MEMBER_ADD":
		if len(d.onGuildMemberAdd) == 0 {
			return
		}
		var e GuildMemberAddEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] GUILD_MEMBER_ADD unmarshal error: %v", err)
			return
		}
		for _, h := range d.onGuildMemberAdd {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "GUILD_MEMBER_REMOVE":
		if len(d.onGuildMemberRemove) == 0 {
			return
		}
		var e GuildMemberRemoveEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] GUILD_MEMBER_REMOVE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onGuildMemberRemove {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "GUILD_MEMBER_UPDATE":
		if len(d.onGuildMemberUpdate) == 0 {
			return
		}
		var e GuildMemberUpdateEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] GUILD_MEMBER_UPDATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onGuildMemberUpdate {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "GUILD_BAN_ADD":
		if len(d.onGuildBanAdd) == 0 {
			return
		}
		var e GuildBanAddEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] GUILD_BAN_ADD unmarshal error: %v", err)
			return
		}
		for _, h := range d.onGuildBanAdd {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "GUILD_BAN_REMOVE":
		if len(d.onGuildBanRemove) == 0 {
			return
		}
		var e GuildBanRemoveEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] GUILD_BAN_REMOVE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onGuildBanRemove {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "CHANNEL_CREATE":
		if len(d.onChannelCreate) == 0 {
			return
		}
		var ch ChannelCreateEvent
		if err := json.Unmarshal(data, &ch); err != nil {
			b.log.Printf("[events] CHANNEL_CREATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onChannelCreate {
			h := h
			safeGo(b, func() { h(b, &ch) })
		}

	case "CHANNEL_UPDATE":
		if len(d.onChannelUpdate) == 0 {
			return
		}
		var ch ChannelUpdateEvent
		if err := json.Unmarshal(data, &ch); err != nil {
			b.log.Printf("[events] CHANNEL_UPDATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onChannelUpdate {
			h := h
			safeGo(b, func() { h(b, &ch) })
		}

	case "CHANNEL_DELETE":
		if len(d.onChannelDelete) == 0 {
			return
		}
		var ch ChannelDeleteEvent
		if err := json.Unmarshal(data, &ch); err != nil {
			b.log.Printf("[events] CHANNEL_DELETE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onChannelDelete {
			h := h
			safeGo(b, func() { h(b, &ch) })
		}

	case "GUILD_UPDATE":
		if len(d.onGuildUpdate) == 0 {
			return
		}
		var g GuildUpdateEvent
		if err := json.Unmarshal(data, &g); err != nil {
			b.log.Printf("[events] GUILD_UPDATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onGuildUpdate {
			h := h
			safeGo(b, func() { h(b, &g) })
		}

	case "GUILD_ROLE_CREATE":
		if len(d.onGuildRoleCreate) == 0 {
			return
		}
		var e GuildRoleCreateEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] GUILD_ROLE_CREATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onGuildRoleCreate {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "GUILD_ROLE_UPDATE":
		if len(d.onGuildRoleUpdate) == 0 {
			return
		}
		var e GuildRoleUpdateEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] GUILD_ROLE_UPDATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onGuildRoleUpdate {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "GUILD_ROLE_DELETE":
		if len(d.onGuildRoleDelete) == 0 {
			return
		}
		var e GuildRoleDeleteEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] GUILD_ROLE_DELETE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onGuildRoleDelete {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "THREAD_CREATE":
		if len(d.onThreadCreate) == 0 {
			return
		}
		var ch ThreadCreateEvent
		if err := json.Unmarshal(data, &ch); err != nil {
			b.log.Printf("[events] THREAD_CREATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onThreadCreate {
			h := h
			safeGo(b, func() { h(b, &ch) })
		}

	case "THREAD_UPDATE":
		if len(d.onThreadUpdate) == 0 {
			return
		}
		var ch ThreadUpdateEvent
		if err := json.Unmarshal(data, &ch); err != nil {
			b.log.Printf("[events] THREAD_UPDATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onThreadUpdate {
			h := h
			safeGo(b, func() { h(b, &ch) })
		}

	case "THREAD_DELETE":
		if len(d.onThreadDelete) == 0 {
			return
		}
		var e ThreadDeleteEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] THREAD_DELETE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onThreadDelete {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "INVITE_CREATE":
		if len(d.onInviteCreate) == 0 {
			return
		}
		var e InviteCreateEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] INVITE_CREATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onInviteCreate {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "INVITE_DELETE":
		if len(d.onInviteDelete) == 0 {
			return
		}
		var e InviteDeleteEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] INVITE_DELETE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onInviteDelete {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "WEBHOOKS_UPDATE":
		if len(d.onWebhooksUpdate) == 0 {
			return
		}
		var e WebhooksUpdateEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] WEBHOOKS_UPDATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onWebhooksUpdate {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "VOICE_STATE_UPDATE":
		if len(d.onVoiceStateUpdate) == 0 {
			return
		}
		var vs VoiceState
		if err := json.Unmarshal(data, &vs); err != nil {
			b.log.Printf("[events] VOICE_STATE_UPDATE unmarshal error: %v", err)
			return
		}
		e := &VoiceStateUpdateEvent{VoiceState: &vs}
		for _, h := range d.onVoiceStateUpdate {
			h := h
			safeGo(b, func() { h(b, e) })
		}

	case "TYPING_START":
		if len(d.onTypingStart) == 0 {
			return
		}
		var e TypingStartEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] TYPING_START unmarshal error: %v", err)
			return
		}
		for _, h := range d.onTypingStart {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "MESSAGE_DELETE_BULK":
		if len(d.onMessageDeleteBulk) == 0 {
			return
		}
		var e MessageDeleteBulkEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] MESSAGE_DELETE_BULK unmarshal error: %v", err)
			return
		}
		for _, h := range d.onMessageDeleteBulk {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "MESSAGE_REACTION_REMOVE_ALL":
		if len(d.onReactionRemoveAll) == 0 {
			return
		}
		var e ReactionRemoveAllEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] MESSAGE_REACTION_REMOVE_ALL unmarshal error: %v", err)
			return
		}
		for _, h := range d.onReactionRemoveAll {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "MESSAGE_REACTION_REMOVE_EMOJI":
		if len(d.onReactionRemoveEmoji) == 0 {
			return
		}
		var e ReactionRemoveEmojiEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] MESSAGE_REACTION_REMOVE_EMOJI unmarshal error: %v", err)
			return
		}
		for _, h := range d.onReactionRemoveEmoji {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "USER_UPDATE":
		if len(d.onUserUpdate) == 0 {
			return
		}
		var e UserUpdateEvent
		if err := json.Unmarshal(data, &e.User); err != nil {
			b.log.Printf("[events] USER_UPDATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onUserUpdate {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "INTEGRATION_CREATE":
		if len(d.onIntegrationCreate) == 0 {
			return
		}
		var e IntegrationCreateEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] INTEGRATION_CREATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onIntegrationCreate {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "INTEGRATION_UPDATE":
		if len(d.onIntegrationUpdate) == 0 {
			return
		}
		var e IntegrationUpdateEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] INTEGRATION_UPDATE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onIntegrationUpdate {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	case "INTEGRATION_DELETE":
		if len(d.onIntegrationDelete) == 0 {
			return
		}
		var e IntegrationDeleteEvent
		if err := json.Unmarshal(data, &e); err != nil {
			b.log.Printf("[events] INTEGRATION_DELETE unmarshal error: %v", err)
			return
		}
		for _, h := range d.onIntegrationDelete {
			h := h
			safeGo(b, func() { h(b, &e) })
		}

	default:
		// Unknown events are silently ignored. This keeps the bot forward-compatible
		// as Discord adds new event types without requiring framework updates.
	}
}
