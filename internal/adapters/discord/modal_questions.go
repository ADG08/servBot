package discord

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// HandleAskQuestion ouvre le modal "Ta question" quand un membre (non orga) clique sur le bouton du post forum.
func (h *Handler) HandleAskQuestion(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByMessageID(ctx, i.Message.ID)
	if err != nil || event == nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.question_event_not_found", nil))
		return
	}
	if event.QuestionsThreadID == "" {
		respondEphemeral(s, i.Interaction, h.translate("errors.question_thread_unavailable", nil))
		return
	}
	userID := interactionUserID(i)
	if userID == event.CreatorID {
		respondEphemeral(s, i.Interaction, h.translate("info.ask_question_organizer", nil))
		return
	}

	customID := fmt.Sprintf("ask_question_modal_%d", event.ID)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: customID,
			Title:    h.translate("ui.modal_ask_question_title", nil),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "question",
							Label:       h.translate("ui.modal_ask_question_label", nil),
							Style:       discordgo.TextInputParagraph,
							Required:    true,
							Placeholder: h.translate("ui.modal_ask_question_placeholder", nil),
						},
					},
				},
			},
		},
	})
}

// HandleAnswerQuestion ouvre le modal "Ta réponse" quand l'organisateur clique sur Répondre dans le thread Questions.
func (h *Handler) HandleAnswerQuestion(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	prefix := "btn_answer_question_"
	if !strings.HasPrefix(customID, prefix) {
		return
	}
	payload := strings.TrimPrefix(customID, prefix)
	parts := strings.Split(payload, "_")
	if len(parts) < 2 {
		return
	}
	eventIDStr, memberID := parts[0], parts[1]
	questionMessageID := ""
	if len(parts) >= 3 {
		questionMessageID = parts[2]
	}
	if eventIDStr == "" || memberID == "" {
		return
	}

	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		return
	}

	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByID(ctx, uint(eventID))
	if err != nil || event == nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.answer_event_not_found", nil))
		return
	}
	userID := interactionUserID(i)
	if userID == "" || userID != event.CreatorID {
		respondEphemeral(s, i.Interaction, h.translate("errors.question_only_organizer_can_answer", nil))
		return
	}

	modalID := fmt.Sprintf("answer_question_modal_%d_%s", event.ID, memberID)
	if questionMessageID != "" {
		modalID += "_" + questionMessageID
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: modalID,
			Title:    h.translate("ui.modal_answer_title", nil),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "answer",
							Label:       h.translate("ui.modal_answer_label", nil),
							Style:       discordgo.TextInputParagraph,
							Required:    true,
							Placeholder: h.translate("ui.modal_answer_placeholder", nil),
						},
					},
				},
			},
		},
	})
}

func extractTextInputValue(data discordgo.ModalSubmitInteractionData, id string) string {
	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			for _, inner := range row.Components {
				if input, ok := inner.(*discordgo.TextInput); ok && input.CustomID == id {
					return input.Value
				}
			}
		}
	}
	return ""
}

// handleAskQuestionModalSubmit reçoit la question, la poste dans le thread Questions et ajoute un bouton Répondre.
func (h *Handler) handleAskQuestionModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
	customID := data.CustomID
	prefix := "ask_question_modal_"
	if !strings.HasPrefix(customID, prefix) {
		return
	}
	eventIDStr := strings.TrimPrefix(customID, prefix)
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.question_invalid_event_id", nil))
		return
	}

	question := extractTextInputValue(data, "question")
	question = strings.TrimSpace(question)
	if question == "" {
		respondEphemeral(s, i.Interaction, h.translate("errors.question_required", nil))
		return
	}

	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByID(ctx, uint(eventID))
	if err != nil || event == nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.question_event_not_found", nil))
		return
	}
	if event.QuestionsThreadID == "" {
		respondEphemeral(s, i.Interaction, h.translate("errors.question_thread_unavailable", nil))
		return
	}

	memberID := interactionUserID(i)
	if memberID == "" {
		respondEphemeral(s, i.Interaction, h.translate("errors.question_member_unknown", nil))
		return
	}

	displayName := ""
	if i.Member != nil {
		displayName = resolveDisplayName(i.Member)
	}
	if displayName == "" {
		displayName = memberID
	}

	var contentBuilder strings.Builder
	contentBuilder.WriteString("<@" + memberID + ">" + h.translate("ui.question_thread_ask_suffix", nil))
	contentBuilder.WriteString("> " + question + "\n")
	contentStr := contentBuilder.String()

	msg, err := s.ChannelMessageSend(event.QuestionsThreadID, contentStr)
	if err != nil || msg == nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.question_send_failed", nil))
		return
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    h.translate("ui.btn_reply", nil),
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("btn_answer_question_%d_%s_%s", event.ID, memberID, msg.ID),
				},
			},
		},
	}
	_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         msg.ID,
		Channel:    event.QuestionsThreadID,
		Content:    &contentStr,
		Components: &components,
	})
	if err != nil {
		log.Printf("❌ Ajout bouton Répondre (question thread): %v", err)
	}

	respondEphemeral(s, i.Interaction, h.translate("success.question_sent", nil))
}

