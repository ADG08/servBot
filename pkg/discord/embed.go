package discord

import (
	"fmt"
	"strings"
	"time"

	"servbot/internal/domain"
	"servbot/internal/domain/entities"
	"servbot/pkg/tz"

	"github.com/bwmarrin/discordgo"
)

const (
	embedColor = 0x5865F2
	embedTitle = "📅 Détails de la sortie"
)

func formatPlaces(maxSlots, confirmedCount int) string {
	if maxSlots == 0 {
		return fmt.Sprintf("%d (Illimité)", confirmedCount)
	}
	return fmt.Sprintf("%d/%d", confirmedCount, maxSlots)
}

func buildDescriptionBase(organizerMention, description string, scheduledAt time.Time, placesText string, waitlistCount int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("**Organisé par :** %s\n\n", organizerMention))
	b.WriteString(description)
	if !scheduledAt.IsZero() {
		b.WriteString(fmt.Sprintf("\n\n**Quand :** %s", scheduledAt.In(tz.Paris).Format("02/01/2006 à 15:04")))
	}
	b.WriteString(fmt.Sprintf("\n\n**Places :** %s", placesText))
	if waitlistCount > 0 {
		b.WriteString(fmt.Sprintf(" • %d en attente", waitlistCount))
	}
	return b.String()
}

// BuildNewEventEmbed builds the initial post embed with the organizer
// already counted as a confirmed participant. Only counters are shown (no public list).
func BuildNewEventEmbed(creatorID, description string, scheduledAt time.Time, slots int, displayName, avatarURL string) *discordgo.MessageEmbed {
	userMention := fmt.Sprintf("<@%s>", creatorID)
	placesText := formatPlaces(slots, 1)
	desc := buildDescriptionBase(userMention, description, scheduledAt, placesText, 0)
	return &discordgo.MessageEmbed{
		Title:       embedTitle,
		Description: desc,
		Color:       embedColor,
		Author:      &discordgo.MessageEmbedAuthor{Name: displayName, IconURL: avatarURL},
		Footer:      &discordgo.MessageEmbedFooter{Text: "Réagis avec ✅ pour t'inscrire"},
	}
}

// UpdateEventEmbed updates the embed with confirmed/waitlist counts only (no public participant list).
func UpdateEventEmbed(embed *discordgo.MessageEmbed, event *entities.Event, confirmedCount, waitlistCount int) {
	organizerMention := fmt.Sprintf("<@%s>", event.CreatorID)
	placesText := formatPlaces(event.MaxSlots, confirmedCount)
	embed.Description = buildDescriptionBase(organizerMention, event.Description, event.ScheduledAt, placesText, waitlistCount)
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
