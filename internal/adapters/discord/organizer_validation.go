package discord

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"servbot/internal/domain"
	"servbot/internal/domain/entities"
	pkgdiscord "servbot/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

const organizerValidationWindow = 48 * time.Hour

func isCasA(eventScheduledAt time.Time, now time.Time) bool {
	return !eventScheduledAt.IsZero() && eventScheduledAt.Sub(now) > organizerValidationWindow
}

func isCasB(eventScheduledAt time.Time, now time.Time) bool {
	return !eventScheduledAt.IsZero() && eventScheduledAt.After(now) && eventScheduledAt.Sub(now) <= organizerValidationWindow
}

func (h *Handler) messageLink(channelID, messageID string) string {
	if h.guildID == "" || channelID == "" || messageID == "" {
		return ""
	}
	return fmt.Sprintf("https://discord.com/channels/%s/%s/%s", h.guildID, channelID, messageID)
}

func (h *Handler) buildOrganizerTriDMContent(event *eventWithParticipants) string {
	participantsSansOrga := make([]entities.Participant, 0, len(event.Participants))
	for _, p := range event.Participants {
		if p.UserID != event.CreatorID {
			participantsSansOrga = append(participantsSansOrga, p)
		}
	}
	confirmed, waitlist := pkgdiscord.FormatParticipants(participantsSansOrga)
	var b strings.Builder
	if link := h.messageLink(event.ChannelID, event.MessageID); link != "" {
		b.WriteString(fmt.Sprintf("**Tri pour la sortie :** [%s](%s)\n\n", event.Title, link))
	} else {
		b.WriteString(fmt.Sprintf("**Tri pour la sortie : %s**\n\n", event.Title))
	}
	if !event.ScheduledAt.IsZero() {
		b.WriteString(fmt.Sprintf("📅 %s\n\n", pkgdiscord.FormatEventDateTime(event.ScheduledAt)))
	}
	if len(confirmed) > 0 {
		b.WriteString("✅ **Potentiels intéressés confirmés :**\n")
		for _, line := range confirmed {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}
	if len(waitlist) > 0 {
		b.WriteString("⏳ **Potentiels intéressés en attente :**\n")
		for _, line := range waitlist {
			b.WriteString(line + "\n")
		}
	}
	b.WriteString("\nClique sur **Finaliser l'étape 1** quand tu as validé la liste.")
	return b.String()
}

type eventWithParticipants struct {
	ID                          uint
	MessageID                   string
	ChannelID                   string
	CreatorID                   string
	Title                       string
	Description                 string
	MaxSlots                    int
	ScheduledAt                 time.Time
	OrganizerValidationDMSentAt time.Time
	OrganizerStep1FinalizedAt   time.Time
	Participants                []entities.Participant
}

func (h *Handler) sendOrganizerValidationDM(s *discordgo.Session, event *eventWithParticipants) error {
	ch, err := s.UserChannelCreate(event.CreatorID)
	if err != nil || ch == nil {
		return fmt.Errorf("create organizer DM channel: %w", err)
	}
	content := h.buildOrganizerTriDMContent(event)
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Finaliser l'étape 1",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("btn_organizer_finalize_%d", event.ID),
				},
			},
		},
	}
	_, err = s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content:    content,
		Components: components,
	})
	return err
}

func (h *Handler) sendOrganizerAcceptRefuseDM(s *discordgo.Session, eventTitle, organizerID, channelID, messageID string, participant *entities.Participant) error {
	ch, err := s.UserChannelCreate(organizerID)
	if err != nil || ch == nil {
		return fmt.Errorf("create organizer DM channel: %w", err)
	}
	var content string
	if link := h.messageLink(channelID, messageID); link != "" {
		content = fmt.Sprintf("**Nouvelle inscription pour :** [%s](%s)\n\n<@%s> (%s) s'est déclaré potentiellement intéressé.\n\nAccepter ou refuser ce potentiel intéressé ?", eventTitle, link, participant.UserID, participant.Username)
	} else {
		content = fmt.Sprintf("**Nouvelle inscription pour : %s**\n\n<@%s> (%s) s'est déclaré potentiellement intéressé.\n\nAccepter ou refuser ce potentiel intéressé ?", eventTitle, participant.UserID, participant.Username)
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Accepter",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("btn_organizer_accept_%d", participant.ID),
				},
				discordgo.Button{
					Label:    "Refuser",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("btn_organizer_refuse_%d", participant.ID),
				},
			},
		},
	}
	_, err = s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
		Content:    content,
		Components: components,
	})
	return err
}

func eventToEventWithParticipants(e *entities.Event) *eventWithParticipants {
	if e == nil {
		return nil
	}
	return &eventWithParticipants{
		ID:                          e.ID,
		MessageID:                   e.MessageID,
		ChannelID:                   e.ChannelID,
		CreatorID:                   e.CreatorID,
		Title:                       e.Title,
		Description:                 e.Description,
		MaxSlots:                    e.MaxSlots,
		ScheduledAt:                 e.ScheduledAt,
		OrganizerValidationDMSentAt: e.OrganizerValidationDMSentAt,
		OrganizerStep1FinalizedAt:   e.OrganizerStep1FinalizedAt,
		Participants:                e.Participants,
	}
}

func interactionUserID(i *discordgo.InteractionCreate) string {
	if i.User != nil {
		return i.User.ID
	}
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	return ""
}

