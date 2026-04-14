package discord

// permissions.go — Discord permission bitflags.
//
// Discord permissions are stored as a 64-bit integer where each bit controls a
// specific capability. Use the Permission type and its constants to build,
// check, and manipulate permission sets without magic numbers.
//
// Quick usage:
//
//	// Discord delivers member permissions as a base-10 string. Use
//	// ParsePermission to convert it to a Permission bitfield safely.
//	perms, err := discord.ParsePermission(member.Permissions)
//	if err != nil {
//	    // member.Permissions was malformed (not a base-10 uint64).
//	}
//	if perms.Has(discord.PermSendMessages, discord.PermEmbedLinks) {
//	    // ...
//	}
//
//	// Build a permission set from scratch.
//	modPerms := discord.Permission(0).Add(
//	    discord.PermKickMembers,
//	    discord.PermBanMembers,
//	    discord.PermManageMessages,
//	)

import (
	"fmt"
	"strconv"
	"strings"
)

// Permission is a 64-bit bitfield representing a set of Discord permissions.
type Permission uint64

// ---------------------------------------------------------------------------
// Permission constants — all 53 flags defined in the Discord documentation.
// https://discord.com/developers/docs/topics/permissions#permissions-bitwise-permission-flags
// ---------------------------------------------------------------------------

const (
	// PermCreateInstantInvite allows creating guild/channel invites.
	PermCreateInstantInvite Permission = 1 << 0

	// PermKickMembers allows kicking members from a guild.
	PermKickMembers Permission = 1 << 1

	// PermBanMembers allows banning and unbanning members from a guild.
	PermBanMembers Permission = 1 << 2

	// PermAdministrator grants all permissions and bypasses channel overwrites.
	// Use with caution — this is effectively a super-admin flag.
	PermAdministrator Permission = 1 << 3

	// PermManageChannels allows creating, editing, and deleting channels.
	PermManageChannels Permission = 1 << 4

	// PermManageGuild allows editing guild settings.
	PermManageGuild Permission = 1 << 5

	// PermAddReactions allows adding reactions to messages.
	PermAddReactions Permission = 1 << 6

	// PermViewAuditLog allows viewing the guild audit log.
	PermViewAuditLog Permission = 1 << 7

	// PermPrioritySpeaker allows using priority speaker in voice channels.
	PermPrioritySpeaker Permission = 1 << 8

	// PermStream allows going live (video streaming) in voice channels.
	PermStream Permission = 1 << 9

	// PermViewChannel allows reading messages in text channels and seeing
	// voice channels.
	PermViewChannel Permission = 1 << 10

	// PermSendMessages allows sending messages in text channels and threads.
	PermSendMessages Permission = 1 << 11

	// PermSendTTSMessages allows sending text-to-speech messages.
	PermSendTTSMessages Permission = 1 << 12

	// PermManageMessages allows deleting others' messages and pinning messages.
	PermManageMessages Permission = 1 << 13

	// PermEmbedLinks allows links posted by members to display as embeds.
	PermEmbedLinks Permission = 1 << 14

	// PermAttachFiles allows uploading files.
	PermAttachFiles Permission = 1 << 15

	// PermReadMessageHistory allows reading message history in channels.
	PermReadMessageHistory Permission = 1 << 16

	// PermMentionEveryone allows mentioning @everyone, @here, and all roles.
	PermMentionEveryone Permission = 1 << 17

	// PermUseExternalEmojis allows using emojis from other guilds.
	PermUseExternalEmojis Permission = 1 << 18

	// PermViewGuildInsights allows viewing guild analytics/insights.
	PermViewGuildInsights Permission = 1 << 19

	// PermConnect allows connecting to voice channels.
	PermConnect Permission = 1 << 20

	// PermSpeak allows speaking in voice channels.
	PermSpeak Permission = 1 << 21

	// PermMuteMembers allows muting members in voice channels.
	PermMuteMembers Permission = 1 << 22

	// PermDeafenMembers allows deafening members in voice channels.
	PermDeafenMembers Permission = 1 << 23

	// PermMoveMembers allows moving members between voice channels.
	PermMoveMembers Permission = 1 << 24

	// PermUseVAD allows using voice-activity detection in voice channels.
	PermUseVAD Permission = 1 << 25

	// PermChangeNickname allows members to change their own nickname.
	PermChangeNickname Permission = 1 << 26

	// PermManageNicknames allows changing other members' nicknames.
	PermManageNicknames Permission = 1 << 27

	// PermManageRoles allows creating, editing, and deleting roles lower than
	// the manager's highest role.
	PermManageRoles Permission = 1 << 28

	// PermManageWebhooks allows creating, editing, and deleting webhooks.
	PermManageWebhooks Permission = 1 << 29

	// PermManageGuildExpressions allows managing guild emojis, stickers, and
	// soundboard sounds.
	PermManageGuildExpressions Permission = 1 << 30

	// PermUseApplicationCommands allows using slash commands and other app
	// commands in a guild.
	PermUseApplicationCommands Permission = 1 << 31

	// PermRequestToSpeak allows requesting to speak in stage channels.
	PermRequestToSpeak Permission = 1 << 32

	// PermManageEvents allows creating, editing, and deleting scheduled events.
	PermManageEvents Permission = 1 << 33

	// PermManageThreads allows deleting and archiving threads, and viewing all
	// private threads.
	PermManageThreads Permission = 1 << 34

	// PermCreatePublicThreads allows creating public and announcement threads.
	PermCreatePublicThreads Permission = 1 << 35

	// PermCreatePrivateThreads allows creating private threads.
	PermCreatePrivateThreads Permission = 1 << 36

	// PermUseExternalStickers allows using stickers from other guilds.
	PermUseExternalStickers Permission = 1 << 37

	// PermSendMessagesInThreads allows sending messages in threads.
	PermSendMessagesInThreads Permission = 1 << 38

	// PermUseEmbeddedActivities allows launching activities (games) in voice
	// channels.
	PermUseEmbeddedActivities Permission = 1 << 39

	// PermModerateMembers allows timing out (Discord-muting) members. This is
	// the permission required for the Discord timeout feature.
	PermModerateMembers Permission = 1 << 40

	// PermViewCreatorMonetizationAnalytics allows viewing role subscription
	// insights.
	PermViewCreatorMonetizationAnalytics Permission = 1 << 41

	// PermUseSoundboard allows using the guild soundboard.
	PermUseSoundboard Permission = 1 << 42

	// PermCreateGuildExpressions allows creating guild emojis, stickers, and
	// soundboard sounds (distinct from managing them).
	PermCreateGuildExpressions Permission = 1 << 43

	// PermCreateEvents allows creating scheduled events (distinct from
	// managing them).
	PermCreateEvents Permission = 1 << 44

	// PermUseExternalSounds allows using sounds from other guilds.
	PermUseExternalSounds Permission = 1 << 45

	// PermSendVoiceMessages allows sending voice messages.
	PermSendVoiceMessages Permission = 1 << 46

	// PermSendPolls allows creating polls.
	PermSendPolls Permission = 1 << 49

	// PermUseExternalApps allows members to interact with apps from other
	// guilds.
	PermUseExternalApps Permission = 1 << 50
)

