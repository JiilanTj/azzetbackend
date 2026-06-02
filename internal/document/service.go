package document

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/events"
	"codeberg.org/azzet/azzetbe/internal/storage"
)

type FeatureChecker interface {
	HasFeature(ctx context.Context, workspaceID, featureKey string) (bool, error)
}

type Service struct {
	Queries         *db.Queries
	Pool            *pgxpool.Pool
	Storage         *storage.R2Client
	FeatureChecker  FeatureChecker
}

func NewService(queries *db.Queries, pool *pgxpool.Pool, storage *storage.R2Client, featureChecker FeatureChecker) *Service {
	return &Service{
		Queries:        queries,
		Pool:           pool,
		Storage:        storage,
		FeatureChecker: featureChecker,
	}
}

func (s *Service) RequestUpload(ctx context.Context, workspaceID, userID string, req *UploadRequest) (*PresignedUploadResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id")
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id")
	}

	if s.FeatureChecker != nil {
		enabled, err := s.FeatureChecker.HasFeature(ctx, workspaceID, "ocr_enabled")
		if err != nil {
			return nil, fmt.Errorf("failed to check OCR feature: %w", err)
		}
		if !enabled {
			return nil, ErrOCRNotEnabled
		}
	}

	if !validDocTypes[req.DocumentType] {
		return nil, ErrInvalidDocumentType
	}
	if req.FileName == "" {
		return nil, fmt.Errorf("file_name is required")
	}
	if req.MimeType == "" {
		return nil, fmt.Errorf("mime_type is required")
	}
	if !IsImageMimeType(req.MimeType) {
		return nil, fmt.Errorf("only image MIME types are supported (image/jpeg, image/png, image/webp)")
	}

	if s.Storage == nil {
		return nil, ErrStorageNotConfigured
	}

	docID := uuid.New()
	fileKey := storage.WorkspaceDocumentKey(workspaceID, docID.String(), req.FileName)

	uploadURL, err := s.Storage.GeneratePresignedPutURL(ctx, fileKey, req.MimeType, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	now := time.Now()
	_, err = s.Queries.CreateDocument(ctx, db.CreateDocumentParams{
		ID:                 docID,
		WorkspaceID:        wsID,
		DocumentType:       req.DocumentType,
		FileKey:            fileKey,
		FileName:           req.FileName,
		FileSize:           req.FileSize,
		MimeType:           req.MimeType,
		UploadStatus:       UploadStatusPending,
		ExtractionStatus:   ExtractionPending,
		VerificationStatus: VerificationUnverified,
		CreatedBy:          uid,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create document record: %w", err)
	}

	return &PresignedUploadResponse{
		DocumentID: docID.String(),
		UploadURL:  uploadURL,
		FileKey:    fileKey,
		ExpiresIn:  900,
	}, nil
}

func (s *Service) ConfirmUpload(ctx context.Context, workspaceID, userID, documentID string) (*DocumentResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id")
	}
	did, err := uuid.Parse(documentID)
	if err != nil {
		return nil, ErrDocumentNotFound
	}

	doc, err := s.Queries.GetDocumentByID(ctx, db.GetDocumentByIDParams{
		ID:          did,
		WorkspaceID: wsID,
	})
	if err != nil {
		return nil, ErrDocumentNotFound
	}

	if doc.UploadStatus != UploadStatusPending {
		return nil, ErrInvalidStatus
	}

	if s.Storage == nil {
		return nil, ErrStorageNotConfigured
	}

	exists, err := s.Storage.ObjectExists(ctx, doc.FileKey)
	if err != nil {
		return nil, fmt.Errorf("failed to verify upload: %w", err)
	}
	if !exists {
		return nil, ErrUploadNotConfirmed
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.Queries.WithTx(tx)

	if err := qtx.MarkWorkspaceDocumentUploaded(ctx, db.MarkWorkspaceDocumentUploadedParams{
		ID:          did,
		WorkspaceID: wsID,
	}); err != nil {
		return nil, fmt.Errorf("failed to mark uploaded: %w", err)
	}

	if err := events.EmitEvent(ctx, tx, events.DocumentUploaded, map[string]string{
		"document_id":  documentID,
		"workspace_id": workspaceID,
		"user_id":      userID,
	}, events.WithWorkspace(workspaceID), events.WithActor(userID)); err != nil {
		return nil, fmt.Errorf("failed to emit event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	updated, err := s.Queries.GetDocumentByID(ctx, db.GetDocumentByIDParams{
		ID:          did,
		WorkspaceID: wsID,
	})
	if err != nil {
		return nil, ErrDocumentNotFound
	}

	resp := documentToResponse(updated)
	return &resp, nil
}

func (s *Service) GetDocument(ctx context.Context, workspaceID, documentID string) (*DocumentResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id")
	}
	did, err := uuid.Parse(documentID)
	if err != nil {
		return nil, ErrDocumentNotFound
	}

	doc, err := s.Queries.GetDocumentByID(ctx, db.GetDocumentByIDParams{
		ID:          did,
		WorkspaceID: wsID,
	})
	if err != nil {
		return nil, ErrDocumentNotFound
	}

	resp := documentToResponse(doc)
	if doc.UploadStatus == UploadStatusUploaded && s.Storage != nil {
		if url, err := s.Storage.GeneratePresignedGetURL(ctx, doc.FileKey, 15*time.Minute); err == nil {
			resp.ViewURL = &url
		}
	}
	return &resp, nil
}

