package discord

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// APIError.Error()
// ---------------------------------------------------------------------------

func TestAPIError_Error_WithCode(t *testing.T) {
	e := &APIError{
		Method:     http.MethodGet,
		Path:       "/guilds/123",
		StatusCode: http.StatusForbidden,
		Code:       ErrCodeMissingPermissions,
		Message:    "Missing Permissions",
	}
	s := e.Error()
	if !strings.Contains(s, "403") {
		t.Errorf("Error() = %q, expected to contain status code 403", s)
	}
	if !strings.Contains(s, "50013") {
		t.Errorf("Error() = %q, expected to contain Discord code 50013", s)
	}
	if !strings.Contains(s, "Missing Permissions") {
		t.Errorf("Error() = %q, expected to contain the message", s)
	}
}

func TestAPIError_Error_WithoutCode(t *testing.T) {
	e := &APIError{
		Method:     http.MethodDelete,
		Path:       "/channels/1",
		StatusCode: http.StatusInternalServerError,
		Message:    "Internal Server Error",
	}
	s := e.Error()
	if strings.Contains(s, "code") {
		t.Errorf("Error() = %q, should not contain 'code' when Code is 0", s)
	}
	if !strings.Contains(s, "500") {
		t.Errorf("Error() = %q, expected to contain status 500", s)
	}
}

// ---------------------------------------------------------------------------
// Predicate methods
// ---------------------------------------------------------------------------

func TestAPIError_Predicates(t *testing.T) {
	cases := []struct {
		status   int
		isNF     bool
		isForbid bool
		isUnauth bool
		isRL     bool
		isSrv    bool
	}{
		{http.StatusNotFound, true, false, false, false, false},
		{http.StatusForbidden, false, true, false, false, false},
		{http.StatusUnauthorized, false, false, true, false, false},
		{http.StatusTooManyRequests, false, false, false, true, false},
		{http.StatusInternalServerError, false, false, false, false, true},
		{http.StatusBadGateway, false, false, false, false, true},
		{http.StatusOK, false, false, false, false, false},
	}

	for _, tc := range cases {
		e := &APIError{StatusCode: tc.status}
		if got := e.IsNotFound(); got != tc.isNF {
			t.Errorf("status %d: IsNotFound() = %v, want %v", tc.status, got, tc.isNF)
		}
		if got := e.IsForbidden(); got != tc.isForbid {
			t.Errorf("status %d: IsForbidden() = %v, want %v", tc.status, got, tc.isForbid)
		}
		if got := e.IsUnauthorized(); got != tc.isUnauth {
			t.Errorf("status %d: IsUnauthorized() = %v, want %v", tc.status, got, tc.isUnauth)
		}
		if got := e.IsRateLimit(); got != tc.isRL {
			t.Errorf("status %d: IsRateLimit() = %v, want %v", tc.status, got, tc.isRL)
		}
		if got := e.IsServerError(); got != tc.isSrv {
			t.Errorf("status %d: IsServerError() = %v, want %v", tc.status, got, tc.isSrv)
		}
	}
}

// APIError must satisfy the error interface and work with errors.As.
func TestAPIError_ErrorsAs(t *testing.T) {
	original := &APIError{StatusCode: http.StatusNotFound, Code: ErrCodeUnknownMember}
	wrapped := errors.New("wrapper: " + original.Error())

	// errors.As on a wrapped value won't unwrap arbitrary errors, but direct
	// assignment to error works — test the common bot-code pattern.
	var target *APIError
	if !errors.As(original, &target) {
		t.Fatal("errors.As should match *APIError directly")
	}
	if target.Code != ErrCodeUnknownMember {
		t.Errorf("Code = %d, want %d", target.Code, ErrCodeUnknownMember)
	}
	// Confirm the wrapped plain error does not satisfy *APIError.
	var target2 *APIError
	if errors.As(wrapped, &target2) {
		t.Error("errors.As should NOT match a plain fmt.Errorf wrapper as *APIError")
	}
}