// ---------------------------------------------------------------------------
// Composite permission sets — common roles assembled from the flags above
// ---------------------------------------------------------------------------

// PermNone is the zero permission — no capabilities granted.
const PermNone Permission = 0

// PermAll is a convenience constant that represents every permission bit set.
// Equivalent to Administrator for practical purposes.
const PermAll Permission = ^Permission(0)

// PermDefaultText is a reasonable permission set for regular members in a
// text channel: view, send, react, read history, and embed links.
const PermDefaultText = PermViewChannel |
	PermSendMessages |
	PermAddReactions |
	PermReadMessageHistory |
	PermEmbedLinks |
	PermAttachFiles |
	PermUseExternalEmojis |
	PermUseApplicationCommands

// PermModerator is a typical moderator permission set.
const PermModerator = PermDefaultText |
	PermKickMembers |
	PermBanMembers |
	PermManageMessages |
	PermViewAuditLog |
	PermModerateMembers |
	PermManageNicknames |
	PermMuteMembers |
	PermDeafenMembers |
	PermMoveMembers

// ---------------------------------------------------------------------------
// Methods
// ---------------------------------------------------------------------------

// ParsePermission parses the decimal string that Discord sends for member and
// role permissions (e.g. "2147483651") and returns the corresponding
// Permission value. It is more robust than fmt.Sscanf and returns a clear
// error on invalid input.
//
//	perms, err := discord.ParsePermission(member.Permissions)
//	if err == nil && perms.Has(discord.PermBanMembers) { … }
func ParsePermission(s string) (Permission, error) {
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("discord: invalid permission string %q: %w", s, err)
	}
	return Permission(v), nil
}

// MustParsePermission is like ParsePermission but panics on error.
// Use only in initialisation contexts where the input is a compile-time constant.
func MustParsePermission(s string) Permission {
	p, err := ParsePermission(s)
	if err != nil {
		panic(err)
	}
	return p
}

// Has reports whether p contains all of the provided flags.
// Returns true only if every flag in flags is set in p.
//
//	perms.Has(PermKickMembers, PermBanMembers) // true if both bits are set
func (p Permission) Has(flags ...Permission) bool {
	for _, f := range flags {
		if p&f != f {
			return false
		}
	}
	return true
}

