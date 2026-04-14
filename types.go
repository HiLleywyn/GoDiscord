package discord

import "encoding/json"

// Snowflake is a Discord unique identifier (uint64 sent as a string over JSON).
type Snowflake = string

// Intents controls which Gateway events the bot receives.
type Intents uint32

const (
	IntentGuilds                 Intents = 1 << 0
	IntentGuildMembers           Intents = 1 << 1
	IntentGuildModeration        Intents = 1 << 2
	IntentGuildEmojisAndStickers Intents = 1 << 3
	IntentGuildIntegrations      Intents = 1 << 4
	IntentGuildWebhooks          Intents = 1 << 5
	IntentGuildInvites           Intents = 1 << 6
	IntentGuildVoiceStates       Intents = 1 << 7
	IntentGuildPresences         Intents = 1 << 8
	IntentGuildMessages          Intents = 1 << 9
	IntentGuildMessageReactions  Intents = 1 << 10
	IntentGuildMessageTyping     Intents = 1 << 11
	IntentDirectMessages         Intents = 1 << 12
	IntentDirectMessageReactions Intents = 1 << 13
	IntentDirectMessageTyping    Intents = 1 << 14
	IntentMessageContent         Intents = 1 << 15
	IntentGuildScheduledEvents   Intents = 1 << 16

	// IntentsDefault is a sensible starting set of intents for most bots.
	IntentsDefault = IntentGuilds | IntentGuildMessages | IntentMessageContent
)

// ChannelType enumerates Discord channel types.
type ChannelType int

const (
	ChannelTypeGuildText          ChannelType = 0
	ChannelTypeDM                 ChannelType = 1
	ChannelTypeGuildVoice         ChannelType = 2
	ChannelTypeGroupDM            ChannelType = 3
	ChannelTypeGuildCategory      ChannelType = 4
	ChannelTypeGuildAnnouncement  ChannelType = 5
	ChannelTypeAnnouncementThread ChannelType = 10
	ChannelTypePublicThread       ChannelType = 11
	ChannelTypePrivateThread      ChannelType = 12
	ChannelTypeGuildForum         ChannelType = 15
)

// ActivityType constants for bot presence.
const (
	ActivityPlaying   = 0
	ActivityStreaming = 1
	ActivityListening = 2
	ActivityWatching  = 3
	ActivityCustom    = 4
	ActivityCompeting = 5
)

// ---------------------------------------------------------------------------
// Core Discord Objects
// ---------------------------------------------------------------------------

// User represents a Discord user account.
type User struct {
	ID            Snowflake `json:"id"`
	Username      string    `json:"username"`
	Discriminator string    `json:"discriminator"`
	GlobalName    string    `json:"global_name"`
	Avatar        string    `json:"avatar"`
	Bot           bool      `json:"bot"`
	PublicFlags   int       `json:"public_flags"`
	AccentColor   int       `json:"accent_color"`
	Banner        string    `json:"banner"`
	System        bool      `json:"system,omitempty"`
}

// Tag returns the user's display tag.
func (u *User) Tag() string {
	if u.Discriminator == "" || u.Discriminator == "0" {
		return u.Username
	}
	return u.Username + "#" + u.Discriminator
}

// Mention returns a Discord @mention string.
func (u *User) Mention() string { return "<@" + u.ID + ">" }

// Member represents a guild member (User + guild-specific data).
type Member struct {
	User                       *User    `json:"user"`
	Nick                       string   `json:"nick"`
	Roles                      []string `json:"roles"`
	JoinedAt                   string   `json:"joined_at"`
	PremiumSince               string   `json:"premium_since"`
	Deaf                       bool     `json:"deaf"`
	Mute                       bool     `json:"mute"`
	Pending                    bool     `json:"pending"`
	Permissions                string   `json:"permissions"`
	CommunicationDisabledUntil string   `json:"communication_disabled_until"`
	// GuildAvatar is the member's guild-specific avatar hash (separate from User.Avatar).
	GuildAvatar string `json:"avatar"`
}

