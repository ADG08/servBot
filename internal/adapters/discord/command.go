package discord

import (
	"github.com/bwmarrin/discordgo"
)

type createEventModalDefaults struct {
	Title, Desc, Date, Time, Slots string
}

func (h *Handler) buildCreateEventModalComponents(d *createEventModalDefaults) []discordgo.MessageComponent {
	if d == nil {
		d = &createEventModalDefaults{}
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "title", Label: h.translate("ui.label_title", nil), Style: discordgo.TextInputShort, Required: true, Placeholder: h.translate("ui.placeholder_title", nil), Value: d.Title},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "desc", Label: h.translate("ui.label_details", nil), Style: discordgo.TextInputParagraph, Required: true, Placeholder: h.translate("ui.placeholder_desc", nil), Value: d.Desc},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "date", Label: h.translate("ui.label_date", nil), Style: discordgo.TextInputShort, Required: true, Placeholder: h.translate("ui.placeholder_date", nil), Value: d.Date},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "time", Label: h.translate("ui.label_time", nil), Style: discordgo.TextInputShort, Required: true, Placeholder: h.translate("ui.placeholder_time", nil), Value: d.Time},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.TextInput{CustomID: "slots", Label: h.translate("ui.label_slots", nil), Style: discordgo.TextInputShort, Required: false, Placeholder: h.translate("ui.placeholder_slots", nil), Value: d.Slots},
		}},
	}
}

func (h *Handler) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   "create_event_modal",
			Title:      h.translate("ui.modal_create_event_title", nil),
			Components: h.buildCreateEventModalComponents(nil),
		},
	})
}

func (h *Handler) HandleTemplateCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "create_event_modal",
			Title:    h.translate("ui.modal_create_event_template_title", nil),
			Components: h.buildCreateEventModalComponents(&createEventModalDefaults{
				Title: h.translate("ui.template_title_default", nil),
				Desc:  h.translate("ui.template_desc_default", nil),
				Date:  h.translate("ui.template_date_default", nil),
				Time:  h.translate("ui.template_time_default", nil),
				Slots: h.translate("ui.template_slots_default", nil),
			}),
		},
	})
}
