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

	// Valeurs par défaut pour la commande /sortie-template (debug).
	templateTitleDefault = "Sortie de test"
	templateDescDefault  = "Lieu de test, adresse, infos pratiques..."
	templateDateDefault  = "01/01/2030"
	templateTimeDefault  = "18:00"
	templateSlotsDefault = "4"
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

// HandleTemplateCommand ouvre le même formulaire que /sortie mais avec des valeurs déjà remplies
// pour accélérer les tests et le debugging.
func (h *Handler) HandleTemplateCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "create_event_modal",
			Title:    "Organiser une sortie (template)",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "title",
						Label:       "Titre",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: placeholderTitle,
						Value:       templateTitleDefault,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "desc",
						Label:       "Détails (Lieu, etc.)",
						Style:       discordgo.TextInputParagraph,
						Required:    true,
						Placeholder: placeholderDesc,
						Value:       templateDescDefault,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "date",
						Label:       "Date",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: placeholderDate,
						Value:       templateDateDefault,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "time",
						Label:       "Heure",
						Style:       discordgo.TextInputShort,
						Required:    true,
						Placeholder: placeholderTime,
						Value:       templateTimeDefault,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "slots",
						Label:       "Nombre de places",
						Style:       discordgo.TextInputShort,
						Required:    false,
						Placeholder: placeholderSlots,
						Value:       templateSlotsDefault,
					},
				}},
			},
		},
	})
}
