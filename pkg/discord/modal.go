package discord

import "github.com/bwmarrin/discordgo"

// Order: 5 rows = title, desc, date, time, slots; 3 rows (legacy) = title, desc, slots.
func ExtractModalData(data discordgo.ModalSubmitInteractionData) (title, desc, dateStr, timeStr, slotsStr string) {
	get := func(i int) string {
		if i >= len(data.Components) {
			return ""
		}
		if row, ok := data.Components[i].(*discordgo.ActionsRow); ok && len(row.Components) > 0 {
			if input, ok := row.Components[0].(*discordgo.TextInput); ok {
				return input.Value
			}
		}
		return ""
	}
	title = get(0)
	desc = get(1)
	if len(data.Components) >= 5 {
		dateStr = get(2)
		timeStr = get(3)
		slotsStr = get(4)
	} else if len(data.Components) >= 3 {
		slotsStr = get(2)
	}
	return
}
