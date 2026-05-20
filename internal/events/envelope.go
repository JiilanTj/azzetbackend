package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event is the standard event envelope used across the system
type Event struct {
	ID             string          `json:"event_id"`
	Type           string          `json:"event_type"`
	Version        int             `json:"event_version"`
	WorkspaceID    *string         `json:"workspace_id,omitempty"`
	ActorID        *string         `json:"actor_id,omitempty"`
	CorrelationID  string          `json:"correlation_id"`
	CausationID    *string         `json:"causation_id,omitempty"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty"`
	OccurredAt     time.Time       `json:"occurred_at"`
	Payload        json.RawMessage `json:"payload"`
	Metadata       *EventMetadata  `json:"metadata,omitempty"`
}

// EventMetadata contains additional context about the event
type EventMetadata struct {
	Source    string `json:"source"`              // "api", "worker", "consumer"
	RequestID string `json:"request_id,omitempty"`
}

// NewEvent creates a new event with a generated ID and correlation ID
func NewEvent(eventType string, payload any, opts ...EventOption) (*Event, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	e := &Event{
		ID:            uuid.New().String(),
		Type:          eventType,
		Version:       1,
		CorrelationID: uuid.New().String(),
		OccurredAt:    time.Now(),
		Payload:       payloadBytes,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e, nil
}

// EventOption is a functional option for configuring events
type EventOption func(*Event)

func WithWorkspace(workspaceID string) EventOption {
	return func(e *Event) {
		e.WorkspaceID = &workspaceID
	}
}

func WithActor(actorID string) EventOption {
	return func(e *Event) {
		e.ActorID = &actorID
	}
}

func WithCorrelation(correlationID string) EventOption {
	return func(e *Event) {
		e.CorrelationID = correlationID
	}
}

func WithCausation(causationID string) EventOption {
	return func(e *Event) {
		e.CausationID = &causationID
	}
}

func WithIdempotencyKey(key string) EventOption {
	return func(e *Event) {
		e.IdempotencyKey = &key
	}
}

func WithMetadata(source, requestID string) EventOption {
	return func(e *Event) {
		e.Metadata = &EventMetadata{
			Source:    source,
			RequestID: requestID,
		}
	}
}

// Marshal serializes the event to JSON
func (e *Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Unmarshal deserializes JSON into an Event
func Unmarshal(data []byte) (*Event, error) {
	var e Event
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// Subject returns the NATS subject for this event (e.g., "accounting.transaction.created")
func (e *Event) Subject() string {
	return e.Type
}
