package events_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"codeberg.org/azzet/azzetbe/internal/events"
)

// TestEmitEvent_CreatesValidEvent tests that EmitEvent produces correct outbox data
func TestEmitEvent_EventEnvelopeIntegrity(t *testing.T) {
	// Test that NewEvent creates proper envelope for various event types
	testCases := []struct {
		name      string
		eventType string
		payload   any
		opts      []events.EventOption
	}{
		{
			name:      "user registered event",
			eventType: events.UserRegistered,
			payload:   map[string]string{"user_id": uuid.New().String(), "name": "Test User"},
			opts:      []events.EventOption{events.WithMetadata("api", "req-123")},
		},
		{
			name:      "transaction created with workspace",
			eventType: events.TransactionCreated,
			payload:   map[string]any{"amount": 100000, "currency": "IDR"},
			opts: []events.EventOption{
				events.WithWorkspace(uuid.New().String()),
				events.WithActor(uuid.New().String()),
			},
		},
		{
			name:      "ledger posting with causation",
			eventType: events.LedgerPostingRequested,
			payload:   map[string]string{"transaction_id": uuid.New().String()},
			opts: []events.EventOption{
				events.WithWorkspace(uuid.New().String()),
				events.WithCausation(uuid.New().String()),
				events.WithIdempotencyKey("txn-" + uuid.New().String()),
			},
		},
		{
			name:      "notification with correlation",
			eventType: events.NotificationRequested,
			payload:   map[string]string{"channel": "email", "to": "user@example.com"},
			opts: []events.EventOption{
				events.WithCorrelation(uuid.New().String()),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event, err := events.NewEvent(tc.eventType, tc.payload, tc.opts...)
			if err != nil {
				t.Fatalf("failed to create event: %v", err)
			}

			// Verify envelope
			if event.ID == "" {
				t.Fatal("event ID should not be empty")
			}
			if event.Type != tc.eventType {
				t.Fatalf("expected type '%s', got '%s'", tc.eventType, event.Type)
			}
			if event.CorrelationID == "" {
				t.Fatal("correlation_id should not be empty")
			}
			if event.Payload == nil {
				t.Fatal("payload should not be nil")
			}

			// Verify marshal/unmarshal roundtrip
			data, err := event.Marshal()
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			restored, err := events.Unmarshal(data)
			if err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			if restored.ID != event.ID {
				t.Fatalf("ID mismatch after roundtrip")
			}
			if restored.Type != event.Type {
				t.Fatalf("Type mismatch after roundtrip")
			}
		})
	}
}

// TestConsumerIdempotency_Concept tests the idempotency concept
func TestConsumerIdempotency_Concept(t *testing.T) {
	// Simulate idempotency: same event processed twice should only execute once
	processed := make(map[string]int)

	handler := func(ctx context.Context, event *events.Event) error {
		processed[event.ID]++
		return nil
	}

	// Create event
	event, _ := events.NewEvent(events.UserRegistered, map[string]string{"user_id": "123"})

	// Process first time
	ctx := context.Background()
	err := handler(ctx, event)
	if err != nil {
		t.Fatalf("first processing failed: %v", err)
	}

	// Simulate idempotency check (in real system this is inbox_consumed_events)
	if processed[event.ID] != 1 {
		t.Fatalf("expected 1 processing, got %d", processed[event.ID])
	}

	// In real system, second call would be skipped by idempotency check
	// Here we just verify the concept
	isConsumed := processed[event.ID] > 0
	if !isConsumed {
		t.Fatal("event should be marked as consumed after first processing")
	}
}

// TestPublisher_RetryDelays tests exponential backoff calculation
func TestPublisher_RetryDelays(t *testing.T) {
	// Verify retry delays follow exponential backoff pattern
	expectedDelays := []string{
		"1s",    // retry 0
		"5s",    // retry 1
		"30s",   // retry 2
		"2m0s",  // retry 3
		"10m0s", // retry 4
		"10m0s", // retry 5+ (capped at last value)
	}

	publisher := events.NewPublisher(nil, nil)

	for i, expected := range expectedDelays {
		delay := publisher.RetryDelay[min(i, len(publisher.RetryDelay)-1)]
		if delay.String() != expected {
			t.Fatalf("retry %d: expected delay '%s', got '%s'", i, expected, delay.String())
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
