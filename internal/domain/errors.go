package domain

import "errors"

// Domain errors.
var (
	ErrEventNotFound           = errors.New("événement non trouvé")
	ErrDateTimeInPast          = errors.New("la date et l'heure doivent être dans le futur")
	ErrParticipantNotFound     = errors.New("participant non trouvé")
	ErrParticipantExists       = errors.New("participant déjà inscrit")
	ErrParticipantNotWaitlist  = errors.New("participant n'est pas en liste d'attente")
	ErrParticipantNotConfirmed = errors.New("participant n'est pas confirmé")
	ErrNoWaitlistParticipant   = errors.New("aucun participant en liste d'attente")
	ErrCannotReduceSlots       = errors.New("impossible de réduire le nombre de places")
	ErrNotOrganizer            = errors.New("seul l'organisateur peut effectuer cette action")
)
