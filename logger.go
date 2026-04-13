package discord

// logger.go — pluggable logger interface for GoDiscord.
//
// By default GoDiscord logs via the standard library's log package. Swap in
// any logger that satisfies the Logger interface using the WithLogger option:
//
//	bot := discord.New(token, intents, discord.WithLogger(myLogger))

import "log"

// ---------------------------------------------------------------------------
// Logger interface
// ---------------------------------------------------------------------------

// Logger is the interface that GoDiscord uses for all internal log output.
// Implement it to route log lines to zap, zerolog, slog, or any other sink.
//
// The method signatures match the standard library's log.Printf / log.Println
// so a thin wrapper around log.Default() satisfies it at zero cost.
type Logger interface {
	// Printf formats according to a format specifier and writes to the log.
	Printf(format string, args ...interface{})

	// Println writes the operands to the log, separated by spaces.
	Println(args ...interface{})
}

// ---------------------------------------------------------------------------
// Default implementation
// ---------------------------------------------------------------------------

// stdLogger wraps the standard library logger to satisfy the Logger interface.
type stdLogger struct{}

func (stdLogger) Printf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (stdLogger) Println(args ...interface{}) {
	log.Println(args...)
}

// defaultLogger is the logger used when none is configured.
var defaultLogger Logger = stdLogger{}

// ---------------------------------------------------------------------------
// Functional option
// ---------------------------------------------------------------------------

// Option is a functional option for configuring a Bot during construction.
// Pass options as variadic arguments to New().
type Option func(*Bot)

// WithLogger replaces the default standard-library logger with the provided
// implementation. Useful for structured logging with zap, zerolog, or slog.
//
// Example — adapting zerolog:
//
//	type zerologAdapter struct{ log zerolog.Logger }
//	func (a zerologAdapter) Printf(f string, args ...interface{}) { a.log.Printf(f, args...) }
//	func (a zerologAdapter) Println(args ...interface{})          { a.log.Print(args...) }
//
//	bot := discord.New(token, intents, discord.WithLogger(zerologAdapter{zlog}))
func WithLogger(l Logger) Option {
	return func(b *Bot) {
		if l != nil {
			b.log = l
		}
	}
}

// ---------------------------------------------------------------------------
// NoopLogger — useful in tests to silence all output
// ---------------------------------------------------------------------------

// NoopLogger discards every log message. Assign it with WithLogger to silence
// GoDiscord during tests:
//
//	bot := discord.New(token, intents, discord.WithLogger(discord.NoopLogger{}))
type NoopLogger struct{}

func (NoopLogger) Printf(string, ...interface{}) {}
func (NoopLogger) Println(...interface{})         {}
