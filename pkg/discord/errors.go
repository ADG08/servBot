package discord

import "servbot/internal/domain"

// TranslateDomainError maps a domain error code to a user-facing message.
// For now, messages are in French; this function is the single place to plug
// a real i18n backend later.
func TranslateDomainError(code string) string {
	switch code {
	case "event_not_found":
		return "Événement non trouvé."
	case "datetime_in_past":
		return "la date et l'heure doivent être dans le futur"
	case "participant_not_found":
		return "Participant introuvable."
	case "participant_exists":
		return "Tu as déjà manifesté ton intérêt."
	case "participant_not_waitlist":
		return "Ce participant n'est plus en liste d'attente."
	case "participant_not_confirmed":
		return "Ce participant n'est pas confirmé."
	case "no_waitlist_participant":
		return "Il n'y a aucun participant en liste d'attente."
	case "cannot_reduce_slots":
		return "Impossible de réduire le nombre de places."
	case "not_organizer":
		return "Seul l'organisateur peut effectuer cette action."
	case "event_already_finalized":
		return "Cette sortie est déjà finalisée."
	default:
		return "Une erreur est survenue."
	}
}

// DomainErrorMessage is a convenience helper that extracts the domain error code
// and immediately resolves it to a user-facing message.
func DomainErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if code := domain.Code(err); code != "" {
		return TranslateDomainError(code)
	}
	return ""
}

