package main

// example/main.go — demonstrates the GoDiscord framework.
//
// Run with:
//
//	DISCORD_TOKEN=your_token_here go run ./example

import (
	"fmt"
	"log"
	"os"
	"strings"

	discord "github.com/hilleywyn/godiscord"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN environment variable is required")
	}

	bot := discord.New(token, discord.IntentsDefault)

	// ── Presence ────────────────────────────────────────────────────────────
	bot.SetActivity("with Go", discord.ActivityPlaying)

	// ── Event handlers ──────────────────────────────────────────────────────

	bot.OnReady(func(b *discord.Bot, e *discord.ReadyEvent) {
		fmt.Printf("✓ Logged in as %s (Gateway v%d)\n", e.User.Tag(), e.V)
	})

	bot.OnGuildCreate(func(b *discord.Bot, g *discord.Guild) {
		fmt.Printf("  Guild available: %s (%s)\n", g.Name, g.ID)
	})

	bot.OnMessageDelete(func(b *discord.Bot, e *discord.MessageDeleteEvent) {
		fmt.Printf("  Message %s deleted in channel %s\n", e.ID, e.ChannelID)
	})

	bot.OnReactionAdd(func(b *discord.Bot, e *discord.MessageReactionAddEvent) {
		fmt.Printf("  %s reacted with %s\n", e.UserID, e.Emoji.Name)
	})

	// ── Command framework ───────────────────────────────────────────────────
	bot.SetPrefix("!")

	bot.AddCommand(&discord.Command{
		Name:        "ping",
		Description: "Responds with Pong! and shows latency info.",
		Handler: func(ctx *discord.CommandContext) {
			_, _ = ctx.ReplyTo("Pong! 🏓")
		},
	})

	bot.AddCommand(&discord.Command{
		Name:        "echo",
		Aliases:     []string{"say"},
		Description: "Repeats whatever you type after the command.",
		Handler: func(ctx *discord.CommandContext) {
			if ctx.RawArgs == "" {
				_, _ = ctx.Reply("Usage: !echo <text>")
				return
			}
			_, _ = ctx.Reply(ctx.RawArgs)
		},
	})

	bot.AddCommand(&discord.Command{
		Name:        "userinfo",
		Aliases:     []string{"whois", "ui"},
		Description: "Shows information about yourself (or a mentioned user).",
		Handler: func(ctx *discord.CommandContext) {
			user := ctx.Message.Author
			embed := discord.Embed{
				Title: "User Info",
				Color: 0x5865F2, // Discord blurple
				Fields: []discord.EmbedField{
					{Name: "Username", Value: user.Tag(), Inline: true},
					{Name: "ID", Value: user.ID, Inline: true},
					{Name: "Bot", Value: fmt.Sprintf("%v", user.Bot), Inline: true},
				},
			}
			_, _ = ctx.ReplyEmbed(embed)
		},
	})

	bot.AddCommand(&discord.Command{
		Name:        "help",
		Aliases:     []string{"commands", "?"},
		Description: "Lists all available commands.",
		Handler: func(ctx *discord.CommandContext) {
			cmds := ctx.Bot.Commands()
			var sb strings.Builder
			sb.WriteString("**Available commands**\n")
			for _, cmd := range cmds {
				sb.WriteString(fmt.Sprintf("• `!%s` — %s\n", cmd.Name, cmd.Description))
			}
			_, _ = ctx.Reply(sb.String())
		},
	})

	// ── Raw event handler alongside commands ────────────────────────────────
	// You can mix command handlers with raw OnMessageCreate handlers freely.
	bot.OnMessageCreate(func(b *discord.Bot, m *discord.Message) {
		// Ignore bot messages.
		if m.Author == nil || m.Author.Bot {
			return
		}
		// Example: react to every message containing "hello".
		if strings.Contains(strings.ToLower(m.Content), "hello") {
			_ = b.Rest.AddReaction(m.ChannelID, m.ID, "👋")
		}
	})

	// ── Start the bot ───────────────────────────────────────────────────────
	fmt.Println("Connecting to Discord…")
	if err := bot.Run(); err != nil {
		log.Fatal(err)
	}
}
