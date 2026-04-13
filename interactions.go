package discord

// interactions.go — Discord Interactions v2: slash commands + message components.
//
// Adds typed structs, handler registration, and REST methods for:
//   - Application commands (slash commands)
//   - Message components (select menus, buttons)
//   - Interaction callbacks and follow-ups

import "net/http"

// ---------------------------------------------------------------------------
// Interaction constants
// ---------------------------------------------------------------------------

// InteractionType identifies the kind of interaction Discord delivered.
type InteractionType int

const (
	InteractionTypePing                           InteractionType = 1
	InteractionTypeApplicationCommand             InteractionType = 2
	InteractionTypeMessageComponent               InteractionType = 3
	InteractionTypeApplicationCommandAutocomplete InteractionType = 4
	InteractionTypeModalSubmit                    InteractionType = 5
)

// ApplicationCommandType is the subtype of a slash/context-menu command.
type ApplicationCommandType int

const (
	ApplicationCommandTypeChatInput ApplicationCommandType = 1
	ApplicationCommandTypeUser      ApplicationCommandType = 2
	ApplicationCommandTypeMessage   ApplicationCommandType = 3
)

// ApplicationCommandOptionType enumerates the types an option may have.
type ApplicationCommandOptionType int

const (
	OptionTypeSubCommand      ApplicationCommandOptionType = 1
	OptionTypeSubCommandGroup ApplicationCommandOptionType = 2
	OptionTypeString          ApplicationCommandOptionType = 3
	OptionTypeInteger         ApplicationCommandOptionType = 4
	OptionTypeBoolean         ApplicationCommandOptionType = 5
	OptionTypeUser            ApplicationCommandOptionType = 6
	OptionTypeChannel         ApplicationCommandOptionType = 7
	OptionTypeRole            ApplicationCommandOptionType = 8
	OptionTypeMentionable     ApplicationCommandOptionType = 9
	OptionTypeNumber          ApplicationCommandOptionType = 10
	OptionTypeAttachment      ApplicationCommandOptionType = 11
)

// InteractionCallbackType is the type field in an interaction response.
type InteractionCallbackType int

const (
	// InteractionCallbackTypePong responds to a Ping.
	InteractionCallbackTypePong InteractionCallbackType = 1
	// InteractionCallbackTypeChannelMessage sends a new message.
	InteractionCallbackTypeChannelMessage InteractionCallbackType = 4
	// InteractionCallbackTypeDeferred defers the response (shows loading state).
	InteractionCallbackTypeDeferred InteractionCallbackType = 5
	// InteractionCallbackTypeDeferredUpdate defers a component update.
	InteractionCallbackTypeDeferredUpdate InteractionCallbackType = 6
	// InteractionCallbackTypeUpdateMessage edits the message a component is attached to.
	InteractionCallbackTypeUpdateMessage InteractionCallbackType = 7
)

// Component type constants.
const (
	ComponentTypeActionRow    = 1
	ComponentTypeButton       = 2
	ComponentTypeStringSelect = 3
	ComponentTypeTextInput    = 4
)

// Button style constants.
const (
	ButtonStylePrimary   = 1
	ButtonStyleSecondary = 2
	ButtonStyleSuccess   = 3
	ButtonStyleDanger    = 4
	ButtonStyleLink      = 5
)

// MessageFlagEphemeral marks a response as visible only to the invoking user.
const MessageFlagEphemeral = 1 << 6 // 64

// ---------------------------------------------------------------------------
// Application command structs
// ---------------------------------------------------------------------------

// ApplicationCommandOptionChoice is a predefined choice value for an option.
type ApplicationCommandOptionChoice struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// ApplicationCommandOption describes one parameter of a slash command.
type ApplicationCommandOption struct {
	Type        ApplicationCommandOptionType      `json:"type"`
	Name        string                            `json:"name"`
	Description string                            `json:"description"`
	Required    bool                              `json:"required,omitempty"`
	Choices     []ApplicationCommandOptionChoice  `json:"choices,omitempty"`
	Options     []ApplicationCommandOption        `json:"options,omitempty"`
}

// ApplicationCommand is the top-level structure for a Discord slash command.
type ApplicationCommand struct {
	ID                       Snowflake                  `json:"id,omitempty"`
	ApplicationID            Snowflake                  `json:"application_id,omitempty"`
	GuildID                  Snowflake                  `json:"guild_id,omitempty"`
	Type                     ApplicationCommandType     `json:"type,omitempty"`
	Name                     string                     `json:"name"`
	Description              string                     `json:"description,omitempty"`
	Options                  []ApplicationCommandOption `json:"options,omitempty"`
	DefaultMemberPermissions *string                    `json:"default_member_permissions,omitempty"`
	DMPermission             *bool                      `json:"dm_permission,omitempty"`
}

// ---------------------------------------------------------------------------
// Component structs
// ---------------------------------------------------------------------------

// SelectMenuOption is a single entry in a string select menu.
type SelectMenuOption struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Emoji       *Emoji `json:"emoji,omitempty"`
	Default     bool   `json:"default,omitempty"`
}