// Any reports whether p contains at least one of the provided flags.
func (p Permission) Any(flags ...Permission) bool {
	for _, f := range flags {
		if p&f != 0 {
			return true
		}
	}
	return false
}

// Add returns a new Permission with all provided flags set.
func (p Permission) Add(flags ...Permission) Permission {
	for _, f := range flags {
		p |= f
	}
	return p
}

// Remove returns a new Permission with all provided flags cleared.
func (p Permission) Remove(flags ...Permission) Permission {
	for _, f := range flags {
		p &^= f
	}
	return p
}

// Toggle returns a new Permission with the provided flags flipped.
func (p Permission) Toggle(flags ...Permission) Permission {
	for _, f := range flags {
		p ^= f
	}
	return p
}

// IsAdmin reports whether the Administrator bit is set.
// Bots with this permission bypass all channel permission overrides.
func (p Permission) IsAdmin() bool {
	return p.Has(PermAdministrator)
}

// PermissionName returns a short human-readable label for a single permission
// bit. These names match the labels shown in the Discord UI. For composite or
// unknown values the empty string is returned.
func PermissionName(perm Permission) string {
	names := map[Permission]string{
		PermAdministrator:          "Administrator",
		PermManageGuild:            "Manage Server",
		PermManageRoles:            "Manage Roles",
		PermManageChannels:         "Manage Channels",
		PermBanMembers:             "Ban Members",
		PermKickMembers:            "Kick Members",
		PermManageWebhooks:         "Manage Webhooks",
		PermManageGuildExpressions: "Manage Expressions",
		PermManageMessages:         "Manage Messages",
		PermMentionEveryone:        "Mention Everyone",
		PermModerateMembers:        "Timeout Members",
		PermManageNicknames:        "Manage Nicknames",
		PermViewAuditLog:           "View Audit Log",
		PermCreateInstantInvite:    "Create Invite",
		PermSendMessages:           "Send Messages",
		PermViewChannel:            "View Channel",
		PermReadMessageHistory:     "Read Message History",
		PermEmbedLinks:             "Embed Links",
		PermAttachFiles:            "Attach Files",
		PermAddReactions:           "Add Reactions",
		PermConnect:                "Connect",
		PermSpeak:                  "Speak",
		PermMuteMembers:            "Mute Members",
		PermDeafenMembers:          "Deafen Members",
		PermMoveMembers:            "Move Members",
		PermManageThreads:          "Manage Threads",
		PermManageEvents:           "Manage Events",
		PermChangeNickname:         "Change Nickname",
	}
	return names[perm]
}

// String returns a human-readable list of permission names.
// Useful for debugging; not suitable for display to end users.
func (p Permission) String() string {
	if p == 0 {
		return "none"
	}

	names := map[Permission]string{
		PermCreateInstantInvite:              "CreateInstantInvite",
		PermKickMembers:                      "KickMembers",
		PermBanMembers:                       "BanMembers",
		PermAdministrator:                    "Administrator",
		PermManageChannels:                   "ManageChannels",
		PermManageGuild:                      "ManageGuild",
		PermAddReactions:                     "AddReactions",
		PermViewAuditLog:                     "ViewAuditLog",
		PermPrioritySpeaker:                  "PrioritySpeaker",
		PermStream:                           "Stream",
		PermViewChannel:                      "ViewChannel",
		PermSendMessages:                     "SendMessages",
		PermSendTTSMessages:                  "SendTTSMessages",
		PermManageMessages:                   "ManageMessages",
		PermEmbedLinks:                       "EmbedLinks",
		PermAttachFiles:                      "AttachFiles",
		PermReadMessageHistory:               "ReadMessageHistory",
		PermMentionEveryone:                  "MentionEveryone",
		PermUseExternalEmojis:                "UseExternalEmojis",
		PermViewGuildInsights:                "ViewGuildInsights",
		PermConnect:                          "Connect",
		PermSpeak:                            "Speak",
		PermMuteMembers:                      "MuteMembers",
		PermDeafenMembers:                    "DeafenMembers",
		PermMoveMembers:                      "MoveMembers",
		PermUseVAD:                           "UseVAD",
		PermChangeNickname:                   "ChangeNickname",
		PermManageNicknames:                  "ManageNicknames",
		PermManageRoles:                      "ManageRoles",
		PermManageWebhooks:                   "ManageWebhooks",
		PermManageGuildExpressions:           "ManageGuildExpressions",
		PermUseApplicationCommands:           "UseApplicationCommands",
		PermRequestToSpeak:                   "RequestToSpeak",
		PermManageEvents:                     "ManageEvents",
		PermManageThreads:                    "ManageThreads",
		PermCreatePublicThreads:              "CreatePublicThreads",
		PermCreatePrivateThreads:             "CreatePrivateThreads",
		PermUseExternalStickers:              "UseExternalStickers",
		PermSendMessagesInThreads:            "SendMessagesInThreads",
		PermUseEmbeddedActivities:            "UseEmbeddedActivities",
		PermModerateMembers:                  "ModerateMembers",
		PermViewCreatorMonetizationAnalytics: "ViewCreatorMonetizationAnalytics",
		PermUseSoundboard:                    "UseSoundboard",
		PermCreateGuildExpressions:           "CreateGuildExpressions",
		PermCreateEvents:                     "CreateEvents",
		PermUseExternalSounds:                "UseExternalSounds",
		PermSendVoiceMessages:                "SendVoiceMessages",
		PermSendPolls:                        "SendPolls",
		PermUseExternalApps:                  "UseExternalApps",
	}

	var parts []string
	for bit := 0; bit < 64; bit++ {
		flag := Permission(1) << bit
		if p&flag != 0 {
			if name, ok := names[flag]; ok {
				parts = append(parts, name)
			} else {
				parts = append(parts, fmt.Sprintf("Unknown(%d)", bit))
			}
		}
	}
	return strings.Join(parts, "|")
}

