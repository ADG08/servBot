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
		respondEphemeral(s, i.Interaction, "❌ Événement non trouvé.")
		return
	}
	if i.Member.User.ID != event.CreatorID {
		respondEphemeral(s, i.Interaction, "❌ Seul l'organisateur peut modifier la sortie.")
		return
	}
	if event.IsEditLocked() {
		respondEphemeral(s, i.Interaction, "🔒 Cette sortie est verrouillée. Aucune modification n'est possible.")
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
			Title:    "Modifier la sortie",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "title", Label: "Titre", Style: discordgo.TextInputShort, Required: true, Value: event.Title, Placeholder: placeholderTitle},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "desc", Label: "Détails (Lieu, etc.)", Style: discordgo.TextInputParagraph, Required: true, Value: event.Description, Placeholder: placeholderDesc},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "date", Label: "Date", Style: discordgo.TextInputShort, Required: true, Value: dateValue, Placeholder: placeholderDate},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "time", Label: "Heure", Style: discordgo.TextInputShort, Required: true, Value: timeValue, Placeholder: placeholderTime},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "slots", Label: "Nombre de places", Style: discordgo.TextInputShort, Required: false, Value: slotsValue, Placeholder: placeholderSlots},
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
			// When the parsing error comes from the domain (e.g. date in the past),
			// resolve it via the i18n adapter; otherwise, fall back to the raw error.
			if msg := pkgdiscord.DomainErrorMessage(parseErr); msg != "" {
				respondEphemeral(s, i.Interaction, "❌ "+msg)
			} else {
				respondEphemeral(s, i.Interaction, "❌ "+parseErr.Error())
			}
			return
		}
	}
	slots, err := parseSlots(slotsStr)
	if err != nil {
		respondEphemeral(s, i.Interaction, "❌ Nombre de places invalide (positif ou vide).")
		return
	}

	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByMessageID(ctx, i.Message.ID)
	if err != nil {
		respondEphemeral(s, i.Interaction, "❌ Événement non trouvé.")
		return
	}
	if i.Member.User.ID != event.CreatorID {
		respondEphemeral(s, i.Interaction, "❌ Seul l'organisateur peut modifier la sortie.")
		return
	}
	if event.IsEditLocked() {
		respondEphemeral(s, i.Interaction, "🔒 Cette sortie est verrouillée. Aucune modification n'est possible.")
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
			respondEphemeral(s, i.Interaction, "🔒 Cette sortie est verrouillée. Aucune modification n'est possible.")
		case errors.Is(err, domain.ErrCannotReduceSlots):
			confirmedParticipants, _ := h.eventUseCase.GetConfirmedParticipants(ctx, event.ID)
			respondEphemeral(s, i.Interaction, fmt.Sprintf("❌ Impossible de réduire à %d places : il y a déjà %d participants confirmés. Retirez d'abord des participants.", slots, len(confirmedParticipants)))
		default:
			log.Printf("❌ Erreur lors de la mise à jour de l'événement: %v", err)
			respondEphemeral(s, i.Interaction, "❌ Erreur lors de la mise à jour.")
		}
		return
	}

	h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	respondEphemeral(s, i.Interaction, "✅ Sortie modifiée avec succès !")
}
