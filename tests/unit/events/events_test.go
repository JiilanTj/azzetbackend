package events_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"codeberg.org/azzet/azzetbe/internal/events"
)

func TestNewEvent(t *testing.T) {
	payload := map[string]string{"user_id": "123", "name": "Test"}

	event, err := events.NewEvent(events.UserRegistered, payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.ID == "" {
		t.Fatal("expected non-empty event ID")
	}
	if event.Type != events.UserRegistered {
		t.Fatalf("expected type '%s', got '%s'", events.UserRegistered, event.Type)
	}
	if event.Version != 1 {
		t.Fatalf("expected version 1, got %d", event.Version)
	}
	if event.CorrelationID == "" {
		t.Fatal("expected non-empty correlation ID")
	}
	if event.OccurredAt.IsZero() {
		t.Fatal("expected non-zero occurred_at")
	}
	if event.Payload == nil {
		t.Fatal("expected non-nil payload")
	}
}

func TestNewEvent_WithOptions(t *testing.T) {
	payload := map[string]string{"key": "value"}

	event, err := events.NewEvent(events.TransactionCreated, payload,
		events.WithWorkspace("ws-123"),
		events.WithActor("actor-456"),
		events.WithCorrelation("corr-789"),
		events.WithCausation("cause-000"),
		events.WithIdempotencyKey("idem-111"),
		events.WithMetadata("api", "req-222"),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.WorkspaceID == nil || *event.WorkspaceID != "ws-123" {
		t.Fatalf("expected workspace_id 'ws-123', got %v", event.WorkspaceID)
	}
	if event.ActorID == nil || *event.ActorID != "actor-456" {
		t.Fatalf("expected actor_id 'actor-456', got %v", event.ActorID)
	}
	if event.CorrelationID != "corr-789" {
		t.Fatalf("expected correlation_id 'corr-789', got '%s'", event.CorrelationID)
	}
	if event.CausationID == nil || *event.CausationID != "cause-000" {
		t.Fatalf("expected causation_id 'cause-000', got %v", event.CausationID)
	}
	if event.IdempotencyKey == nil || *event.IdempotencyKey != "idem-111" {
		t.Fatalf("expected idempotency_key 'idem-111', got %v", event.IdempotencyKey)
	}
	if event.Metadata == nil {
		t.Fatal("expected non-nil metadata")
	}
	if event.Metadata.Source != "api" {
		t.Fatalf("expected source 'api', got '%s'", event.Metadata.Source)
	}
	if event.Metadata.RequestID != "req-222" {
		t.Fatalf("expected request_id 'req-222', got '%s'", event.Metadata.RequestID)
	}
}

func TestEvent_MarshalUnmarshal(t *testing.T) {
	payload := map[string]string{"user_id": "abc"}

	original, _ := events.NewEvent(events.UserRegistered, payload,
		events.WithWorkspace("ws-1"),
		events.WithActor("actor-1"),
	)

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	restored, err := events.Unmarshal(data)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if restored.ID != original.ID {
		t.Fatalf("expected ID '%s', got '%s'", original.ID, restored.ID)
	}
	if restored.Type != original.Type {
		t.Fatalf("expected type '%s', got '%s'", original.Type, restored.Type)
	}
	if restored.CorrelationID != original.CorrelationID {
		t.Fatalf("expected correlation '%s', got '%s'", original.CorrelationID, restored.CorrelationID)
	}
	if *restored.WorkspaceID != *original.WorkspaceID {
		t.Fatalf("expected workspace '%s', got '%s'", *original.WorkspaceID, *restored.WorkspaceID)
	}
}

func TestEvent_Subject(t *testing.T) {
	event, _ := events.NewEvent(events.LedgerPosted, map[string]string{})

	if event.Subject() != events.LedgerPosted {
		t.Fatalf("expected subject '%s', got '%s'", events.LedgerPosted, event.Subject())
	}
}

func TestEvent_PayloadParsing(t *testing.T) {
	type CustomPayload struct {
		Amount float64 `json:"amount"`
		Note   string  `json:"note"`
	}

	original := CustomPayload{Amount: 100000, Note: "test transaction"}
	event, _ := events.NewEvent(events.TransactionCreated, original)

	var parsed CustomPayload
	if err := json.Unmarshal(event.Payload, &parsed); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}

	if parsed.Amount != 100000 {
		t.Fatalf("expected amount 100000, got %f", parsed.Amount)
	}
	if parsed.Note != "test transaction" {
		t.Fatalf("expected note 'test transaction', got '%s'", parsed.Note)
	}
}

func TestEvent_UniqueIDs(t *testing.T) {
	e1, _ := events.NewEvent(events.UserRegistered, map[string]string{})
	e2, _ := events.NewEvent(events.UserRegistered, map[string]string{})

	if e1.ID == e2.ID {
		t.Fatal("expected unique event IDs")
	}
	if e1.CorrelationID == e2.CorrelationID {
		t.Fatal("expected unique correlation IDs")
	}
}

func TestEvent_OccurredAtIsRecent(t *testing.T) {
	before := time.Now().Add(-1 * time.Second)
	event, _ := events.NewEvent(events.UserRegistered, map[string]string{})
	after := time.Now().Add(1 * time.Second)

	if event.OccurredAt.Before(before) || event.OccurredAt.After(after) {
		t.Fatalf("occurred_at should be recent, got %v", event.OccurredAt)
	}
}

func TestEventTypes_Constants(t *testing.T) {
	// Verify key event types are defined
	types := []string{
		events.UserRegistered,
		events.TransactionCreated,
		events.LedgerPostingRequested,
		events.LedgerPosted,
		events.CompanyClaimRequested,
		events.CompanyClaimApproved,
		events.DocumentUploaded,
		events.DocumentExtracted,
		events.NotificationRequested,
		events.ReportGenerationReq,
		events.WebhookDeliveryRequested,
		events.SubscriptionCreated,
	}

	for _, typ := range types {
		if typ == "" {
			t.Fatal("event type constant should not be empty")
		}
	}
}

func TestStreamConfig(t *testing.T) {
	// Verify all streams have subjects
	if len(events.StreamConfig) == 0 {
		t.Fatal("expected non-empty stream config")
	}

	for name, subjects := range events.StreamConfig {
		if name == "" {
			t.Fatal("stream name should not be empty")
		}
		if len(subjects) == 0 {
			t.Fatalf("stream '%s' should have at least one subject", name)
		}
	}
}

func TestTaskTypes_Constants(t *testing.T) {
	tasks := []string{
		events.TaskEmailSend,
		events.TaskEmailVerification,
		events.TaskImageOCR,
		events.TaskWebhookDeliver,
		events.TaskWebhookRetry,
		events.TaskInvoiceGenerate,
		events.TaskReportGenerate,
		events.TaskCleanupSessions,
		events.TaskCleanupOutbox,
		events.TaskSubscriptionCheck,
	}

	for _, task := range tasks {
		if task == "" {
			t.Fatal("task type constant should not be empty")
		}
	}
}

// ============================================================
// ACCOUNTING EVENT TESTS
// ============================================================

func TestAccountingEvents_TransactionCreated(t *testing.T) {
	txID := uuid.New().String()
	wsID := uuid.New().String()
	actorID := uuid.New().String()

	payload := map[string]string{
		"transaction_id": txID,
		"workspace_id":   wsID,
	}

	event, err := events.NewEvent(events.TransactionCreated, payload,
		events.WithWorkspace(wsID),
		events.WithActor(actorID),
	)
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	// Verify type
	if event.Type != "accounting.transaction.created" {
		t.Fatalf("expected type 'accounting.transaction.created', got '%s'", event.Type)
	}

	// Verify subject routes to ACCOUNTING stream
	subject := event.Subject()
	found := false
	for _, filter := range events.StreamConfig[events.StreamAccounting] {
		// "accounting.>" should match "accounting.transaction.created"
		if filter == "accounting.>" {
			found = true
		}
	}
	if !found {
		t.Fatalf("subject '%s' should match ACCOUNTING stream filter", subject)
	}

	// Verify payload roundtrip
	var parsed map[string]string
	if err := json.Unmarshal(event.Payload, &parsed); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}
	if parsed["transaction_id"] != txID {
		t.Fatalf("expected transaction_id '%s', got '%s'", txID, parsed["transaction_id"])
	}
	if parsed["workspace_id"] != wsID {
		t.Fatalf("expected workspace_id '%s', got '%s'", wsID, parsed["workspace_id"])
	}
}

