package discord

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
)

// RunScheduledTasks runs periodic tasks every 10 minutes: H-48 organizer DMs and edit-lock embed refresh.
func (h *Handler) RunScheduledTasks(s *discordgo.Session) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		ctx := context.Background()
		now := time.Now()
		h.processH48OrganizerDMs(s, ctx, now)
		h.processEditLock(s, ctx, now)
	}
}
