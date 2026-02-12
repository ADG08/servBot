package discord

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"servbot/internal/domain"
	"servbot/internal/domain/entities"

	"github.com/bwmarrin/discordgo"
)

func parseParticipantID(value, prefix string) (uint, bool) {
	idStr, ok := strings.CutPrefix(value, prefix)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}

// ── Waitlist (promote) ──────────────────────────────────────────────────────

const maxSelectOptions = 25 // limite Discord par menu
const maxSelectMenus = 5    // 5×25 = 125 max en un message
const maxSelectLabelLen = 100

func displayAndUsername(s *discordgo.Session, guildID, userID, fallback string) (display, username string) {
	display = fallback
	username = fallback
	if guildID == "" {
		return display, username
	}
	member, err := s.GuildMember(guildID, userID)
	if err != nil || member == nil || member.User == nil {
		return display, username
	}
	username = member.User.Username
	display = resolveDisplayName(member)
	if display == "" {
		display = username
	}
	return display, username
}

func truncateLabel(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func waitlistOptionLabel(display, username string) string {
	if display == username {
		return display
	}
	return display + " • " + username
}

func (h *Handler) HandleManageWaitlist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByMessageID(ctx, i.Message.ID)
	if err != nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.event_not_found", nil))
		return
	}
	if i.Member.User.ID != event.CreatorID {
		respondEphemeral(s, i.Interaction, h.translate("errors.only_organizer_can_manage_waitlist", nil))
		return
	}

	waitlistParticipants, err := h.eventUseCase.GetWaitlistParticipants(ctx, event.ID)
	if err != nil || len(waitlistParticipants) == 0 {
		respondEphemeral(s, i.Interaction, h.translate("info.waitlist.empty", nil))
		return
	}

	content := h.translate("ui.waitlist_manage_intro", nil)

	options := make([]discordgo.SelectMenuOption, 0, len(waitlistParticipants))
	for _, p := range waitlistParticipants {
		if p.ID == 0 {
			continue
		}
		display, username := displayAndUsername(s, h.guildID, p.UserID, p.Username)
		label := truncateLabel(waitlistOptionLabel(display, username), maxSelectLabelLen)
		options = append(options, discordgo.SelectMenuOption{
			Label:       label,
			Value:       fmt.Sprintf("promote_%d", p.ID),
			Description: h.translate("ui.waitlist_option_promote", nil),
		})
	}

	if len(options) == 0 {
		respondEphemeral(s, i.Interaction, h.translate("info.waitlist.empty", nil))
		return
	}

	var components []discordgo.MessageComponent
	for i := 0; i < maxSelectMenus && i*maxSelectOptions < len(options); i++ {
		start := i * maxSelectOptions
		end := min(start+maxSelectOptions, len(options))
		chunk := options[start:end]
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    fmt.Sprintf("select_promote_%d", i),
					Placeholder: h.translate("ui.waitlist_placeholder_range", map[string]any{"Start": start + 1, "End": end}),
					Options:     chunk,
				},
			},
		})
	}

	if len(options) > maxSelectOptions*maxSelectMenus {
		content += h.translate("ui.waitlist_manage_truncated", map[string]any{"Max": maxSelectOptions * maxSelectMenus})
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: components,
		},
	})
}

func (h *Handler) HandlePromote(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		respondEphemeral(s, i.Interaction, h.translate("errors.no_selection", nil))
		return
	}
	participantID, ok := parseParticipantID(data.Values[0], "promote_")
	if !ok {
		respondEphemeral(s, i.Interaction, h.translate("errors.invalid_selection", nil))
		return
	}

	participant, quotaIncreased, err := h.participantUseCase.PromoteParticipant(ctx, participantID, i.Member.User.ID)
	if err != nil {
		var key string
		switch {
		case errors.Is(err, domain.ErrNotOrganizer):
			key = "errors.only_organizer_can_accept"
		case errors.Is(err, domain.ErrParticipantNotWaitlist):
			key = "errors.participant_not_waitlist"
		case errors.Is(err, domain.ErrParticipantNotFound):
			key = "errors.participant_not_found"
		default:
			key = "errors.finalize_generic"
		}
		respondEphemeral(s, i.Interaction, h.translate(key, nil))
		return
	}

	event, _ := h.eventUseCase.GetEventByID(ctx, participant.EventID)
	if event != nil {
		sendDM(s, participant.UserID, h.translate("dm.waitlist.promoted_by_organizer", map[string]any{"EventTitle": event.Title}))
		if shouldGrantPrivateChannelOnPromote(event, time.Now()) {
			grantPrivateChannelAccess(s, event.PrivateChannelID, participant.UserID)
		}
		h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	}

	msg := h.translate("success.participant_promoted", map[string]any{"Username": participant.Username})
	if quotaIncreased {
		msg = h.translate("success.participant_promoted_with_quota", map[string]any{"Username": participant.Username})
	}
	respondEphemeral(s, i.Interaction, msg)
}