// Role represents a Discord role.
type Role struct {
	ID           Snowflake `json:"id"`
	Name         string    `json:"name"`
	Color        int       `json:"color"`
	Hoist        bool      `json:"hoist"`
	Permissions  string    `json:"permissions"`
	Position     int       `json:"position"`
	Managed      bool      `json:"managed"`
	Mentionable  bool      `json:"mentionable"`
	Icon         string    `json:"icon,omitempty"`
	UnicodeEmoji string    `json:"unicode_emoji,omitempty"`
}

// Channel represents a Discord channel.
type Channel struct {
	ID                         Snowflake             `json:"id"`
	Type                       ChannelType           `json:"type"`
	GuildID                    Snowflake             `json:"guild_id"`
	Name                       string                `json:"name"`
	Topic                      string                `json:"topic"`
	Position                   int                   `json:"position"`
	NSFW                       bool                  `json:"nsfw"`
	ParentID                   Snowflake             `json:"parent_id"`
	Bitrate                    int                   `json:"bitrate"`
	UserLimit                  int                   `json:"user_limit"`
	RateLimitPerUser           int                   `json:"rate_limit_per_user"`
	LastMessageID              string                `json:"last_message_id"`
	DefaultAutoArchiveDuration int                   `json:"default_auto_archive_duration"`
	PermissionOverwrites       []PermissionOverwrite `json:"permission_overwrites"`
	ThreadMetadata             *ThreadMetadata       `json:"thread_metadata,omitempty"`
	Member                     *ThreadMember         `json:"member,omitempty"`
}

// Mention returns a Discord #channel mention string.
func (c *Channel) Mention() string { return "<#" + c.ID + ">" }

// Guild represents a Discord server.
type Guild struct {
	ID                          Snowflake `json:"id"`
	Name                        string    `json:"name"`
	Icon                        string    `json:"icon"`
	OwnerID                     Snowflake `json:"owner_id"`
	MemberCount                 int       `json:"member_count"`
	Channels                    []Channel `json:"channels"`
	Roles                       []Role    `json:"roles"`
	Members                     []Member  `json:"members"`
	Unavailable                 bool      `json:"unavailable"`
	PreferredLocale             string    `json:"preferred_locale"`
	Description                 string    `json:"description"`
	Features                    []string  `json:"features"`
	VanityURLCode               string    `json:"vanity_url_code"`
	PremiumTier                 int       `json:"premium_tier"`
	PremiumSubscriptionCount    int       `json:"premium_subscription_count"`
	SystemChannelID             string    `json:"system_channel_id"`
	RulesChannelID              string    `json:"rules_channel_id"`
	ApproximateMemberCount      int       `json:"approximate_member_count,omitempty"`
	MaxMembers                  int       `json:"max_members"`
	Banner                      string    `json:"banner"`
	Splash                      string    `json:"splash"`
	AFKChannelID                string    `json:"afk_channel_id"`
	AFKTimeout                  int       `json:"afk_timeout"`
	VerificationLevel           int       `json:"verification_level"`
	MFALevel                    int       `json:"mfa_level"`
	ExplicitContentFilter       int       `json:"explicit_content_filter"`
	DefaultMessageNotifications int       `json:"default_message_notifications"`
	NSFWLevel                   int       `json:"nsfw_level"`
	PublicUpdatesChannelID      string    `json:"public_updates_channel_id"`
}

// GuildUnavailable is sent when a guild becomes unavailable (outage).
type GuildUnavailable struct {
	ID          Snowflake `json:"id"`
	Unavailable bool      `json:"unavailable"`
}

// Emoji represents a Discord emoji (standard or custom).
type Emoji struct {
	ID            Snowflake `json:"id"`
	Name          string    `json:"name"`
	Roles         []string  `json:"roles,omitempty"`
	User          *User     `json:"user,omitempty"`
	RequireColons bool      `json:"require_colons"`
	Managed       bool      `json:"managed"`
	Animated      bool      `json:"animated"`
	Available     bool      `json:"available"`
}

// String returns the emoji in message format.
func (e *Emoji) String() string {
	if e.ID == "" {
		return e.Name
	}
	if e.Animated {
		return "<a:" + e.Name + ":" + e.ID + ">"
	}
	return "<:" + e.Name + ":" + e.ID + ">"
}

// Attachment represents a file attached to a message.
type Attachment struct {
	ID       Snowflake `json:"id"`
	Filename string    `json:"filename"`
	Size     int       `json:"size"`
	URL      string    `json:"url"`
	ProxyURL string    `json:"proxy_url"`
	Width    int       `json:"width,omitempty"`
	Height   int       `json:"height,omitempty"`
}

