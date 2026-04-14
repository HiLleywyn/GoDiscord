package discord

// commands.go — prefix-based command framework.
//
// Register commands with Bot.AddCommand(). When a message starts with the
// configured prefix and a known command name (or alias), the command's Handler
// is called with a CommandContext that provides the bot, the message, parsed
// arguments, and convenience reply methods.
//
// # Argument Parsing
//
// Arguments are split on whitespace by default. Double-quoted strings are
// treated as a single argument regardless of internal spaces:
//
//	!ban @user "being very rude in general"
//	// Args: ["@user", "being very rude in general"]
//
// # Middleware
//
// Register middleware with Bot.Use(). Middleware wraps every command handler:
//
//	bot.Use(func(next discord.HandlerFunc) discord.HandlerFunc {
//	    return func(ctx *discord.CommandContext) {
//	        log.Printf("[cmd] %s by %s", ctx.Command.Name, ctx.Message.Author.Username)
//	        next(ctx)
//	    }
//	})

import (
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

// HandlerFunc is the type of a command handler function. Middleware functions
// accept a HandlerFunc and return a wrapped HandlerFunc.
type HandlerFunc func(*CommandContext)

// MiddlewareFunc wraps a HandlerFunc, allowing pre/post processing around
// every command invocation.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// ---------------------------------------------------------------------------
// Command
// ---------------------------------------------------------------------------

// Command describes a single bot command.
type Command struct {
	// Name is the primary name used to invoke the command (case-insensitive).
	Name string

	// Aliases are alternative names for the command.
	Aliases []string

	// Description is a human-readable summary shown in help output.
	Description string

	// Usage describes the argument syntax, e.g. "@user [days] [reason]".
	// Shown in help embeds if non-empty.
	Usage string

	// Handler is called when the command is matched. Do not set this field
	// directly when using middleware — use commandHandler.build() internally.
	Handler func(*CommandContext)

	// RequiredPermissions, if non-zero, gates this command behind a Discord
	// permission check. All specified bits must be present in the invoking
	// member's computed guild permissions. A failed check invokes the bot's
	// command-denied callback (see Bot.SetCommandDenied) if one is registered.
	RequiredPermissions Permission

	// PermCheck is an optional custom gate evaluated after RequiredPermissions.
	// Return false to block the invocation. Runs synchronously in the dispatch
	// goroutine, so it must not block.
	PermCheck func(*CommandContext) bool
}

// ---------------------------------------------------------------------------
// CommandContext
// ---------------------------------------------------------------------------

// CommandContext is passed to a command handler and provides everything needed
// to respond to the invoking message.
type CommandContext struct {
	// Bot is the running bot instance.
	Bot *Bot

	// Message is the message that triggered the command.
	Message *Message

	// Command is the matched Command struct.
	Command *Command

	// Args contains the parsed tokens after the command name.
	// Quoted strings are kept together as a single argument:
	//   !ban @user "too many warnings" → Args: ["@user", "too many warnings"]
	Args []string

	// RawArgs is everything after the command name, unsplit and untouched.
	RawArgs string

	// GuildID is the ID of the guild the command was invoked in. Empty for DMs.
	GuildID string

	// ChannelID is the ID of the channel the command was invoked in.
	ChannelID string

	// AuthorID is the ID of the user who invoked the command. Empty if Author is nil.
	AuthorID string

	// Member is the guild member record of the invoking user. Nil for DMs.
	Member *Member
}

// Reply sends a plain-text message to the same channel.
func (ctx *CommandContext) Reply(content string) (*Message, error) {
	return ctx.Bot.Rest.SendMessage(ctx.Message.ChannelID, content)
}

// ReplyEmbed sends an embed to the same channel.
func (ctx *CommandContext) ReplyEmbed(embed Embed) (*Message, error) {
	return ctx.Bot.Rest.SendEmbed(ctx.Message.ChannelID, embed)
}

// ReplyTo sends a message that replies (with a message reference) to the
// invoking message.
func (ctx *CommandContext) ReplyTo(content string) (*Message, error) {
	return ctx.Bot.Rest.ReplyTo(ctx.Message, content)
}

// ---------------------------------------------------------------------------
// commandHandler
// ---------------------------------------------------------------------------

// commandHandler manages registered commands and routes messages.
type commandHandler struct {
	prefix     string
	mu         sync.RWMutex
	commands   map[string]*Command // keyed by lower-cased name and aliases
	middleware []MiddlewareFunc
	onDenied   func(*CommandContext, string) // called when a permission check fails
}

func newCommandHandler(prefix string) *commandHandler {
	return &commandHandler{
		prefix:   prefix,
		commands: make(map[string]*Command),
	}
}

// use appends middleware to the chain. Order matters: the first Use() call
// becomes the outermost wrapper.
func (h *commandHandler) use(mw ...MiddlewareFunc) {
	h.mu.Lock()
	h.middleware = append(h.middleware, mw...)
	h.mu.Unlock()
}

// register adds a command (and its aliases) to the routing table.
func (h *commandHandler) register(cmd *Command) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.commands[strings.ToLower(cmd.Name)] = cmd
	for _, alias := range cmd.Aliases {
		h.commands[strings.ToLower(alias)] = cmd
	}
}

