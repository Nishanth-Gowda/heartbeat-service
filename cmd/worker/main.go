package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"time"

	"github.com/nishanth-gowda/heartbeat-service/internal/models"
	"github.com/nishanth-gowda/heartbeat-service/internal/repository"
	"github.com/redis/go-redis/v9"
)

// Configuration (Move these to env variables in production)
const (
	RedisAddr      = "localhost:6379"
	PostgresDSN    = "postgres://user:password@localhost:5432/heartbeat_db?sslmode=disable"
	CheckInterval  = 5 * time.Second  // How often we run the check
	FailureTimeout = 10 * time.Second // How long silence is allowed before marking DOWN
)

func main() {
	// Initialise Redis and Postgres connections
	redis_db := redis.NewClient(&redis.Options{Addr: RedisAddr})

	db, err := sql.Open("postgres", PostgresDSN)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Initialise repositories
	pg_repo := repository.NewPostgresRepository(db)
	redis_repo := repository.NewRedisRepository(redis_db)

	slog.Info("Failure Detector Worker Started...")

	ticker := time.NewTicker(CheckInterval)
	defer ticker.Stop()

	// Run the detector loop
	for range ticker.C {
		detectFailures(pg_repo, redis_repo)
	}
}

func detectFailures(pg_repo *repository.PostgresRepository, redis_repo *repository.RedisRepository) {

	ctx := context.Background()

	threshold := time.Now().Add(-FailureTimeout).Unix()

	// Get services that have not registered a heartbeat in the last FailureTimeout
	dead_service_ids, err := redis_repo.GetDeadServices(ctx, threshold)
	if err != nil {
		panic(err)
	}

	if len(dead_service_ids) == 0 {
		return
	}

	slog.Info("Detected failures", "count", len(dead_service_ids), "service_ids", dead_service_ids)

	for _, service_id := range dead_service_ids {
		incident := models.Incident{
			ServiceID: service_id,
			Type:      models.EventWentDown,
			Time:      time.Now().Format(time.RFC3339),
			Details:   "Heartbeat timeout exceeded",
		}
		if err := pg_repo.LogIncident(ctx, incident); err != nil {
			slog.Error("Failed to log incident", "service_id", service_id, "error", err)
			continue // Don't remove from Redis if DB write failed (try again next tick)
		}

		// Mark as DOWN in Redis Set (NEW STEP!)
		// This allows the API to detect when it comes back up.
		if err := redis_repo.MarkNodeDown(ctx, service_id); err != nil {
			log.Printf("Failed to mark node down in Redis: %v", err)
		}

		// remove from redis
		if err := redis_repo.RemoveService(ctx, service_id); err != nil {
			slog.Error("Failed to remove service from Redis", "service_id", service_id, "error", err)
		}

		// send alert
		slog.Info("Sending alert", "service_id", service_id)
	}
}
