package discord

import (
	"context"
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
		if err == domain.ErrParticipantExists {
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

func (h *Handler) HandleCreateSortieChat(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByMessageID(ctx, i.Message.ID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Événement non trouvé.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	if i.Member.User.ID != event.CreatorID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Seul l'organisateur peut créer un salon pour la sortie.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	confirmed, err := h.eventUseCase.GetConfirmedParticipants(ctx, event.ID)
	if err != nil || len(confirmed) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Aucun participant confirmé pour créer un salon.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	allowedIDs := map[string]struct{}{event.CreatorID: {}}
	for _, p := range confirmed {
		allowedIDs[p.UserID] = struct{}{}
	}

	guildID := i.GuildID
	parentID := ""
	if ch, err := s.Channel(event.ChannelID); err == nil && ch.ParentID != "" {
		if parent, err := s.Channel(ch.ParentID); err == nil && parent.ParentID != "" {
			parentID = parent.ParentID
		}
	}

	overwrites := []*discordgo.PermissionOverwrite{
		{ID: guildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: discordgo.PermissionViewChannel},
	}
	for uid := range allowedIDs {
		overwrites = append(overwrites, &discordgo.PermissionOverwrite{
			ID:    uid,
			Type:  discordgo.PermissionOverwriteTypeMember,
			Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages,
		})
	}

	name := sanitizeChannelName(event.Title)
	if name == "" {
		name = "sortie"
	}
	data := discordgo.GuildChannelCreateData{
		Name:                 name,
		Type:                 discordgo.ChannelTypeGuildText,
		PermissionOverwrites: overwrites,
	}
	if parentID != "" {
		data.ParentID = parentID
	}

	ch, err := s.GuildChannelCreateComplex(guildID, data)
	if err != nil {
		log.Printf("❌ Création salon sortie: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Erreur lors de la création du salon. Vérifie les permissions du bot.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	_, _ = s.ChannelMessageSend(ch.ID, fmt.Sprintf("💬 Salon créé pour la sortie **%s**. Visible uniquement par les participants.", event.Title))

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Salon créé : <#%s> (visible par toi et les participants confirmés).", ch.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

var channelNameSanitize = regexp.MustCompile(`[^a-z0-9\-]`)

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
		rowComponents = append(rowComponents, discordgo.Button{Label: "💬 Créer un salon", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("btn_create_sortie_chat_%s", messageID)})
	}
	if len(rowComponents) > 0 {
		components = append(components, discordgo.ActionsRow{Components: rowComponents})
	}
	return components
}
