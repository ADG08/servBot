package discord

import (
	"context"
	"errors"
	"log"
	"strconv"

	"servbot/internal/domain/entities"
	pkgdiscord "servbot/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

// parseSlots convertit la valeur du champ "Nombre de places" en entier
// (0 ou vide = illimité).
func parseSlots(slotsStr string) (int, error) {
	if slotsStr == "" || slotsStr == "0" {
		return 0, nil
	}
	n, err := strconv.Atoi(slotsStr)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, errors.New("invalid")
	}
	return n, nil
}

// handleCreateEventModalSubmit gère la soumission du modal de création de sortie.
func (h *Handler) handleCreateEventModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
	title, desc, dateStr, timeStr, slotsStr := pkgdiscord.ExtractModalData(data)

	if dateStr == "" || timeStr == "" {
		respondEphemeral(s, i.Interaction, "❌ Date et heure requises (JJ/MM/AAAA et HH:MM).")
		return
	}
	scheduledAt, err := pkgdiscord.ParseEventDateTime(dateStr, timeStr)
	if err != nil {
		respondEphemeral(s, i.Interaction, "❌ "+err.Error())
		return
	}
	slots, err := parseSlots(slotsStr)
	if err != nil {
		respondEphemeral(s, i.Interaction, "❌ Nombre de places invalide (positif ou vide).")
		return
	}

	respondEphemeral(s, i.Interaction, "Création du post dans le forum...")

	user := i.Member.User
	displayName := resolveDisplayName(i.Member)
	embed := pkgdiscord.BuildNewEventEmbed(user.ID, desc, scheduledAt, slots, displayName, user.AvatarURL("256"))

	threadData := &discordgo.ThreadStart{
		Name:                title,
		AutoArchiveDuration: 1440,
		Type:                discordgo.ChannelTypeGuildPublicThread,
	}
	messageData := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
	}

	thread, err := s.ForumThreadStartComplex(h.forumChannelID, threadData, messageData)
	if err != nil {
		log.Println("❌ Erreur création forum post:", err)
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Erreur lors de la création du post (Vérifie que le Bot a la permission 'Créer des messages publics' et 'Créer des fils').",
		})
		return
	}

	message, err := s.ChannelMessage(thread.ID, thread.ID)
	msgID := thread.ID
	if err == nil && message != nil {
		msgID = message.ID
	}

	creatorID := i.Member.User.ID
	guildID := i.GuildID
	botID := s.State.User.ID

	parentID := ""
	if ch, err := s.Channel(h.forumChannelID); err == nil && ch != nil && ch.ParentID != "" {
		parentID = ch.ParentID
	}
	overwrites := []*discordgo.PermissionOverwrite{
		{ID: guildID, Type: discordgo.PermissionOverwriteTypeRole, Deny: discordgo.PermissionViewChannel},
	}
	privChannelName := sanitizeChannelName(title)
	if privChannelName == "" {
		privChannelName = "sortie"
	}
	privData := discordgo.GuildChannelCreateData{
		Name:                 privChannelName,
		Type:                 discordgo.ChannelTypeGuildText,
		PermissionOverwrites: overwrites,
	}
	if parentID != "" {
		privData.ParentID = parentID
	}
	privCh, err := s.GuildChannelCreateComplex(guildID, privData)
	if err != nil {
		log.Printf("❌ Création salon privé sortie: %v", err)
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ Post forum créé mais erreur lors de la création du salon privé. Vérifie les permissions du bot (Gérer les salons).",
		})
		return
	}
	grantPrivateChannelAccess(s, privCh.ID, creatorID)
	grantPrivateChannelAccess(s, privCh.ID, botID)

	_, _ = s.ChannelMessageSend(privCh.ID, "💬 Salon privé pour cette sortie. Les questions des participants te seront relayées ici par le bot (thread **Questions**).")

	questionsThreadID := ""
	questionsThread, threadErr := s.ThreadStart(privCh.ID, "Questions", discordgo.ChannelTypeGuildPrivateThread, 1440)
	if threadErr != nil {
		log.Printf("❌ Création thread privé Questions: %v", threadErr)
	} else {
		questionsThreadID = questionsThread.ID
		_ = s.ThreadMemberAdd(questionsThread.ID, creatorID)
		_ = s.ThreadMemberAdd(questionsThread.ID, botID)
		_, _ = s.ChannelMessageSend(questionsThread.ID, "Les questions des participants te seront relayées ici par le bot.")
	}

	event := &entities.Event{
		MessageID:         msgID,
		ChannelID:         thread.ID,
		CreatorID:         creatorID,
		Title:             title,
		Description:       desc,
		MaxSlots:          slots,
		ScheduledAt:       scheduledAt,
		PrivateChannelID:  privCh.ID,
		QuestionsThreadID: questionsThreadID,
	}

	ctx := context.Background()
	if err := h.eventUseCase.CreateEvent(ctx, event, displayName); err != nil {
		log.Printf("❌ Erreur lors de la sauvegarde de l'événement: %v", err)
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ Erreur lors de la sauvegarde de l'événement.",
		})
		return
	}

	h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	_ = s.MessageReactionAdd(event.ChannelID, event.MessageID, "✅")
}
