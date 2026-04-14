package discord

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// discordEpoch is the Discord epoch in milliseconds (2015-01-01T00:00:00.000Z).
const discordEpoch int64 = 1420070400000

// avatarHashPattern matches valid Discord avatar hash strings. Animated
// avatars are prefixed with "a_" followed by hex digits; static avatars are
// plain hex strings.
var avatarHashPattern = regexp.MustCompile(`^(?:a_[A-Fa-f0-9]+|[A-Fa-f0-9]+)$`)

// SnowflakeUnix converts a Discord snowflake ID to a Unix timestamp (seconds).
// Returns 0 if the input is empty or cannot be parsed.
func SnowflakeUnix(snowflake string) int64 {
	if snowflake == "" {
		return 0
	}
	id, err := strconv.ParseUint(snowflake, 10, 64)
	if err != nil || id == 0 {
		return 0
	}
	return (int64(id>>22) + discordEpoch) / 1000
}

// IsSnowflake reports whether s looks like a Discord snowflake ID (non-empty
// all-digit string).
func IsSnowflake(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// ParseUserID extracts a raw user ID from either a <@id> / <@!id> mention or
// a plain snowflake string. Returns the input unchanged when it is not a
// mention.
func ParseUserID(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return ""
	}
	if s[0] != '<' {
		return s
	}
	// Strip <@, <@!, and trailing >
	s = strings.TrimPrefix(s, "<@!")
	s = strings.TrimPrefix(s, "<@")
	s = strings.TrimSuffix(s, ">")
	return s
}

// ParseRoleMention extracts a raw role ID from a <@&id> mention or returns s
// unchanged when it is already a plain ID.
func ParseRoleMention(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "<@&")
	s = strings.TrimSuffix(s, ">")
	return s
}

// ParseChannelMention extracts a raw channel ID from a <#id> mention or
// returns s unchanged when it is already a plain ID.
func ParseChannelMention(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "<#")
	s = strings.TrimSuffix(s, ">")
	return s
}

// FormatAge converts a duration into a human-friendly "age" string, matching
// the way Discord account ages are typically displayed in moderation tooling.
// Examples: "< 1 day old", "3 day(s) old", "2 month(s) old", "1 year(s) 6 month(s) old".
func FormatAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days < 1 {
		return "< 1 day old"
	}
	if days < 30 {
		return fmt.Sprintf("%d day(s) old", days)
	}
	months := days / 30
	if months < 12 {
		return fmt.Sprintf("%d month(s) old", months)
	}
	years := months / 12
	remaining := months % 12
	if remaining == 0 {
		return fmt.Sprintf("%d year(s) old", years)
	}
	return fmt.Sprintf("%d year(s) %d month(s) old", years, remaining)
}

// AvatarURL returns the Discord CDN URL for a user's avatar. When avatarHash
// is a valid hash the per-user avatar is returned; otherwise the coloured
// default avatar for the user's snowflake is used.
func AvatarURL(userID, avatarHash string) string {
	if avatarHash != "" && avatarHashPattern.MatchString(avatarHash) {
		return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png?size=128", userID, avatarHash)
	}
	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil || id == 0 {
		return "https://cdn.discordapp.com/embed/avatars/0.png"
	}
	return fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", (id>>22)%6)
}
