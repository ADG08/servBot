package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Token          string
	ForumChannelID string
	DatabaseURL    string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		// .env is optional when env vars are set externally (e.g. Docker)
	}

	token := os.Getenv("TOKEN")
	if token == "" {
		log.Fatal("Le token n'est pas défini dans le fichier .env")
	}

	forumChannelID := os.Getenv("FORUM_CHANNEL_ID")
	if forumChannelID == "" {
		forumChannelID = "1464818337388167389"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://localhost:5432/servbot?sslmode=disable"
	}

	return &Config{
		Token:          token,
		ForumChannelID: forumChannelID,
		DatabaseURL:    databaseURL,
	}
}
