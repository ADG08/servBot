package discord

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"servbot/internal/domain"
	"servbot/internal/domain/entities"

	"github.com/bwmarrin/discordgo"
)

const reactionJoinEmoji = "✅"

func sendDM(s *discordgo.Session, userID string, content string) {
	ch, _ := s.UserChannelCreate(userID)
	if ch != nil {
		_, _ = s.ChannelMessageSend(ch.ID, content)
	}
}

// shouldForceWaitlistForJoin returns true when a new joiner must go to waitlist (manual only).
// Auto: never forced here (application layer uses slot count). Manual: forced if finalized, Cas B (<48h),
// or when there are already waitlist participants (preserve their priority).
func (h *Handler) shouldForceWaitlistForJoin(ctx context.Context, event *entities.Event, now time.Time) bool {
	if event.WaitlistAuto {
		return false
	}
	if event.IsFinalized() || isCasB(event.ScheduledAt, now) {
		return true
	}
	waitlist, err := h.eventUseCase.GetWaitlistParticipants(ctx, event.ID)
	return err == nil && len(waitlist) > 0
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
	now := time.Now()
	forceWaitlist := h.shouldForceWaitlistForJoin(ctx, event, now)
	reply, err := h.participantUseCase.JoinEvent(ctx, event.ID, userID, username, forceWaitlist)
	if err != nil {
		if errors.Is(err, domain.ErrParticipantExists) {
			sendDM(s, userID, reply)
		}
		return
	}
	h.updateEmbed(ctx, s, channelID, messageID)

	if event.IsFinalized() {
		p, _ := h.participantUseCase.GetParticipantByEventIDAndUserID(ctx, event.ID, userID)
		if p != nil && p.Status == domain.StatusConfirmed {
			grantPrivateChannelAccess(s, event.PrivateChannelID, userID)
		}
	}

	if isCasA(event.ScheduledAt, now) {
		eventFull, _ := h.eventUseCase.GetEventByID(ctx, event.ID)
		if eventFull != nil {
			confirmedCount, _ := h.eventUseCase.GetConfirmedParticipants(ctx, event.ID)
			isComplet := eventFull.MaxSlots > 0 && len(confirmedCount) >= eventFull.MaxSlots
			if isComplet && eventFull.OrganizerValidationDMSentAt.IsZero() {
				evWP := eventToEventWithParticipants(eventFull)
				if err := h.sendOrganizerValidationDM(s, evWP); err != nil {
					log.Printf("❌ Envoi MP validation organisateur (Cas A complet): %v", err)
				} else {
					_ = h.eventUseCase.MarkOrganizerValidationDMSent(ctx, event.ID)
				}
			}
		}
	} else if isCasB(event.ScheduledAt, now) {
		participant, _ := h.participantUseCase.GetParticipantByEventIDAndUserID(ctx, event.ID, userID)
		if participant != nil && participant.Status == domain.StatusConfirmed {
			if err := h.sendOrganizerAcceptRefuseDM(s, event.Title, event.CreatorID, channelID, messageID, participant); err != nil {
				log.Printf("❌ Envoi MP Accepter/Refuser organisateur (Cas B): %v", err)
			}
		}
	}

	sendDM(s, userID, reply)
}

func (h *Handler) promoteNextFromWaitlist(s *discordgo.Session, ctx context.Context, event *entities.Event) {
	luckyWinner, err := h.participantUseCase.GetNextWaitlistParticipant(ctx, event.ID)
	if err != nil {
		return
	}
	sendDM(s, luckyWinner.UserID, fmt.Sprintf("🎉 **Bonne nouvelle !** Une place s'est libérée pour **%s**, tu es maintenant parmi les confirmés !", event.Title))
	if shouldGrantPrivateChannelOnPromote(event, time.Now()) {
		grantPrivateChannelAccess(s, event.PrivateChannelID, luckyWinner.UserID)
	}
}

// onSlotFreed is called when a confirmed participant leaves or is removed.
// Auto: promote next waitlist. Manual + Cas A: no promo. Manual + Cas B: DM orga Accept/Ignore.
func (h *Handler) onSlotFreed(s *discordgo.Session, ctx context.Context, event *entities.Event) {
	if event.WaitlistAuto {
		h.promoteNextFromWaitlist(s, ctx, event)
	}
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
		return
	}
	revokePrivateChannelAccess(s, event.PrivateChannelID, userID)
	msg := "🗑️ Tu t'es désisté."
	if wasConfirmed {
		h.onSlotFreed(s, ctx, event)
	}
	h.updateEmbed(ctx, s, channelID, messageID)
	sendDM(s, userID, msg)
}
