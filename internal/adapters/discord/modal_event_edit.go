package discord

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"servbot/internal/domain"
	pkgdiscord "servbot/pkg/discord"
	"servbot/pkg/tz"

	"github.com/bwmarrin/discordgo"
)

// HandleEditEvent ouvre le modal d'édition d'une sortie existante.
func (h *Handler) HandleEditEvent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByMessageID(ctx, i.Message.ID)
	if err != nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.event_not_found", nil))
		return
	}
	if i.Member.User.ID != event.CreatorID {
		respondEphemeral(s, i.Interaction, h.translate("errors.only_organizer_can_edit", nil))
		return
	}
	if event.IsEditLocked() {
		respondEphemeral(s, i.Interaction, h.translate("errors.event_locked", nil))
		return
	}

	slotsValue := ""
	if event.MaxSlots > 0 {
		slotsValue = fmt.Sprintf("%d", event.MaxSlots)
	}
	dateValue, timeValue := "", ""
	if !event.ScheduledAt.IsZero() {
		t := event.ScheduledAt.In(tz.Paris)
		dateValue = t.Format("02/01/2006")
		timeValue = t.Format("15:04")
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "edit_event_modal",
			Title:    h.translate("ui.modal_edit_event_title", nil),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "title", Label: h.translate("ui.label_title", nil), Style: discordgo.TextInputShort, Required: true, Value: event.Title, Placeholder: h.translate("ui.placeholder_title", nil)},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "desc", Label: h.translate("ui.label_details", nil), Style: discordgo.TextInputParagraph, Required: true, Value: event.Description, Placeholder: h.translate("ui.placeholder_desc", nil)},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "date", Label: h.translate("ui.label_date", nil), Style: discordgo.TextInputShort, Required: true, Value: dateValue, Placeholder: h.translate("ui.placeholder_date", nil)},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "time", Label: h.translate("ui.label_time", nil), Style: discordgo.TextInputShort, Required: true, Value: timeValue, Placeholder: h.translate("ui.placeholder_time", nil)},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "slots", Label: h.translate("ui.label_slots", nil), Style: discordgo.TextInputShort, Required: false, Value: slotsValue, Placeholder: h.translate("ui.placeholder_slots", nil)},
				}},
			},
		},
	})
}

// HandleEditModalSubmit traite la soumission du modal d'édition.
func (h *Handler) HandleEditModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	title, desc, dateStr, timeStr, slotsStr := pkgdiscord.ExtractModalData(data)

	var scheduledAt time.Time
	if dateStr != "" && timeStr != "" {
		var parseErr error
		scheduledAt, parseErr = pkgdiscord.ParseEventDateTime(dateStr, timeStr)
		if parseErr != nil {
			if code := domain.Code(parseErr); code != "" {
				respondEphemeral(s, i.Interaction, h.translate("errors."+code, nil))
			} else {
				respondEphemeral(s, i.Interaction, h.translate("errors.generic", nil))
			}
			return
		}
	}
	slots, err := parseSlots(slotsStr)
	if err != nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.invalid_slots", nil))
		return
	}

	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByMessageID(ctx, i.Message.ID)
	if err != nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.event_not_found", nil))
		return
	}
	if i.Member.User.ID != event.CreatorID {
		respondEphemeral(s, i.Interaction, h.translate("errors.only_organizer_can_edit", nil))
		return
	}
	if event.IsEditLocked() {
		respondEphemeral(s, i.Interaction, h.translate("errors.event_locked", nil))
		return
	}

	event.Title = title
	event.Description = desc
	event.MaxSlots = slots
	if !scheduledAt.IsZero() {
		event.ScheduledAt = scheduledAt
	}

	if err := h.eventUseCase.UpdateEvent(ctx, event); err != nil {
		switch {
		case errors.Is(err, domain.ErrEventAlreadyFinalized):
			respondEphemeral(s, i.Interaction, h.translate("errors.event_locked", nil))
		case errors.Is(err, domain.ErrCannotReduceSlots):
			confirmedParticipants, _ := h.eventUseCase.GetConfirmedParticipants(ctx, event.ID)
			respondEphemeral(s, i.Interaction, h.translate("errors.cannot_reduce_slots", map[string]any{
				"Slots":          slots,
				"ConfirmedCount": len(confirmedParticipants),
			}))
		default:
			log.Printf("❌ Erreur lors de la mise à jour de l'événement: %v", err)
			respondEphemeral(s, i.Interaction, h.translate("errors.generic", nil))
		}
		return
	}

	h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	respondEphemeral(s, i.Interaction, h.translate("success.generic", nil))
}
