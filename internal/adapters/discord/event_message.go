package discord

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	pkgdiscord "servbot/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

// Garde lettres (y compris accentuées), chiffres, tiret. Le reste → tiret.
var channelNameSanitize = regexp.MustCompile(`[^\p{L}\p{N}-]+`)

func sanitizeChannelName(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = channelNameSanitize.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 100 {
		s = s[:100]
	}
	return s
}

func (h *Handler) updateEmbed(ctx context.Context, s *discordgo.Session, channelID, messageID string) {
	event, err := h.eventUseCase.GetEventByMessageID(ctx, messageID)
	if err != nil {
		log.Printf("❌ Erreur lors de la récupération de l'événement: %v", err)
		return
	}
	confirmedParticipants, _ := h.eventUseCase.GetConfirmedParticipants(ctx, event.ID)
	waitlistParticipants, _ := h.eventUseCase.GetWaitlistParticipants(ctx, event.ID)
	confirmedCount := len(confirmedParticipants)
	waitlistCount := len(waitlistParticipants)

	origMsg, err := s.ChannelMessage(channelID, messageID)
	if err != nil || origMsg == nil || len(origMsg.Embeds) == 0 {
		log.Printf("❌ Erreur lors de la récupération du message: %v", err)
		return
	}

	newEmbed := *origMsg.Embeds[0]
	pkgdiscord.UpdateEventEmbed(&newEmbed, event, confirmedCount, waitlistCount)

	components := h.buildComponents(messageID, waitlistCount, confirmedCount)

	embeds := []*discordgo.MessageEmbed{&newEmbed}
	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         messageID,
		Channel:    channelID,
		Embeds:     &embeds,
		Components: &components,
	}); err != nil {
		log.Printf("❌ Erreur lors de la mise à jour de l'embed: %v", err)
	}
}

func (h *Handler) buildComponents(messageID string, waitlistCount, confirmedCount int) []discordgo.MessageComponent {
	rowComponents := []discordgo.MessageComponent{
		discordgo.Button{Label: "✏️ Modifier la sortie", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("btn_edit_event_%s", messageID)},
	}
	if waitlistCount > 0 {
		rowComponents = append(rowComponents, discordgo.Button{Label: "⚙️ Gérer la liste d'attente", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("btn_manage_waitlist_%s", messageID)})
	}
	if confirmedCount > 0 {
		rowComponents = append(rowComponents, discordgo.Button{Label: "🗑️ Retirer un participant", Style: discordgo.DangerButton, CustomID: fmt.Sprintf("btn_remove_participant_%s", messageID)})
	}
	var components []discordgo.MessageComponent
	if len(rowComponents) > 0 {
		components = append(components, discordgo.ActionsRow{Components: rowComponents})
	}
	return components
}
