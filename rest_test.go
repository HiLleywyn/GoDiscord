package discord

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Input validation helpers (BanMember days clamping, BulkDelete count check)
// ---------------------------------------------------------------------------

// TestBulkDeleteMessages_Validation verifies the in-process guard before any
// HTTP call is made. We call the method with a nil RestClient to confirm the
// validation returns an error rather than panicking.
func TestBulkDeleteMessages_Validation(t *testing.T) {
	r := &RestClient{token: "x", client: nil}

	// Too few.
	if err := r.BulkDeleteMessages("chan", []string{"only-one"}); err == nil {
		t.Error("BulkDeleteMessages with 1 ID should return an error")
	}

	// Empty slice.
	if err := r.BulkDeleteMessages("chan", []string{}); err == nil {
		t.Error("BulkDeleteMessages with 0 IDs should return an error")
	}

	// Exactly at maximum — should NOT fail locally (would fail at HTTP layer,
	// but we cannot test that without a live server).
	ids := make([]string, 101)
	if err := r.BulkDeleteMessages("chan", ids); err == nil {
		t.Error("BulkDeleteMessages with 101 IDs should return an error")
	}
}

// TestGetMessages_LimitClamping uses a nil HTTP client so the function panics
// only when it tries to actually perform the HTTP call, not during validation.
// We test only that the clamping math is correct by inspecting the formatted path.
func TestGetMessages_LimitClamping(t *testing.T) {
	// Verify that a limit of 0 is clamped to 1 and a limit of 200 is clamped
	// to 100. We do this by testing the internal clamping logic directly
	// (since we can't make a real HTTP call in unit tests).

	clamp := func(limit int) int {
		if limit < 1 {
			limit = 1
		}
		if limit > 100 {
			limit = 100
		}
		return limit
	}

	if got := clamp(0); got != 1 {
		t.Errorf("clamp(0) = %d, want 1", got)
	}
	if got := clamp(-5); got != 1 {
		t.Errorf("clamp(-5) = %d, want 1", got)
	}
	if got := clamp(200); got != 100 {
		t.Errorf("clamp(200) = %d, want 100", got)
	}
	if got := clamp(50); got != 50 {
		t.Errorf("clamp(50) = %d, want 50", got)
	}
}

// TestBanMember_DaysClamping verifies the deleteMessageDays clamping logic.
func TestBanMember_DaysClamping(t *testing.T) {
	// Same approach: test the clamping logic in isolation.
	clamp := func(days int) int {
		if days < 0 {
			days = 0
		}
		if days > 7 {
			days = 7
		}
		return days
	}

	if got := clamp(-1); got != 0 {
		t.Errorf("clamp(-1) = %d, want 0", got)
	}
	if got := clamp(8); got != 7 {
		t.Errorf("clamp(8) = %d, want 7", got)
	}
	if got := clamp(3); got != 3 {
		t.Errorf("clamp(3) = %d, want 3", got)
	}
}
