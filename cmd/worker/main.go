package main

import (
	"context"
	"database/sql"
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

	slog.Info("Detected %d failures: %v", len(dead_service_ids), dead_service_ids)

	for _, service_id := range dead_service_ids {
		incident := models.Incident{
			ServiceID: service_id,
			Type:      models.EventWentDown,
			Time:      time.Now().Format(time.RFC3339),
			Details:   "Heartbeat timeout exceeded",
		}
		if err := pg_repo.LogIncident(ctx, incident); err != nil {
			slog.Info("Failed to log incident for node %d: %v", service_id, err)
			continue // Don't remove from Redis if DB write failed (try again next tick)
		}

		// Update Service Status
		if err := pg_repo.SetServiceStatus(ctx, service_id, models.StatusDown); err != nil {
			slog.Info("Failed to update status for node %d: %v", service_id, err)
		}

		// remove from redis
		if err := redis_repo.RemoveService(ctx, service_id); err != nil {
			slog.Info("Failed to remove service %d from Redis: %v", service_id, err)
		}

		// send alert
		slog.Info("Sending alert for service %d", service_id)
	}
}
