package relay

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	ErrEndpointNotFound = errors.New("endpoint not found")
	ErrDuplicateEvent   = errors.New("duplicate idempotency key")
	ErrEventNotFound    = errors.New("event not found")
)

type Store struct {
	mu        sync.RWMutex
	endpoints map[string]Endpoint
	events    map[string]Event
	keys      map[string]string
	audit     []AuditEntry
	nextAudit int64
}

func NewStore() *Store {
	return &Store{endpoints: make(map[string]Endpoint), events: make(map[string]Event), keys: make(map[string]string)}
}

func (s *Store) RegisterEndpoint(id, url, secret string, ratePerMin int) (Endpoint, error) {
	if id == "" || url == "" || secret == "" {
		return Endpoint{}, errors.New("id, url, and secret are required")
	}
	if ratePerMin < 1 {
		ratePerMin = 60
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.endpoints[id]; ok {
		old.URL, old.Secret, old.Active, old.RatePerMin = url, secret, true, ratePerMin
		s.endpoints[id] = old
		return old, nil
	}
	endpoint := Endpoint{ID: id, URL: url, Secret: secret, Active: true, RatePerMin: ratePerMin, CreatedAt: time.Now().UTC()}
	s.endpoints[id] = endpoint
	s.recordLocked("", id, "endpoint_registered", "endpoint is active")
	return endpoint, nil
}

func (s *Store) EnqueueEvent(endpointID, key string, payload []byte, now time.Time) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	endpoint, ok := s.endpoints[endpointID]
	if !ok || !endpoint.Active {
		return Event{}, ErrEndpointNotFound
	}
	if key == "" {
		return Event{}, errors.New("idempotency key is required")
	}
	if existingID, exists := s.keys[endpointID+"\x00"+key]; exists {
		return s.events[existingID], ErrDuplicateEvent
	}
	id := fmt.Sprintf("evt_%06d", len(s.events)+1)
	event := Event{ID: id, EndpointID: endpointID, IdempotencyKey: key, Payload: append([]byte(nil), payload...), NextAttemptAt: now.UTC(), Status: "pending", CreatedAt: now.UTC()}
	s.events[id] = event
	s.keys[endpointID+"\x00"+key] = id
	s.recordLocked(id, endpointID, "event_queued", "delivery scheduled")
	return event, nil
}

func (s *Store) GetEvent(id string) (Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event, ok := s.events[id]
	if !ok {
		return Event{}, ErrEventNotFound
	}
	return event, nil
}

func (s *Store) MarkDelivery(id, status, detail string, now time.Time) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, ok := s.events[id]
	if !ok {
		return Event{}, ErrEventNotFound
	}
	if status != "delivered" && status != "pending" && status != "dead" {
		return Event{}, errors.New("invalid delivery status")
	}
	event.Attempt++
	event.Status = status
	event.LastError = detail
	if status == "pending" {
		delay := time.Second * time.Duration(1<<(min(event.Attempt, 6)-1))
		event.NextAttemptAt = now.UTC().Add(delay)
	}
	s.events[id] = event
	s.recordLocked(id, event.EndpointID, "delivery_"+status, detail)
	return event, nil
}

func (s *Store) ListDeadLetters(endpointID string, limit int) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit < 1 {
		limit = 50
	}
	result := make([]Event, 0, limit)
	for _, event := range s.events {
		if event.Status == "dead" {
			result = append(result, event)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	if len(result) > limit {
		result = result[:limit]
	}
	if endpointID != "" {
		filtered := result[:0]
		for _, event := range result {
			if event.EndpointID == endpointID {
				filtered = append(filtered, event)
			}
		}
		result = filtered
	}
	return result
}

func (s *Store) ListAudit(endpointID, eventID string, limit int) []AuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit < 1 {
		limit = 50
	}
	result := make([]AuditEntry, 0, limit)
	for i := len(s.audit) - 1; i >= 0; i-- {
		entry := s.audit[i]
		if (endpointID == "" || entry.EndpointID == endpointID) && (eventID == "" || entry.EventID == eventID) {
			result = append(result, entry)
			if len(result) == limit {
				break
			}
		}
	}
	return result
}

func (s *Store) recordLocked(eventID, endpointID, action, detail string) {
	s.nextAudit++
	s.audit = append(s.audit, AuditEntry{ID: s.nextAudit, EventID: eventID, EndpointID: endpointID, Action: action, Detail: detail, At: time.Now().UTC()})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