// Component is a generic Discord UI component.
// Use the ComponentType* constants to populate the Type field.
type Component struct {
	Type int `json:"type"`

	// ActionRow children.
	Components []Component `json:"components,omitempty"`

	// Button fields.
	Style    int    `json:"style,omitempty"`
	Label    string `json:"label,omitempty"`
	Emoji    *Emoji `json:"emoji,omitempty"`
	CustomID string `json:"custom_id,omitempty"`
	URL      string `json:"url,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`

	// Select menu fields.
	Placeholder string             `json:"placeholder,omitempty"`
	MinValues   *int               `json:"min_values,omitempty"`
	MaxValues   *int               `json:"max_values,omitempty"`
	Options     []SelectMenuOption `json:"options,omitempty"`
}

// ActionRow wraps a slice of components in a single ActionRow component.
func ActionRow(children ...Component) Component {
	return Component{Type: ComponentTypeActionRow, Components: children}
}

// StringSelect builds a string-select component.
func StringSelect(customID, placeholder string, options []SelectMenuOption) Component {
	return Component{
		Type:        ComponentTypeStringSelect,
		CustomID:    customID,
		Placeholder: placeholder,
		Options:     options,
	}
}

// ---------------------------------------------------------------------------
// Interaction payload structs
// ---------------------------------------------------------------------------

// InteractionOption is a resolved option from a slash-command invocation.
type InteractionOption struct {
	Name    string                       `json:"name"`
	Type    ApplicationCommandOptionType `json:"type"`
	Value   interface{}                  `json:"value"`
	Options []InteractionOption          `json:"options"`
}

// InteractionData carries the command-specific or component-specific payload.
type InteractionData struct {
	// Application command fields.
	ID      Snowflake              `json:"id"`
	Name    string                 `json:"name"`
	Type    ApplicationCommandType `json:"type"`
	Options []InteractionOption    `json:"options"`

	// Message component fields.
	ComponentType int      `json:"component_type"`
	CustomID      string   `json:"custom_id"`
	Values        []string `json:"values"` // selected values for select menus
}

// Interaction is the full Discord Interaction event object.
type Interaction struct {
	ID            Snowflake        `json:"id"`
	ApplicationID Snowflake        `json:"application_id"`
	Type          InteractionType  `json:"type"`
	Data          *InteractionData `json:"data"`
	GuildID       Snowflake        `json:"guild_id"`
	ChannelID     Snowflake        `json:"channel_id"`
	Member        *Member          `json:"member"`
	User          *User            `json:"user"`
	Token         string           `json:"token"`
	Version       int              `json:"version"`
	Message       *Message         `json:"message"`
}

// Author returns the user who triggered this interaction (works in both guild and DM context).
func (i *Interaction) Author() *User {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User
	}
	return i.User
}

// ---------------------------------------------------------------------------
// Interaction response structs
// ---------------------------------------------------------------------------

// InteractionResponseData is the data payload of an interaction callback.
type InteractionResponseData struct {
	TTS        bool        `json:"tts,omitempty"`
	Content    string      `json:"content,omitempty"`
	Embeds     []Embed     `json:"embeds,omitempty"`
	Flags      int         `json:"flags,omitempty"`
	Components []Component `json:"components,omitempty"`
}

// InteractionResponse is the full payload sent to Discord's callback endpoint.
type InteractionResponse struct {
	Type InteractionCallbackType  `json:"type"`
	Data *InteractionResponseData `json:"data,omitempty"`
}

// ---------------------------------------------------------------------------
// REST — application commands
// ---------------------------------------------------------------------------

// BulkOverwriteGuildCommands atomically replaces all guild-scoped commands.
// Pass an empty slice to remove all guild commands.
func (r *RestClient) BulkOverwriteGuildCommands(appID, guildID string, cmds []ApplicationCommand) ([]ApplicationCommand, error) {
	var result []ApplicationCommand
	if err := r.do(http.MethodPut, "/applications/"+appID+"/guilds/"+guildID+"/commands", cmds, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetGuildCommands returns all guild-scoped application commands.
func (r *RestClient) GetGuildCommands(appID, guildID string) ([]ApplicationCommand, error) {
	var result []ApplicationCommand
	if err := r.get("/applications/"+appID+"/guilds/"+guildID+"/commands", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateGuildCommand registers a single guild-scoped application command.
func (r *RestClient) CreateGuildCommand(appID, guildID string, cmd ApplicationCommand) (*ApplicationCommand, error) {
	var result ApplicationCommand
	if err := r.post("/applications/"+appID+"/guilds/"+guildID+"/commands", cmd, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ---------------------------------------------------------------------------
// REST — interaction callbacks
// ---------------------------------------------------------------------------

// CreateInteractionResponse responds to an interaction.
// Must be called within 3 seconds of receiving the interaction token.
func (r *RestClient) CreateInteractionResponse(interactionID, token string, resp InteractionResponse) error {
	return r.post("/interactions/"+interactionID+"/"+token+"/callback", resp, nil)
}

// EditInteractionResponse edits the original response sent to an interaction.
func (r *RestClient) EditInteractionResponse(appID, token string, data InteractionResponseData) error {
	return r.patch("/webhooks/"+appID+"/"+token+"/messages/@original", data, nil)
}

// DeleteInteractionResponse deletes the original interaction response.
func (r *RestClient) DeleteInteractionResponse(appID, token string) error {
	return r.delete("/webhooks/" + appID + "/" + token + "/messages/@original")
}
