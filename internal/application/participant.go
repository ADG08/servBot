package application

import (
	"context"
	"fmt"
	"time"

	"servbot/internal/domain"
	"servbot/internal/domain/entities"
	"servbot/internal/ports/output"
)

type ParticipantService struct {
	participantRepo output.ParticipantRepository
	eventRepo       output.EventRepository
	translator      output.T
}

func NewParticipantService(
	participantRepo output.ParticipantRepository,
	eventRepo output.EventRepository,
	translator output.T,
) *ParticipantService {
	return &ParticipantService{
		participantRepo: participantRepo,
		eventRepo:       eventRepo,
		translator:      translator,
	}
}

func (s *ParticipantService) JoinEvent(ctx context.Context, locale string, eventID uint, userID, username string, forceWaitlist bool) (string, error) {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return "", domain.ErrEventNotFound
	}
	existing, _ := s.participantRepo.FindByEventIDAndUserID(ctx, eventID, userID)
	if existing != nil {
		msgKey := "dm.join.already_interested"
		if existing.Status == domain.StatusWaitlist {
			msgKey = "dm.join.already_waitlist"
		}
		return s.translator.T(locale, msgKey, nil), domain.ErrParticipantExists
	}
	confirmedCount, err := s.participantRepo.CountByEventIDAndStatus(ctx, eventID, domain.StatusConfirmed)
	if err != nil {
		return "", fmt.Errorf("count confirmed: %w", err)
	}
	status := domain.StatusConfirmed
	replyKey := "dm.join.confirmed"
	if event.MaxSlots > 0 && int(confirmedCount) >= event.MaxSlots {
		status = domain.StatusWaitlist
		replyKey = "dm.join.waitlist_full"
	} else if forceWaitlist {
		status = domain.StatusWaitlist
		replyKey = "dm.join.waitlist_forced"
	}
	participant := &entities.Participant{
		EventID:  eventID,
		UserID:   userID,
		Username: username,
		Status:   status,
		JoinedAt: time.Now(),
	}
	if err := s.participantRepo.Create(ctx, participant); err != nil {
		return "", fmt.Errorf("create participant: %w", err)
	}
	return s.translator.T(locale, replyKey, nil), nil
}

func (s *ParticipantService) GetParticipantByEventIDAndUserID(ctx context.Context, eventID uint, userID string) (*entities.Participant, error) {
	return s.participantRepo.FindByEventIDAndUserID(ctx, eventID, userID)
}

func (s *ParticipantService) GetParticipantByID(ctx context.Context, id uint) (*entities.Participant, error) {
	return s.participantRepo.FindByID(ctx, id)
}

func (s *ParticipantService) LeaveEvent(ctx context.Context, eventID uint, userID string) (bool, error) {
	participant, err := s.participantRepo.FindByEventIDAndUserID(ctx, eventID, userID)
	if err != nil {
		return false, domain.ErrParticipantNotFound
	}
	wasConfirmed := participant.Status == domain.StatusConfirmed
	if err := s.participantRepo.Delete(ctx, participant); err != nil {
		return false, fmt.Errorf("delete participant: %w", err)
	}
	return wasConfirmed, nil
}

// PromoteParticipant promotes a waitlist participant to confirmed; if the event was full, MaxSlots is increased by 1.
func (s *ParticipantService) PromoteParticipant(ctx context.Context, participantID uint, creatorID string) (*entities.Participant, bool, error) {
	participant, err := s.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, false, domain.ErrParticipantNotFound
	}
	if participant.Status != domain.StatusWaitlist {
		return nil, false, domain.ErrParticipantNotWaitlist
	}
	event, err := s.eventRepo.FindByID(ctx, participant.EventID)
	if err != nil {
		return nil, false, domain.ErrEventNotFound
	}
	if event.CreatorID != creatorID {
		return nil, false, domain.ErrNotOrganizer
	}
	confirmedCount, err := s.participantRepo.CountByEventIDAndStatus(ctx, event.ID, domain.StatusConfirmed)
	if err != nil {
		return nil, false, fmt.Errorf("count confirmed: %w", err)
	}
	quotaIncreased := false
	if event.MaxSlots > 0 && int(confirmedCount) >= event.MaxSlots {
		event.MaxSlots = int(confirmedCount) + 1
		if err := s.eventRepo.Update(ctx, event); err != nil {
			return nil, false, fmt.Errorf("update event: %w", err)
		}
		quotaIncreased = true
	}
	participant.Status = domain.StatusConfirmed
	if err := s.participantRepo.Update(ctx, participant); err != nil {
		return nil, false, fmt.Errorf("update participant: %w", err)
	}
	return participant, quotaIncreased, nil
}

func (s *ParticipantService) RemoveParticipant(ctx context.Context, participantID uint, creatorID string) (*entities.Participant, error) {
	participant, err := s.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, domain.ErrParticipantNotFound
	}
	event, err := s.eventRepo.FindByID(ctx, participant.EventID)
	if err != nil {
		return nil, domain.ErrEventNotFound
	}
	if event.CreatorID != creatorID {
		return nil, domain.ErrNotOrganizer
	}
	if participant.Status != domain.StatusConfirmed {
		return nil, domain.ErrParticipantNotConfirmed
	}
	if err := s.participantRepo.Delete(ctx, participant); err != nil {
		return nil, fmt.Errorf("delete participant: %w", err)
	}
	return participant, nil
}

func (s *ParticipantService) GetNextWaitlistParticipant(ctx context.Context, eventID uint) (*entities.Participant, error) {
	participants, err := s.participantRepo.FindByEventIDAndStatus(ctx, eventID, domain.StatusWaitlist)
	if err != nil {
		return nil, fmt.Errorf("find waitlist: %w", err)
	}
	if len(participants) == 0 {
		return nil, domain.ErrNoWaitlistParticipant
	}
	oldest := participants[0]
	oldest.Status = domain.StatusConfirmed
	if err := s.participantRepo.Update(ctx, &oldest); err != nil {
		return nil, fmt.Errorf("update participant: %w", err)
	}
	return &oldest, nil
}