// unregister removes a command and all its aliases from the routing table.
func (h *commandHandler) unregister(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	lower := strings.ToLower(name)
	cmd, ok := h.commands[lower]
	if !ok {
		return
	}
	// Remove primary name and all aliases that point to the same Command.
	for k, v := range h.commands {
		if v == cmd {
			delete(h.commands, k)
		}
	}
}

// list returns all unique registered commands (no alias duplicates).
func (h *commandHandler) list() []*Command {
	h.mu.RLock()
	defer h.mu.RUnlock()
	seen := make(map[*Command]struct{})
	var out []*Command
	for _, cmd := range h.commands {
		if _, ok := seen[cmd]; !ok {
			seen[cmd] = struct{}{}
			out = append(out, cmd)
		}
	}
	return out
}

// handle inspects a message and invokes the matching command handler, if any.
func (h *commandHandler) handle(b *Bot, msg *Message) {
	if h == nil || msg.Author == nil || msg.Author.Bot {
		return
	}

	h.mu.RLock()
	prefix := h.prefix
	h.mu.RUnlock()

	if prefix == "" || !strings.HasPrefix(msg.Content, prefix) {
		return
	}

	rest := strings.TrimPrefix(msg.Content, prefix)
	if rest == "" {
		return
	}

	// Parse the command name from the first whitespace-delimited token.
	tokens := strings.Fields(rest)
	if len(tokens) == 0 {
		return
	}

	cmdName := strings.ToLower(tokens[0])
	rawArgs := ""
	if idx := strings.Index(rest, tokens[0]); idx >= 0 {
		after := rest[idx+len(tokens[0]):]
		rawArgs = strings.TrimSpace(after)
	}

	h.mu.RLock()
	cmd, ok := h.commands[cmdName]
	mw := h.middleware
	h.mu.RUnlock()

	if !ok {
		return
	}

	// Use the quoted-aware parser for Args so callers can do:
	//   !reason <id> "new reason with spaces"
	args := parseArgs(rawArgs)

	ctx := &CommandContext{
		Bot:     b,
		Message: msg,
		Command: cmd,
		Args:    args,
		RawArgs: rawArgs,
	}

	ctx.GuildID = msg.GuildID
	ctx.ChannelID = msg.ChannelID
	if msg.Author != nil {
		ctx.AuthorID = msg.Author.ID
	}
	ctx.Member = msg.Member

	// Permission gate: Discord bitfield check.
	if cmd.RequiredPermissions != 0 {
		allowed := false
		if msg.Member != nil {
			if perms, err := ParsePermission(msg.Member.Permissions); err == nil {
				allowed = perms.Has(cmd.RequiredPermissions)
			}
		}
		if !allowed {
			h.mu.RLock()
			denied := h.onDenied
			h.mu.RUnlock()
			if denied != nil {
				denied(ctx, "missing required permissions")
			}
			return
		}
	}

	// Custom permission gate.
	if cmd.PermCheck != nil && !cmd.PermCheck(ctx) {
		h.mu.RLock()
		denied := h.onDenied
		h.mu.RUnlock()
		if denied != nil {
			denied(ctx, "permission check failed")
		}
		return
	}

	// Build the middleware chain and invoke.
	final := buildChain(cmd.Handler, mw)
	final(ctx)
}

// buildChain wraps handler with middleware in reverse order so the first
// middleware in the slice is the outermost wrapper.
func buildChain(handler HandlerFunc, mw []MiddlewareFunc) HandlerFunc {
	for i := len(mw) - 1; i >= 0; i-- {
		handler = mw[i](handler)
	}
	return handler
}

// ---------------------------------------------------------------------------
// Quoted argument parser
// ---------------------------------------------------------------------------

// parseArgs splits s into arguments, respecting double-quoted strings.
// Quoted strings may contain spaces and are returned without their quotes.
// A backslash before a double-quote escapes it inside a quoted string.
//
// Examples:
//
//	parseArgs(`@user spamming`)                      → ["@user", "spamming"]
//	parseArgs(`@user "repeated rule violations"`)    → ["@user", "repeated rule violations"]
//	parseArgs(`@user "said \"hi\" 3 times"`)         → ["@user", `said "hi" 3 times`]
func parseArgs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	var args []string
	var cur strings.Builder
	inQuote := false
	escaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if escaped {
			cur.WriteByte(c)
			escaped = false
			continue
		}

		switch c {
		case '\\':
			if inQuote {
				escaped = true
			} else {
				cur.WriteByte(c)
			}
		case '"':
			if inQuote {
				inQuote = false
			} else {
				inQuote = true
			}
		case ' ', '\t':
			if inQuote {
				cur.WriteByte(c)
			} else if cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}

	if cur.Len() > 0 {
		args = append(args, cur.String())
	}

	return args
}
