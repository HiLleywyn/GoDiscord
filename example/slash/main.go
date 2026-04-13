// Package main — GoDiscord slash-command example.
//
// Demonstrates:
//   - Registering guild-scoped slash commands on GUILD_CREATE.
//   - Responding to APPLICATION_COMMAND and MESSAGE_COMPONENT interactions.
//   - Ephemeral responses and in-place component updates (UpdateMessage).
//   - Select menu navigation with custom_id allowlisting.
//
// Usage:
//
//	DISCORD_TOKEN=your_token go run .
package main

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

	intents := discord.IntentGuilds | discord.IntentGuildMessages

	bot := discord.New(token, intents)

	// ── Register guild commands on GUILD_CREATE ───────────────────────────────

	bot.OnGuildCreate(func(b *discord.Bot, g *discord.Guild) {
		self := b.Self()
		if self == nil {
			return // called before READY in rare cases; will be retried next GUILD_CREATE
		}

		commands := []discord.ApplicationCommand{
			{
				Name:        "hello",
				Description: "Say hello with a colour selector",
			},
			{
				Name:        "roll",
				Description: "Roll a number between 1 and the given max",
				Options: []discord.ApplicationCommandOption{
					{
						Type:        discord.OptionTypeInteger,
						Name:        "max",
						Description: "The upper bound (inclusive)",
						Required:    true,
					},
				},
			},
			{
				Name:        "info",
				Description: "Show information about this bot (ephemeral)",
			},
		}

		if _, err := b.Rest.BulkOverwriteGuildCommands(self.ID, g.ID, commands); err != nil {
			log.Printf("[slash] failed to register commands in %s: %v", g.Name, err)
		} else {
			log.Printf("[slash] commands registered in %s", g.Name)
		}
	})

	// ── Interaction handler ───────────────────────────────────────────────────

	bot.OnInteractionCreate(func(b *discord.Bot, i *discord.Interaction) {
		switch i.Type {
		case discord.InteractionTypeApplicationCommand:
			handleCommand(b, i)
		case discord.InteractionTypeMessageComponent:
			handleComponent(b, i)
		}
	})

	// ── Run ───────────────────────────────────────────────────────────────────

	bot.OnReady(func(b *discord.Bot, e *discord.ReadyEvent) {
		fmt.Printf("[ready] Logged in as %s\n", e.User.Tag())
	})

	log.Println("[bot] starting…")
	if err := bot.Run(); err != nil {
		log.Fatal(err)
	}
}

// handleCommand routes slash command interactions.
func handleCommand(b *discord.Bot, i *discord.Interaction) {
	if i.Data == nil {
		return
	}

	switch i.Data.Name {
	case "hello":
		// Respond with a greeting and a colour-picker select menu.
		b.Rest.CreateInteractionResponse(i.ID, i.Token, discord.InteractionResponse{
			Type: discord.InteractionCallbackTypeChannelMessage,
			Data: &discord.InteractionResponseData{
				Content: fmt.Sprintf("Hello, **%s**! Pick a colour:", i.Author().Username),
				Components: []discord.Component{
					discord.ActionRow(
						discord.StringSelect("colour:pick", "Choose a colour…", []discord.SelectMenuOption{
							{Label: "Blurple", Value: "blurple", Description: "#5865F2"},
							{Label: "Green", Value: "green", Description: "#57F287"},
							{Label: "Red", Value: "red", Description: "#ED4245"},
							{Label: "Yellow", Value: "yellow", Description: "#FEE75C"},
							{Label: "Fuchsia", Value: "fuchsia", Description: "#EB459E"},
						}),
					),
				},
			},
		})

	case "roll":
		max := int64(6) // default
		for _, opt := range i.Data.Options {
			if opt.Name == "max" {
				if v, ok := opt.Value.(float64); ok {
					max = int64(v)
				}
			}
		}
		if max < 1 {
			max = 1
		}
		// Derive a deterministic demo result from the interaction ID's last digit.
		// A real implementation should use math/rand or crypto/rand.
		lastDigit := i.ID[len(i.ID)-1]
		result := int64(lastDigit)%max + 1
		b.Rest.CreateInteractionResponse(i.ID, i.Token, discord.InteractionResponse{
			Type: discord.InteractionCallbackTypeChannelMessage,
			Data: &discord.InteractionResponseData{
				Content: fmt.Sprintf("🎲 You rolled **%d** (1–%d)", result, max),
			},
		})

	case "info":
		self := b.Self()
		name := "GoDiscord Bot"
		if self != nil {
			name = self.Tag()
		}
		embed := discord.Embed{
			Title:       name,
			Description: "A Discord bot powered by [GoDiscord](https://github.com/hilleywyn/godiscord) — a zero-dependency Go framework.",
			Color:       0x5865F2,
			Fields: []discord.EmbedField{
				{Name: "Framework", Value: "GoDiscord", Inline: true},
				{Name: "Language", Value: "Go 1.21+", Inline: true},
				{Name: "Dependencies", Value: "None", Inline: true},
			},
			Footer: &discord.EmbedFooter{Text: "GoDiscord slash example"},
		}
		b.Rest.CreateInteractionResponse(i.ID, i.Token, discord.InteractionResponse{
			Type: discord.InteractionCallbackTypeChannelMessage,
			Data: &discord.InteractionResponseData{
				Embeds: []discord.Embed{embed},
				Flags:  discord.MessageFlagEphemeral,
			},
		})
	}
}

// handleComponent routes MESSAGE_COMPONENT interactions (buttons, select menus).
func handleComponent(b *discord.Bot, i *discord.Interaction) {
	if i.Data == nil {
		return
	}

	customID := i.Data.CustomID

	// Allowlist: only handle custom IDs with known prefixes.
	if !strings.HasPrefix(customID, "colour:") {
		return
	}

	if customID == "colour:pick" && len(i.Data.Values) > 0 {
		colourMap := map[string]int{
			"blurple": 0x5865F2,
			"green":   0x57F287,
			"red":     0xED4245,
			"yellow":  0xFEE75C,
			"fuchsia": 0xEB459E,
		}

		selected := i.Data.Values[0]
		colour, ok := colourMap[selected]
		if !ok {
			return
		}

		// Capitalise the colour name without the deprecated strings.Title.
		displayName := strings.ToUpper(selected[:1]) + selected[1:]
		embed := discord.Embed{
			Title:       "Your colour: " + displayName,
			Description: fmt.Sprintf("Colour code: `#%06X`", colour),
			Color:       colour,
		}

		// Update the original message in-place (no new message is sent).
		b.Rest.CreateInteractionResponse(i.ID, i.Token, discord.InteractionResponse{
			Type: discord.InteractionCallbackTypeUpdateMessage,
			Data: &discord.InteractionResponseData{
				Content:    "",
				Embeds:     []discord.Embed{embed},
				Components: []discord.Component{}, // remove the select menu
			},
		})
	}
}