func (s *Service) ListDocuments(ctx context.Context, workspaceID string, limit, offset int32) (*DocumentListResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id")
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	total, err := s.Queries.CountDocumentsByWorkspace(ctx, wsID)
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	docs, err := s.Queries.ListDocumentsByWorkspace(ctx, db.ListDocumentsByWorkspaceParams{
		WorkspaceID: wsID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	resp := make([]DocumentResponse, 0, len(docs))
	for i := range docs {
		resp = append(resp, documentToResponse(docs[i]))
	}

	return &DocumentListResponse{
		Documents: resp,
		Total:     total,
	}, nil
}

func (s *Service) SaveExtractionResult(ctx context.Context, workspaceID, documentID string, result *ExtractionResult, txID *uuid.UUID, extractErr error) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace_id")
	}
	did, err := uuid.Parse(documentID)
	if err != nil {
		return ErrDocumentNotFound
	}

	var extractedData []byte
	var confidence pgtype.Numeric
	var errMsg pgtype.Text
	status := ExtractionCompleted

	if extractErr != nil {
		status = ExtractionFailed
		errMsg = pgtype.Text{String: extractErr.Error(), Valid: true}
	} else if result != nil {
		extractedData, _ = json.Marshal(result)
		confidence = floatToNumeric(result.Confidence)
	}

	if err := s.Queries.UpdateDocumentExtraction(ctx, db.UpdateDocumentExtractionParams{
		ID:                   did,
		WorkspaceID:          wsID,
		ExtractionStatus:     status,
		ExtractedData:        extractedData,
		ExtractionConfidence: confidence,
		ExtractionError:      errMsg,
	}); err != nil {
		return fmt.Errorf("failed to update extraction: %w", err)
	}

	if txID != nil {
		if err := s.Queries.LinkDocumentTransaction(ctx, db.LinkDocumentTransactionParams{
			ID:            did,
			WorkspaceID:   wsID,
			TransactionID: pgtype.UUID{Bytes: *txID, Valid: true},
		}); err != nil {
			return fmt.Errorf("failed to link transaction: %w", err)
		}
	}

	return nil
}

func (s *Service) MarkExtractionProcessing(ctx context.Context, workspaceID, documentID string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	did, err := uuid.Parse(documentID)
	if err != nil {
		return err
	}
	return s.Queries.SetDocumentExtractionProcessing(ctx, db.SetDocumentExtractionProcessingParams{
		ID:          did,
		WorkspaceID: wsID,
	})
}

func documentToResponse(d db.Document) DocumentResponse {
	resp := DocumentResponse{
		ID:                 d.ID.String(),
		WorkspaceID:        d.WorkspaceID.String(),
		DocumentType:       d.DocumentType,
		FileName:           d.FileName,
		FileSize:           d.FileSize,
		MimeType:           d.MimeType,
		UploadStatus:       d.UploadStatus,
		ExtractionStatus:   d.ExtractionStatus,
		VerificationStatus: d.VerificationStatus,
		CreatedAt:          d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          d.UpdatedAt.Format(time.RFC3339),
	}

	if len(d.ExtractedData) > 0 {
		var data map[string]interface{}
		if json.Unmarshal(d.ExtractedData, &data) == nil {
			resp.ExtractedData = data
		}
	}
	if d.ExtractionConfidence.Valid {
		f := numericToFloat(d.ExtractionConfidence)
		resp.ExtractionConfidence = &f
	}
	if d.ExtractionError.Valid {
		resp.ExtractionError = &d.ExtractionError.String
	}
	if d.TransactionID.Valid {
		id := uuid.UUID(d.TransactionID.Bytes).String()
		resp.TransactionID = &id
	}
	if d.UploadedAt != nil {
		s := d.UploadedAt.Format(time.RFC3339)
		resp.UploadedAt = &s
	}
	if d.ProcessedAt != nil {
		s := d.ProcessedAt.Format(time.RFC3339)
		resp.ProcessedAt = &s
	}

	return resp
}

func floatToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.4f", f))
	return n
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	if !f.Valid {
		return 0
	}
	return f.Float64
}
