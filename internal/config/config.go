package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Token          string
	ForumChannelID string
	DatabaseURL    string
	GuildID        string
}

// Load charge la configuration depuis les variables d'environnement et la valide.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		// .env est optionnel lorsque les variables sont fournies par l'environnement (Docker, CI, etc.).
	}

	cfg := &Config{
		Token:          os.Getenv("TOKEN"),
		ForumChannelID: os.Getenv("FORUM_CHANNEL_ID"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		GuildID:        os.Getenv("GUILD_ID"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate applique toutes les règles métier sur la configuration chargée.
func (c *Config) validate() error {
	if strings.TrimSpace(c.Token) == "" {
		return fmt.Errorf("config: TOKEN est requis et ne peut pas être vide")
	}

	if strings.TrimSpace(c.ForumChannelID) == "" {
		return fmt.Errorf("config: FORUM_CHANNEL_ID est requis et ne peut pas être vide")
	}

	for _, r := range c.ForumChannelID {
		if r < '0' || r > '9' {
			return fmt.Errorf("config: FORUM_CHANNEL_ID doit être un ID de salon Discord (chiffres uniquement)")
		}
	}

	if strings.TrimSpace(c.DatabaseURL) == "" {
		// Valeur par défaut utile en local lorsque DATABASE_URL n'est pas fournie.
		c.DatabaseURL = "postgres://localhost:5432/servbot?sslmode=disable"
	}

	parsed, err := url.Parse(c.DatabaseURL)
	if err != nil {
		return fmt.Errorf("config: DATABASE_URL invalide (%q): %w", c.DatabaseURL, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("config: DATABASE_URL invalide (%q): scheme ou host manquant", c.DatabaseURL)
	}

	return nil
}
