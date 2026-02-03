package discord

import (
	"github.com/bwmarrin/discordgo"
)

const (
	placeholderTitle = "Ex: Resto, Ciné, Soirée jeux..."
	placeholderDesc  = "Lieu, adresse, infos pratiques..."
	placeholderDate  = "Ex: 15/02/2026 (jour/mois/année)"
	placeholderTime  = "Ex: 14:00 ou 18:30"
	placeholderSlots = "Ex: 4 ou vide = illimité"
)

func (h *Handler) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "create_event_modal",
			Title:    "Organiser une sortie",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "title", Label: "Titre", Style: discordgo.TextInputShort, Required: true, Placeholder: placeholderTitle},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "desc", Label: "Détails (Lieu, etc.)", Style: discordgo.TextInputParagraph, Required: true, Placeholder: placeholderDesc},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "date", Label: "Date", Style: discordgo.TextInputShort, Required: true, Placeholder: placeholderDate},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "time", Label: "Heure", Style: discordgo.TextInputShort, Required: true, Placeholder: placeholderTime},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "slots", Label: "Nombre de places", Style: discordgo.TextInputShort, Required: false, Placeholder: placeholderSlots},
				}},
			},
		},
	})
}
