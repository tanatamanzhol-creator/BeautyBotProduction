package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"beauty-bot/internal/bot"
	"beauty-bot/internal/config"
	"beauty-bot/internal/db"
	"beauty-bot/internal/health"
	"beauty-bot/internal/logger"
	"beauty-bot/internal/repository"
	"beauty-bot/internal/scheduler"
	"beauty-bot/internal/types"
)

func main() {
	// ── Init logger ───────────────────────────────────────────────────────
	logger.Init()

	// ── Load config ───────────────────────────────────────────────────────
	cfg := config.Load()

	// ── Connect to database ───────────────────────────────────────────────
	pool := db.Connect(cfg.DatabaseURL)
	defer pool.Close()

	// ── Run migrations ────────────────────────────────────────────────────
	db.RunMigrations(pool)

	// ── Start health check server ─────────────────────────────────────────
	// Responds to GET /health and GET /ping
	// Point UptimeRobot at: http://your-server-ip:8080/ping
	health.StartServer(cfg.HealthPort, pool)

	// ── Build repos ───────────────────────────────────────────────────────
	repos := &types.Repos{
		Master:      repository.NewMasterRepo(pool),
		Client:      repository.NewClientRepo(pool),
		Service:     repository.NewServiceRepo(pool),
		Booking:     repository.NewBookingRepo(pool),
		Review:      repository.NewReviewRepo(pool),
		BlockedSlot: repository.NewBlockedSlotRepo(pool),
	}

	// ── Build manager ─────────────────────────────────────────────────────
	manager := bot.NewManager(repos, cfg.AdminTelegramID)

	// ── Context with graceful shutdown ────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Start all active bots ─────────────────────────────────────────────
	if err := manager.StartAll(ctx); err != nil {
		log.Fatalf("Failed to start bots: %v", err)
	}

	// ── Start scheduler ───────────────────────────────────────────────────
	sched := scheduler.New(manager, repos)
	sched.Start()
	defer sched.Stop()

	log.Println("✅ Beauty Bot platform started")
	log.Printf("Health check: http://localhost:%d/health", cfg.HealthPort)

	// ── Wait for shutdown signal ──────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Printf("Received signal %v, shutting down gracefully...", sig)
	cancel()
	log.Println("Done.")
}
