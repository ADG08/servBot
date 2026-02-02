package discord

import (
	"github.com/bwmarrin/discordgo"
)

func (h *Handler) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "create_event_modal",
			Title:    "Organiser une sortie",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "title", Label: "Titre", Style: discordgo.TextInputShort, Required: true, Placeholder: "Resto, Ciné..."},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "desc", Label: "Détails (Date, Heure, Lieu)", Style: discordgo.TextInputParagraph, Required: true},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "slots", Label: "Nombre de places (laisser vide pour illimité)", Style: discordgo.TextInputShort, Required: false, Placeholder: "4 ou laisser vide"},
				}},
			},
		},
	})
}
