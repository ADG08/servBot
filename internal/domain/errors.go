package domain

import "errors"

// Error is the domain error type carrying a stable code.
// The code is used by outer layers (adapters/i18n) to resolve
// a localized, user-facing message.
type Error struct {
	code string
}

func (e *Error) Error() string {
	// The error string is intentionally the machine-readable code.
	// Adapters must not display this directly to end users.
	return e.code
}

// Code returns the stable machine-readable code for this domain error.
func (e *Error) Code() string {
	return e.code
}

// Code extracts the domain error code from an error value.
// Returns an empty string when the error is not a domain.Error.
func Code(err error) string {
	if err == nil {
		return ""
	}
	var de *Error
	if errors.As(err, &de) {
		return de.code
	}
	return ""
}

// Domain error sentinels.
// These are used with errors.Is and mapped to localized messages
// by the adapters (i18n layer).
var (
	ErrEventNotFound           = &Error{code: "event_not_found"}
	ErrDateTimeInPast          = &Error{code: "datetime_in_past"}
	ErrParticipantNotFound     = &Error{code: "participant_not_found"}
	ErrParticipantExists       = &Error{code: "participant_exists"}
	ErrParticipantNotWaitlist  = &Error{code: "participant_not_waitlist"}
	ErrParticipantNotConfirmed = &Error{code: "participant_not_confirmed"}
	ErrNoWaitlistParticipant   = &Error{code: "no_waitlist_participant"}
	ErrCannotReduceSlots       = &Error{code: "cannot_reduce_slots"}
	ErrNotOrganizer            = &Error{code: "not_organizer"}
	ErrEventAlreadyFinalized   = &Error{code: "event_already_finalized"}
)
