package discord

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"servbot/internal/domain"
	"servbot/internal/domain/entities"
	pkgdiscord "servbot/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

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

func (h *Handler) HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
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
		{ID: creatorID, Type: discordgo.PermissionOverwriteTypeMember, Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages},
		{ID: botID, Type: discordgo.PermissionOverwriteTypeMember, Allow: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages},
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
	if !event.OrganizerStep1FinalizedAt.IsZero() {
		respondEphemeral(s, i.Interaction, "🔒 Cette sortie est verrouillée (étape 1 finalisée). Aucune modification n'est possible.")
		return
	}

	slotsValue := ""
	if event.MaxSlots > 0 {
		slotsValue = fmt.Sprintf("%d", event.MaxSlots)
	}
	dateValue := ""
	timeValue := ""
	if !event.ScheduledAt.IsZero() {
		dateValue = event.ScheduledAt.Format("02/01/2006")
		timeValue = event.ScheduledAt.Format("15:04")
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

func (h *Handler) HandleEditModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	title, desc, dateStr, timeStr, slotsStr := pkgdiscord.ExtractModalData(data)

	var scheduledAt time.Time
	if dateStr != "" && timeStr != "" {
		var parseErr error
		scheduledAt, parseErr = pkgdiscord.ParseEventDateTime(dateStr, timeStr)
		if parseErr != nil {
			respondEphemeral(s, i.Interaction, "❌ "+parseErr.Error())
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

	event.Title = title
	event.Description = desc
	event.MaxSlots = slots
	if !scheduledAt.IsZero() {
		event.ScheduledAt = scheduledAt
	}

	if err := h.eventUseCase.UpdateEvent(ctx, event); err != nil {
		if errors.Is(err, domain.ErrCannotReduceSlots) {
			confirmedParticipants, _ := h.eventUseCase.GetConfirmedParticipants(ctx, event.ID)
			respondEphemeral(s, i.Interaction, fmt.Sprintf("❌ Impossible de réduire à %d places : il y a déjà %d participants confirmés. Retirez d'abord des participants.", slots, len(confirmedParticipants)))
			return
		}
		log.Printf("❌ Erreur lors de la mise à jour de l'événement: %v", err)
		respondEphemeral(s, i.Interaction, "❌ Erreur lors de la mise à jour.")
		return
	}

	h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	respondEphemeral(s, i.Interaction, "✅ Sortie modifiée avec succès !")
}
