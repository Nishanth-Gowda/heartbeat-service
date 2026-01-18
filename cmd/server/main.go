package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"time"

	_ "github.com/lib/pq"
	"github.com/nishanth-gowda/heartbeat-service/api/pb"
	"github.com/nishanth-gowda/heartbeat-service/internal/models"
	"github.com/nishanth-gowda/heartbeat-service/internal/repository"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

type Server struct {
	pb.UnimplementedMonitorServer
	redis_repo *repository.RedisRepository
	pg_repo    *repository.PostgresRepository
}

// 2. Implement the SendHeartbeat RPC
func (s *Server) SendHeartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	nodeID := int(req.ServiceId)

	// High-Speed Write: Update the heartbeat timestamp in Redis (ZSET)
	err := s.redis_repo.RegisterHeartbeat(ctx, nodeID)
	if err != nil {
		slog.Error("Failed to register heartbeat", "nodeID", nodeID, "err", err)
		return &pb.HeartbeatResponse{Success: false}, err
	}

	// Recovery Check: Did this node just come back from the dead?
	// This uses a fast Redis Set check (SISMEMBER)
	if s.redis_repo.IsNodeDown(ctx, nodeID) {
		go s.handleRecovery(nodeID) // Handle in background to keep latency low
	}

	return &pb.HeartbeatResponse{Success: true}, nil
}

// 3. Handle Recovery (The "Healer" Logic)
func (s *Server) handleRecovery(nodeID int) {
	ctx := context.Background()
	slog.Info("Service has RECOVERED!", "nodeID", nodeID)

	// Update Postgres Status
	if err := s.pg_repo.SetServiceStatus(ctx, nodeID, models.StatusUp); err != nil {
		slog.Error("DB Error marking UP", "err", err)
	}

	// Log the Incident
	incident := &models.Incident{
		ServiceID: nodeID,
		Type:      models.EventCameUp,
		Time:      time.Now().Format(time.RFC3339), // Helper needed or use time.Now()
		Details:   "Heartbeat received after failure",
	}
	err := s.pg_repo.LogIncident(ctx, *incident)
	if err != nil {
		slog.Error("DB Error logging incident", "err", err)
	}

	// Remove from the "Down" list in Redis so we don't trigger this again
	err = s.redis_repo.MarkNodeUp(ctx, nodeID)
	if err != nil {
		slog.Error("Redis Error marking node up", "err", err)
	}
}

// 4. Main Entry Point
func main() {
	// Setup DB Connections
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/heartbeat_db?sslmode=disable")
	if err != nil {
		slog.Error("Failed to open database connection", "err", err)
		return
	}

	// Setup Repos
	redis_repo := repository.NewRedisRepository(rdb)
	pg_repo := repository.NewPostgresRepository(db)

	// Start TCP Listener
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		slog.Error("failed to listen", "err", err)
		return
	}

	// Start gRPC Server
	s := grpc.NewServer()
	pb.RegisterMonitorServer(s, &Server{
		redis_repo: redis_repo,
		pg_repo:    pg_repo,
	})

	slog.Info("🚀 gRPC Heartbeat Server listening on :9090")
	if err := s.Serve(lis); err != nil {
		slog.Error("failed to serve", "err", err)
	}
}
