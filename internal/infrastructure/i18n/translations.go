package i18n

import (
	"embed"
	"log"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"

	"servbot/internal/ports/output"
)

//go:embed active.*.toml
var localeFS embed.FS

// Ensure Translator implements the output.Translator port.
var _ output.T = (*Translator)(nil)

// Translator is a thin wrapper around go-i18n's Bundle/Localizer.
type Translator struct {
	bundle          *i18n.Bundle
	defaultLanguage language.Tag
}

// NewTranslator builds a Translator backed by go-i18n using the given default
// locale (e.g. "fr").
//
// It currently loads translations from the embedded active.*.toml files.
func NewTranslator(defaultLocale string) *Translator {
	tag, err := language.Parse(defaultLocale)
	if err != nil {
		tag = language.English
	}
	bundle := i18n.NewBundle(tag)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	for _, file := range []string{"active.fr.toml", "active.en.toml"} {
		if _, err := bundle.LoadMessageFileFS(localeFS, file); err != nil {
			log.Printf("i18n: failed to load %s: %v", file, err)
		}
	}

	return &Translator{
		bundle:          bundle,
		defaultLanguage: tag,
	}
}

// T renders the message identified by key for the given locale.
// If the key/locale is not found, it falls back to the default locale,
// then finally to the key itself.
func (t *Translator) T(locale, key string, data map[string]any) string {
	if key == "" {
		return ""
	}

	languages := []string{}
	if locale != "" {
		languages = append(languages, locale)
	}
	languages = append(languages, t.defaultLanguage.String())

	localizer := i18n.NewLocalizer(t.bundle, languages...)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: data,
	})
	if err != nil {
		log.Printf("i18n: localize failed (key=%s, locales=%v): %v", key, languages, err)
		return key
	}
	return msg
}
