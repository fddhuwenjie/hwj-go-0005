package relay

import "time"

type Endpoint struct {
	ID         string    `json:"id"`
	URL        string    `json:"url"`
	Secret     string    `json:"secret"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	RatePerMin int       `json:"rate_per_min"`
}

type Event struct {
	ID             string    `json:"id"`
	EndpointID     string    `json:"endpoint_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	Payload        []byte    `json:"payload"`
	Attempt        int       `json:"attempt"`
	NextAttemptAt  time.Time `json:"next_attempt_at"`
	Status         string    `json:"status"`
	LastError      string    `json:"last_error,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type AuditEntry struct {
	ID         int64     `json:"id"`
	EventID    string    `json:"event_id"`
	EndpointID string    `json:"endpoint_id"`
	Action     string    `json:"action"`
	Detail     string    `json:"detail"`
	At         time.Time `json:"at"`
}
