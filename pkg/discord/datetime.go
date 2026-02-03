package discord

import (
	"fmt"
	"time"

	"servbot/internal/domain"
)

// ParseEventDateTime parses date (JJ/MM/AAAA) and time (HH:MM) in local timezone.
// Returns an error if format is invalid or if the date/time is in the past.
func ParseEventDateTime(dateStr, timeStr string) (time.Time, error) {
	dateStr = trimSpace(dateStr)
	timeStr = trimSpace(timeStr)
	if dateStr == "" || timeStr == "" {
		return time.Time{}, fmt.Errorf("date et heure requises (JJ/MM/AAAA et HH:MM)")
	}
	tDate, err := time.Parse("02/01/2006", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("date invalide (attendu JJ/MM/AAAA, ex: 15/02/2025)")
	}
	tTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("heure invalide (attendu HH:MM, ex: 14:00)")
	}
	loc := time.Local
	dt := time.Date(tDate.Year(), tDate.Month(), tDate.Day(),
		tTime.Hour(), tTime.Minute(), 0, 0, loc)
	if dt.Before(time.Now()) {
		return time.Time{}, domain.ErrDateTimeInPast
	}
	return dt, nil
}

func FormatEventDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02/01/2006 à 15:04")
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
