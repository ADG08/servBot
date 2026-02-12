package output

// Translator exposes a minimal i18n contract for user-facing messages.
// Implementations provide message lookup + templating for a given locale.
type T interface {
	// T renders the message identified by key for the given locale.
	// data is an optional map used for template placeholders (may be nil).
	T(locale, key string, data map[string]any) string
}
