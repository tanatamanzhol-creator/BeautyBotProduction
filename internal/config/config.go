package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL     string
	AdminTelegramID int64
	HealthPort      int
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	adminID, err := strconv.ParseInt(os.Getenv("ADMIN_TELEGRAM_ID"), 10, 64)
	if err != nil {
		log.Fatal("ADMIN_TELEGRAM_ID must be a valid integer")
	}

	healthPort := 8080
	if p := os.Getenv("HEALTH_PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			healthPort = parsed
		}
	}

	return &Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		AdminTelegramID: adminID,
		HealthPort:      healthPort,
	}
}
