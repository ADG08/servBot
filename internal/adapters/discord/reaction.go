package discord

import (
	"context"
	"errors"
	"fmt"

	"servbot/internal/domain"

	"github.com/bwmarrin/discordgo"
)

const reactionJoinEmoji = "✅"

func sendDM(s *discordgo.Session, userID string, content string) {
	ch, _ := s.UserChannelCreate(userID)
	if ch != nil {
		_, _ = s.ChannelMessageSend(ch.ID, content)
	}
}

func (h *Handler) HandleReactionJoin(s *discordgo.Session, channelID, messageID, userID, username string) {
	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByMessageID(ctx, messageID)
	if err != nil {
		return
	}
	if userID == event.CreatorID {
		_ = s.MessageReactionRemove(channelID, messageID, reactionJoinEmoji, userID)
		return
	}
	reply, err := h.participantUseCase.JoinEvent(ctx, event.ID, userID, username)
	if err != nil {
		if errors.Is(err, domain.ErrParticipantExists) {
			sendDM(s, userID, reply)
		}
		return
	}
	h.updateEmbed(ctx, s, channelID, messageID)
	sendDM(s, userID, reply)
}

func (h *Handler) HandleReactionLeave(s *discordgo.Session, channelID, messageID, userID string) {
	ctx := context.Background()
	event, err := h.eventUseCase.GetEventByMessageID(ctx, messageID)
	if err != nil {
		return
	}
	if userID == event.CreatorID {
		return
	}
	wasConfirmed, err := h.participantUseCase.LeaveEvent(ctx, event.ID, userID)
	if err != nil {
		sendDM(s, userID, "Tu n'étais pas inscrit.")
		return
	}
	msg := "🗑️ Tu t'es désisté."
	if wasConfirmed {
		luckyWinner, err := h.participantUseCase.GetNextWaitlistParticipant(ctx, event.ID)
		if err == nil {
			sendDM(s, luckyWinner.UserID, fmt.Sprintf("🎉 **Bonne nouvelle !** Une place s'est libérée pour **%s**, tu es maintenant inscrit !", event.Title))
		}
	}
	h.updateEmbed(ctx, s, channelID, messageID)
	sendDM(s, userID, msg)
}
