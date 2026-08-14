package relay

import (
	"errors"
	"testing"
	"time"
)

func TestStoreLifecycle(t *testing.T) {
	s := NewStore()
	now := time.Unix(100, 0)
	if _, err := s.RegisterEndpoint("billing", "https://example.test/hook", "secret", 10); err != nil {
		t.Fatal(err)
	}
	event, err := s.EnqueueEvent("billing", "order-1", []byte(`{"ok":true}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.EnqueueEvent("billing", "order-1", []byte(`{"ok":true}`), now); !errors.Is(err, ErrDuplicateEvent) {
		t.Fatalf("duplicate error = %v", err)
	}
	if event.Status != "pending" || !event.NextAttemptAt.Equal(now) {
		t.Fatalf("unexpected event: %+v", event)
	}
	updated, err := s.MarkDelivery(event.ID, "pending", "temporary", now)
	if err != nil || updated.Attempt != 1 || !updated.NextAttemptAt.Equal(now.Add(time.Second)) {
		t.Fatalf("retry = %+v, err=%v", updated, err)
	}
	if _, err := s.MarkDelivery(event.ID, "dead", "permanent", now); err != nil {
		t.Fatal(err)
	}
	dead := s.ListDeadLetters("billing", 10)
	if len(dead) != 1 || dead[0].ID != event.ID {
		t.Fatalf("dead letters = %+v", dead)
	}
	if len(s.ListAudit("billing", "", 10)) < 4 {
		t.Fatal("expected audit entries")
	}
}
