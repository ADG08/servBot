package main

import (
	"context"
	"log"
	"os"

	"servbot/internal/adapters/discord"
	"servbot/internal/config"
	"servbot/internal/infrastructure/database"
	"servbot/internal/infrastructure/database/sqlc_generated"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Erreur lors de l'initialisation de la base de données: %v", err)
	}
	defer pool.Close()

	q := sqlc_generated.New(pool)
	eventRepo := database.NewEventRepository(q)
	participantRepo := database.NewParticipantRepository(q)

	bot := discord.NewBot(cfg, eventRepo, participantRepo)
	if err := bot.Start(); err != nil {
		log.Printf("❌ Erreur lors du démarrage du bot: %v", err)
		os.Exit(1)
	}
}
