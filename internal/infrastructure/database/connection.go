package database

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates a pgx connection pool for PostgreSQL.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	log.Println("✅ Base de données PostgreSQL connectée.")
	return pool, nil
}
