package relay

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerPublicAPI(t *testing.T) {
	server := NewServer(NewStore())
	register := httptest.NewRequest(http.MethodPost, "/v1/endpoints", bytes.NewBufferString(`{"id":"orders","url":"https://example.test/hook","secret":"s"}`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, register)
	if response.Code != http.StatusCreated {
		t.Fatalf("register status = %d", response.Code)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/endpoints/orders/events", bytes.NewBufferString(`{"id":1}`))
	req.Header.Set("Idempotency-Key", "a")
	response = httptest.NewRecorder()
	server.ServeHTTP(response, req)
	if response.Code != http.StatusAccepted {
		t.Fatalf("enqueue status = %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Event     Event  `json:"event"`
		Signature string `json:"signature"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Event.ID == "" || !Verify("s", []byte(`{"id":1}`), body.Signature) {
		t.Fatalf("response = %s", response.Body.String())
	}
	get := httptest.NewRequest(http.MethodGet, "/v1/events/"+body.Event.ID, nil)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, get)
	if response.Code != http.StatusOK {
		t.Fatalf("get status = %d", response.Code)
	}
}
