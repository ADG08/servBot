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

func grantPrivateChannelAccess(s *discordgo.Session, channelID, userID string) {
	if channelID == "" || userID == "" {
		return
	}
	err := s.ChannelPermissionSet(channelID, userID, discordgo.PermissionOverwriteTypeMember,
		discordgo.PermissionViewChannel|discordgo.PermissionSendMessages, 0)
	if err != nil {
		log.Printf("❌ Ajout accès salon privé (channel=%s, user=%s): %v", channelID, userID, err)
	}
}

func revokePrivateChannelAccess(s *discordgo.Session, channelID, userID string) {
	if channelID == "" || userID == "" {
		return
	}
	err := s.ChannelPermissionDelete(channelID, userID)
	if err != nil {
		log.Printf("❌ Retrait accès salon privé (channel=%s, user=%s): %v", channelID, userID, err)
	}
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

	components := h.buildComponents(messageID, waitlistCount, confirmedCount, event.IsEditLocked())

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

const buttonsPerRow = 2

func (h *Handler) buildComponents(messageID string, waitlistCount, confirmedCount int, editLocked bool) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent
	if !editLocked {
		buttons = append(buttons, discordgo.Button{Label: "✏️ Modifier la sortie", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("btn_edit_event_%s", messageID)})
	}
	buttons = append(buttons, discordgo.Button{
		Label:    "❓ Poser une question",
		Style:    discordgo.SecondaryButton,
		CustomID: fmt.Sprintf("btn_ask_question_%s", messageID),
	})
	if waitlistCount > 0 {
		buttons = append(buttons, discordgo.Button{Label: "⚙️ Gérer la liste d'attente", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("btn_manage_waitlist_%s", messageID)})
	}
	if confirmedCount > 0 {
		buttons = append(buttons, discordgo.Button{Label: "🗑️ Retirer un participant", Style: discordgo.DangerButton, CustomID: fmt.Sprintf("btn_remove_participant_%s", messageID)})
	}
	var components []discordgo.MessageComponent
	for i := 0; i < len(buttons); i += buttonsPerRow {
		end := min(i+buttonsPerRow, len(buttons))
		components = append(components, discordgo.ActionsRow{Components: buttons[i:end]})
	}
	return components
}