func (h *Handler) HandleOrganizerFinalizeStep1(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	customID := i.MessageComponentData().CustomID
	prefix := "btn_organizer_finalize_"
	if !strings.HasPrefix(customID, prefix) {
		return
	}
	eventIDStr := strings.TrimPrefix(customID, prefix)
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		return
	}
	userID := interactionUserID(i)

	event, err := h.eventUseCase.FinalizeOrganizerStep1(ctx, uint(eventID), userID)
	if err != nil {
		msg := "❌ Erreur lors de la finalisation."
		switch {
		case errors.Is(err, domain.ErrNotOrganizer):
			msg = "❌ Seul l'organisateur de cette sortie peut finaliser l'étape 1."
		case errors.Is(err, domain.ErrEventAlreadyFinalized):
			msg = "ℹ️ Cette sortie a déjà été finalisée."
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: msg, Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	for _, p := range event.Participants {
		if p.Status != domain.StatusConfirmed || p.UserID == event.CreatorID {
			continue
		}
		var dmContent string
		if link := h.messageLink(event.ChannelID, event.MessageID); link != "" {
			dmContent = fmt.Sprintf("🎉 **Ta participation à [%s](%s) est confirmée !**", event.Title, link)
		} else {
			dmContent = fmt.Sprintf("🎉 **Ta participation à %s est confirmée !**", event.Title)
		}
		if !event.ScheduledAt.IsZero() {
			dmContent += fmt.Sprintf("\n📅 %s", pkgdiscord.FormatEventDateTime(event.ScheduledAt))
		}
		dmContent += "\nÀ bientôt !"
		sendDM(s, p.UserID, dmContent)
	}

	if h.guildID != "" && !event.ScheduledAt.IsZero() {
		h.createDiscordScheduledEvent(s, event)
	}

	h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ Étape 1 finalisée ! Les participants confirmés ont été notifiés et l'événement a été ajouté au calendrier Discord.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *Handler) createDiscordScheduledEvent(s *discordgo.Session, event *entities.Event) {
	startTime := event.ScheduledAt
	endTime := startTime.Add(2 * time.Hour)

	location := event.Description
	if len(location) > 100 {
		location = location[:97] + "..."
	}
	if location == "" {
		location = "Voir les détails de la sortie"
	}

	_, err := s.GuildScheduledEventCreate(h.guildID, &discordgo.GuildScheduledEventParams{
		Name:               event.Title,
		Description:        event.Description,
		ScheduledStartTime: &startTime,
		ScheduledEndTime:   &endTime,
		PrivacyLevel:       discordgo.GuildScheduledEventPrivacyLevelGuildOnly,
		EntityType:         discordgo.GuildScheduledEventEntityTypeExternal,
		EntityMetadata: &discordgo.GuildScheduledEventEntityMetadata{
			Location: location,
		},
	})
	if err != nil {
		log.Printf("❌ Création événement calendrier Discord (event %d): %v", event.ID, err)
	}
}

func (h *Handler) HandleOrganizerAccept(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	customID := i.MessageComponentData().CustomID
	prefix := "btn_organizer_accept_"
	if !strings.HasPrefix(customID, prefix) {
		return
	}
	participantIDStr := strings.TrimPrefix(customID, prefix)
	participantID, err := strconv.ParseUint(participantIDStr, 10, 32)
	if err != nil {
		return
	}
	participant, err := h.participantUseCase.GetParticipantByID(ctx, uint(participantID))
	if err != nil {
		return
	}
	event, err := h.eventUseCase.GetEventByID(ctx, participant.EventID)
	if err != nil {
		return
	}
	userID := interactionUserID(i)
	if event.CreatorID != userID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Seul l'organisateur peut accepter ce potentiel intéressé.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ %s a été accepté.", participant.Username),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *Handler) HandleOrganizerRefuse(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	customID := i.MessageComponentData().CustomID
	prefix := "btn_organizer_refuse_"
	if !strings.HasPrefix(customID, prefix) {
		return
	}
	participantIDStr := strings.TrimPrefix(customID, prefix)
	participantID, err := strconv.ParseUint(participantIDStr, 10, 32)
	if err != nil {
		return
	}
	userID := interactionUserID(i)
	participant, err := h.participantUseCase.RemoveParticipant(ctx, uint(participantID), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotOrganizer) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Seul l'organisateur peut refuser ce potentiel intéressé.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
		return
	}
	event, _ := h.eventUseCase.GetEventByID(ctx, participant.EventID)
	if event != nil {
		_ = s.MessageReactionRemove(event.ChannelID, event.MessageID, reactionJoinEmoji, participant.UserID)
		ch, _ := s.UserChannelCreate(participant.UserID)
		if ch != nil {
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("L'organisateur de **%s** a refusé ton inscription.", event.Title))
		}
		h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ %s a été refusé et retiré de la sortie.", participant.Username),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *Handler) RunOrganizerValidationScheduler(s *discordgo.Session) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	ctx := context.Background()
	for range ticker.C {
		now := time.Now()
		events, err := h.eventUseCase.EventsNeedingH48OrganizerDM(ctx, now)
		if err != nil {
			log.Printf("❌ Scheduler H-48: %v", err)
			continue
		}
		for _, e := range events {
			eventFull, err := h.eventUseCase.GetEventByID(ctx, e.ID)
			if err != nil || eventFull == nil {
				continue
			}
			evWP := eventToEventWithParticipants(eventFull)
			if err := h.sendOrganizerValidationDM(s, evWP); err != nil {
				log.Printf("❌ Envoi MP H-48 organisateur (event %d): %v", e.ID, err)
				continue
			}
			if err := h.eventUseCase.MarkOrganizerValidationDMSent(ctx, e.ID); err != nil {
				log.Printf("❌ MarkOrganizerValidationDMSent (event %d): %v", e.ID, err)
			}
		}
	}
}
