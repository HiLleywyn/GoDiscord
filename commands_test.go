package discord

import (
	"reflect"
	"testing"
)

// ---------------------------------------------------------------------------
// parseArgs
// ---------------------------------------------------------------------------

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace only",
			input: "   ",
			want:  nil,
		},
		{
			name:  "single token",
			input: "hello",
			want:  []string{"hello"},
		},
		{
			name:  "multiple tokens",
			input: "foo bar baz",
			want:  []string{"foo", "bar", "baz"},
		},
		{
			name:  "quoted string with spaces",
			input: `@user "some reason"`,
			want:  []string{"@user", "some reason"},
		},
		{
			name:  "quoted string with escaped quote",
			input: `@user "said \"hi\""`,
			want:  []string{"@user", `said "hi"`},
		},
		{
			name:  "extra whitespace between tokens",
			input: "  foo   bar  ",
			want:  []string{"foo", "bar"},
		},
		{
			name:  "tab as separator",
			input: "foo\tbar",
			want:  []string{"foo", "bar"},
		},
		{
			name:  "multiple quoted args",
			input: `"first arg" "second arg"`,
			want:  []string{"first arg", "second arg"},
		},
		{
			name:  "mixed quoted and unquoted",
			input: `user 7 "posting malware links"`,
			want:  []string{"user", "7", "posting malware links"},
		},
		{
			name:  "backslash outside quote is kept literal",
			input: `foo\bar`,
			want:  []string{`foo\bar`},
		},
		{
			name:  "unterminated quote is flushed at end",
			input: `hello "world`,
			want:  []string{"hello", "world"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseArgs(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseArgs(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// buildChain
// ---------------------------------------------------------------------------

func TestBuildChain_NoMiddleware(t *testing.T) {
	called := false
	handler := HandlerFunc(func(_ *CommandContext) { called = true })
	final := buildChain(handler, nil)
	final(nil)
	if !called {
		t.Error("handler was not called with empty middleware chain")
	}
}

func TestBuildChain_Order(t *testing.T) {
	var order []int
	mw1 := MiddlewareFunc(func(next HandlerFunc) HandlerFunc {
		return func(ctx *CommandContext) {
			order = append(order, 1)
			next(ctx)
			order = append(order, -1)
		}
	})
	mw2 := MiddlewareFunc(func(next HandlerFunc) HandlerFunc {
		return func(ctx *CommandContext) {
			order = append(order, 2)
			next(ctx)
			order = append(order, -2)
		}
	})
	handler := HandlerFunc(func(_ *CommandContext) {
		order = append(order, 0)
	})

	final := buildChain(handler, []MiddlewareFunc{mw1, mw2})
	final(nil)

	// mw1 is the outermost wrapper, so it runs first on entry and last on exit.
	want := []int{1, 2, 0, -2, -1}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("middleware execution order = %v, want %v", order, want)
	}
}
