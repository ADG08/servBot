package discord

import (
	"fmt"
	"strings"
	"time"

	"servbot/internal/domain"
	"servbot/pkg/tz"
)

// ParseEventDateTime parses date (JJ/MM/AAAA) + time (HH:MM) in Europe/Paris.
// Returns an error if format is invalid or date/time is in the past.
func ParseEventDateTime(dateStr, timeStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	timeStr = strings.TrimSpace(timeStr)
	if dateStr == "" || timeStr == "" {
		return time.Time{}, fmt.Errorf("date et heure requises (JJ/MM/AAAA et HH:MM)")
	}
	tDate, err := time.Parse("02/01/2006", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("date invalide (attendu JJ/MM/AAAA, ex: 15/03/2026)")
	}
	tTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("heure invalide (attendu HH:MM, ex: 14:00)")
	}
	dt := time.Date(tDate.Year(), tDate.Month(), tDate.Day(),
		tTime.Hour(), tTime.Minute(), 0, 0, tz.Paris)
	now := time.Now().In(tz.Paris)
	if dt.Before(now.Add(-time.Minute)) {
		return time.Time{}, domain.ErrDateTimeInPast
	}
	return dt, nil
}