// extractQuestionFromThreadMessage extracts the question text from the thread message.
// Supports both FR (" te demande :\n") and EN (" asks:\n") suffixes.
func extractQuestionFromThreadMessage(content string) string {
	for _, suffix := range []string{" te demande :\n", " asks:\n"} {
		idx := strings.Index(content, suffix)
		if idx != -1 {
			block := strings.TrimSpace(content[idx+len(suffix):])
			block = strings.TrimPrefix(block, "> ")
			block = strings.ReplaceAll(block, "\n> ", "\n")
			return strings.TrimSpace(block)
		}
	}
	return ""
}

// handleAnswerQuestionModalSubmit envoie la réponse en MP au membre, précédée de sa question.
func (h *Handler) handleAnswerQuestionModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
	customID := data.CustomID
	prefix := "answer_question_modal_"
	if !strings.HasPrefix(customID, prefix) {
		return
	}
	payload := strings.TrimPrefix(customID, prefix)
	parts := strings.Split(payload, "_")
	if len(parts) < 2 {
		respondEphemeral(s, i.Interaction, h.translate("errors.answer_invalid_payload", nil))
		return
	}
	eventIDStr, memberID := parts[0], parts[1]
	questionMessageID := ""
	if len(parts) >= 3 {
		questionMessageID = parts[2]
	}
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.answer_invalid_event_id", nil))
		return
	}

	answer := extractTextInputValue(data, "answer")
	answer = strings.TrimSpace(answer)
	if answer == "" {
		respondEphemeral(s, i.Interaction, h.translate("errors.answer_required", nil))
		return
	}

	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByID(ctx, uint(eventID))
	if err != nil || event == nil {
		respondEphemeral(s, i.Interaction, h.translate("errors.answer_event_not_found", nil))
		return
	}
	userID := interactionUserID(i)
	if userID == "" || userID != event.CreatorID {
		respondEphemeral(s, i.Interaction, h.translate("errors.question_only_organizer_can_answer", nil))
		return
	}

	questionText := ""
	if event.QuestionsThreadID != "" && questionMessageID != "" {
		if qMsg, err := s.ChannelMessage(event.QuestionsThreadID, questionMessageID); err == nil && qMsg != nil {
			questionText = extractQuestionFromThreadMessage(qMsg.Content)
		}
	}

	var dmBuilder strings.Builder
	if link := h.messageLink(event.ChannelID, event.MessageID); link != "" {
		dmBuilder.WriteString(h.translate("ui.dm_answer_header_link", map[string]any{"EventTitle": event.Title, "Link": link}))
	} else {
		dmBuilder.WriteString(h.translate("ui.dm_answer_header", map[string]any{"EventTitle": event.Title}))
	}
	if questionText != "" {
		dmBuilder.WriteString(h.translate("ui.dm_answer_your_question", nil))
		dmBuilder.WriteString(questionText + "\n\n")
	}
	dmBuilder.WriteString(h.translate("ui.dm_answer_label", nil))
	dmBuilder.WriteString(answer)

	sendDM(s, memberID, dmBuilder.String())
	respondEphemeral(s, i.Interaction, h.translate("success.answer_sent", nil))
}
