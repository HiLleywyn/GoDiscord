package discord

// guild_incidents.go - Guild Security Actions (Pause Invites / Pause DMs).
//
// Wraps Discord REST API v10's
//     PUT /guilds/{guild.id}/incident-actions
// endpoint, which sets the same "Security Actions" panel that staff operate
// from the guild's Safety Setup settings. Requires the MANAGE_GUILD
// permission. Each timer is capped at 24 hours by Discord - to keep the
// panel active indefinitely the caller must re-issue the request before the
// current window elapses.

import (
	"net/http"
	"time"
)

// IncidentsData mirrors Discord's guild "incidents_data" object. All four
// fields are RFC3339 timestamp strings (or nil) and are pointers so the
// caller can explicitly tell the difference between "leave unchanged"
// (absent) and "clear the timer" (null). DMSpamDetectedAt and
// RaidDetectedAt are read-only fields populated by Discord when it
// auto-detects activity; they are included here so the decoded response
// is not silently dropped.
type IncidentsData struct {
	InvitesDisabledUntil *string `json:"invites_disabled_until,omitempty"`
	DMsDisabledUntil     *string `json:"dms_disabled_until,omitempty"`
	DMSpamDetectedAt     *string `json:"dm_spam_detected_at,omitempty"`
	RaidDetectedAt       *string `json:"raid_detected_at,omitempty"`
}

// ModifyGuildIncidentActions toggles the guild's "Security Actions" timers
// (Pause Invites / Pause DMs). Pass a non-nil time.Time to set a timer
// until that instant; pass nil to clear the corresponding timer. Discord
// caps each timer at 24 hours; values further out are still accepted by
// the API but will be truncated to the 24-hour limit server-side.
//
// Requires MANAGE_GUILD.
func (r *RestClient) ModifyGuildIncidentActions(guildID string, invitesDisabledUntil, dmsDisabledUntil *time.Time) (*IncidentsData, error) {
	body := map[string]interface{}{
		"invites_disabled_until": nil,
		"dms_disabled_until":     nil,
	}
	if invitesDisabledUntil != nil {
		body["invites_disabled_until"] = invitesDisabledUntil.UTC().Format(time.RFC3339)
	}
	if dmsDisabledUntil != nil {
		body["dms_disabled_until"] = dmsDisabledUntil.UTC().Format(time.RFC3339)
	}
	var out IncidentsData
	if err := r.do(http.MethodPut, "/guilds/"+guildID+"/incident-actions", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
