// Package main — GoDiscord basic example.
//
// Demonstrates prefix commands, middleware, and event handling.
//
// Usage:
//
//	DISCORD_TOKEN=your_token go run .
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	discord "github.com/hilleywyn/godiscord"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN environment variable is required")
	}

	intents := discord.IntentGuilds |
		discord.IntentGuildMessages |
		discord.IntentMessageContent |
		discord.IntentGuildMembers

	bot := discord.New(token, intents)

	// ── Presence ────────────────────────────────────────────────────────────

	bot.SetActivity("over the server", discord.ActivityWatching)

	// ── Middleware ───────────────────────────────────────────────────────────

	// Log every command invocation.
	bot.Use(func(next discord.HandlerFunc) discord.HandlerFunc {
		return func(ctx *discord.CommandContext) {
			start := time.Now()
			log.Printf("[cmd] %-12s  user=%s  guild=%s",
				ctx.Command.Name, ctx.Message.Author.Username, ctx.Message.GuildID)
			next(ctx)
			log.Printf("[cmd] %-12s  done in %s", ctx.Command.Name, time.Since(start))
		}
	})

	// ── Events ───────────────────────────────────────────────────────────────

	bot.OnReady(func(b *discord.Bot, e *discord.ReadyEvent) {
		fmt.Printf("[ready] Logged in as %s\n", e.User.Tag())
	})

	bot.OnGuildCreate(func(b *discord.Bot, g *discord.Guild) {
		fmt.Printf("[guild] Available: %s (%s)\n", g.Name, g.ID)
	})

	bot.OnGuildMemberAdd(func(b *discord.Bot, e *discord.GuildMemberAddEvent) {
		fmt.Printf("[join] %s joined guild %s\n", e.User.Tag(), e.GuildID)
	})

	// ── Commands ─────────────────────────────────────────────────────────────

	bot.SetPrefix("!")

	bot.AddCommand(&discord.Command{
		Name:        "ping",
		Description: "Check that the bot is alive and measure latency",
		Handler: func(ctx *discord.CommandContext) {
			start := time.Now()
			msg, err := ctx.Reply("🏓 Pong!")
			if err != nil {
				log.Printf("[ping] error: %v", err)
				return
			}
			elapsed := time.Since(start)
			ctx.Bot.Rest.EditMessage(msg.ChannelID, msg.ID,
				fmt.Sprintf("🏓 Pong! (%dms)", elapsed.Milliseconds()))
		},
	})

	bot.AddCommand(&discord.Command{
		Name:        "echo",
		Aliases:     []string{"say"},
		Description: "Repeat the given text",
		Usage:       "<text>",
		Handler: func(ctx *discord.CommandContext) {
			if ctx.RawArgs == "" {
				ctx.Reply("Usage: !echo <text>")
				return
			}
			// Delete the invoking message, then echo.
			_ = ctx.Bot.Rest.DeleteMessage(ctx.Message.ChannelID, ctx.Message.ID)
			ctx.Reply(ctx.RawArgs)
		},
	})

	bot.AddCommand(&discord.Command{
		Name:        "userinfo",
		Description: "Show information about a guild member",
		Usage:       "[@user]",
		Handler: func(ctx *discord.CommandContext) {
			// Use the invoking user if no mention was provided.
			userID := ctx.Message.Author.ID
			if len(ctx.Args) > 0 {
				mention := ctx.Args[0]
				// Strip <@…> mention syntax.
				userID = strings.TrimRight(strings.TrimLeft(strings.TrimLeft(mention, "<@!"), "<@"), ">")
			}

			member, err := ctx.Bot.Rest.GetGuildMember(ctx.Message.GuildID, userID)
			if err != nil {
				var apiErr *discord.APIError
				if errors.As(err, &apiErr) && apiErr.IsNotFound() {
					ctx.Reply("User not found in this guild.")
					return
				}
				ctx.Reply("Failed to fetch user: " + err.Error())
				return
			}

			perms, _ := discord.ParsePermission(member.Permissions)

			embed := discord.Embed{
				Title: member.User.Tag(),
				Color: 0x5865F2,
				Fields: []discord.EmbedField{
					{Name: "ID", Value: member.User.ID, Inline: true},
					{Name: "Joined", Value: member.JoinedAt, Inline: true},
					{Name: "Roles", Value: fmt.Sprintf("%d roles", len(member.Roles)), Inline: true},
					{Name: "Bot", Value: fmt.Sprintf("%v", member.User.Bot), Inline: true},
					{Name: "Admin", Value: fmt.Sprintf("%v", perms.IsAdmin()), Inline: true},
				},
				Footer: &discord.EmbedFooter{Text: "GoDiscord basic example"},
			}
			ctx.ReplyEmbed(embed)
		},
	})

	bot.AddCommand(&discord.Command{
		Name:        "help",
		Description: "List all available commands",
		Handler: func(ctx *discord.CommandContext) {
			cmds := ctx.Bot.Commands()
			var lines []string
			for _, cmd := range cmds {
				line := fmt.Sprintf("`!%s`", cmd.Name)
				if cmd.Usage != "" {
					line += " `" + cmd.Usage + "`"
				}
				if cmd.Description != "" {
					line += " — " + cmd.Description
				}
				lines = append(lines, line)
			}
			embed := discord.Embed{
				Title:       "Commands",
				Description: strings.Join(lines, "\n"),
				Color:       0x57F287,
			}
			ctx.ReplyEmbed(embed)
		},
	})

	// ── Run ──────────────────────────────────────────────────────────────────

	log.Println("[bot] starting…")
	if err := bot.Run(); err != nil {
		log.Fatal(err)
	}
}
