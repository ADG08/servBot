package discord

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// HandleModalSubmit route les différents modals en fonction de leur CustomID.
func (h *Handler) HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	switch {
	case data.CustomID == "create_event_modal":
		h.handleCreateEventModalSubmit(s, i, data)
	case strings.HasPrefix(data.CustomID, "ask_question_modal_"):
		h.handleAskQuestionModalSubmit(s, i, data)
	case strings.HasPrefix(data.CustomID, "answer_question_modal_"):
		h.handleAnswerQuestionModalSubmit(s, i, data)
	default:
		// Modal inconnu : on ignore silencieusement pour rester robuste.
	}
}