// ── Remove participants ─────────────────────────────────────────────────────

func (h *Handler) respondRemoveSelect(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, event *entities.Event) {
	confirmed, err := h.eventUseCase.GetConfirmedParticipants(ctx, event.ID)
	if err != nil || len(confirmed) == 0 {
		respondEphemeral(s, i.Interaction, h.translate("info.no_confirmed_to_remove", nil))
		return
	}

	options := make([]discordgo.SelectMenuOption, 0, len(confirmed))
	for _, p := range confirmed {
		if p.UserID == event.CreatorID {
			continue
		}
		display, username := displayAndUsername(s, h.guildID, p.UserID, p.Username)
		label := truncateLabel(waitlistOptionLabel(display, username), maxSelectLabelLen)
		options = append(options, discordgo.SelectMenuOption{
			Label:       label,
			Value:       fmt.Sprintf("remove_%d", p.ID),
			Description: h.translate("ui.remove_option_description", nil),
		})
	}

	if len(options) == 0 {
		respondEphemeral(s, i.Interaction, h.translate("info.no_confirmed_to_remove", nil))
		return
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: h.translate("ui.remove_select_intro", nil),
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "select_remove_user",
							Placeholder: h.translate("ui.remove_placeholder", nil),
							Options:     options,
							MaxValues:   len(options),
						},
					},
				},
			},
		},
	})
}

// HandleRemoveParticipant is triggered by the embed "Retirer" button.
func (h *Handler) HandleRemoveParticipant(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByMessageID(ctx, i.Message.ID)
	if err != nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.event_not_found", nil))
		return
	}
	if i.Member.User.ID != event.CreatorID {
		respondEphemeral(s, i.Interaction, h.translate("errors.only_organizer_can_remove", nil))
		return
	}

	h.respondRemoveSelect(ctx, s, i, event)
}

// HandleRemoveCommand is triggered by the /retirer slash command from the private channel.
func (h *Handler) HandleRemoveCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	event, err := h.eventUseCase.GetEventByPrivateChannelID(ctx, i.ChannelID)
	if err != nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.remove_command_wrong_channel", nil))
		return
	}

	if i.Member.User.ID != event.CreatorID {
		respondEphemeral(s, i.Interaction, h.translate("errors.only_organizer_can_remove", nil))
		return
	}

	h.respondRemoveSelect(ctx, s, i, event)
}

// HandleRemoveUserSelect processes the remove select menu (shared by button and /retirer).
func (h *Handler) HandleRemoveUserSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}

	var event *entities.Event
	removed := make([]string, 0, len(data.Values))

	for _, val := range data.Values {
		pID, ok := parseParticipantID(val, "remove_")
		if !ok {
			continue
		}

		participant, err := h.participantUseCase.GetParticipantByID(ctx, pID)
		if err != nil {
			continue
		}

		// Resolve event once from the first valid participant.
		if event == nil {
			event, err = h.eventUseCase.GetEventByID(ctx, participant.EventID)
			if err != nil {
				respondEphemeral(s, i.Interaction, h.translate("errors.event_not_found", nil))
				return
			}
		}

		wasConfirmed, err := h.participantUseCase.LeaveEvent(ctx, event.ID, participant.UserID)
		if err != nil {
			continue
		}

		_ = s.MessageReactionRemove(event.ChannelID, event.MessageID, reactionJoinEmoji, participant.UserID)
		revokePrivateChannelAccess(s, event.PrivateChannelID, participant.UserID)

		if wasConfirmed {
			h.onSlotFreed(s, ctx, event)
		}

		sendDM(s, participant.UserID, h.translate("dm.removed_by_organizer", map[string]any{"EventTitle": event.Title}))
		removed = append(removed, fmt.Sprintf("<@%s>", participant.UserID))
	}

	if event != nil {
		h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	}

	if len(removed) == 0 {
		respondEphemeral(s, i.Interaction, h.translate("errors.no_participant_removed", nil))
		return
	}

	msg := h.translate("success.participant_removed_single", map[string]any{"Mention": removed[0]})
	if len(removed) > 1 {
		msg = h.translate("success.participant_removed_many", map[string]any{
			"Count":    len(removed),
			"Mentions": strings.Join(removed, ", "),
		})
	}
	respondEphemeral(s, i.Interaction, msg)
}
