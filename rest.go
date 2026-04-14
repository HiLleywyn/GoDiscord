package discord

// rest.go — Discord REST API v10 client.
//
// Features:
//   - All requests carry the Authorization and User-Agent headers.
//   - Failed requests return *APIError with the HTTP status and Discord JSON
//     error code, so callers can branch with errors.As().
//   - 429 Too Many Requests is handled transparently: the client sleeps for
//     Retry-After seconds and retries up to maxRateLimitRetries times, then
//     returns an *APIError with StatusCode == 429 once the budget is
//     exhausted so callers can back off at a higher level.
//   - All public methods have descriptive godoc comments.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// maxRateLimitRetries is the maximum number of times a single REST call will
// be retried after a 429 Too Many Requests response. This prevents an infinite
// recursion when Discord continuously rate-limits a request.
const maxRateLimitRetries = 3

const apiBase = "https://discord.com/api/v10"

// RestClient is an authenticated HTTP client for the Discord REST API.
// Obtain one via bot.Rest — do not construct directly.
type RestClient struct {
	token  string
	bot    *Bot
	client *http.Client
}

func newRestClient(token string, bot *Bot) *RestClient {
	return &RestClient{
		token:  token,
		bot:    bot,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// discordErrorBody is the JSON shape Discord returns for API errors.
type discordErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// do performs an HTTP request and decodes the JSON response into out (may be nil).
// On non-2xx responses it returns *APIError. 429 Too Many Requests responses
// are retried up to maxRateLimitRetries times after sleeping for Retry-After.
func (r *RestClient) do(method, path string, body interface{}, out interface{}) error {
	return r.doWithRetry(method, path, body, out, 0)
}

func (r *RestClient) doWithRetry(method, path string, body interface{}, out interface{}, retries int) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, apiBase+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+r.token)
	req.Header.Set("User-Agent", "GoDiscord (https://github.com/hilleywyn/godiscord, 1.0)")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Transparent rate-limit handling: sleep and retry up to maxRateLimitRetries.
	if resp.StatusCode == http.StatusTooManyRequests {
		if retries >= maxRateLimitRetries {
			return &APIError{
				Method:     method,
				Path:       path,
				StatusCode: resp.StatusCode,
				Message:    "rate limit retry budget exhausted",
			}
		}
		retryAfter := resp.Header.Get("Retry-After")
		secs, _ := strconv.ParseFloat(retryAfter, 64)
		if secs <= 0 {
			secs = 1
		}
		if r.bot != nil {
			r.bot.log.Printf("[rest] rate limited on %s %s — retrying in %.2fs (attempt %d/%d)",
				method, path, secs, retries+1, maxRateLimitRetries)
		}
		time.Sleep(time.Duration(secs*1000) * time.Millisecond)
		return r.doWithRetry(method, path, body, out, retries+1)
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{
			Method:     method,
			Path:       path,
			StatusCode: resp.StatusCode,
			Message:    string(raw),
		}
		// Try to decode the Discord JSON error body for the code and message.
		// Use errBody to avoid shadowing the outer body parameter.
		var errBody discordErrorBody
		if err := json.Unmarshal(raw, &errBody); err == nil && errBody.Code != 0 {
			apiErr.Code = errBody.Code
			apiErr.Message = errBody.Message
		}
		return apiErr
	}

	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

func (r *RestClient) get(path string, out interface{}) error {
	return r.do(http.MethodGet, path, nil, out)
}

func (r *RestClient) post(path string, body, out interface{}) error {
	return r.do(http.MethodPost, path, body, out)
}

func (r *RestClient) patch(path string, body, out interface{}) error {
	return r.do(http.MethodPatch, path, body, out)
}

func (r *RestClient) delete(path string) error {
	return r.do(http.MethodDelete, path, nil, nil)
}

func (r *RestClient) put(path string, body interface{}) error {
	return r.do(http.MethodPut, path, body, nil)
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// SendMessage sends a plain-text message to a channel.
func (r *RestClient) SendMessage(channelID, content string) (*Message, error) {
	return r.SendMessageComplex(channelID, &MessageSend{Content: content})
}

// SendMessageComplex sends a message with full control over the payload
// (embeds, components, reply references, TTS, etc.).
func (r *RestClient) SendMessageComplex(channelID string, msg *MessageSend) (*Message, error) {
	var m Message
	if err := r.post("/channels/"+channelID+"/messages", msg, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ReplyTo sends a reply to an existing message.
func (r *RestClient) ReplyTo(msg *Message, content string) (*Message, error) {
	return r.SendMessageComplex(msg.ChannelID, &MessageSend{
		Content: content,
		MessageReference: &MessageReference{
			MessageID: msg.ID,
			ChannelID: msg.ChannelID,
			GuildID:   msg.GuildID,
		},
	})
}

// SendEmbed sends a message containing a single embed.
func (r *RestClient) SendEmbed(channelID string, embed Embed) (*Message, error) {
	return r.SendMessageComplex(channelID, &MessageSend{Embeds: []Embed{embed}})
}

// EditMessage edits the content of a message authored by the bot.
func (r *RestClient) EditMessage(channelID, messageID, content string) (*Message, error) {
	var m Message
	if err := r.patch("/channels/"+channelID+"/messages/"+messageID,
		&MessageEdit{Content: &content}, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// EditMessageEmbed replaces the embeds on a bot-authored message.
func (r *RestClient) EditMessageEmbed(channelID, messageID string, embed Embed) (*Message, error) {
	var m Message
	if err := r.patch("/channels/"+channelID+"/messages/"+messageID,
		&MessageEdit{Embeds: []Embed{embed}}, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// EditMessageComplex edits a message with full control over the payload.
func (r *RestClient) EditMessageComplex(channelID, messageID string, edit *MessageEdit) (*Message, error) {
	var m Message
	if err := r.patch("/channels/"+channelID+"/messages/"+messageID, edit, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// DeleteMessage deletes a message.
func (r *RestClient) DeleteMessage(channelID, messageID string) error {
	return r.delete("/channels/" + channelID + "/messages/" + messageID)
}

// GetMessage fetches a single message by ID.
func (r *RestClient) GetMessage(channelID, messageID string) (*Message, error) {
	var m Message
	if err := r.get("/channels/"+channelID+"/messages/"+messageID, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// PinMessage pins a message in a channel.
func (r *RestClient) PinMessage(channelID, messageID string) error {
	return r.put("/channels/"+channelID+"/pins/"+messageID, nil)
}

// UnpinMessage unpins a message from a channel.
func (r *RestClient) UnpinMessage(channelID, messageID string) error {
	return r.delete("/channels/" + channelID + "/pins/" + messageID)
}

// ---------------------------------------------------------------------------
// Reactions
// ---------------------------------------------------------------------------

// AddReaction adds a reaction emoji to a message.
// emoji should be a unicode character (e.g. "👍") or "name:id" for custom
// emojis. The value is URL-encoded before being placed in the path.
func (r *RestClient) AddReaction(channelID, messageID, emoji string) error {
	return r.put("/channels/"+channelID+"/messages/"+messageID+"/reactions/"+url.PathEscape(emoji)+"/@me", nil)
}

// RemoveReaction removes the bot's own reaction from a message.
// emoji should be a unicode character (e.g. "👍") or "name:id" for custom emojis.
func (r *RestClient) RemoveReaction(channelID, messageID, emoji string) error {
	return r.delete("/channels/" + channelID + "/messages/" + messageID + "/reactions/" + url.PathEscape(emoji) + "/@me")
}

// ---------------------------------------------------------------------------
// Channels
// ---------------------------------------------------------------------------

// GetChannel fetches a channel by ID.
func (r *RestClient) GetChannel(channelID string) (*Channel, error) {
	var c Channel
	if err := r.get("/channels/"+channelID, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateDM opens (or returns an existing) DM channel with a user.
func (r *RestClient) CreateDM(userID string) (*Channel, error) {
	var c Channel
	if err := r.post("/users/@me/channels", map[string]string{"recipient_id": userID}, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// SendDM sends a plain-text direct message to a user.
func (r *RestClient) SendDM(userID, content string) (*Message, error) {
	ch, err := r.CreateDM(userID)
	if err != nil {
		return nil, err
	}
	return r.SendMessage(ch.ID, content)
}

// SendEmbedDM sends an embed via direct message to a user.
func (r *RestClient) SendEmbedDM(userID string, embed Embed) (*Message, error) {
	ch, err := r.CreateDM(userID)
	if err != nil {
		return nil, err
	}
	return r.SendEmbed(ch.ID, embed)
}

// ---------------------------------------------------------------------------
// Guilds
// ---------------------------------------------------------------------------

// GetGuild fetches a guild (server) by ID.
func (r *RestClient) GetGuild(guildID string) (*Guild, error) {
	var g Guild
	if err := r.get("/guilds/"+guildID, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// GetGuildMember fetches a specific member from a guild.
func (r *RestClient) GetGuildMember(guildID, userID string) (*Member, error) {
	var m Member
	if err := r.get("/guilds/"+guildID+"/members/"+userID, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// SearchGuildMembers searches guild members whose username or nickname starts
// with query. limit is capped at 1000 by Discord; pass a small value (e.g. 5)
// for interactive lookups. Returns nil slice (not an error) when no members match.
func (r *RestClient) SearchGuildMembers(guildID, query string, limit int) ([]*Member, error) {
	if limit <= 0 {
		limit = 5
	}
	path := fmt.Sprintf("/guilds/%s/members/search?query=%s&limit=%d",
		guildID, url.QueryEscape(query), limit)
	var members []*Member
	if err := r.get(path, &members); err != nil {
		return nil, err
	}
	return members, nil
}

// KickMember removes a member from a guild.
func (r *RestClient) KickMember(guildID, userID string) error {
	return r.delete("/guilds/" + guildID + "/members/" + userID)
}

// BanMember bans a user from a guild.
// deleteMessageDays is the number of days of message history to purge (0–7).
// Values outside the 0–7 range are clamped to the nearest valid boundary.
func (r *RestClient) BanMember(guildID, userID string, deleteMessageDays int) error {
	if deleteMessageDays < 0 {
		deleteMessageDays = 0
	}
	if deleteMessageDays > 7 {
		deleteMessageDays = 7
	}
	return r.put("/guilds/"+guildID+"/bans/"+userID,
		map[string]int{"delete_message_days": deleteMessageDays})
}

// UnbanMember removes a ban from a guild.
func (r *RestClient) UnbanMember(guildID, userID string) error {
	return r.delete("/guilds/" + guildID + "/bans/" + userID)
}

// AddMemberRole adds a role to a guild member.
func (r *RestClient) AddMemberRole(guildID, userID, roleID string) error {
	return r.put("/guilds/"+guildID+"/members/"+userID+"/roles/"+roleID, nil)
}

// RemoveMemberRole removes a role from a guild member.
func (r *RestClient) RemoveMemberRole(guildID, userID, roleID string) error {
	return r.delete("/guilds/" + guildID + "/members/" + userID + "/roles/" + roleID)
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

// GetUser fetches a user by ID.
func (r *RestClient) GetUser(userID string) (*User, error) {
	var u User
	if err := r.get("/users/"+userID, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// ---------------------------------------------------------------------------
// Guild member management
// ---------------------------------------------------------------------------

// ModifyGuildMember updates attributes of a guild member.
// Accepted keys: nick, roles, mute, deaf, channel_id, communication_disabled_until.
func (r *RestClient) ModifyGuildMember(guildID, userID string, data map[string]interface{}) (*Member, error) {
	var m Member
	if err := r.patch("/guilds/"+guildID+"/members/"+userID, data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// TimeoutMember applies a Discord communication timeout to a member.
// Pass an empty string for until to remove an existing timeout.
// until must be an RFC3339 UTC timestamp, e.g.:
//
//	time.Now().Add(5 * time.Minute).UTC().Format(time.RFC3339)
func (r *RestClient) TimeoutMember(guildID, userID, until string) error {
	var val interface{}
	if until != "" {
		val = until
	}
	return r.patch("/guilds/"+guildID+"/members/"+userID,
		map[string]interface{}{"communication_disabled_until": val}, nil)
}

// GetGuildRoles returns all roles for a guild.
func (r *RestClient) GetGuildRoles(guildID string) ([]Role, error) {
	var roles []Role
	if err := r.get("/guilds/"+guildID+"/roles", &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

// GetGuildChannels returns all channels for a guild.
func (r *RestClient) GetGuildChannels(guildID string) ([]Channel, error) {
	var channels []Channel
	if err := r.get("/guilds/"+guildID+"/channels", &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

// GetGuildBan fetches a specific ban record.
func (r *RestClient) GetGuildBan(guildID, userID string) (*Ban, error) {
	var b Ban
	if err := r.get("/guilds/"+guildID+"/bans/"+userID, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// GetGuildBans returns all ban records for a guild.
func (r *RestClient) GetGuildBans(guildID string) ([]Ban, error) {
	var bans []Ban
	if err := r.get("/guilds/"+guildID+"/bans", &bans); err != nil {
		return nil, err
	}
	return bans, nil
}

// ---------------------------------------------------------------------------
// Messages — bulk delete and listing
// ---------------------------------------------------------------------------

// GetMessages fetches up to limit (1–100) recent messages from a channel.
// limit is clamped to the range [1, 100].
func (r *RestClient) GetMessages(channelID string, limit int) ([]Message, error) {
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}
	var msgs []Message
	path := fmt.Sprintf("/channels/%s/messages?limit=%d", channelID, limit)
	if err := r.get(path, &msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

// BulkDeleteMessages deletes 2–100 messages at once.
// Messages older than 14 days cannot be bulk-deleted (Discord restriction).
// Returns an error immediately if fewer than 2 or more than 100 IDs are provided.
func (r *RestClient) BulkDeleteMessages(channelID string, messageIDs []string) error {
	if len(messageIDs) < 2 {
		return fmt.Errorf("discord: BulkDeleteMessages requires at least 2 message IDs (got %d)", len(messageIDs))
	}
	if len(messageIDs) > 100 {
		return fmt.Errorf("discord: BulkDeleteMessages accepts at most 100 message IDs (got %d)", len(messageIDs))
	}
	return r.post("/channels/"+channelID+"/messages/bulk-delete",
		map[string]interface{}{"messages": messageIDs}, nil)
}

// ---------------------------------------------------------------------------
// Channels — modification
// ---------------------------------------------------------------------------

// ModifyChannel updates channel settings.
// Accepted keys: name, topic, nsfw, rate_limit_per_user, position, permission_overwrites, etc.
func (r *RestClient) ModifyChannel(channelID string, data map[string]interface{}) (*Channel, error) {
	var ch Channel
	if err := r.patch("/channels/"+channelID, data, &ch); err != nil {
		return nil, err
	}
	return &ch, nil
}

// EditChannelPermissions sets a permission overwrite on a channel.
// overwriteID is a role or user ID; typ is 0 for role, 1 for member.
// allow and deny are permission bitfield strings (e.g. "2048").
func (r *RestClient) EditChannelPermissions(channelID, overwriteID string, allow, deny string, typ int) error {
	return r.put("/channels/"+channelID+"/permissions/"+overwriteID,
		map[string]interface{}{"allow": allow, "deny": deny, "type": typ})
}

// DeleteChannelPermission removes a permission overwrite from a channel.
func (r *RestClient) DeleteChannelPermission(channelID, overwriteID string) error {
	return r.delete("/channels/" + channelID + "/permissions/" + overwriteID)
}

// ---------------------------------------------------------------------------
// Webhooks
// ---------------------------------------------------------------------------

// Webhook represents a Discord webhook.
type Webhook struct {
	ID        Snowflake `json:"id"`
	Type      int       `json:"type"`
	GuildID   Snowflake `json:"guild_id"`
	ChannelID Snowflake `json:"channel_id"`
	Name      string    `json:"name"`
	Avatar    string    `json:"avatar"`
	Token     string    `json:"token"`
	URL       string    `json:"url"`
}

// WebhookSend is the payload for executing a webhook.
type WebhookSend struct {
	Content   string  `json:"content,omitempty"`
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	TTS       bool    `json:"tts,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

// CreateWebhook creates a new webhook in a channel.
// name is required; avatarDataURI is optional (pass "" to skip).
func (r *RestClient) CreateWebhook(channelID, name, avatarDataURI string) (*Webhook, error) {
	body := map[string]interface{}{"name": name}
	if avatarDataURI != "" {
		body["avatar"] = avatarDataURI
	}
	var wh Webhook
	if err := r.post("/channels/"+channelID+"/webhooks", body, &wh); err != nil {
		return nil, err
	}
	return &wh, nil
}

// GetWebhook fetches a webhook by ID.
func (r *RestClient) GetWebhook(webhookID string) (*Webhook, error) {
	var wh Webhook
	if err := r.get("/webhooks/"+webhookID, &wh); err != nil {
		return nil, err
	}
	return &wh, nil
}

// GetChannelWebhooks returns all webhooks for a channel.
func (r *RestClient) GetChannelWebhooks(channelID string) ([]Webhook, error) {
	var whs []Webhook
	if err := r.get("/channels/"+channelID+"/webhooks", &whs); err != nil {
		return nil, err
	}
	return whs, nil
}

// DeleteWebhook deletes a webhook by ID.
func (r *RestClient) DeleteWebhook(webhookID string) error {
	return r.delete("/webhooks/" + webhookID)
}

// ExecuteWebhook sends a message via a webhook using its ID and token.
// Pass wait=true to receive the created Message back.
func (r *RestClient) ExecuteWebhook(webhookID, webhookToken string, msg *WebhookSend, wait bool) (*Message, error) {
	path := fmt.Sprintf("/webhooks/%s/%s", webhookID, webhookToken)
	if wait {
		path += "?wait=true"
	}
	var m Message
	var out interface{}
	if wait {
		out = &m
	}
	if err := r.post(path, msg, out); err != nil {
		return nil, err
	}
	if wait {
		return &m, nil
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Current User
// ---------------------------------------------------------------------------

// GetCurrentUser returns the bot's own user object.
func (r *RestClient) GetCurrentUser() (*User, error) {
	var u User
	if err := r.get("/users/@me", &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// ModifyCurrentUser updates the bot's username or avatar.
// Accepted keys: username, avatar (data URI string).
func (r *RestClient) ModifyCurrentUser(data map[string]interface{}) (*User, error) {
	var u User
	if err := r.patch("/users/@me", data, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetCurrentUserGuilds returns guilds the current user is a member of.
// Pass limit=0 to omit the limit param. before and after are guild IDs for pagination.
func (r *RestClient) GetCurrentUserGuilds(limit int, before, after string) ([]*Guild, error) {
	path := "/users/@me/guilds"
	params := []string{}
	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}
	if before != "" {
		params = append(params, "before="+before)
	}
	if after != "" {
		params = append(params, "after="+after)
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}
	var guilds []*Guild
	if err := r.get(path, &guilds); err != nil {
		return nil, err
	}
	return guilds, nil
}

// LeaveGuild removes the current user from a guild.
func (r *RestClient) LeaveGuild(guildID string) error {
	return r.delete("/users/@me/guilds/" + guildID)
}

// ---------------------------------------------------------------------------
// Guild management
// ---------------------------------------------------------------------------

// ModifyGuild updates a guild's settings.
// Accepted keys: name, region, icon, verification_level, etc.
func (r *RestClient) ModifyGuild(guildID string, data map[string]interface{}) (*Guild, error) {
	var g Guild
	if err := r.patch("/guilds/"+guildID, data, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// CreateChannel creates a new channel in a guild.
// Accepted keys: name, type, topic, position, permission_overwrites, parent_id, nsfw, etc.
func (r *RestClient) CreateChannel(guildID string, data map[string]interface{}) (*Channel, error) {
	var ch Channel
	if err := r.post("/guilds/"+guildID+"/channels", data, &ch); err != nil {
		return nil, err
	}
	return &ch, nil
}

// CreateRole creates a new role in a guild.
// Accepted keys: name, permissions, color, hoist, mentionable.
func (r *RestClient) CreateRole(guildID string, data map[string]interface{}) (*Role, error) {
	var role Role
	if err := r.post("/guilds/"+guildID+"/roles", data, &role); err != nil {
		return nil, err
	}
	return &role, nil
}

// ModifyRole updates an existing role.
// Accepted keys: name, permissions, color, hoist, mentionable.
func (r *RestClient) ModifyRole(guildID, roleID string, data map[string]interface{}) (*Role, error) {
	var role Role
	if err := r.patch("/guilds/"+guildID+"/roles/"+roleID, data, &role); err != nil {
		return nil, err
	}
	return &role, nil
}

// DeleteRole deletes a role from a guild.
func (r *RestClient) DeleteRole(guildID, roleID string) error {
	return r.delete("/guilds/" + guildID + "/roles/" + roleID)
}

// ModifyRolePositions bulk-updates role positions. Each entry must contain
// "id" (role ID) and "position" (int).
func (r *RestClient) ModifyRolePositions(guildID string, positions []map[string]interface{}) ([]*Role, error) {
	var roles []*Role
	if err := r.patch("/guilds/"+guildID+"/roles", positions, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

// GetGuildBansPaginated returns ban records for a guild with optional pagination.
// Pass limit=0 to omit the limit param. before and after are user IDs.
func (r *RestClient) GetGuildBansPaginated(guildID string, limit int, before, after string) ([]*BanEntry, error) {
	path := "/guilds/" + guildID + "/bans"
	params := []string{}
	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}
	if before != "" {
		params = append(params, "before="+before)
	}
	if after != "" {
		params = append(params, "after="+after)
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}
	var bans []*BanEntry
	if err := r.get(path, &bans); err != nil {
		return nil, err
	}
	return bans, nil
}

// GetGuildInvites returns all active invites for a guild.
func (r *RestClient) GetGuildInvites(guildID string) ([]*Invite, error) {
	var invites []*Invite
	if err := r.get("/guilds/"+guildID+"/invites", &invites); err != nil {
		return nil, err
	}
	return invites, nil
}

// GetGuildEmojis returns all custom emojis for a guild.
func (r *RestClient) GetGuildEmojis(guildID string) ([]*Emoji, error) {
	var emojis []*Emoji
	if err := r.get("/guilds/"+guildID+"/emojis", &emojis); err != nil {
		return nil, err
	}
	return emojis, nil
}

// CreateEmoji creates a new custom emoji in a guild.
// Accepted keys: name, image (data URI), roles.
func (r *RestClient) CreateEmoji(guildID string, data map[string]interface{}) (*Emoji, error) {
	var e Emoji
	if err := r.post("/guilds/"+guildID+"/emojis", data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// ModifyEmoji updates a custom emoji's name or allowed roles.
// Accepted keys: name, roles.
func (r *RestClient) ModifyEmoji(guildID, emojiID string, data map[string]interface{}) (*Emoji, error) {
	var e Emoji
	if err := r.patch("/guilds/"+guildID+"/emojis/"+emojiID, data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// DeleteEmoji deletes a custom emoji from a guild.
func (r *RestClient) DeleteEmoji(guildID, emojiID string) error {
	return r.delete("/guilds/" + guildID + "/emojis/" + emojiID)
}

// ---------------------------------------------------------------------------
// Channel management
// ---------------------------------------------------------------------------

// DeleteChannel deletes a channel by ID.
func (r *RestClient) DeleteChannel(channelID string) error {
	return r.delete("/channels/" + channelID)
}

// GetChannelMessages fetches messages from a channel with optional pagination.
// Pass limit=0 to omit the limit param. before and after are message IDs.
func (r *RestClient) GetChannelMessages(channelID string, limit int, before, after string) ([]*Message, error) {
	path := "/channels/" + channelID + "/messages"
	params := []string{}
	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}
	if before != "" {
		params = append(params, "before="+before)
	}
	if after != "" {
		params = append(params, "after="+after)
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}
	var msgs []*Message
	if err := r.get(path, &msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

// GetPinnedMessages returns all pinned messages in a channel.
func (r *RestClient) GetPinnedMessages(channelID string) ([]*Message, error) {
	var msgs []*Message
	if err := r.get("/channels/"+channelID+"/pins", &msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

// CreateChannelInvite creates an invite for a channel.
// Accepted keys: max_age, max_uses, temporary, unique, target_type, target_user_id.
func (r *RestClient) CreateChannelInvite(channelID string, data map[string]interface{}) (*Invite, error) {
	var inv Invite
	if err := r.post("/channels/"+channelID+"/invites", data, &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

// GetChannelInvites returns all invites for a channel.
func (r *RestClient) GetChannelInvites(channelID string) ([]*Invite, error) {
	var invites []*Invite
	if err := r.get("/channels/"+channelID+"/invites", &invites); err != nil {
		return nil, err
	}
	return invites, nil
}

// TriggerTypingIndicator triggers the typing indicator in a channel.
func (r *RestClient) TriggerTypingIndicator(channelID string) error {
	return r.do(http.MethodPost, "/channels/"+channelID+"/typing", nil, nil)
}

// TriggerTyping triggers the typing indicator in a channel.
// This is an alias for TriggerTypingIndicator.
func (r *RestClient) TriggerTyping(channelID string) error {
	return r.do(http.MethodPost, "/channels/"+channelID+"/typing", nil, nil)
}

// ---------------------------------------------------------------------------
// Invites
// ---------------------------------------------------------------------------

// GetInvite fetches an invite by its code. Pass withCounts=true to include
// approximate member and presence counts.
func (r *RestClient) GetInvite(code string, withCounts bool) (*Invite, error) {
	path := "/invites/" + code
	if withCounts {
		path += "?with_counts=true"
	}
	var inv Invite
	if err := r.get(path, &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

// DeleteInvite revokes an invite by its code.
func (r *RestClient) DeleteInvite(code string) error {
	return r.delete("/invites/" + code)
}

// ---------------------------------------------------------------------------
// Threads
// ---------------------------------------------------------------------------

// CreateThreadFromMessage creates a thread from an existing message.
// Accepted keys: name, auto_archive_duration, rate_limit_per_user.
func (r *RestClient) CreateThreadFromMessage(channelID, messageID string, data map[string]interface{}) (*Channel, error) {
	var ch Channel
	if err := r.post("/channels/"+channelID+"/messages/"+messageID+"/threads", data, &ch); err != nil {
		return nil, err
	}
	return &ch, nil
}

// CreateThreadWithoutMessage creates a thread not attached to any message.
// Accepted keys: name, auto_archive_duration, type, invitable, rate_limit_per_user.
func (r *RestClient) CreateThreadWithoutMessage(channelID string, data map[string]interface{}) (*Channel, error) {
	var ch Channel
	if err := r.post("/channels/"+channelID+"/threads", data, &ch); err != nil {
		return nil, err
	}
	return &ch, nil
}

// JoinThread adds the current user to a thread.
func (r *RestClient) JoinThread(channelID string) error {
	return r.put("/channels/"+channelID+"/thread-members/@me", nil)
}

// LeaveThread removes the current user from a thread.
func (r *RestClient) LeaveThread(channelID string) error {
	return r.delete("/channels/" + channelID + "/thread-members/@me")
}

// AddThreadMember adds a user to a thread.
func (r *RestClient) AddThreadMember(channelID, userID string) error {
	return r.put("/channels/"+channelID+"/thread-members/"+userID, nil)
}

// RemoveThreadMember removes a user from a thread.
func (r *RestClient) RemoveThreadMember(channelID, userID string) error {
	return r.delete("/channels/" + channelID + "/thread-members/" + userID)
}

// GetThreadMembers returns all members of a thread.
func (r *RestClient) GetThreadMembers(channelID string) ([]*ThreadMember, error) {
	var members []*ThreadMember
	if err := r.get("/channels/"+channelID+"/thread-members", &members); err != nil {
		return nil, err
	}
	return members, nil
}

// activeThreadsResponse is the shape of the active threads endpoint response.
type activeThreadsResponse struct {
	Threads []*Channel      `json:"threads"`
	Members []*ThreadMember `json:"members"`
}

// GetActiveThreads returns all active threads in a guild.
func (r *RestClient) GetActiveThreads(guildID string) ([]*Channel, error) {
	var resp activeThreadsResponse
	if err := r.get("/guilds/"+guildID+"/threads/active", &resp); err != nil {
		return nil, err
	}
	return resp.Threads, nil
}

// ---------------------------------------------------------------------------
// Reactions - extended
// ---------------------------------------------------------------------------

// GetReactions returns users who reacted with a specific emoji on a message.
// emoji should be a unicode character (e.g. "👍") or "name:id" for custom emojis.
// Pass limit=0 to omit the limit param.
func (r *RestClient) GetReactions(channelID, messageID, emoji string, limit int) ([]*User, error) {
	path := "/channels/" + channelID + "/messages/" + messageID + "/reactions/" + url.QueryEscape(emoji)
	if limit > 0 {
		path += fmt.Sprintf("?limit=%d", limit)
	}
	var users []*User
	if err := r.get(path, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// DeleteAllReactions removes all reactions from a message.
func (r *RestClient) DeleteAllReactions(channelID, messageID string) error {
	return r.delete("/channels/" + channelID + "/messages/" + messageID + "/reactions")
}

// DeleteAllReactionsForEmoji removes all reactions of a specific emoji from a message.
// emoji should be a unicode character (e.g. "👍") or "name:id" for custom emojis.
func (r *RestClient) DeleteAllReactionsForEmoji(channelID, messageID, emoji string) error {
	return r.delete("/channels/" + channelID + "/messages/" + messageID + "/reactions/" + url.QueryEscape(emoji))
}

// ---------------------------------------------------------------------------
// Voice
// ---------------------------------------------------------------------------

// GetVoiceRegions returns available voice regions.
func (r *RestClient) GetVoiceRegions() ([]map[string]interface{}, error) {
	var regions []map[string]interface{}
	if err := r.get("/voice/regions", &regions); err != nil {
		return nil, err
	}
	return regions, nil
}

// ---------------------------------------------------------------------------
// Audit Log
// ---------------------------------------------------------------------------

// GetGuildAuditLog fetches the audit log for a guild, optionally filtered by
// action type. Pass 0 for actionType to return all action types.
// limit controls how many entries to return (1–100; 50 is the API default).
func (r *RestClient) GetGuildAuditLog(guildID string, actionType int, limit int) (*AuditLog, error) {
	params := url.Values{}
	if actionType != 0 {
		params.Set("action_type", strconv.Itoa(actionType))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	path := "/guilds/" + guildID + "/audit-logs"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	var al AuditLog
	if err := r.get(path, &al); err != nil {
		return nil, err
	}
	return &al, nil
}
