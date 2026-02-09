package discord

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

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

	if len(options) == 0 {
		respondEphemeral(s, i.Interaction, "ℹ️ Il n'y a personne en liste d'attente.")
		return
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Choisissez une personne à promouvoir :",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "select_promote",
							Placeholder: "Sélectionner une personne à promouvoir",
							Options:     options,
						},
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
		if event.IsFinalized() {
			grantPrivateChannelAccess(s, event.PrivateChannelID, participant.UserID)
		}
		h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	}
	respondEphemeral(s, i.Interaction, fmt.Sprintf("✅ %s a été promu de la liste d'attente !", participant.Username))
}

// ── Remove participants ─────────────────────────────────────────────────────

func (h *Handler) respondRemoveSelect(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, event *entities.Event) {
	confirmed, err := h.eventUseCase.GetConfirmedParticipants(ctx, event.ID)
	if err != nil || len(confirmed) == 0 {
		respondEphemeral(s, i.Interaction, "ℹ️ Il n'y a aucun participant confirmé à retirer.")
		return
	}

	options := make([]discordgo.SelectMenuOption, 0, len(confirmed))
	for _, p := range confirmed {
		if p.UserID == event.CreatorID {
			continue
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       p.Username,
			Value:       fmt.Sprintf("remove_%d", p.ID),
			Description: fmt.Sprintf("Retirer %s de la sortie", p.Username),
		})
	}

	if len(options) == 0 {
		respondEphemeral(s, i.Interaction, "ℹ️ Il n'y a aucun participant à retirer.")
		return
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Sélectionne le(s) membre(s) à retirer de la sortie :",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "select_remove_user",
							Placeholder: "Choisir un ou plusieurs membres",
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
		respondEphemeral(s, i.Interaction, "❌ Événement non trouvé.")
		return
	}
	if i.Member.User.ID != event.CreatorID {
		respondEphemeral(s, i.Interaction, "❌ Seul l'organisateur peut retirer des participants.")
		return
	}

	h.respondRemoveSelect(ctx, s, i, event)
}

// HandleRemoveCommand is triggered by the /retirer slash command from the private channel.
func (h *Handler) HandleRemoveCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	event, err := h.eventUseCase.GetEventByPrivateChannelID(ctx, i.ChannelID)
	if err != nil {
		respondEphemeral(s, i.Interaction, "❌ Cette commande doit être utilisée dans le salon privé d'une sortie.")
		return
	}

	if i.Member.User.ID != event.CreatorID {
		respondEphemeral(s, i.Interaction, "❌ Seul l'organisateur peut retirer des participants.")
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
				respondEphemeral(s, i.Interaction, "❌ Événement non trouvé.")
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
			h.promoteNextFromWaitlist(s, ctx, event)
		}

		sendDM(s, participant.UserID, "🚪 Tu as été retiré de la sortie **"+event.Title+"** par l'organisateur.")
		removed = append(removed, fmt.Sprintf("<@%s>", participant.UserID))
	}

	if event != nil {
		h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	}

	if len(removed) == 0 {
		respondEphemeral(s, i.Interaction, "❌ Aucun participant n'a pu être retiré.")
		return
	}

	msg := fmt.Sprintf("✅ %s a été retiré de la sortie.", removed[0])
	if len(removed) > 1 {
		msg = fmt.Sprintf("✅ %d participants retirés : %s", len(removed), strings.Join(removed, ", "))
	}
	respondEphemeral(s, i.Interaction, msg)
}
