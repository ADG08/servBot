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

// Bot is the Discord adapter.
type Bot struct {
	session *discordgo.Session
	config  *config.Config
	handler *Handler
}

// NewBot creates a Bot and wires ports: output adapters -> application (use cases) -> handler.
func NewBot(cfg *config.Config, eventRepo output.EventRepository, participantRepo output.ParticipantRepository) *Bot {
	eventUC := application.NewEventService(eventRepo, participantRepo)
	participantUC := application.NewParticipantService(participantRepo, eventRepo)

	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		log.Fatal("❌ Erreur lors de la création de la session Discord:", err)
	}

	handler := NewHandler(eventUC, participantUC, cfg.ForumChannelID)

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
}

func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		cmdData := i.ApplicationCommandData()
		if cmdData.Name == "sortie" {
			b.handler.HandleCommand(s, i)
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

		if strings.HasPrefix(customID, "btn_manage_waitlist_") {
			b.handler.HandleManageWaitlist(s, i)
		} else if strings.HasPrefix(customID, "btn_remove_participant_") {
			b.handler.HandleRemoveParticipant(s, i)
		} else if strings.HasPrefix(customID, "btn_edit_event_") {
			b.handler.HandleEditEvent(s, i)
		} else if strings.HasPrefix(customID, "btn_create_sortie_chat_") {
			b.handler.HandleCreateSortieChat(s, i)
		} else {
			switch customID {
			case "btn_join":
				b.handler.HandleJoin(s, i)
			case "btn_leave":
				b.handler.HandleLeave(s, i)
			case "select_promote":
				b.handler.HandlePromote(s, i)
			case "select_remove":
				b.handler.HandleRemove(s, i)
			}
		}
	}
}

// Start runs the bot until interrupted.
func (b *Bot) Start() error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("erreur lors de l'ouverture de la session: %w", err)
	}
	defer b.session.Close()

	commands := []*discordgo.ApplicationCommand{
		{Name: "sortie", Description: "Créer une nouvelle sortie"},
	}

	for _, cmd := range commands {
		if _, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", cmd); err != nil {
			log.Printf("⚠️ Erreur lors de l'enregistrement de la commande %s: %v", cmd.Name, err)
		}
	}

	fmt.Println("🤖 Bot en ligne ! Appuyez sur CTRL+C pour quitter.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	return nil
}
