package discord

import (
	"context"
	"errors"
	"log"
	"strconv"

	"servbot/internal/domain"
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
		respondEphemeral(s, i.Interaction, h.translate("errors.datetime_required", nil))
		return
	}
	scheduledAt, err := pkgdiscord.ParseEventDateTime(dateStr, timeStr)
	if err != nil {
		// When ParseEventDateTime returns a domain error, resolve it via the domain code.
		if code := domain.Code(err); code != "" {
			respondEphemeral(s, i.Interaction, h.translate("errors."+code, nil))
		} else {
			respondEphemeral(s, i.Interaction, h.translate("errors.generic", nil))
		}
		return
	}
	slots, err := parseSlots(slotsStr)
	if err != nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.invalid_slots", nil))
		return
	}
	respondEphemeral(s, i.Interaction, h.translate("info.create_forum_post", nil))

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
			Content: h.translate("errors.create_forum_failed", nil),
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
		privChannelName = h.translate("ui.default_private_channel_name", nil)
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
			Content: h.translate("errors.create_private_channel_failed", nil),
		})
		return
	}
	grantPrivateChannelAccess(s, privCh.ID, creatorID)
	grantPrivateChannelAccess(s, privCh.ID, botID)

	_, _ = s.ChannelMessageSend(privCh.ID, h.translate("info.private_channel_intro", nil))

	questionsThreadID := ""
	questionsThread, threadErr := s.ThreadStart(privCh.ID, "Questions", discordgo.ChannelTypeGuildPrivateThread, 1440)
	if threadErr != nil {
		log.Printf("❌ Création thread privé Questions: %v", threadErr)
	} else {
		questionsThreadID = questionsThread.ID
		_ = s.ThreadMemberAdd(questionsThread.ID, creatorID)
		_ = s.ThreadMemberAdd(questionsThread.ID, botID)
		_, _ = s.ChannelMessageSend(questionsThread.ID, h.translate("info.questions_thread_intro", nil))
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
		WaitlistAuto:      true,
	}

	ctx := context.Background()
	if err := h.eventUseCase.CreateEvent(ctx, event, displayName); err != nil {
		log.Printf("❌ Erreur lors de la sauvegarde de l'événement: %v", err)
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: h.translate("errors.create_event_save_failed", nil),
		})
		return
	}

	h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	_ = s.MessageReactionAdd(event.ChannelID, event.MessageID, "✅")
}
