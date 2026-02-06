package discord

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"servbot/internal/domain"

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

func (h *Handler) HandleManageWaitlist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByMessageID(ctx, i.Message.ID)
	if err != nil {
		respondEphemeral(s, i.Interaction, "❌ Événement non trouvé.")
		return
	}
	if i.Member.User.ID != event.CreatorID {
		respondEphemeral(s, i.Interaction, "❌ Seul l'organisateur peut gérer la liste d'attente.")
		return
	}

	waitlistParticipants, err := h.eventUseCase.GetWaitlistParticipants(ctx, event.ID)
	if err != nil || len(waitlistParticipants) == 0 {
		respondEphemeral(s, i.Interaction, "ℹ️ Il n'y a personne en liste d'attente.")
		return
	}

	options := make([]discordgo.SelectMenuOption, 0, len(waitlistParticipants))
	for _, p := range waitlistParticipants {
		if p.ID == 0 {
			continue
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       p.Username,
			Value:       fmt.Sprintf("promote_%d", p.ID),
			Description: fmt.Sprintf("Promouvoir %s de la liste d'attente", p.Username),
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Choisissez une personne à promouvoir :",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{CustomID: "select_promote", Placeholder: "Sélectionner une personne à promouvoir", Options: options},
					},
				},
			},
		},
	})
}

func (h *Handler) HandlePromote(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}
	participantID, ok := parseParticipantID(data.Values[0], "promote_")
	if !ok {
		return
	}

	participant, err := h.participantUseCase.PromoteParticipant(ctx, participantID, i.Member.User.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotOrganizer) {
			respondEphemeral(s, i.Interaction, "❌ Seul l'organisateur peut promouvoir des participants.")
		}
		return
	}

	event, _ := h.eventUseCase.GetEventByID(ctx, participant.EventID)
	if event != nil {
		sendDM(s, participant.UserID, fmt.Sprintf("🎉 **Bonne nouvelle !** Tu as été promu pour **%s** par l'organisateur !", event.Title))
		h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	}
	respondEphemeral(s, i.Interaction, fmt.Sprintf("✅ %s a été promu de la liste d'attente !", participant.Username))
}

func (h *Handler) HandleRemoveParticipant(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByMessageID(ctx, i.Message.ID)
	if err != nil {
		respondEphemeral(s, i.Interaction, "❌ Événement non trouvé.")
		return
	}
	if i.Member.User.ID != event.CreatorID {
		respondEphemeral(s, i.Interaction, "❌ Seul l'organisateur peut retirer des participants.")
		return
	}

	confirmedParticipants, err := h.eventUseCase.GetConfirmedParticipants(ctx, event.ID)
	if err != nil || len(confirmedParticipants) == 0 {
		respondEphemeral(s, i.Interaction, "ℹ️ Il n'y a aucun participant confirmé à retirer.")
		return
	}

	options := make([]discordgo.SelectMenuOption, 0, len(confirmedParticipants))
	for _, p := range confirmedParticipants {
		if p.ID == 0 {
			continue
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       p.Username,
			Value:       fmt.Sprintf("remove_%d", p.ID),
			Description: fmt.Sprintf("Retirer %s de l'événement", p.Username),
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Choisissez un participant à retirer :",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{CustomID: "select_remove", Placeholder: "Sélectionner un participant à retirer", Options: options},
					},
				},
			},
		},
	})
}

func (h *Handler) HandleRemove(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}
	participantID, ok := parseParticipantID(data.Values[0], "remove_")
	if !ok {
		return
	}

	participant, err := h.participantUseCase.RemoveParticipant(ctx, participantID, i.Member.User.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotOrganizer) {
			respondEphemeral(s, i.Interaction, "❌ Seul l'organisateur peut retirer des participants.")
		}
		return
	}

	event, _ := h.eventUseCase.GetEventByID(ctx, participant.EventID)
	if event != nil {
		luckyWinner, err := h.participantUseCase.GetNextWaitlistParticipant(ctx, event.ID)
		if err == nil {
			sendDM(s, luckyWinner.UserID, fmt.Sprintf("🎉 **Bonne nouvelle !** Une place s'est libérée pour **%s**, tu es maintenant parmi les confirmés !", event.Title))
		}
		h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	}
	respondEphemeral(s, i.Interaction, fmt.Sprintf("✅ %s a été retiré de l'événement.", participant.Username))
}
