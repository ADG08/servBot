package discord

import (
	"github.com/bwmarrin/discordgo"
)

func respondEphemeral(s *discordgo.Session, i *discordgo.Interaction, content string) {
	_ = s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
