package db

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTimeout is the default timeout for all database operations.
// If a query takes longer — it's cancelled automatically.
const DBTimeout = 5 * time.Second

// NewContext returns a context with the standard DB timeout.
// Use this in every repository method:
//
//	ctx, cancel := db.NewContext(ctx)
//	defer cancel()
func NewContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, DBTimeout)
}

func Connect(databaseURL string) *pgxpool.Pool {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("Unable to parse database URL: %v", err)
	}

	// Pool settings
	config.MaxConns = 20              // max simultaneous connections
	config.MinConns = 2               // keep 2 connections warm
	config.MaxConnLifetime = time.Hour // recycle connections every hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	// Verify connection with timeout
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	log.Println("Connected to PostgreSQL")
	return pool
}

func RunMigrations(pool *pgxpool.Pool) {
	// Run all migration files in order
	files := []string{
		"internal/db/migrations/001_init.sql",
		"internal/db/migrations/002_master_telegram_id.sql",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, f := range files {
		migration, err := os.ReadFile(f)
		if err != nil {
			log.Fatalf("Could not read migration file %s: %v", f, err)
		}
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			log.Fatalf("Could not run migration %s: %v", f, err)
		}
		log.Printf("Migration applied: %s", f)
	}
}