func TestAccountingEvents_LedgerPosted(t *testing.T) {
	event, err := events.NewEvent(events.LedgerPosted, map[string]string{
		"transaction_id": uuid.New().String(),
		"workspace_id":   uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	if event.Type != "accounting.ledger.posted" {
		t.Fatalf("expected type 'accounting.ledger.posted', got '%s'", event.Type)
	}
}

func TestAccountingEvents_ReversalTransaction(t *testing.T) {
	originalTxID := uuid.New().String()
	reversalTxID := uuid.New().String()
	wsID := uuid.New().String()

	payload := map[string]string{
		"transaction_id": reversalTxID,
		"workspace_id":   wsID,
		"is_reversal":    "true",
	}

	event, err := events.NewEvent(events.TransactionCreated, payload,
		events.WithWorkspace(wsID),
		events.WithCausation(originalTxID),
		events.WithIdempotencyKey("reversal-"+originalTxID),
	)
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	// Reversal should have causation pointing to original
	if event.CausationID == nil || *event.CausationID != originalTxID {
		t.Fatalf("expected causation_id '%s', got %v", originalTxID, event.CausationID)
	}

	// Should have idempotency key to prevent double-reversal
	if event.IdempotencyKey == nil || *event.IdempotencyKey != "reversal-"+originalTxID {
		t.Fatalf("expected idempotency_key 'reversal-%s', got %v", originalTxID, event.IdempotencyKey)
	}

	// Verify payload contains is_reversal flag
	var parsed map[string]string
	json.Unmarshal(event.Payload, &parsed)
	if parsed["is_reversal"] != "true" {
		t.Fatal("expected is_reversal=true in payload")
	}
}

func TestAccountingEvents_WorkspaceCreated_ForCOASeed(t *testing.T) {
	wsID := uuid.New().String()
	actorID := uuid.New().String()

	event, err := events.NewEvent(events.WorkspaceCreated, map[string]string{
		"workspace_id": wsID,
	},
		events.WithWorkspace(wsID),
		events.WithActor(actorID),
	)
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	if event.Type != "workspace.created" {
		t.Fatalf("expected type 'workspace.created', got '%s'", event.Type)
	}

	// Should route to USER stream (workspace.> filter)
	subject := event.Subject()
	found := false
	for _, filter := range events.StreamConfig[events.StreamUser] {
		if filter == "workspace.>" {
			found = true
		}
	}
	if !found {
		t.Fatalf("subject '%s' should match USER stream filter (workspace.>)", subject)
	}

	// Verify workspace_id in payload
	var parsed map[string]string
	json.Unmarshal(event.Payload, &parsed)
	if parsed["workspace_id"] != wsID {
		t.Fatalf("expected workspace_id '%s' in payload", wsID)
	}
}

func TestAccountingEvents_AllTypesDefinedCorrectly(t *testing.T) {
	// Verify all accounting event types follow naming convention
	accountingEvents := map[string]string{
		"TransactionCreated":     events.TransactionCreated,
		"LedgerPostingRequested": events.LedgerPostingRequested,
		"LedgerPosted":           events.LedgerPosted,
		"JournalEntryCreated":    events.JournalEntryCreated,
	}

	for name, value := range accountingEvents {
		if value == "" {
			t.Fatalf("%s should not be empty", name)
		}
		// All accounting events should start with "accounting."
		if len(value) < 11 || value[:11] != "accounting." {
			t.Fatalf("%s should start with 'accounting.', got '%s'", name, value)
		}
	}
}

func TestEventEnvelope_FullIntegrity(t *testing.T) {
	// Simulate a complete accounting flow event chain
	wsID := uuid.New().String()
	userID := uuid.New().String()
	correlationID := uuid.New().String()

	// Step 1: Transaction created
	txEvent, _ := events.NewEvent(events.TransactionCreated,
		map[string]string{"transaction_id": "tx-001", "workspace_id": wsID},
		events.WithWorkspace(wsID),
		events.WithActor(userID),
		events.WithCorrelation(correlationID),
		events.WithMetadata("api", "req-abc"),
	)

	// Step 2: Ledger posted (caused by transaction created)
	ledgerEvent, _ := events.NewEvent(events.LedgerPosted,
		map[string]string{"transaction_id": "tx-001", "workspace_id": wsID},
		events.WithWorkspace(wsID),
		events.WithCorrelation(correlationID),
		events.WithCausation(txEvent.ID),
	)

	// Verify chain integrity
	if ledgerEvent.CorrelationID != txEvent.CorrelationID {
		t.Fatal("events in same flow should share correlation_id")
	}
	if *ledgerEvent.CausationID != txEvent.ID {
		t.Fatal("ledger event should reference transaction event as cause")
	}
	if *ledgerEvent.WorkspaceID != *txEvent.WorkspaceID {
		t.Fatal("events in same flow should share workspace_id")
	}

	// Verify temporal ordering
	if ledgerEvent.OccurredAt.Before(txEvent.OccurredAt) {
		t.Fatal("ledger event should occur after transaction event")
	}
}
