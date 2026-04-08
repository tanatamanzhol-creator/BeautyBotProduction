package health

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Status struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Uptime    string            `json:"uptime"`
	Checks    map[string]string `json:"checks"`
}

var startTime = time.Now()

// StartServer starts a lightweight HTTP server for health checks.
// UptimeRobot, Telegram, or any monitoring tool can ping GET /health
// to verify the service is alive.
//
// Returns 200 OK with JSON when healthy.
// Returns 503 Service Unavailable when DB is unreachable.
func StartServer(port int, pool *pgxpool.Pool) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		checks := make(map[string]string)
		overallOK := true

		// Check database
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			checks["database"] = fmt.Sprintf("ERROR: %v", err)
			overallOK = false
		} else {
			checks["database"] = "OK"
		}

		// Check pool stats
		stats := pool.Stat()
		checks["db_connections"] = fmt.Sprintf(
			"total=%d idle=%d acquired=%d",
			stats.TotalConns(),
			stats.IdleConns(),
			stats.AcquiredConns(),
		)

		status := Status{
			Status:    "OK",
			Timestamp: time.Now().Format(time.RFC3339),
			Uptime:    time.Since(startTime).Round(time.Second).String(),
			Checks:    checks,
		}

		if !overallOK {
			status.Status = "ERROR"
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})

	// Simple ping for UptimeRobot
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Health check server started on %s", addr)

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("Health server error: %v", err)
		}
	}()
}
