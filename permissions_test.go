package discord

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ParsePermission
// ---------------------------------------------------------------------------

func TestParsePermission(t *testing.T) {
	tests := []struct {
		input   string
		want    Permission
		wantErr bool
	}{
		{"0", 0, false},
		{"", 0, false},
		{"8", PermAdministrator, false},
		{"2147483651", Permission(2147483651), false},
		{"18446744073709551615", Permission(^uint64(0)), false}, // max uint64
		{"notanumber", 0, true},
		{"-1", 0, true},
	}

	for _, tc := range tests {
		got, err := ParsePermission(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParsePermission(%q): expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParsePermission(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParsePermission(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestMustParsePermission_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParsePermission with invalid input did not panic")
		}
	}()
	MustParsePermission("bad")
}

// ---------------------------------------------------------------------------
// Permission.Has
// ---------------------------------------------------------------------------

func TestPermission_Has(t *testing.T) {
	p := PermKickMembers | PermBanMembers

	if !p.Has(PermKickMembers) {
		t.Error("Has(PermKickMembers) should be true")
	}
	if !p.Has(PermKickMembers, PermBanMembers) {
		t.Error("Has(PermKickMembers, PermBanMembers) should be true")
	}
	if p.Has(PermAdministrator) {
		t.Error("Has(PermAdministrator) should be false")
	}
	if p.Has(PermKickMembers, PermAdministrator) {
		t.Error("Has(PermKickMembers, PermAdministrator) should be false (all-or-nothing)")
	}
}

func TestPermission_Any(t *testing.T) {
	p := PermKickMembers

	if !p.Any(PermKickMembers, PermBanMembers) {
		t.Error("Any(PermKickMembers, PermBanMembers) should be true")
	}
	if p.Any(PermAdministrator, PermManageGuild) {
		t.Error("Any(PermAdministrator, PermManageGuild) should be false")
	}
}

func TestPermission_AddRemove(t *testing.T) {
	p := Permission(0).Add(PermKickMembers, PermBanMembers)
	if !p.Has(PermKickMembers, PermBanMembers) {
		t.Error("Add did not set the expected flags")
	}

	p = p.Remove(PermKickMembers)
	if p.Has(PermKickMembers) {
		t.Error("Remove did not clear PermKickMembers")
	}
	if !p.Has(PermBanMembers) {
		t.Error("Remove should leave PermBanMembers untouched")
	}
}

func TestPermission_Toggle(t *testing.T) {
	p := PermKickMembers
	p = p.Toggle(PermKickMembers) // clear it
	if p.Has(PermKickMembers) {
		t.Error("Toggle should have cleared PermKickMembers")
	}
	p = p.Toggle(PermKickMembers) // set it again
	if !p.Has(PermKickMembers) {
		t.Error("Toggle should have set PermKickMembers again")
	}
}

func TestPermission_IsAdmin(t *testing.T) {
	if !PermAdministrator.IsAdmin() {
		t.Error("PermAdministrator.IsAdmin() should be true")
	}
	if PermKickMembers.IsAdmin() {
		t.Error("PermKickMembers.IsAdmin() should be false")
	}
}

func TestPermission_String_None(t *testing.T) {
	if got := PermNone.String(); got != "none" {
		t.Errorf("PermNone.String() = %q, want %q", got, "none")
	}
}

func TestPermission_String_ContainsNames(t *testing.T) {
	p := PermKickMembers | PermBanMembers
	s := p.String()
	if !strings.Contains(s, "KickMembers") {
		t.Errorf("String() = %q, expected to contain KickMembers", s)
	}
	if !strings.Contains(s, "BanMembers") {
		t.Errorf("String() = %q, expected to contain BanMembers", s)
	}
}

func TestPermission_String_UnknownBit(t *testing.T) {
	// Bit 47 is currently unassigned; String() should label it Unknown(47).
	p := Permission(1) << 47
	s := p.String()
	if !strings.Contains(s, "Unknown(47)") {
		t.Errorf("String() = %q, expected to contain Unknown(47)", s)
	}
}
