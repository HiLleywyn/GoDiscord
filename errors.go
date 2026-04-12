package discord

// errors.go — structured error types for GoDiscord.
//
// REST API calls return *APIError instead of a raw error string so callers can
// inspect the HTTP status code and Discord error code programmatically:
//
//	_, err := bot.Rest.GetGuildMember(guildID, userID)
//	var apiErr *discord.APIError
//	if errors.As(err, &apiErr) && apiErr.IsNotFound() {
//	    // user not in guild
//	}

import (
	"fmt"
	"net/http"
)

// ---------------------------------------------------------------------------
// APIError
// ---------------------------------------------------------------------------

// APIError represents a failed Discord REST API call. It implements the error
// interface and carries structured metadata so callers can branch on specific
// HTTP statuses or Discord JSON error codes without string parsing.
type APIError struct {
	// Method is the HTTP verb used (GET, POST, PATCH, …).
	Method string

	// Path is the request path, e.g. "/guilds/123/members/456".
	Path string

	// StatusCode is the HTTP response status code.
	StatusCode int

	// Code is the Discord JSON error code from the response body, if present.
	// See https://discord.com/developers/docs/topics/opcodes-and-status-codes#json
	// Zero means Discord did not include a JSON error body.
	Code int

	// Message is the human-readable error description from Discord, or the
	// raw response body if the response was not valid JSON.
	Message string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("discord: %s %s — HTTP %d, code %d: %s",
			e.Method, e.Path, e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("discord: %s %s — HTTP %d: %s",
		e.Method, e.Path, e.StatusCode, e.Message)
}

// ---------------------------------------------------------------------------
// Convenience predicates
// ---------------------------------------------------------------------------

// IsNotFound reports whether the error is a 404 Not Found response.
// Common causes: unknown guild/channel/user ID, or the bot cannot see the
// resource due to missing permissions.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsForbidden reports whether the error is a 403 Forbidden response.
// The bot is authenticated but lacks the required permission.
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == http.StatusForbidden
}

// IsUnauthorized reports whether the error is a 401 Unauthorized response.
// Usually caused by an invalid or revoked bot token.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsRateLimit reports whether the error is a 429 Too Many Requests response.
// GoDiscord respects Retry-After headers automatically; you should only see
// this error in edge cases where automatic retry is not possible.
func (e *APIError) IsRateLimit() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// IsServerError reports whether the error originated on Discord's side
// (5xx status codes). Retrying after a short back-off is usually appropriate.
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500
}

// ---------------------------------------------------------------------------
// Common Discord JSON error codes
// https://discord.com/developers/docs/topics/opcodes-and-status-codes#json-json-error-codes
// ---------------------------------------------------------------------------

const (
	// ErrCodeUnknownAccount — 10001
	ErrCodeUnknownAccount = 10001

	// ErrCodeUnknownApplication — 10002
	ErrCodeUnknownApplication = 10002

	// ErrCodeUnknownChannel — 10003
	ErrCodeUnknownChannel = 10003

	// ErrCodeUnknownGuild — 10004
	ErrCodeUnknownGuild = 10004

	// ErrCodeUnknownIntegration — 10005
	ErrCodeUnknownIntegration = 10005

	// ErrCodeUnknownInvite — 10006
	ErrCodeUnknownInvite = 10006

	// ErrCodeUnknownMember — 10007
	ErrCodeUnknownMember = 10007

	// ErrCodeUnknownMessage — 10008
	ErrCodeUnknownMessage = 10008

	// ErrCodeUnknownOverwrite — 10009
	ErrCodeUnknownOverwrite = 10009

	// ErrCodeUnknownProvider — 10010
	ErrCodeUnknownProvider = 10010

	// ErrCodeUnknownRole — 10011
	ErrCodeUnknownRole = 10011

	// ErrCodeUnknownToken — 10012
	ErrCodeUnknownToken = 10012

	// ErrCodeUnknownUser — 10013
	ErrCodeUnknownUser = 10013

	// ErrCodeUnknownEmoji — 10014
	ErrCodeUnknownEmoji = 10014

	// ErrCodeUnknownWebhook — 10015
	ErrCodeUnknownWebhook = 10015

	// ErrCodeUnknownBan — 10026
	ErrCodeUnknownBan = 10026

	// ErrCodeUnknownSKU — 10027
	ErrCodeUnknownSKU = 10027

	// ErrCodeUnknownStoreListing — 10028
	ErrCodeUnknownStoreListing = 10028

	// ErrCodeUnknownEntitlement — 10029
	ErrCodeUnknownEntitlement = 10029

	// ErrCodeUnknownBuild — 10030
	ErrCodeUnknownBuild = 10030

	// ErrCodeUnknownLobby — 10031
	ErrCodeUnknownLobby = 10031

	// ErrCodeUnknownBranch — 10032
	ErrCodeUnknownBranch = 10032

	// ErrCodeUnknownApplicationCommand — 10063
	ErrCodeUnknownApplicationCommand = 10063

	// ErrCodeBotsCannotUseEndpoint — 20001
	ErrCodeBotsCannotUseEndpoint = 20001

	// ErrCodeOnlyBotsCanUseEndpoint — 20002
	ErrCodeOnlyBotsCanUseEndpoint = 20002

	// ErrCodeCannotSendToUser — 50007
	ErrCodeCannotSendToUser = 50007

	// ErrCodeCannotSendInVoice — 50008
	ErrCodeCannotSendInVoice = 50008

	// ErrCodeMissingAccess — 50001
	ErrCodeMissingAccess = 50001

	// ErrCodeInvalidAccountType — 50002
	ErrCodeInvalidAccountType = 50002

	// ErrCodeCannotExecuteOnDM — 50003
	ErrCodeCannotExecuteOnDM = 50003

	// ErrCodeMissingPermissions — 50013
	ErrCodeMissingPermissions = 50013

	// ErrCodeInvalidToken — 50014
	ErrCodeInvalidToken = 50014

	// ErrCodeBulkDeleteTooFew — 50016
	ErrCodeBulkDeleteTooFew = 50016

	// ErrCodeBulkDeleteTooMany — 50034
	ErrCodeBulkDeleteTooMany = 50034

	// ErrCodeInvalidFormBody — 50035
	ErrCodeInvalidFormBody = 50035

	// ErrCodeInteractionAlreadyAcknowledged — 40060
	ErrCodeInteractionAlreadyAcknowledged = 40060

	// ErrCodeMaxGuilds — 30001
	ErrCodeMaxGuilds = 30001

	// ErrCodeMaxFriends — 30002
	ErrCodeMaxFriends = 30002

	// ErrCodeMaxPins — 30003
	ErrCodeMaxPins = 30003

	// ErrCodeMaxRoles — 30005
	ErrCodeMaxRoles = 30005

	// ErrCodeMaxWebhooks — 30007
	ErrCodeMaxWebhooks = 30007

	// ErrCodeMaxEmojis — 30008
	ErrCodeMaxEmojis = 30008

	// ErrCodeMaxReactions — 30010
	ErrCodeMaxReactions = 30010

	// ErrCodeMaxChannels — 30013
	ErrCodeMaxChannels = 30013

	// ErrCodeMaxInvites — 30016
	ErrCodeMaxInvites = 30016
)
