package discord

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"servbot/internal/domain/entities"
	pkgdiscord "servbot/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

// Garde lettres (y compris accentuées), chiffres, tiret. Le reste → tiret.
var channelNameSanitize = regexp.MustCompile(`[^\p{L}\p{N}-]+`)

func sanitizeChannelName(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = channelNameSanitize.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 100 {
		s = s[:100]
	}
	return s
}

func grantPrivateChannelAccess(s *discordgo.Session, channelID, userID string) {
	if channelID == "" || userID == "" {
		return
	}
	err := s.ChannelPermissionSet(channelID, userID, discordgo.PermissionOverwriteTypeMember,
		discordgo.PermissionViewChannel|discordgo.PermissionSendMessages, 0)
	if err != nil {
		log.Printf("❌ Ajout accès salon privé (channel=%s, user=%s): %v", channelID, userID, err)
	}
}

func revokePrivateChannelAccess(s *discordgo.Session, channelID, userID string) {
	if channelID == "" || userID == "" {
		return
	}
	err := s.ChannelPermissionDelete(channelID, userID)
	if err != nil {
		log.Printf("❌ Retrait accès salon privé (channel=%s, user=%s): %v", channelID, userID, err)
	}
}

// HandleToggleWaitlistMode toggles the waitlist auto/manual mode for an event.
// Only the organizer can change this setting.
func (h *Handler) HandleToggleWaitlistMode(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	userID := interactionUserID(i)
	if userID == "" {
		return
	}

	msgID := i.Message.ID
	if msgID == "" {
		return
	}

	event, err := h.eventUseCase.GetEventByMessageID(ctx, msgID)
	if err != nil || event == nil {
		respondEphemeral(s, i.Interaction, "❌ Événement non trouvé.")
		return
	}
	if event.CreatorID != userID {
		respondEphemeral(s, i.Interaction, "❌ Seul l'organisateur peut changer le mode de liste d'attente.")
		return
	}
	if event.IsEditLocked() {
		respondEphemeral(s, i.Interaction, "🔒 Cette sortie est verrouillée. Le mode de liste d'attente ne peut plus être modifié.")
		return
	}

	event.WaitlistAuto = !event.WaitlistAuto
	if err := h.eventUseCase.UpdateEvent(ctx, event); err != nil {
		log.Printf("❌ Erreur lors du changement de mode waitlist: %v", err)
		respondEphemeral(s, i.Interaction, "❌ Erreur lors de la mise à jour du mode de liste d'attente.")
		return
	}

	// Refresh embed + components to update the button label.
	h.updateEmbed(ctx, s, event.ChannelID, event.MessageID)

	modeLabel := "auto"
	if !event.WaitlistAuto {
		modeLabel = "manuel"
	}
	respondEphemeral(s, i.Interaction, fmt.Sprintf("✅ Mode liste d'attente: **%s**.", modeLabel))
}

func (h *Handler) updateEmbed(ctx context.Context, s *discordgo.Session, channelID, messageID string) {
	event, err := h.eventUseCase.GetEventByMessageID(ctx, messageID)
	if err != nil {
		log.Printf("❌ Erreur lors de la récupération de l'événement: %v", err)
		return
	}
	confirmedParticipants, _ := h.eventUseCase.GetConfirmedParticipants(ctx, event.ID)
	waitlistParticipants, _ := h.eventUseCase.GetWaitlistParticipants(ctx, event.ID)
	confirmedCount := len(confirmedParticipants)
	waitlistCount := len(waitlistParticipants)

	origMsg, err := s.ChannelMessage(channelID, messageID)
	if err != nil || origMsg == nil || len(origMsg.Embeds) == 0 {
		log.Printf("❌ Erreur lors de la récupération du message: %v", err)
		return
	}

	newEmbed := *origMsg.Embeds[0]
	pkgdiscord.UpdateEventEmbed(&newEmbed, event, confirmedCount, waitlistCount)

	components := h.buildComponents(event, waitlistCount, confirmedCount)

	embeds := []*discordgo.MessageEmbed{&newEmbed}
	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         messageID,
		Channel:    channelID,
		Embeds:     &embeds,
		Components: &components,
	}); err != nil {
		log.Printf("❌ Erreur lors de la mise à jour de l'embed: %v", err)
	}
}

const buttonsPerRow = 2

func (h *Handler) buildComponents(event *entities.Event, waitlistCount, confirmedCount int) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent
	if !event.IsEditLocked() {
		buttons = append(buttons, discordgo.Button{Label: "✏️ Modifier la sortie", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("btn_edit_event_%s", event.MessageID)})
	}
	buttons = append(buttons, discordgo.Button{
		Label:    "❓ Poser une question",
		Style:    discordgo.SecondaryButton,
		CustomID: fmt.Sprintf("btn_ask_question_%s", event.MessageID),
	})
	// Toggle waitlist mode (organizer only action).
	// Label reflects the *current* mode, with distinct colors.
	modeLabel := "⏳ Auto"
	modeStyle := discordgo.SuccessButton // vert = auto
	if !event.WaitlistAuto {
		modeLabel = "⏳ Manuel"
		modeStyle = discordgo.PrimaryButton // bleu = manuel
	}
	buttons = append(buttons, discordgo.Button{
		Label:    modeLabel,
		Style:    modeStyle,
		CustomID: fmt.Sprintf("btn_toggle_waitlist_%s", event.MessageID),
	})
	if waitlistCount > 0 {
		buttons = append(buttons, discordgo.Button{Label: "⚙️ Gérer la liste d'attente", Style: discordgo.SecondaryButton, CustomID: fmt.Sprintf("btn_manage_waitlist_%s", event.MessageID)})
	}
	if confirmedCount > 0 {
		buttons = append(buttons, discordgo.Button{Label: "🗑️ Retirer un participant", Style: discordgo.DangerButton, CustomID: fmt.Sprintf("btn_remove_participant_%s", event.MessageID)})
	}
	var components []discordgo.MessageComponent
	for i := 0; i < len(buttons); i += buttonsPerRow {
		end := min(i+buttonsPerRow, len(buttons))
		components = append(components, discordgo.ActionsRow{Components: buttons[i:end]})
	}
	return components
}
