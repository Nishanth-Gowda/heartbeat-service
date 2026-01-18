package models

// Define a custom type for type safety in Go
type ServiceStatus string
type EventType string

// Constants matching Postgres ENUMs exactly
const (
	StatusUp       ServiceStatus = "UP"
	StatusDown     ServiceStatus = "DOWN"
	StatusDegraded ServiceStatus = "DEGRADED"
)

const (
	EventWentDown EventType = "WENT_DOWN"
	EventCameUp   EventType = "CAME_UP"
)

// Incident represents a row in the incident_log table
type Incident struct {
	ID        int       `json:"id"`
	ServiceID int       `json:"service_id"`
	Type      EventType `json:"event_type"` // Uses the custom type
	Time      string    `json:"event_time"`
	Details   string    `json:"details"`
}

type Service struct {
	ID            int           `json:"id"`
	Name          string        `json:"service_name"`
	URL           string        `json:"service_url"`
	Region        string        `json:"region"`
	Status        ServiceStatus `json:"status"`
	LastHeartbeat string        `json:"last_heartbeat"`
	CreatedAt     string        `json:"created_at"`
}
