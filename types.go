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
	ActivityStreaming  = 1
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
	User     *User    `json:"user"`
	Nick     string   `json:"nick"`
	Roles    []string `json:"roles"`
	JoinedAt string   `json:"joined_at"`
	Deaf     bool     `json:"deaf"`
	Mute     bool     `json:"mute"`
}

// Role represents a Discord role.
type Role struct {
	ID          Snowflake `json:"id"`
	Name        string    `json:"name"`
	Color       int       `json:"color"`
	Hoist       bool      `json:"hoist"`
	Permissions string    `json:"permissions"`
	Position    int       `json:"position"`
	Managed     bool      `json:"managed"`
	Mentionable bool      `json:"mentionable"`
}

// Channel represents a Discord channel.
type Channel struct {
	ID       Snowflake   `json:"id"`
	Type     ChannelType `json:"type"`
	GuildID  Snowflake   `json:"guild_id"`
	Name     string      `json:"name"`
	Topic    string      `json:"topic"`
	Position int         `json:"position"`
	NSFW     bool        `json:"nsfw"`
	ParentID Snowflake   `json:"parent_id"`
}

// Mention returns a Discord #channel mention string.
func (c *Channel) Mention() string { return "<#" + c.ID + ">" }

// Guild represents a Discord server.
type Guild struct {
	ID          Snowflake `json:"id"`
	Name        string    `json:"name"`
	Icon        string    `json:"icon"`
	OwnerID     Snowflake `json:"owner_id"`
	MemberCount int       `json:"member_count"`
	Channels    []Channel `json:"channels"`
	Roles       []Role    `json:"roles"`
	Members     []Member  `json:"members"`
}

// GuildUnavailable is sent when a guild becomes unavailable (outage).
type GuildUnavailable struct {
	ID          Snowflake `json:"id"`
	Unavailable bool      `json:"unavailable"`
}

// Emoji represents a Discord emoji (standard unicode or custom).
type Emoji struct {
	ID       Snowflake `json:"id"`
	Name     string    `json:"name"`
	Animated bool      `json:"animated"`
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
	ID               Snowflake         `json:"id"`
	ChannelID        Snowflake         `json:"channel_id"`
	GuildID          Snowflake         `json:"guild_id"`
	Author           *User             `json:"author"`
	Member           *Member           `json:"member"`
	Content          string            `json:"content"`
	Timestamp        string            `json:"timestamp"`
	EditedTimestamp  string            `json:"edited_timestamp"`
	TTS              bool              `json:"tts"`
	Pinned           bool              `json:"pinned"`
	Embeds           []Embed           `json:"embeds"`
	Attachments      []Attachment      `json:"attachments"`
	Reactions        []Reaction        `json:"reactions"`
	MessageReference *MessageReference `json:"message_reference"`
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
	MessageReference *MessageReference `json:"message_reference,omitempty"`
}

// MessageEdit is the payload for editing an existing message.
type MessageEdit struct {
	Content *string `json:"content,omitempty"`
	Embeds  []Embed `json:"embeds,omitempty"`
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
