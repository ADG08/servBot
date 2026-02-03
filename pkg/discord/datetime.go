package discord

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"servbot/internal/domain"
)

var (
	parisLoc     *time.Location
	parisLocErr  error
	parisLocOnce sync.Once
)

func parisLocation() (*time.Location, error) {
	parisLocOnce.Do(func() {
		parisLoc, parisLocErr = time.LoadLocation("Europe/Paris")
	})
	return parisLoc, parisLocErr
}

// ParseEventDateTime returns an error if format is invalid or date/time is in the past (Europe/Paris).
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
	loc, err := parisLocation()
	if err != nil {
		return time.Time{}, err
	}
	dt := time.Date(tDate.Year(), tDate.Month(), tDate.Day(),
		tTime.Hour(), tTime.Minute(), 0, 0, loc)
	// Grace period to avoid rejecting times that became "past" due to processing delay.
	now := time.Now()
	if dt.Before(now.Add(-time.Minute)) {
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
