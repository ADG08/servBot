package discord

import (
	"github.com/bwmarrin/discordgo"
)

const (
	placeholderTitle  = "Ex: Resto, Ciné, Soirée jeux..."
	placeholderDesc   = "Lieu, adresse, infos pratiques..."
	placeholderDate   = "Ex: 15/02/2026 (jour/mois/année)"
	placeholderTime   = "Ex: 14:00 ou 18:30"
	placeholderSlots  = "Ex: 4 ou vide = illimité"
	templateTitleDefault  = "Sortie de test"
	templateDescDefault   = "Lieu de test, adresse, infos pratiques..."
	templateDateDefault   = "01/01/2030"
	templateTimeDefault   = "18:00"
	templateSlotsDefault  = "4"
)

type createEventModalDefaults struct {
	Title, Desc, Date, Time, Slots string
}

func buildCreateEventModalComponents(d *createEventModalDefaults) []discordgo.MessageComponent {
	if d == nil {
		d = &createEventModalDefaults{}
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "title", Label: "Titre", Style: discordgo.TextInputShort, Required: true, Placeholder: placeholderTitle, Value: d.Title},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "desc", Label: "Détails (Lieu, etc.)", Style: discordgo.TextInputParagraph, Required: true, Placeholder: placeholderDesc, Value: d.Desc},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "date", Label: "Date", Style: discordgo.TextInputShort, Required: true, Placeholder: placeholderDate, Value: d.Date},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "time", Label: "Heure", Style: discordgo.TextInputShort, Required: true, Placeholder: placeholderTime, Value: d.Time},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "slots", Label: "Nombre de places", Style: discordgo.TextInputShort, Required: false, Placeholder: placeholderSlots, Value: d.Slots},
		}},
	}
}

func (h *Handler) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   "create_event_modal",
			Title:      "Organiser une sortie",
			Components: buildCreateEventModalComponents(nil),
		},
	})
}

func (h *Handler) HandleTemplateCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "create_event_modal",
			Title:    "Organiser une sortie (template)",
			Components: buildCreateEventModalComponents(&createEventModalDefaults{
				Title: templateTitleDefault,
				Desc:  templateDescDefault,
				Date:  templateDateDefault,
				Time:  templateTimeDefault,
				Slots: templateSlotsDefault,
			}),
		},
	})
}