// Reaction represents an emoji reaction on a message.
type Reaction struct {
	Count int   `json:"count"`
	Me    bool  `json:"me"`
	Emoji Emoji `json:"emoji"`
}

// MessageReference is used for replies and crossposting.
type MessageReference struct {
	MessageID Snowflake `json:"message_id,omitempty"`
	ChannelID Snowflake `json:"channel_id,omitempty"`
	GuildID   Snowflake `json:"guild_id,omitempty"`
}

// Message represents a Discord message.
type Message struct {
	ID                Snowflake         `json:"id"`
	ChannelID         Snowflake         `json:"channel_id"`
	GuildID           Snowflake         `json:"guild_id"`
	Author            *User             `json:"author"`
	Member            *Member           `json:"member"`
	Content           string            `json:"content"`
	Timestamp         string            `json:"timestamp"`
	EditedTimestamp   string            `json:"edited_timestamp"`
	TTS               bool              `json:"tts"`
	Pinned            bool              `json:"pinned"`
	MentionEveryone   bool              `json:"mention_everyone"`
	MentionRoles      []string          `json:"mention_roles"`
	Mentions          []*User           `json:"mentions"`
	Embeds            []Embed           `json:"embeds"`
	Attachments       []Attachment      `json:"attachments"`
	Reactions         []Reaction        `json:"reactions"`
	MessageReference  *MessageReference `json:"message_reference"`
	WebhookID         string            `json:"webhook_id,omitempty"`
	Type              int               `json:"type"`
	Flags             int               `json:"flags"`
	ReferencedMessage *Message          `json:"referenced_message,omitempty"`
	Thread            *Channel          `json:"thread,omitempty"`
	Components        []Component       `json:"components,omitempty"`
}

// ---------------------------------------------------------------------------
// Embeds
// ---------------------------------------------------------------------------

