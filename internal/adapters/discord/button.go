package discord

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"servbot/internal/domain"
	pkgdiscord "servbot/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

func (h *Handler) HandleJoin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByMessageID(ctx, i.Message.ID)
	if err != nil {
		return
	}
	userID := i.Member.User.ID
	username := i.Member.User.Username

	reply, err := h.participantUseCase.JoinEvent(ctx, event.ID, userID, username)
	if err != nil {
		if errors.Is(err, domain.ErrParticipantExists) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: reply,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
		return
	}

	h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: reply,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *Handler) HandleLeave(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByMessageID(ctx, i.Message.ID)
	if err != nil {
		return
	}
	userID := i.Member.User.ID
	wasConfirmed, err := h.participantUseCase.LeaveEvent(ctx, event.ID, userID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Tu n'étais pas inscrit.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	msg := "🗑️ Tu t'es désisté."
	if wasConfirmed {
		luckyWinner, err := h.participantUseCase.GetNextWaitlistParticipant(ctx, event.ID)
		if err == nil {
			ch, err := s.UserChannelCreate(luckyWinner.UserID)
			if err == nil && ch != nil {
				s.ChannelMessageSend(ch.ID, fmt.Sprintf("🎉 **Bonne nouvelle !** Une place s'est libérée pour **%s**, tu es maintenant inscrit !", event.Title))
			}
		}
	}

	h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

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
	confirmed, waitlist := pkgdiscord.FormatParticipants(event.Participants)

	origMsg, err := s.ChannelMessage(channelID, messageID)
	if err != nil || origMsg == nil || len(origMsg.Embeds) == 0 {
		log.Printf("❌ Erreur lors de la récupération du message: %v", err)
		return
	}

	newEmbed := *origMsg.Embeds[0]
	pkgdiscord.UpdateEventEmbed(&newEmbed, event, confirmed, waitlist)

	waitlistParticipants, _ := h.eventUseCase.GetWaitlistParticipants(ctx, event.ID)
	confirmedParticipants, _ := h.eventUseCase.GetConfirmedParticipants(ctx, event.ID)
	components := h.buildComponents(messageID, len(waitlistParticipants), len(confirmedParticipants))

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
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Je participe", Style: discordgo.SuccessButton, CustomID: "btn_join"},
			discordgo.Button{Label: "Se désister", Style: discordgo.DangerButton, CustomID: "btn_leave"},
		}},
	}
	rowComponents := []discordgo.MessageComponent{
		discordgo.Button{Label: "✏️ Modifier la sortie", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("btn_edit_event_%s", messageID)},
	}
	if waitlistCount > 0 {
		rowComponents = append(rowComponents, discordgo.Button{Label: "⚙️ Gérer la liste d'attente", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("btn_manage_waitlist_%s", messageID)})
	}
	if confirmedCount > 0 {
		rowComponents = append(rowComponents, discordgo.Button{Label: "🗑️ Retirer un participant", Style: discordgo.DangerButton, CustomID: fmt.Sprintf("btn_remove_participant_%s", messageID)})
	}
	if len(rowComponents) > 0 {
		components = append(components, discordgo.ActionsRow{Components: rowComponents})
	}
	return components
}
