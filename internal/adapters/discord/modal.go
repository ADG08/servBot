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

func (h *Handler) HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	title, desc, dateStr, timeStr, slotsStr := pkgdiscord.ExtractModalData(data)

	scheduledAt, err := pkgdiscord.ParseEventDateTime(dateStr, timeStr)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ " + err.Error(),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var slots int
	if slotsStr == "" || slotsStr == "0" {
		slots = 0
	} else {
		var parseErr error
		slots, parseErr = strconv.Atoi(slotsStr)
		if parseErr != nil || slots < 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Nombre de places invalide (positif ou vide).",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Création du post dans le forum...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	user := i.Member.User
	displayName := user.Username
	if i.Member.Nick != "" {
		displayName = i.Member.Nick
	}
	embed := pkgdiscord.BuildNewEventEmbed(user.ID, desc, scheduledAt, slots, displayName, user.AvatarURL("256"))

	btns := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Je participe", Style: discordgo.SuccessButton, CustomID: "btn_join"},
			discordgo.Button{Label: "Se désister", Style: discordgo.DangerButton, CustomID: "btn_leave"},
		}},
	}

	threadData := &discordgo.ThreadStart{
		Name:                title,
		AutoArchiveDuration: 1440,
		Type:                discordgo.ChannelTypeGuildPublicThread,
	}
	messageData := &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: btns,
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

	event := &entities.Event{
		MessageID:   msgID,
		ChannelID:   thread.ID,
		CreatorID:   i.Member.User.ID,
		Title:       title,
		Description: desc,
		MaxSlots:    slots,
		ScheduledAt: scheduledAt,
	}

	ctx := context.Background()
	if err := h.eventUseCase.CreateEvent(ctx, event); err != nil {
		log.Printf("❌ Erreur lors de la sauvegarde de l'événement: %v", err)
	}
}

func (h *Handler) HandleEditEvent(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
				Content: "❌ Seul l'organisateur peut modifier la sortie.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
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
		var err error
		scheduledAt, err = pkgdiscord.ParseEventDateTime(dateStr, timeStr)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ " + err.Error(),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

	var slots int
	if slotsStr == "" || slotsStr == "0" {
		slots = 0
	} else {
		var err error
		slots, err = strconv.Atoi(slotsStr)
		if err != nil || slots < 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Nombre de places invalide (positif ou vide).",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

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
				Content: "❌ Seul l'organisateur peut modifier la sortie.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	event.Title = title
	event.Description = desc
	event.MaxSlots = slots
	event.ScheduledAt = scheduledAt

	if err := h.eventUseCase.UpdateEvent(ctx, event); err != nil {
		if errors.Is(err, domain.ErrCannotReduceSlots) {
			confirmedParticipants, _ := h.eventUseCase.GetConfirmedParticipants(ctx, event.ID)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Impossible de réduire à %d places : il y a déjà %d participants confirmés. Retirez d'abord des participants.", slots, len(confirmedParticipants)),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		log.Printf("❌ Erreur lors de la mise à jour de l'événement: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Erreur lors de la mise à jour.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ Sortie modifiée avec succès !",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