// EmbedAuthor holds author information for an embed.
type EmbedAuthor struct {
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// EmbedFooter holds footer information for an embed.
type EmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// EmbedField is a single field inside an embed.
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// EmbedImage holds an image inside an embed.
type EmbedImage struct {
	URL string `json:"url"`
}

// EmbedThumbnail holds a thumbnail inside an embed.
type EmbedThumbnail struct {
	URL string `json:"url"`
}

// Embed is a Discord rich embed object.
type Embed struct {
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	URL         string          `json:"url,omitempty"`
	Color       int             `json:"color,omitempty"`
	Timestamp   string          `json:"timestamp,omitempty"`
	Author      *EmbedAuthor    `json:"author,omitempty"`
	Footer      *EmbedFooter    `json:"footer,omitempty"`
	Image       *EmbedImage     `json:"image,omitempty"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail,omitempty"`
	Fields      []EmbedField    `json:"fields,omitempty"`
}

// ---------------------------------------------------------------------------
// Send / Edit helpers
// ---------------------------------------------------------------------------

// MessageSend is the payload for creating a new message.
type MessageSend struct {
	Content          string            `json:"content,omitempty"`
	TTS              bool              `json:"tts,omitempty"`
	Embeds           []Embed           `json:"embeds,omitempty"`
	Components       []Component       `json:"components,omitempty"`
	MessageReference *MessageReference `json:"message_reference,omitempty"`
}

// MessageEdit is the payload for editing an existing message.
type MessageEdit struct {
	Content    *string     `json:"content,omitempty"`
	Embeds     []Embed     `json:"embeds,omitempty"`
	Components []Component `json:"components,omitempty"`
}

// ---------------------------------------------------------------------------
// Gateway internals
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Moderation / Guild management objects
// ---------------------------------------------------------------------------

// Ban represents a guild ban entry.
type Ban struct {
	Reason string `json:"reason"`
	User   *User  `json:"user"`
}

// PermissionOverwrite is a channel-level permission override for a role or member.
type PermissionOverwrite struct {
	ID    Snowflake `json:"id"`
	Type  int       `json:"type"` // 0 = role, 1 = member
	Allow string    `json:"allow"`
	Deny  string    `json:"deny"`
}

// GuildMemberAddEvent is dispatched when a user joins a guild.
// Embeds all Member fields plus the GuildID.
type GuildMemberAddEvent struct {
	GuildID  Snowflake `json:"guild_id"`
	User     *User     `json:"user"`
	Nick     string    `json:"nick"`
	Roles    []string  `json:"roles"`
	JoinedAt string    `json:"joined_at"`
	Deaf     bool      `json:"deaf"`
	Mute     bool      `json:"mute"`
	Pending  bool      `json:"pending"`
}

// GuildMemberRemoveEvent is dispatched when a user leaves (or is kicked/banned from) a guild.
type GuildMemberRemoveEvent struct {
	GuildID Snowflake `json:"guild_id"`
	User    *User     `json:"user"`
}

// GuildMemberUpdateEvent is dispatched when a guild member's state changes
// (roles, nick, timeout, etc.).
type GuildMemberUpdateEvent struct {
	GuildID                    Snowflake `json:"guild_id"`
	Roles                      []string  `json:"roles"`
	User                       *User     `json:"user"`
	Nick                       string    `json:"nick"`
	JoinedAt                   string    `json:"joined_at"`
	PremiumSince               string    `json:"premium_since"`
	Deaf                       bool      `json:"deaf"`
	Mute                       bool      `json:"mute"`
	Pending                    bool      `json:"pending"`
	CommunicationDisabledUntil string    `json:"communication_disabled_until"`
}

// GuildBanAddEvent is dispatched when a user is banned from a guild.
type GuildBanAddEvent struct {
	GuildID Snowflake `json:"guild_id"`
	User    *User     `json:"user"`
}

// GuildBanRemoveEvent is dispatched when a user is unbanned from a guild.
type GuildBanRemoveEvent struct {
	GuildID Snowflake `json:"guild_id"`
	User    *User     `json:"user"`
}

// ---------------------------------------------------------------------------
// Gateway internals
// ---------------------------------------------------------------------------

// gatewayPayload is the raw envelope sent/received over the Gateway WebSocket.
type gatewayPayload struct {
	Op       int             `json:"op"`
	Data     json.RawMessage `json:"d"`
	Sequence *int64          `json:"s"`
	Type     string          `json:"t"`
}

// presence is sent to Discord to update the bot's status.
type presence struct {
	Status     string     `json:"status"`
	Activities []activity `json:"activities"`
	Since      *int64     `json:"since"`
	AFK        bool       `json:"afk"`
}

type activity struct {
	Name string `json:"name"`
	Type int    `json:"type"`
}

// ---------------------------------------------------------------------------
// Event payloads
// ---------------------------------------------------------------------------

// ReadyEvent is dispatched when the bot connects and identifies successfully.
type ReadyEvent struct {
	V                int    `json:"v"`
	User             User   `json:"user"`
	SessionID        string `json:"session_id"`
	ResumeGatewayURL string `json:"resume_gateway_url"`
}

// MessageDeleteEvent carries the IDs of a deleted message.
type MessageDeleteEvent struct {
	ID        Snowflake `json:"id"`
	ChannelID Snowflake `json:"channel_id"`
	GuildID   Snowflake `json:"guild_id"`
}

// MessageReactionAddEvent is dispatched when a user adds a reaction.
type MessageReactionAddEvent struct {
	UserID    Snowflake `json:"user_id"`
	ChannelID Snowflake `json:"channel_id"`
	MessageID Snowflake `json:"message_id"`
	GuildID   Snowflake `json:"guild_id"`
	Member    *Member   `json:"member"`
	Emoji     Emoji     `json:"emoji"`
}

// MessageReactionRemoveEvent is dispatched when a reaction is removed.
type MessageReactionRemoveEvent struct {
	UserID    Snowflake `json:"user_id"`
	ChannelID Snowflake `json:"channel_id"`
	MessageID Snowflake `json:"message_id"`
	GuildID   Snowflake `json:"guild_id"`
	Emoji     Emoji     `json:"emoji"`
}

// ---------------------------------------------------------------------------
// Additional types
// ---------------------------------------------------------------------------

// VoiceState represents a user's connection to a voice channel.
type VoiceState struct {
	GuildID   string  `json:"guild_id"`
	ChannelID string  `json:"channel_id"`
	UserID    string  `json:"user_id"`
	Member    *Member `json:"member,omitempty"`
	SessionID string  `json:"session_id"`
	Deaf      bool    `json:"deaf"`
	Mute      bool    `json:"mute"`
	SelfDeaf  bool    `json:"self_deaf"`
	SelfMute  bool    `json:"self_mute"`
	SelfVideo bool    `json:"self_video"`
	Suppress  bool    `json:"suppress"`
}

// BanEntry is the paginated ban record returned by the bans list endpoint.
type BanEntry struct {
	Reason string `json:"reason"`
	User   *User  `json:"user"`
}

// Invite represents a Discord invite.
type Invite struct {
	Code                     string   `json:"code"`
	Guild                    *Guild   `json:"guild,omitempty"`
	Channel                  *Channel `json:"channel"`
	Inviter                  *User    `json:"inviter,omitempty"`
	MaxAge                   int      `json:"max_age"`
	MaxUses                  int      `json:"max_uses"`
	Uses                     int      `json:"uses"`
	Temporary                bool     `json:"temporary"`
	CreatedAt                string   `json:"created_at"`
	ApproximateMemberCount   int      `json:"approximate_member_count,omitempty"`
	ApproximatePresenceCount int      `json:"approximate_presence_count,omitempty"`
}

// ThreadMetadata holds archival state for a thread channel.
type ThreadMetadata struct {
	Archived            bool   `json:"archived"`
	AutoArchiveDuration int    `json:"auto_archive_duration"`
	ArchiveTimestamp    string `json:"archive_timestamp"`
	Locked              bool   `json:"locked"`
	Invitable           bool   `json:"invitable,omitempty"`
	CreateTimestamp     string `json:"create_timestamp,omitempty"`
}

// ThreadMember represents a user who has joined a thread.
type ThreadMember struct {
	ID            Snowflake `json:"id,omitempty"`
	UserID        Snowflake `json:"user_id,omitempty"`
	JoinTimestamp string    `json:"join_timestamp"`
	Flags         int       `json:"flags"`
}

// ---------------------------------------------------------------------------
// New gateway event payloads
// ---------------------------------------------------------------------------

// ChannelCreateEvent is dispatched when a channel is created.
// It is an alias for Channel — the Gateway sends the full channel object.
type ChannelCreateEvent = Channel

// ChannelUpdateEvent is dispatched when a channel is updated.
// It is an alias for Channel — the Gateway sends the full channel object.
type ChannelUpdateEvent = Channel

// ChannelDeleteEvent is dispatched when a channel is deleted.
// It is an alias for Channel — the Gateway sends the full channel object.
type ChannelDeleteEvent = Channel

// GuildUpdateEvent is dispatched when a guild's settings change.
// It is an alias for Guild — the Gateway sends the full guild object.
type GuildUpdateEvent = Guild

// GuildRoleCreateEvent is dispatched when a role is created in a guild.
type GuildRoleCreateEvent struct {
	GuildID string `json:"guild_id"`
	Role    Role   `json:"role"`
}

// GuildRoleUpdateEvent is dispatched when a role is updated in a guild.
type GuildRoleUpdateEvent struct {
	GuildID string `json:"guild_id"`
	Role    Role   `json:"role"`
}

// GuildRoleDeleteEvent is dispatched when a role is deleted from a guild.
type GuildRoleDeleteEvent struct {
	GuildID string `json:"guild_id"`
	RoleID  string `json:"role_id"`
}

// ThreadCreateEvent is dispatched when a thread is created.
// It is an alias for Channel — the Gateway sends the full channel object.
type ThreadCreateEvent = Channel

// ThreadUpdateEvent is dispatched when a thread is updated.
// It is an alias for Channel — the Gateway sends the full channel object.
type ThreadUpdateEvent = Channel

// ThreadDeleteEvent is dispatched when a thread is deleted.
// It is an alias for Channel — the Gateway sends the full channel object.
type ThreadDeleteEvent = Channel

// InviteCreateEvent is dispatched when an invite is created.
type InviteCreateEvent struct {
	Code      string `json:"code"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
	Inviter   *User  `json:"inviter"`
	MaxAge    int    `json:"max_age"`
	MaxUses   int    `json:"max_uses"`
	Temporary bool   `json:"temporary"`
	CreatedAt string `json:"created_at"`
}

// InviteDeleteEvent is dispatched when an invite is deleted.
type InviteDeleteEvent struct {
	Code      string `json:"code"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
}

// WebhooksUpdateEvent is dispatched when a channel's webhooks change.
type WebhooksUpdateEvent struct {
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
}

// VoiceStateUpdateEvent is dispatched when a user's voice state changes.
type VoiceStateUpdateEvent struct {
	*VoiceState
}

// TypingStartEvent is dispatched when a user starts typing.
type TypingStartEvent struct {
	ChannelID string  `json:"channel_id"`
	GuildID   string  `json:"guild_id"`
	UserID    string  `json:"user_id"`
	Timestamp int64   `json:"timestamp"`
	Member    *Member `json:"member"`
}

// MessageDeleteBulkEvent is dispatched when multiple messages are deleted at once.
type MessageDeleteBulkEvent struct {
	IDs       []string `json:"ids"`
	ChannelID string   `json:"channel_id"`
	GuildID   string   `json:"guild_id"`
}

// ReactionRemoveAllEvent is dispatched when all reactions are removed from a message.
type ReactionRemoveAllEvent struct {
	ChannelID string `json:"channel_id"`
	MessageID string `json:"message_id"`
	GuildID   string `json:"guild_id"`
}

// ReactionRemoveEmojiEvent is dispatched when all reactions for a specific emoji are removed.
type ReactionRemoveEmojiEvent struct {
	ChannelID string `json:"channel_id"`
	MessageID string `json:"message_id"`
	GuildID   string `json:"guild_id"`
	Emoji     *Emoji `json:"emoji"`
}

// ---------------------------------------------------------------------------
// Audit Log
// ---------------------------------------------------------------------------

// Integration represents a guild integration (Twitch, YouTube, Discord, etc.).
type Integration struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"` // "twitch", "youtube", "discord", "guild_subscription"
	Enabled bool   `json:"enabled"`
	GuildID string `json:"guild_id"`
	Account struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"account"`
}

// UserUpdateEvent is dispatched when the current user's properties change.
type UserUpdateEvent struct {
	User User `json:"user"`
}

// IntegrationCreateEvent is dispatched when an integration is created in a guild.
type IntegrationCreateEvent struct {
	GuildID     string      `json:"guild_id"`
	Integration Integration `json:"integration"`
}

// IntegrationUpdateEvent is dispatched when an integration is updated in a guild.
type IntegrationUpdateEvent struct {
	GuildID     string      `json:"guild_id"`
	Integration Integration `json:"integration"`
}

// IntegrationDeleteEvent is dispatched when an integration is deleted from a guild.
type IntegrationDeleteEvent struct {
	ID            string `json:"id"`
	GuildID       string `json:"guild_id"`
	ApplicationID string `json:"application_id,omitempty"`
}

// AuditLog is the response from GET /guilds/{id}/audit-logs.
type AuditLog struct {
	AuditLogEntries []AuditLogEntry `json:"audit_log_entries"`
	Users           []User          `json:"users"`
}

// AuditLogEntry is a single entry in the audit log.
type AuditLogEntry struct {
	ID         string           `json:"id"`
	TargetID   string           `json:"target_id"`
	UserID     string           `json:"user_id"`
	ActionType int              `json:"action_type"`
	Options    *AuditLogOptions `json:"options,omitempty"`
	Changes    []AuditLogChange `json:"changes,omitempty"`
	Reason     string           `json:"reason,omitempty"`
}

// AuditLogOptions holds optional extra information for certain action types.
type AuditLogOptions struct {
	ChannelID        string `json:"channel_id,omitempty"`
	Count            string `json:"count,omitempty"`
	DeleteMemberDays string `json:"delete_member_days,omitempty"`
	ID               string `json:"id,omitempty"`
	MembersRemoved   string `json:"members_removed,omitempty"`
	MessageID        string `json:"message_id,omitempty"`
	RoleName         string `json:"role_name,omitempty"`
	Type             string `json:"type,omitempty"`
}

// AuditLogChange describes a single changed field within an audit log entry.
type AuditLogChange struct {
	Key      string      `json:"key"`
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
}
