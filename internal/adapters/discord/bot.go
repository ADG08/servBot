package discord

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"

	"servbot/internal/application"
	"servbot/internal/config"
	"servbot/internal/ports/output"
)

type Bot struct {
	session *discordgo.Session
	config  *config.Config
	handler *Handler
}

// NewBot wires output adapters, application services, and handler (composition root).
func NewBot(cfg *config.Config, eventRepo output.EventRepository, participantRepo output.ParticipantRepository) *Bot {
	eventUC := application.NewEventService(eventRepo, participantRepo)
	participantUC := application.NewParticipantService(participantRepo, eventRepo)

	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		log.Fatal("❌ Erreur lors de la création de la session Discord:", err)
	}

	handler := NewHandler(eventUC, participantUC, cfg.ForumChannelID, cfg.GuildID)

	bot := &Bot{
		session: s,
		config:  cfg,
		handler: handler,
	}
	bot.setupHandlers()
	return bot
}

func (b *Bot) setupHandlers() {
	b.session.AddHandler(b.handleInteraction)
	b.session.AddHandler(b.handleMessageReactionAdd)
	b.session.AddHandler(b.handleMessageReactionRemove)
}

func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		cmdData := i.ApplicationCommandData()
		switch cmdData.Name {
		case "sortie":
			b.handler.HandleCommand(s, i)
		case "sortie-template":
			b.handler.HandleTemplateCommand(s, i)
		case "retirer":
			b.handler.HandleRemoveCommand(s, i)
		}
	case discordgo.InteractionModalSubmit:
		modalData := i.ModalSubmitData()
		if modalData.CustomID == "edit_event_modal" {
			b.handler.HandleEditModalSubmit(s, i)
		} else {
			b.handler.HandleModalSubmit(s, i)
		}
	case discordgo.InteractionMessageComponent:
		componentData := i.MessageComponentData()
		customID := componentData.CustomID
		if strings.HasPrefix(customID, "btn_organizer_") {
			switch {
			case strings.HasPrefix(customID, "btn_organizer_finalize_"):
				b.handler.HandleOrganizerFinalizeStep1(s, i)
			case strings.HasPrefix(customID, "btn_organizer_accept_"):
				b.handler.HandleOrganizerAccept(s, i)
			case strings.HasPrefix(customID, "btn_organizer_refuse_"):
				b.handler.HandleOrganizerRefuse(s, i)
			}
		} else if strings.HasPrefix(customID, "btn_") {
			switch {
			case strings.HasPrefix(customID, "btn_manage_waitlist_"):
				b.handler.HandleManageWaitlist(s, i)
			case strings.HasPrefix(customID, "btn_remove_participant_"):
				b.handler.HandleRemoveParticipant(s, i)
			case strings.HasPrefix(customID, "btn_edit_event_"):
				b.handler.HandleEditEvent(s, i)
			}
		} else {
			switch customID {
			case "select_promote":
				b.handler.HandlePromote(s, i)
			case "select_remove_user":
				b.handler.HandleRemoveUserSelect(s, i)
			}
		}
	}
}

func (b *Bot) handleMessageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.Emoji.Name != reactionJoinEmoji || r.UserID == s.State.User.ID {
		return
	}
	displayName := resolveDisplayName(r.Member)
	if displayName == "" {
		displayName = r.UserID
	}
	b.handler.HandleReactionJoin(s, r.ChannelID, r.MessageID, r.UserID, displayName)
}

func (b *Bot) handleMessageReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r.Emoji.Name != reactionJoinEmoji || r.UserID == s.State.User.ID {
		return
	}
	b.handler.HandleReactionLeave(s, r.ChannelID, r.MessageID, r.UserID)
}

func (b *Bot) deleteAllCommands(appID, guildID string) {
	scope := "global"
	if guildID != "" {
		scope = "guild"
	}
	existing, err := b.session.ApplicationCommands(appID, guildID)
	if err != nil {
		log.Printf("⚠️ Erreur lors de la récupération des commandes (%s): %v", scope, err)
		return
	}
	for _, cmd := range existing {
		if err := b.session.ApplicationCommandDelete(appID, guildID, cmd.ID); err != nil {
			log.Printf("⚠️ Erreur lors de la suppression de la commande %s (%s): %v", cmd.Name, scope, err)
		}
	}
}

func (b *Bot) Start() error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("erreur lors de l'ouverture de la session: %w", err)
	}
	defer b.session.Close()

	appID := b.session.State.User.ID
	targetGuildID := b.config.GuildID
	b.deleteAllCommands(appID, "")
	if targetGuildID != "" {
		b.deleteAllCommands(appID, targetGuildID)
	}

	commands := []*discordgo.ApplicationCommand{
		{Name: "sortie", Description: "Créer une nouvelle sortie"},
		{
			Name:        "sortie-template",
			Description: "Ouvrir le formulaire de sortie pré-rempli pour le debug",
		},
		{
			Name:        "retirer",
			Description: "Retirer un membre d'une sortie (à utiliser dans le salon privé)",
		},
	}

	// Si GUILD_ID est défini, on enregistre les commandes au niveau du serveur
	// pour qu'elles soient disponibles immédiatement (pratique pour le debug).
	// Sinon, fallback sur des commandes globales (peuvent prendre plusieurs minutes).
	for _, cmd := range commands {
		if _, err := b.session.ApplicationCommandCreate(appID, targetGuildID, cmd); err != nil {
			scope := "global"
			if targetGuildID != "" {
				scope = "guild"
			}
			log.Printf("⚠️ Erreur lors de l'enregistrement de la commande %s (%s): %v", cmd.Name, scope, err)
		}
	}

	go b.handler.RunOrganizerValidationScheduler(b.session)

	fmt.Println("🤖 Bot en ligne ! Appuyez sur CTRL+C pour quitter.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	return nil
}
