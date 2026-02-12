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
	"servbot/pkg/tz"

	"github.com/bwmarrin/discordgo"
)

const organizerValidationWindow = 48 * time.Hour

func isCasA(eventScheduledAt time.Time, now time.Time) bool {
	return !eventScheduledAt.IsZero() && eventScheduledAt.Sub(now) > organizerValidationWindow
}

func isCasB(eventScheduledAt time.Time, now time.Time) bool {
	return !eventScheduledAt.IsZero() && eventScheduledAt.After(now) && eventScheduledAt.Sub(now) <= organizerValidationWindow
}

func shouldGrantPrivateChannelOnPromote(event *entities.Event, now time.Time) bool {
	return event != nil && event.PrivateChannelID != "" && (event.IsFinalized() || isCasB(event.ScheduledAt, now))
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
		b.WriteString(h.translate("ui.dm_organizer_tri_title_link", map[string]any{"EventTitle": event.Title, "Link": link}))
	} else {
		b.WriteString(h.translate("ui.dm_organizer_tri_title", map[string]any{"EventTitle": event.Title}))
	}
	if !event.ScheduledAt.IsZero() {
		b.WriteString(h.translate("ui.dm_organizer_tri_date", map[string]any{"Date": event.ScheduledAt.In(tz.Paris).Format("02/01/2006 15:04")}))
	}
	if len(confirmed) > 0 {
		b.WriteString(h.translate("ui.dm_organizer_tri_confirmed_header", nil))
		for _, line := range confirmed {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}
	if len(waitlist) > 0 {
		b.WriteString(h.translate("ui.dm_organizer_tri_waitlist_header", nil))
		for _, line := range waitlist {
			b.WriteString(line + "\n")
		}
	}
	b.WriteString(h.translate("ui.dm_organizer_tri_footer", nil))
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
					Label:    h.translate("ui.btn_finalize_step1", nil),
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
	data := map[string]any{"EventTitle": eventTitle, "UserID": participant.UserID, "Username": participant.Username}
	if link := h.messageLink(channelID, messageID); link != "" {
		data["Link"] = link
		content = h.translate("ui.dm_organizer_new_registration_link", data)
	} else {
		content = h.translate("ui.dm_organizer_new_registration", data)
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    h.translate("ui.btn_accept", nil),
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("btn_organizer_accept_%d", participant.ID),
				},
				discordgo.Button{
					Label:    h.translate("ui.btn_refuse", nil),
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

func (h *Handler) sendWaitlistSlotFreedDM(s *discordgo.Session, event *entities.Event, next *entities.Participant) error {
	ch, err := s.UserChannelCreate(event.CreatorID)
	if err != nil || ch == nil {
		return fmt.Errorf("create organizer DM channel: %w", err)
	}
	data := map[string]any{"EventTitle": event.Title, "UserID": next.UserID, "Username": next.Username}
	var content string
	if link := h.messageLink(event.ChannelID, event.MessageID); link != "" {
		data["Link"] = link
		content = h.translate("ui.dm_waitlist_slot_freed_link", data)
	} else {
		content = h.translate("ui.dm_waitlist_slot_freed", data)
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    h.translate("ui.btn_accept", nil),
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("btn_waitlist_slot_accept_%d", next.ID),
				},
				discordgo.Button{
					Label:    h.translate("ui.btn_ignore", nil),
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("btn_waitlist_slot_ignore_%d", next.ID),
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

	// Acknowledge immediately to avoid the 3-second timeout.
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})

	event, err := h.eventUseCase.FinalizeOrganizerStep1(ctx, uint(eventID), userID)
	if err != nil {
		key := "errors.finalize_generic"
		switch {
		case errors.Is(err, domain.ErrNotOrganizer):
			key = "errors.finalize_only_organizer"
		case errors.Is(err, domain.ErrEventAlreadyFinalized):
			key = "errors.finalize_already_done"
		}
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: h.translate(key, nil),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	for _, p := range event.Participants {
		if p.Status != domain.StatusConfirmed || p.UserID == event.CreatorID {
			continue
		}
		var dmContent string
		if link := h.messageLink(event.ChannelID, event.MessageID); link != "" {
			dmContent = h.translate("dm.finalize_confirmed_with_link", map[string]any{
				"EventTitle": event.Title,
				"Link":       link,
			})
		} else {
			dmContent = h.translate("dm.finalize_confirmed_no_link", map[string]any{
				"EventTitle": event.Title,
			})
		}
		if !event.ScheduledAt.IsZero() {
			dmContent += "\n" + h.translate("ui.dm_date_line", map[string]any{"Date": event.ScheduledAt.In(tz.Paris).Format("02/01/2006 15:04")})
		}
		dmContent += h.translate("dm.finalize_confirmed_footer", nil)
		sendDM(s, p.UserID, dmContent)
		grantPrivateChannelAccess(s, event.PrivateChannelID, p.UserID)
	}

	if h.guildID != "" && !event.ScheduledAt.IsZero() {
		h.createDiscordScheduledEvent(s, event)
	}

	h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: h.translate("success.finalize_step1", nil),
		Flags:   discordgo.MessageFlagsEphemeral,
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
		location = h.translate("ui.calendar_location_placeholder", nil)
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
					Content: h.translate("errors.only_organizer_can_accept_candidate", nil),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
		})
		return
	}

	if participant.Status == domain.StatusWaitlist {
		promoted, _, err := h.participantUseCase.PromoteParticipant(ctx, participant.ID, userID)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: h.translate("errors.generic", nil),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		participant = promoted
	}

	sendDM(s, participant.UserID, h.translate("dm.organizer_accepted", map[string]any{
		"EventTitle": event.Title,
	}))
	grantPrivateChannelAccess(s, event.PrivateChannelID, participant.UserID)

	h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: h.translate("success.organizer_accepted", map[string]any{"Username": participant.Username}),
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
					Content: h.translate("errors.only_organizer_can_refuse_candidate", nil),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
		return
	}
	event, _ := h.eventUseCase.GetEventByID(ctx, participant.EventID)
	if event != nil {
		revokePrivateChannelAccess(s, event.PrivateChannelID, participant.UserID)
		_ = s.MessageReactionRemove(event.ChannelID, event.MessageID, reactionJoinEmoji, participant.UserID)
		ch, _ := s.UserChannelCreate(participant.UserID)
		if ch != nil {
			s.ChannelMessageSend(ch.ID, h.translate("dm.organizer_refused", map[string]any{
				"EventTitle": event.Title,
			}))
		}
		h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: h.translate("success.organizer_refused", map[string]any{"Username": participant.Username}),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *Handler) HandleWaitlistSlotAccept(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	customID := i.MessageComponentData().CustomID
	prefix := "btn_waitlist_slot_accept_"
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
		respondEphemeral(s, i.Interaction, h.translate("errors.only_organizer_can_accept", nil))
		return
	}
	if participant.Status != domain.StatusWaitlist {
		respondEphemeral(s, i.Interaction, h.translate("errors.participant_not_waitlist", nil))
		return
	}
	promoted, _, err := h.participantUseCase.PromoteParticipant(ctx, participant.ID, userID)
	if err != nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.finalize_generic", nil))
		return
	}
	sendDM(s, promoted.UserID, h.translate("dm.waitlist.promoted_auto", map[string]any{
		"EventTitle": event.Title,
	}))
	if shouldGrantPrivateChannelOnPromote(event, time.Now()) {
		grantPrivateChannelAccess(s, event.PrivateChannelID, promoted.UserID)
	}
	h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	respondEphemeral(s, i.Interaction, h.translate("success.participant_promoted", map[string]any{
		"Username": promoted.Username,
	}))
}

func (h *Handler) HandleWaitlistSlotIgnore(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	prefix := "btn_waitlist_slot_ignore_"
	if !strings.HasPrefix(customID, prefix) {
		return
	}
	respondEphemeral(s, i.Interaction, h.translate("info.no_promotion_done", nil))
}

func (h *Handler) processH48OrganizerDMs(s *discordgo.Session, ctx context.Context, now time.Time) {
	events, err := h.eventUseCase.EventsNeedingH48OrganizerDM(ctx, now)
	if err != nil {
		log.Printf("❌ Scheduler H-48: %v", err)
		return
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

func (h *Handler) processEditLock(s *discordgo.Session, ctx context.Context, now time.Time) {
	started, err := h.eventUseCase.FindStartedNonFinalizedEvents(ctx, now)
	if err != nil {
		log.Printf("❌ Scheduler edit-lock: %v", err)
		return
	}
	for _, e := range started {
		h.updateEmbed(ctx, s, e.ChannelID, e.MessageID)
	}
}
