package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"servbot/internal/domain"
	"servbot/internal/domain/entities"
)

const maxDisplayed = 10

// BuildEventEmbed builds an embed for an event.
func BuildEventEmbed(event *entities.Event, desc string) *discordgo.MessageEmbed {
	userMention := fmt.Sprintf("<@%s>", event.CreatorID)
	placesText := "Illimité"
	if event.MaxSlots > 0 {
		placesText = fmt.Sprintf("0/%d", event.MaxSlots)
	}
	return &discordgo.MessageEmbed{
		Title:       "📅 Détails de la sortie",
		Description: fmt.Sprintf("**Organisé par :** %s\n\n%s\n\n**Places :** %s", userMention, desc, placesText),
		Color:       0x5865F2,
	}
}

// UpdateEventEmbed updates an embed with event and participant data.
func UpdateEventEmbed(embed *discordgo.MessageEmbed, event *entities.Event, confirmed, waitlist []string) {
	organizerMention := fmt.Sprintf("<@%s>", event.CreatorID)
	var descBuilder strings.Builder
	descBuilder.WriteString(fmt.Sprintf("**Organisé par :** %s\n\n", organizerMention))
	descBuilder.WriteString(event.Description)
	placesText := fmt.Sprintf("%d/%d", len(confirmed), event.MaxSlots)
	if event.MaxSlots == 0 {
		placesText = fmt.Sprintf("%d (Illimité)", len(confirmed))
	}
	descBuilder.WriteString(fmt.Sprintf("\n\n**Places :** %s", placesText))
	if len(confirmed) > 0 {
		descBuilder.WriteString("\n\n✅ **Participants :**\n")
		displayCount := len(confirmed)
		if displayCount > maxDisplayed {
			displayCount = maxDisplayed
		}
		descBuilder.WriteString(strings.Join(confirmed[:displayCount], "\n"))
		if len(confirmed) > maxDisplayed {
			descBuilder.WriteString(fmt.Sprintf("\n*... et %d autre(s)*", len(confirmed)-maxDisplayed))
		}
	}
	if len(waitlist) > 0 {
		descBuilder.WriteString("\n\n⏳ **Liste d'attente :**\n")
		displayCount := len(waitlist)
		if displayCount > maxDisplayed {
			displayCount = maxDisplayed
		}
		descBuilder.WriteString(strings.Join(waitlist[:displayCount], "\n"))
		if len(waitlist) > maxDisplayed {
			descBuilder.WriteString(fmt.Sprintf("\n*... et %d autre(s)*", len(waitlist)-maxDisplayed))
		}
	}
	embed.Description = descBuilder.String()
}

// FormatParticipants splits participants into confirmed and waitlist mention strings.
func FormatParticipants(participants []entities.Participant) (confirmed, waitlist []string) {
	confirmed = make([]string, 0, len(participants))
	waitlist = make([]string, 0, len(participants))
	for _, p := range participants {
		mention := fmt.Sprintf("<@%s>", p.UserID)
		switch p.Status {
		case domain.StatusConfirmed:
			confirmed = append(confirmed, "- "+mention)
		case domain.StatusWaitlist:
			waitlist = append(waitlist, "- "+mention)
		}
	}
	return confirmed, waitlist
}
