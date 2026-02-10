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
		respondEphemeral(s, i.Interaction, "❌ Impossible de retrouver la sortie pour cette question.")
		return
	}
	if event.QuestionsThreadID == "" {
		respondEphemeral(s, i.Interaction, "❌ Le thread privé des questions n'est pas disponible pour cette sortie.")
		return
	}
	userID := interactionUserID(i)
	if userID == event.CreatorID {
		respondEphemeral(s, i.Interaction, "En tant qu'organisateur, tu reçois les questions dans le thread **Questions** du salon privé ; inutile de t'en poser une.")
		return
	}

	customID := fmt.Sprintf("ask_question_modal_%d", event.ID)
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: customID,
			Title:    "Poser une question",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "question",
							Label:       "Ta question",
							Style:       discordgo.TextInputParagraph,
							Required:    true,
							Placeholder: "Ex: Où se retrouve-t-on exactement ?",
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
		respondEphemeral(s, i.Interaction, "❌ Impossible de retrouver la sortie pour cette réponse.")
		return
	}
	userID := interactionUserID(i)
	if userID == "" || userID != event.CreatorID {
		respondEphemeral(s, i.Interaction, "❌ Seul l'organisateur peut répondre à cette question.")
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
			Title:    "Ta réponse",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "answer",
							Label:       "Ta réponse",
							Style:       discordgo.TextInputParagraph,
							Required:    true,
							Placeholder: "Ta réponse sera envoyée en MP au membre.",
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
		respondEphemeral(s, i.Interaction, "❌ Identifiant de sortie invalide pour la question.")
		return
	}

	question := extractTextInputValue(data, "question")
	question = strings.TrimSpace(question)
	if question == "" {
		respondEphemeral(s, i.Interaction, "❌ Merci de saisir une question.")
		return
	}

	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByID(ctx, uint(eventID))
	if err != nil || event == nil {
		respondEphemeral(s, i.Interaction, "❌ Impossible de retrouver la sortie pour cette question.")
		return
	}
	if event.QuestionsThreadID == "" {
		respondEphemeral(s, i.Interaction, "❌ Le thread privé des questions n'est pas disponible pour cette sortie.")
		return
	}

	memberID := interactionUserID(i)
	if memberID == "" {
		respondEphemeral(s, i.Interaction, "❌ Impossible d'identifier le membre qui pose la question.")
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
	contentBuilder.WriteString(fmt.Sprintf("<@%s> te demande :\n", memberID))
	contentBuilder.WriteString(fmt.Sprintf("> %s\n", question))
	contentStr := contentBuilder.String()

	msg, err := s.ChannelMessageSend(event.QuestionsThreadID, contentStr)
	if err != nil || msg == nil {
		respondEphemeral(s, i.Interaction, "❌ Impossible d'envoyer la question à l'organisateur.")
		return
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Répondre",
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

	respondEphemeral(s, i.Interaction, "✅ Ta question a été envoyée à l'organisateur.")
}

// extractQuestionFromThreadMessage extrait le texte de la question depuis le message posté dans le thread (format: "<@id> te demande :\n> question").
func extractQuestionFromThreadMessage(content string) string {
	const prefix = "te demande :\n"
	idx := strings.Index(content, prefix)
	if idx == -1 {
		return ""
	}
	block := strings.TrimSpace(content[idx+len(prefix):])
	// Enlever le préfixe "> " de chaque ligne (citation Discord)
	block = strings.TrimPrefix(block, "> ")
	block = strings.ReplaceAll(block, "\n> ", "\n")
	return strings.TrimSpace(block)
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
		respondEphemeral(s, i.Interaction, "❌ Données de réponse invalides.")
		return
	}
	eventIDStr, memberID := parts[0], parts[1]
	questionMessageID := ""
	if len(parts) >= 3 {
		questionMessageID = parts[2]
	}
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		respondEphemeral(s, i.Interaction, "❌ Identifiant de sortie invalide pour la réponse.")
		return
	}

	answer := extractTextInputValue(data, "answer")
	answer = strings.TrimSpace(answer)
	if answer == "" {
		respondEphemeral(s, i.Interaction, "❌ Merci de saisir une réponse.")
		return
	}

	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByID(ctx, uint(eventID))
	if err != nil || event == nil {
		respondEphemeral(s, i.Interaction, "❌ Impossible de retrouver la sortie pour cette réponse.")
		return
	}
	userID := interactionUserID(i)
	if userID == "" || userID != event.CreatorID {
		respondEphemeral(s, i.Interaction, "❌ Seul l'organisateur peut répondre à cette question.")
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
		dmBuilder.WriteString(fmt.Sprintf("✉️ **[%s](%s)** – Réponse de l'organisateur\n\n", event.Title, link))
	} else {
		dmBuilder.WriteString(fmt.Sprintf("✉️ **%s** – Réponse de l'organisateur\n\n", event.Title))
	}
	if questionText != "" {
		dmBuilder.WriteString(fmt.Sprintf("**Ta question :**\n%s\n\n", questionText))
	}
	dmBuilder.WriteString("**Réponse :**\n")
	dmBuilder.WriteString(answer)

	sendDM(s, memberID, dmBuilder.String())
	respondEphemeral(s, i.Interaction, "✅ Réponse envoyée en MP au membre.")
}
