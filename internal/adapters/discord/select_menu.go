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

const maxSelectOptions = 25  // limite Discord par menu
const maxSelectMenus = 5     // 5×25 = 125 max en un message
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

	content := "**Liste d'attente** — Choisissez qui faire monter :"

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
			Description: "Faire monter",
		})
	}

	if len(options) == 0 {
		respondEphemeral(s, i.Interaction, "ℹ️ Il n'y a personne en liste d'attente.")
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
					Placeholder: fmt.Sprintf("Faire monter un membre (%d–%d)", start+1, end),
					Options:     chunk,
				},
			},
		})
	}

	if len(options) > maxSelectOptions*maxSelectMenus {
		content += fmt.Sprintf("\n\n_(Seuls les %d premiers sont dans les menus ; rouvrez « Gérer la liste d'attente » après des promotions pour voir la suite.)_", maxSelectOptions*maxSelectMenus)
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
		respondEphemeral(s, i.Interaction, "❌ Aucune sélection.")
		return
	}
	participantID, ok := parseParticipantID(data.Values[0], "promote_")
	if !ok {
		respondEphemeral(s, i.Interaction, "❌ Sélection invalide.")
		return
	}

	participant, quotaIncreased, err := h.participantUseCase.PromoteParticipant(ctx, participantID, i.Member.User.ID)
	if err != nil {
		var msg string
		switch {
		case errors.Is(err, domain.ErrNotOrganizer):
			msg = "❌ Seul l'organisateur peut faire monter des participants."
		case errors.Is(err, domain.ErrParticipantNotWaitlist):
			msg = "❌ Ce participant n'est plus en liste d'attente."
		case errors.Is(err, domain.ErrParticipantNotFound):
			msg = "❌ Participant introuvable."
		default:
			msg = "❌ Erreur lors de la promotion."
		}
		respondEphemeral(s, i.Interaction, msg)
		return
	}

	event, _ := h.eventUseCase.GetEventByID(ctx, participant.EventID)
	if event != nil {
		sendDM(s, participant.UserID, fmt.Sprintf("🎉 **Bonne nouvelle !** Tu as été promu pour **%s** par l'organisateur !", event.Title))
		if shouldGrantPrivateChannelOnPromote(event, time.Now()) {
			grantPrivateChannelAccess(s, event.PrivateChannelID, participant.UserID)
		}
		h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	}

	msg := fmt.Sprintf("✅ **%s** a été fait monter de la liste d'attente.", participant.Username)
	if quotaIncreased {
		msg += " Le nombre de places a été augmenté de 1 automatiquement."
	}
	respondEphemeral(s, i.Interaction, msg)
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
		display, username := displayAndUsername(s, h.guildID, p.UserID, p.Username)
		label := truncateLabel(waitlistOptionLabel(display, username), maxSelectLabelLen)
		options = append(options, discordgo.SelectMenuOption{
			Label:       label,
			Value:       fmt.Sprintf("remove_%d", p.ID),
			Description: "Retirer de la sortie",
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
			h.onSlotFreed(s, ctx, event)
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
