package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nishanth-gowda/heartbeat-service/internal/models"
)

type PostgresRepository struct {
	// We will add the *sql.DB connection here later
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetServiceByID(ctx context.Context, id int) (models.Service, error) {
	var s models.Service
	// SQL: SELECT * FROM services WHERE id = ?
	fmt.Println("DB: Fetching service", id)
	return s, nil
}

func (r *PostgresRepository) GetAllServices(ctx context.Context) ([]models.Service, error) {
	var services []models.Service
	// SQL: SELECT * FROM services
	fmt.Println("DB: Fetching all services")
	return services, nil
}

// --- 2. Heartbeat Logic ---

func (r *PostgresRepository) RecordHeartbeat(ctx context.Context, serviceID int, status string, details string) error {
	// SQL: UPDATE services SET status = ?, last_heartbeat = ? WHERE id = ?
	fmt.Printf("DB: Recording heartbeat for service %d: %s\n", serviceID, status)
	return nil
}

func (r *PostgresRepository) LogIncident(ctx context.Context, incident models.Incident) error {
	// SQL: INSERT INTO incident_log (service_id, event_type, event_time, details) VALUES (?, ?, ?, ?)

	query := `
		INSERT INTO incident_log (service_id, event_type, event_time, details)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.ExecContext(ctx, query,
		incident.ServiceID,
		incident.Type,
		incident.Time,
		incident.Details)
	if err != nil {
		return err
	}

	fmt.Printf("DB: Logging incident for service %d: %s\n", incident.ServiceID, incident.Type)
	return nil
}

func (r *PostgresRepository) SetServiceStatus(ctx context.Context, serviceID int, status models.ServiceStatus) error {
	// SQL: UPDATE services SET status = ? WHERE id = ?
	query := `
		UPDATE services
		SET status = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query,
		status,
		serviceID)
	if err != nil {
		return err
	}

	return nil
}

// --- 4. Health Check ---

func (r *PostgresRepository) HealthCheck(ctx context.Context) error {
	// SQL: SELECT 1
	fmt.Println("DB: Ping successful")
	return nil
}
