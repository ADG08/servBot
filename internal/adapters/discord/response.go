package discord

import "github.com/bwmarrin/discordgo"

// Nick > GlobalName > Username
func resolveDisplayName(member *discordgo.Member) string {
	if member == nil || member.User == nil {
		return ""
	}
	if member.Nick != "" {
		return member.Nick
	}
	if member.User.GlobalName != "" {
		return member.User.GlobalName
	}
	return member.User.Username
}

func respondEphemeral(s *discordgo.Session, i *discordgo.Interaction, content string) {
	_ = s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// currentLocale returns the effective locale for this handler (bot-wide for now).
func (h *Handler) currentLocale() string {
	if h == nil {
		return ""
	}
	return h.defaultLocale
}

// translate resolves a message key using the handler's current locale.
func (h *Handler) translate(key string, data map[string]any) string {
	return h.translator.T(h.currentLocale(), key, data)
}