// ---------------------------------------------------------------------------
// Permission* aliases — convenience names used by plugins.
//
// These are identical in value to the Perm* constants above; they exist so
// plugin authors can choose the style that reads most naturally to them
// (e.g. discord.PermissionManageMessages vs. discord.PermManageMessages).
// ---------------------------------------------------------------------------

const (
	PermissionCreateInstantInvite   = PermCreateInstantInvite
	PermissionKickMembers           = PermKickMembers
	PermissionBanMembers            = PermBanMembers
	PermissionAdministrator         = PermAdministrator
	PermissionManageChannels        = PermManageChannels
	PermissionManageGuild           = PermManageGuild
	PermissionAddReactions          = PermAddReactions
	PermissionViewAuditLog          = PermViewAuditLog
	PermissionPrioritySpeaker       = PermPrioritySpeaker
	PermissionStream                = PermStream
	PermissionViewChannel           = PermViewChannel
	PermissionSendMessages          = PermSendMessages
	PermissionSendTTSMessages       = PermSendTTSMessages
	PermissionManageMessages        = PermManageMessages
	PermissionEmbedLinks            = PermEmbedLinks
	PermissionAttachFiles           = PermAttachFiles
	PermissionReadMessageHistory    = PermReadMessageHistory
	PermissionMentionEveryone       = PermMentionEveryone
	PermissionUseExternalEmojis     = PermUseExternalEmojis
	PermissionViewGuildInsights     = PermViewGuildInsights
	PermissionConnect               = PermConnect
	PermissionSpeak                 = PermSpeak
	PermissionMuteMembers           = PermMuteMembers
	PermissionDeafenMembers         = PermDeafenMembers
	PermissionMoveMembers           = PermMoveMembers
	PermissionUseVAD                = PermUseVAD
	PermissionChangeNickname        = PermChangeNickname
	PermissionManageNicknames       = PermManageNicknames
	PermissionManageRoles           = PermManageRoles
	PermissionManageWebhooks        = PermManageWebhooks
	PermissionManageGuildExpressions = PermManageGuildExpressions
	PermissionUseApplicationCommands = PermUseApplicationCommands
	PermissionRequestToSpeak        = PermRequestToSpeak
	PermissionManageEvents          = PermManageEvents
	PermissionManageThreads         = PermManageThreads
	PermissionCreatePublicThreads   = PermCreatePublicThreads
	PermissionCreatePrivateThreads  = PermCreatePrivateThreads
	PermissionUseExternalStickers   = PermUseExternalStickers
	PermissionSendMessagesInThreads = PermSendMessagesInThreads
	PermissionUseEmbeddedActivities = PermUseEmbeddedActivities
	PermissionModerateMembers       = PermModerateMembers
	PermissionSendVoiceMessages     = PermSendVoiceMessages
	PermissionSendPolls             = PermSendPolls
	PermissionUseExternalApps       = PermUseExternalApps
)

// HasPermission reports whether the member holds the given permission bit.
//
// It parses the member's Permissions string (the decimal integer Discord
// includes on members received via gateway events) and checks with Has().
// Administrator bypasses all other permission checks per Discord's rules.
// Returns false on parse errors or when the member has no permissions set.
//
//	if ctx.Member.HasPermission(discord.PermissionManageMessages) { … }
func (m *Member) HasPermission(perm Permission) bool {
	if m == nil {
		return false
	}
	p, err := ParsePermission(m.Permissions)
	if err != nil {
		return false
	}
	// Administrator grants everything.
	if p.Has(PermAdministrator) {
		return true
	}
	return p.Has(perm)
}
