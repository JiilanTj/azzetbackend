package document

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"codeberg.org/azzet/azzetbe/internal/accounting"
	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/events"
	"codeberg.org/azzet/azzetbe/internal/identity"
	"codeberg.org/azzet/azzetbe/internal/storage"
	"codeberg.org/azzet/azzetbe/internal/workspace"
)

type DocumentWorker struct {
	Queries           *db.Queries
	Pool              *pgxpool.Pool
	Storage           *storage.R2Client
	Extractor         *Extractor
	AccountingService *accounting.Service
	IdentityService   *identity.Service
	WorkspaceService  *workspace.Service
	DocumentService   *Service
}

func NewDocumentWorker(
	queries *db.Queries,
	pool *pgxpool.Pool,
	storage *storage.R2Client,
	extractor *Extractor,
	accountingService *accounting.Service,
	identityService *identity.Service,
	workspaceService *workspace.Service,
	documentService *Service,
) *DocumentWorker {
	return &DocumentWorker{
		Queries:           queries,
		Pool:              pool,
		Storage:           storage,
		Extractor:         extractor,
		AccountingService: accountingService,
		IdentityService:   identityService,
		WorkspaceService:  workspaceService,
		DocumentService:   documentService,
	}
}

func (w *DocumentWorker) HandleDocumentUploaded(ctx context.Context, event *events.Event) error {
	var payload struct {
		DocumentID  string `json:"document_id"`
		WorkspaceID string `json:"workspace_id"`
		UserID      string `json:"user_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse document.uploaded payload: %w", err)
	}

	wsID, err := uuid.Parse(payload.WorkspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace_id: %w", err)
	}
	docID, err := uuid.Parse(payload.DocumentID)
	if err != nil {
		return fmt.Errorf("invalid document_id: %w", err)
	}
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return fmt.Errorf("invalid user_id: %w", err)
	}

	doc, err := w.Queries.GetDocumentByID(ctx, db.GetDocumentByIDParams{
		ID:          docID,
		WorkspaceID: wsID,
	})
	if err != nil {
		return fmt.Errorf("document not found: %w", err)
	}

	if doc.ExtractionStatus != ExtractionPending {
		slog.Info("document-worker: extraction already processed", "document_id", payload.DocumentID, "status", doc.ExtractionStatus)
		return nil
	}

	if err := w.DocumentService.MarkExtractionProcessing(ctx, payload.WorkspaceID, payload.DocumentID); err != nil {
		slog.Warn("document-worker: failed to mark processing", "error", err)
	}

	if !IsImageMimeType(doc.MimeType) {
		err := fmt.Errorf("OCR hanya mendukung file gambar (JPEG, PNG, WebP), got %s", doc.MimeType)
		_ = w.DocumentService.SaveExtractionResult(ctx, payload.WorkspaceID, payload.DocumentID, nil, nil, err)
		return nil // permanent validation failure — no retry
	}

	if w.Storage == nil || w.Extractor == nil {
		err := fmt.Errorf("storage or AI not configured")
		return err
	}

	imageURL, err := w.Storage.GeneratePresignedGetURL(ctx, doc.FileKey, 10*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to generate view URL: %w", err)
	}

	result, err := w.Extractor.ExtractFromImageURL(ctx, imageURL, doc.DocumentType)
	if err != nil {
		_ = w.DocumentService.SaveExtractionResult(ctx, payload.WorkspaceID, payload.DocumentID, nil, nil, err)
		return fmt.Errorf("extraction failed: %w", err)
	}

	counterpartyEntityID, counterpartyName, err := w.resolveCounterparty(ctx, payload.WorkspaceID, payload.UserID, result)
	if err != nil {
		slog.Warn("document-worker: counterparty resolution failed, using name only", "error", err)
		counterpartyName = result.VendorName
	}

	includesTax := accounting.IsPPNCategory(result.Category) ||
		doc.DocumentType == DocTypeFaktur ||
		doc.DocumentType == DocTypeInvoice

	txReq := &accounting.CreateTransactionRequest{
		TransactionType:      result.TransactionType,
		InputMode:            accounting.InputModeOCR,
		Description:          buildDescription(result),
		TransactionDate:      result.Date,
		Amount:               accounting.FloatToNumeric(result.Amount),
		Category:             result.Category,
		CounterpartyEntityID: counterpartyEntityID,
		CounterpartyName:     counterpartyName,
		PaymentMethod:        result.PaymentMethod,
		IncludesTax:          includesTax,
	}

	txResp, err := w.AccountingService.CreateTransaction(ctx, wsID, userID, txReq)
	if err != nil {
		_ = w.DocumentService.SaveExtractionResult(ctx, payload.WorkspaceID, payload.DocumentID, result, nil, err)
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	txUUID, _ := uuid.Parse(txResp.ID)
	if err := w.DocumentService.SaveExtractionResult(ctx, payload.WorkspaceID, payload.DocumentID, result, &txUUID, nil); err != nil {
		return fmt.Errorf("failed to save extraction result: %w", err)
	}

	slog.Info("document-worker: transaction created from OCR",
		"document_id", payload.DocumentID,
		"transaction_id", txResp.ID,
		"amount", result.Amount,
	)

	return nil
}

func (w *DocumentWorker) resolveCounterparty(ctx context.Context, workspaceID, userID string, result *ExtractionResult) (entityID, name string, err error) {
	name = result.VendorName

	matches, err := w.IdentityService.SearchFuzzy(ctx, workspaceID, result.VendorName, 5, 0)
	if err == nil && len(matches) > 0 && matches[0].MatchScore >= 0.75 {
		return matches[0].ID, matches[0].NamaUtama, nil
	}

	relationType := workspace.RelationVendor
	if result.TransactionType == accounting.TxTypeSales || result.TransactionType == accounting.TxTypeCashIn {
		relationType = workspace.RelationPelanggan
	}

	entityType := "BADAN_USAHA"
	if result.VendorNPWP == "" && len(result.VendorName) < 30 {
		entityType = "ORANG_PRIBADI"
	}

	cp, err := w.WorkspaceService.AddCounterparty(ctx, workspaceID, &workspace.AddCounterpartyRequest{
		RelationType: relationType,
		NamaUtama:    &result.VendorName,
		NikNpwp:      strPtr(result.VendorNPWP),
		EntityType:   &entityType,
	})
	if err != nil {
		if errors.Is(err, workspace.ErrRelationExists) {
			if len(matches) > 0 {
				return matches[0].ID, matches[0].NamaUtama, nil
			}
			return "", name, nil
		}
		return "", name, err
	}

	return cp.EntityID, cp.EntityName, nil
}

func buildDescription(r *ExtractionResult) string {
	if r.Description != "" {
		return r.Description
	}
	return fmt.Sprintf("OCR: %s - %s", r.VendorName, r.Date)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
