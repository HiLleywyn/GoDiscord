package discord

// commands.go — prefix-based command framework.
//
// Register commands with Bot.AddCommand(). When a message starts with the
// configured prefix and a known command name (or alias), the command's Handler
// is called with a CommandContext that provides the bot, the message, parsed
// arguments, and convenience reply methods.

import (
	"strings"
	"sync"
)

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

	// Handler is called when the command is matched.
	Handler func(*CommandContext)
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

	// Args contains the whitespace-split tokens after the command name.
	// e.g. "!ban @user spamming" → Args: ["@user", "spamming"]
	Args []string

	// RawArgs is everything after the command name, unsplit.
	RawArgs string
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

// commandHandler manages the registered commands and routes messages.
type commandHandler struct {
	prefix   string
	mu       sync.RWMutex
	commands map[string]*Command // keyed by lower-cased name and aliases
}

func newCommandHandler(prefix string) *commandHandler {
	return &commandHandler{
		prefix:   prefix,
		commands: make(map[string]*Command),
	}
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

// list returns all unique registered commands (no duplicates from aliases).
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

	// Strip the prefix and split into tokens.
	rest := strings.TrimPrefix(msg.Content, prefix)
	if rest == "" {
		return
	}

	tokens := strings.Fields(rest)
	if len(tokens) == 0 {
		return
	}

	cmdName := strings.ToLower(tokens[0])
	args := tokens[1:]
	rawArgs := ""
	if len(args) > 0 {
		// Preserve original spacing for rawArgs.
		idx := strings.Index(rest, tokens[0]) + len(tokens[0])
		rawArgs = strings.TrimSpace(rest[idx:])
	}

	h.mu.RLock()
	cmd, ok := h.commands[cmdName]
	h.mu.RUnlock()

	if !ok {
		return
	}

	ctx := &CommandContext{
		Bot:     b,
		Message: msg,
		Command: cmd,
		Args:    args,
		RawArgs: rawArgs,
	}
	cmd.Handler(ctx)
}
