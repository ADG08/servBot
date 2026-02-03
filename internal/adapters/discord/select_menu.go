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

func (h *Handler) HandleManageWaitlist(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
				Content: "❌ Seul l'organisateur peut gérer la liste d'attente.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	waitlistParticipants, err := h.eventUseCase.GetWaitlistParticipants(ctx, event.ID)
	if err != nil || len(waitlistParticipants) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "ℹ️ Il n'y a personne en liste d'attente.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
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
	selectedValue := data.Values[0]
	idStr, ok := strings.CutPrefix(selectedValue, "promote_")
	if !ok {
		return
	}
	participantID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return
	}

	participant, err := h.participantUseCase.PromoteParticipant(ctx, uint(participantID), i.Member.User.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotOrganizer) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Seul l'organisateur peut promouvoir des participants.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
		return
	}

	event, _ := h.eventUseCase.GetEventByID(ctx, participant.EventID)
	ch, _ := s.UserChannelCreate(participant.UserID)
	if ch != nil && event != nil {
		s.ChannelMessageSend(ch.ID, fmt.Sprintf("🎉 **Bonne nouvelle !** Tu as été promu pour **%s** par l'organisateur !", event.Title))
	}
	if event != nil {
		h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ %s a été promu de la liste d'attente !", participant.Username),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *Handler) HandleRemoveParticipant(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
				Content: "❌ Seul l'organisateur peut retirer des participants.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	confirmedParticipants, err := h.eventUseCase.GetConfirmedParticipants(ctx, event.ID)
	if err != nil || len(confirmedParticipants) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "ℹ️ Il n'y a aucun participant confirmé à retirer.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
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
	selectedValue := data.Values[0]
	idStr, ok := strings.CutPrefix(selectedValue, "remove_")
	if !ok {
		return
	}
	participantID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return
	}

	participant, err := h.participantUseCase.RemoveParticipant(ctx, uint(participantID), i.Member.User.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotOrganizer) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Seul l'organisateur peut retirer des participants.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
		return
	}

	event, _ := h.eventUseCase.GetEventByID(ctx, participant.EventID)
	if event != nil {
		luckyWinner, err := h.participantUseCase.GetNextWaitlistParticipant(ctx, event.ID)
		if err == nil {
			ch, err := s.UserChannelCreate(luckyWinner.UserID)
			if err == nil && ch != nil {
				s.ChannelMessageSend(ch.ID, fmt.Sprintf("🎉 **Bonne nouvelle !** Une place s'est libérée pour **%s**, tu es maintenant inscrit !", event.Title))
			}
		}
		h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ %s a été retiré de l'événement.", participant.Username),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
