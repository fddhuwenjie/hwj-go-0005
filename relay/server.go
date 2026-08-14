package relay

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Server struct{ store *Store }

func NewServer(store *Store) *Server { return &Server{store: store} }

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if r.Method == http.MethodPost && path == "v1/endpoints" {
		s.register(w, r)
		return
	}
	if len(parts) == 3 && parts[0] == "v1" && parts[1] == "endpoints" && parts[2] == "events" && r.Method == http.MethodPost {
		s.enqueue(w, r, r.Header.Get("X-Endpoint-ID"))
		return
	}
	if len(parts) == 4 && parts[0] == "v1" && parts[1] == "endpoints" && parts[3] == "events" && r.Method == http.MethodPost {
		s.enqueue(w, r, parts[2])
		return
	}
	if len(parts) == 3 && parts[0] == "v1" && parts[1] == "events" && r.Method == http.MethodGet {
		s.event(w, parts[2])
		return
	}
	if path == "v1/dead-letters" && r.Method == http.MethodGet {
		s.deadLetters(w, r)
		return
	}
	if path == "v1/audit" && r.Method == http.MethodGet {
		s.audit(w, r)
		return
	}
	http.Error(w, "not found", http.StatusNotFound)
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID, URL, Secret string
		RatePerMin      int `json:"rate_per_min"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	endpoint, err := s.store.RegisterEndpoint(input.ID, input.URL, input.Secret, input.RatePerMin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, endpoint)
}

func (s *Server) enqueue(w http.ResponseWriter, r *http.Request, endpointID string) {
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	event, err := s.store.EnqueueEvent(endpointID, r.Header.Get("Idempotency-Key"), payload, time.Now())
	if err == ErrDuplicateEvent {
		writeJSON(w, http.StatusConflict, event)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"event": event, "signature": SignForEndpoint(s.store, endpointID, payload)})
}

func SignForEndpoint(store *Store, endpointID string, payload []byte) string {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return Sign(store.endpoints[endpointID].Secret, payload)
}

func (s *Server) event(w http.ResponseWriter, id string) {
	event, err := s.store.GetEvent(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (s *Server) deadLetters(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	endpointID := r.URL.Query().Get("endpoint_id")
	writeJSON(w, http.StatusOK, s.store.ListDeadLetters(endpointID, limit))
}

func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	endpointID := r.URL.Query().Get("endpoint_id")
	eventID := r.URL.Query().Get("event_id")
	writeJSON(w, http.StatusOK, s.store.ListAudit(endpointID, eventID, limit))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
