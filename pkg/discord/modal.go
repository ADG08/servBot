package discord

import "github.com/bwmarrin/discordgo"

func ExtractModalData(data discordgo.ModalSubmitInteractionData) (title, desc, slotsStr string) {
	if len(data.Components) >= 3 {
		if row, ok := data.Components[0].(*discordgo.ActionsRow); ok && len(row.Components) > 0 {
			if input, ok := row.Components[0].(*discordgo.TextInput); ok {
				title = input.Value
			}
		}
		if row, ok := data.Components[1].(*discordgo.ActionsRow); ok && len(row.Components) > 0 {
			if input, ok := row.Components[0].(*discordgo.TextInput); ok {
				desc = input.Value
			}
		}
		if row, ok := data.Components[2].(*discordgo.ActionsRow); ok && len(row.Components) > 0 {
			if input, ok := row.Components[0].(*discordgo.TextInput); ok {
				slotsStr = input.Value
			}
		}
	}
	return
}
