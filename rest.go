package discord

// rest.go — Discord REST API v10 client.
//
// Handles authentication headers, JSON encoding/decoding, and basic
// rate-limit awareness (respects Retry-After when Discord returns 429).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const apiBase = "https://discord.com/api/v10"

// RestClient is an authenticated HTTP client for the Discord REST API.
type RestClient struct {
	token  string
	client *http.Client
}

func newRestClient(token string) *RestClient {
	return &RestClient{
		token:  token,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// do performs an HTTP request and decodes the JSON response into out (may be nil).
func (r *RestClient) do(method, path string, body interface{}, out interface{}) error {
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

	// Handle rate limiting transparently.
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		secs, _ := strconv.ParseFloat(retryAfter, 64)
		if secs == 0 {
			secs = 1
		}
		time.Sleep(time.Duration(secs*1000) * time.Millisecond)
		return r.do(method, path, body, out)
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord api: %s %s → %d %s", method, path, resp.StatusCode, string(raw))
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
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
// (embeds, reply references, TTS, etc.).
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

// EditMessageEmbed replaces the embeds on a message authored by the bot.
func (r *RestClient) EditMessageEmbed(channelID, messageID string, embed Embed) (*Message, error) {
	var m Message
	if err := r.patch("/channels/"+channelID+"/messages/"+messageID,
		&MessageEdit{Embeds: []Embed{embed}}, &m); err != nil {
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
// emoji should be the unicode character (e.g. "👍") or "name:id" for custom emojis.
func (r *RestClient) AddReaction(channelID, messageID, emoji string) error {
	return r.put("/channels/"+channelID+"/messages/"+messageID+"/reactions/"+emoji+"/@me", nil)
}

// RemoveReaction removes the bot's reaction from a message.
func (r *RestClient) RemoveReaction(channelID, messageID, emoji string) error {
	return r.delete("/channels/" + channelID + "/messages/" + messageID + "/reactions/" + emoji + "/@me")
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

// SendDM sends a direct message to a user.
func (r *RestClient) SendDM(userID, content string) (*Message, error) {
	ch, err := r.CreateDM(userID)
	if err != nil {
		return nil, err
	}
	return r.SendMessage(ch.ID, content)
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

// KickMember removes a member from a guild.
func (r *RestClient) KickMember(guildID, userID string) error {
	return r.delete("/guilds/" + guildID + "/members/" + userID)
}

// BanMember bans a user from a guild.
// deleteMessageDays is the number of days of messages to delete (0–7).
func (r *RestClient) BanMember(guildID, userID string, deleteMessageDays int) error {
	return r.put("/guilds/"+guildID+"/bans/"+userID,
		map[string]int{"delete_message_days": deleteMessageDays})
}

// UnbanMember removes a ban from a guild.
func (r *RestClient) UnbanMember(guildID, userID string) error {
	return r.delete("/guilds/" + guildID + "/bans/" + userID)
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
// Pass a zero-value time.Time (or nil) to remove the timeout.
// until must be an RFC3339 timestamp string, e.g. time.Now().Add(5*time.Minute).UTC().Format(time.RFC3339).
// Pass an empty string to remove the timeout.
func (r *RestClient) TimeoutMember(guildID, userID, until string) error {
	var val interface{}
	if until == "" {
		val = nil
	} else {
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

// GetMessages fetches up to `limit` (max 100) recent messages from a channel.
func (r *RestClient) GetMessages(channelID string, limit int) ([]Message, error) {
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
func (r *RestClient) BulkDeleteMessages(channelID string, messageIDs []string) error {
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
func (r *RestClient) EditChannelPermissions(channelID, overwriteID string, allow, deny string, typ int) error {
	return r.put("/channels/"+channelID+"/permissions/"+overwriteID,
		map[string]interface{}{"allow": allow, "deny": deny, "type": typ})
}

// DeleteChannelPermission removes a permission overwrite from a channel.
func (r *RestClient) DeleteChannelPermission(channelID, overwriteID string) error {
	return r.delete("/channels/" + channelID + "/permissions/" + overwriteID)
}
