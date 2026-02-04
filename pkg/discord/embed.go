package discord

import (
	"fmt"
	"strings"
	"time"

	"servbot/internal/domain"
	"servbot/internal/domain/entities"

	"github.com/bwmarrin/discordgo"
)

const (
	embedColor   = 0x5865F2
	embedTitle   = "📅 Détails de la sortie"
	maxDisplayed = 10
)

func formatPlaces(maxSlots, confirmedCount int) string {
	if maxSlots == 0 {
		return fmt.Sprintf("%d (Illimité)", confirmedCount)
	}
	return fmt.Sprintf("%d/%d", confirmedCount, maxSlots)
}

func buildDescriptionBase(organizerMention, description string, scheduledAt time.Time, placesText string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("**Organisé par :** %s\n\n", organizerMention))
	b.WriteString(description)
	if !scheduledAt.IsZero() {
		b.WriteString(fmt.Sprintf("\n\n**Quand :** %s", FormatEventDateTime(scheduledAt)))
	}
	b.WriteString(fmt.Sprintf("\n\n**Places :** %s", placesText))
	return b.String()
}

// BuildNewEventEmbed builds the initial post embed with the organizer
// already counted as a confirmed participant (e.g. 1/4) and listed
// in the participants section.
func BuildNewEventEmbed(creatorID, description string, scheduledAt time.Time, slots int, displayName, avatarURL string) *discordgo.MessageEmbed {
	userMention := fmt.Sprintf("<@%s>", creatorID)
	placesText := formatPlaces(slots, 1)
	desc := buildDescriptionBase(userMention, description, scheduledAt, placesText)
	desc += "\n\n✅ **Participants :**\n- " + userMention
	return &discordgo.MessageEmbed{
		Title:       embedTitle,
		Description: desc,
		Color:       embedColor,
		Author:      &discordgo.MessageEmbedAuthor{Name: displayName, IconURL: avatarURL},
		Footer:      &discordgo.MessageEmbedFooter{Text: "Organisé par " + userMention},
	}
}

func UpdateEventEmbed(embed *discordgo.MessageEmbed, event *entities.Event, confirmed, waitlist []string) {
	organizerMention := fmt.Sprintf("<@%s>", event.CreatorID)
	placesText := formatPlaces(event.MaxSlots, len(confirmed))
	desc := buildDescriptionBase(organizerMention, event.Description, event.ScheduledAt, placesText)
	var b strings.Builder
	b.WriteString(desc)
	if len(confirmed) > 0 {
		b.WriteString("\n\n✅ **Participants :**\n")
		displayCount := min(len(confirmed), maxDisplayed)
		b.WriteString(strings.Join(confirmed[:displayCount], "\n"))
		if len(confirmed) > maxDisplayed {
			b.WriteString(fmt.Sprintf("\n*... et %d autre(s)*", len(confirmed)-maxDisplayed))
		}
	}
	if len(waitlist) > 0 {
		b.WriteString("\n\n⏳ **Liste d'attente :**\n")
		displayCount := min(len(waitlist), maxDisplayed)
		b.WriteString(strings.Join(waitlist[:displayCount], "\n"))
		if len(waitlist) > maxDisplayed {
			b.WriteString(fmt.Sprintf("\n*... et %d autre(s)*", len(waitlist)-maxDisplayed))
		}
	}
	embed.Description = b.String()
}

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
