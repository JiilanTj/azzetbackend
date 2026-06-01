package claim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/events"
	"codeberg.org/azzet/azzetbe/internal/identity"
	"codeberg.org/azzet/azzetbe/internal/storage"
)

var (
	ErrClaimNotFound       = errors.New("claim not found")
	ErrClaimExists         = errors.New("a claim already exists for this entity")
	ErrInvalidStatus       = errors.New("invalid claim status transition")
	ErrNotOwner            = errors.New("not authorized for this claim")
	ErrDocumentsMissing    = errors.New("at least one document is required before submission")
	ErrNotShadow           = errors.New("entity is not a shadow entity")
	ErrDocNotFound         = errors.New("document not found")
	ErrUploadNotConfirmed  = errors.New("document not found in storage")
	ErrEntityNotFound      = errors.New("entity not found")
)

// WorkspaceBootstrapper creates a workspace after a claim is approved.
type WorkspaceBootstrapper interface {
	EnsureWorkspaceForClaimedEntity(ctx context.Context, userID string, entityID uuid.UUID) error
}

var validDocTypes = map[string]bool{
	"NPWP": true, "NIB": true, "SIUP": true,
	"AKTA_PENDIRIAN": true, "AKTA_PERUBAHAN": true,
	"KTP_DIREKTUR": true, "SURAT_KUASA": true, "OTHER": true,
}

type Service struct {
	Queries         *db.Queries
	Pool            *pgxpool.Pool
	Storage         *storage.R2Client
	IdentityService *identity.Service
	Workspace       WorkspaceBootstrapper
}

func NewService(queries *db.Queries, pool *pgxpool.Pool, storageClient *storage.R2Client, identityService *identity.Service, workspace WorkspaceBootstrapper) *Service {
	return &Service{
		Queries:         queries,
		Pool:            pool,
		Storage:         storageClient,
		IdentityService: identityService,
		Workspace:       workspace,
	}
}

// --- User-Facing Methods ---

func (s *Service) CreateClaim(ctx context.Context, userID string, req *CreateClaimRequest) (*ClaimResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id")
	}

	eid, err := uuid.Parse(req.EntityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity_id")
	}

	e, err := s.Queries.GetEntityByID(ctx, eid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEntityNotFound
		}
		return nil, err
	}

	if !e.IsShadow {
		return nil, ErrNotShadow
	}

	hasClaim, err := s.Queries.HasClaimForEntity(ctx, eid)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing claim: %w", err)
	}
	if hasClaim {
		return nil, ErrClaimExists
	}

	personal, err := s.Queries.GetEntityByUserID(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("personal entity not found")
	}

	now := time.Now()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.Queries.WithTx(tx)

	claim, err := qtx.CreateCompanyClaim(ctx, db.CreateCompanyClaimParams{
		ID:               uuid.New(),
		EntityID:         eid,
		ClaimantUserID:   uid,
		ClaimantEntityID: personal.ID,
		Status:           StatusDraft,
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create claim: %w", err)
	}

	if err := s.ensureVerificationRecordTx(ctx, qtx, eid); err != nil {
		return nil, err
	}

	details, _ := json.Marshal(map[string]string{"entity_id": req.EntityID})
	_ = qtx.CreateClaimAuditEntry(ctx, db.CreateClaimAuditEntryParams{
		ID:        uuid.New(),
		ClaimID:   claim.ID,
		ActorID:   uid,
		ActorType: ActorUser,
		Action:    ActionCreated,
		NewStatus: pgtype.Text{String: StatusDraft, Valid: true},
		Details:   details,
		CreatedAt: now,
	})

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return &ClaimResponse{
		ID:         claim.ID.String(),
		EntityID:   claim.EntityID.String(),
		EntityName: e.NamaUtama,
		EntityType: e.EntityType,
		Status:     claim.Status,
		CreatedAt:  claim.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  claim.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Service) SubmitClaim(ctx context.Context, userID, claimID string) (*ClaimResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id")
	}

	cid, err := uuid.Parse(claimID)
	if err != nil {
		return nil, ErrClaimNotFound
	}

	claim, err := s.Queries.GetCompanyClaimByID(ctx, cid)
	if err != nil {
		return nil, ErrClaimNotFound
	}

	if claim.ClaimantUserID != uid {
		return nil, ErrNotOwner
	}

	oldStatus := claim.Status
	action := ActionSubmitted
	if claim.Status != StatusDraft && claim.Status != StatusRejected {
		return nil, ErrInvalidStatus
	}
	if oldStatus == StatusRejected {
		action = ActionResubmitted
	}

	count, err := s.Queries.CountClaimDocuments(ctx, cid)
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}
	if count == 0 {
		return nil, ErrDocumentsMissing
	}

	now := time.Now()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.Queries.WithTx(tx)

	if oldStatus == StatusRejected {
		if err := qtx.ResubmitCompanyClaim(ctx, cid); err != nil {
			return nil, fmt.Errorf("failed to resubmit claim: %w", err)
		}
	} else if err := qtx.UpdateClaimStatus(ctx, db.UpdateClaimStatusParams{
		ID:     cid,
		Status: StatusSubmitted,
	}); err != nil {
		return nil, fmt.Errorf("failed to update claim status: %w", err)
	}

	if err := s.ensureVerificationRecordTx(ctx, qtx, claim.EntityID); err != nil {
		return nil, err
	}

	if err := qtx.UpdateEntityVerificationStatus(ctx, db.UpdateEntityVerificationStatusParams{
		EntityID: claim.EntityID,
		Status:   identity.StatusPending,
	}); err != nil {
		return nil, fmt.Errorf("failed to update verification status: %w", err)
	}

	details, _ := json.Marshal(map[string]string{"from": oldStatus})
	_ = qtx.CreateClaimAuditEntry(ctx, db.CreateClaimAuditEntryParams{
		ID:        uuid.New(),
		ClaimID:   cid,
		ActorID:   uid,
		ActorType: ActorUser,
		Action:    action,
		OldStatus: pgtype.Text{String: oldStatus, Valid: true},
		NewStatus: pgtype.Text{String: StatusSubmitted, Valid: true},
		Details:   details,
		CreatedAt: now,
	})

	_ = events.EmitEvent(ctx, tx, events.CompanyClaimRequested, map[string]string{
		"claim_id":  claimID,
		"entity_id": claim.EntityID.String(),
		"user_id":   uid.String(),
	},
		events.WithWorkspace(claim.EntityID.String()),
		events.WithActor(uid.String()),
	)

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	resp := &ClaimResponse{
		ID:       claimID,
		EntityID: claim.EntityID.String(),
		Status:   StatusSubmitted,
	}
	if e, err := s.Queries.GetEntityByID(ctx, claim.EntityID); err == nil {
		resp.EntityName = e.NamaUtama
		resp.EntityType = e.EntityType
	}
	return resp, nil
}

func (s *Service) RequestDocumentUpload(ctx context.Context, userID, claimID string, req *DocumentUploadRequest) (*PresignedUploadResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id")
	}

	cid, err := uuid.Parse(claimID)
	if err != nil {
		return nil, ErrClaimNotFound
	}

	claim, err := s.Queries.GetCompanyClaimByID(ctx, cid)
	if err != nil {
		return nil, ErrClaimNotFound
	}

	if claim.ClaimantUserID != uid {
		return nil, ErrNotOwner
	}

	if claim.Status != StatusDraft && claim.Status != StatusRejected && claim.Status != StatusDisputed {
		return nil, ErrInvalidStatus
	}

	if !validDocTypes[req.DocumentType] {
		return nil, fmt.Errorf("invalid document_type")
	}

	if req.FileName == "" {
		return nil, fmt.Errorf("file_name is required")
	}

	if req.MimeType == "" {
		req.MimeType = "application/pdf"
	}

	docID := uuid.New()
	fileKey := storage.ClaimDocumentKey(claimID, docID.String(), req.FileName)

	if s.Storage == nil {
		return nil, fmt.Errorf("storage not configured")
	}

	uploadURL, err := s.Storage.GeneratePresignedPutURL(ctx, fileKey, req.MimeType, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	now := time.Now()
	doc, err := s.Queries.CreateClaimDocument(ctx, db.CreateClaimDocumentParams{
		ID:           docID,
		ClaimID:      cid,
		DocumentType: req.DocumentType,
		FileKey:      fileKey,
		FileName:     req.FileName,
		FileSize:     req.FileSize,
		MimeType:     req.MimeType,
		UploadStatus: DocStatusPending,
		CreatedAt:    now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create document record: %w", err)
	}

	s.createAuditEntry(ctx, claimID, uid.String(), ActorUser, ActionDocumentUploaded, claim.Status, claim.Status, map[string]string{
		"document_id":   doc.ID.String(),
		"document_type": req.DocumentType,
	})

	return &PresignedUploadResponse{
		DocumentID: doc.ID.String(),
		UploadURL:  uploadURL,
		FileKey:    fileKey,
		ExpiresIn:  900,
	}, nil
}

func (s *Service) ConfirmDocumentUpload(ctx context.Context, userID, claimID, documentID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user_id")
	}

	cid, err := uuid.Parse(claimID)
	if err != nil {
		return ErrClaimNotFound
	}

	did, err := uuid.Parse(documentID)
	if err != nil {
		return ErrDocNotFound
	}

	claim, err := s.Queries.GetCompanyClaimByID(ctx, cid)
	if err != nil {
		return ErrClaimNotFound
	}

	if claim.ClaimantUserID != uid {
		return ErrNotOwner
	}

	doc, err := s.Queries.GetClaimDocumentByID(ctx, did)
	if err != nil {
		return ErrDocNotFound
	}

	if doc.ClaimID != cid {
		return ErrDocNotFound
	}

	if s.Storage == nil {
		return fmt.Errorf("storage not configured")
	}

	exists, err := s.Storage.ObjectExists(ctx, doc.FileKey)
	if err != nil {
		return fmt.Errorf("failed to verify upload: %w", err)
	}
	if !exists {
		return ErrUploadNotConfirmed
	}

	return s.Queries.MarkDocumentUploaded(ctx, did)
}

func (s *Service) GetClaimDocuments(ctx context.Context, userID, claimID string) ([]DocumentResponse, error) {
	if err := s.ensureClaimOwner(ctx, userID, claimID); err != nil {
		return nil, err
	}

	cid, err := uuid.Parse(claimID)
	if err != nil {
		return nil, ErrClaimNotFound
	}

	docs, err := s.Queries.GetClaimDocuments(ctx, cid)
	if err != nil {
		return nil, err
	}

	resp := make([]DocumentResponse, 0, len(docs))
	for i := range docs {
		resp = append(resp, s.docToResponse(&docs[i]))
	}
	return resp, nil
}

func (s *Service) GetMyClaims(ctx context.Context, userID string) ([]ClaimResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id")
	}

	claims, err := s.Queries.GetClaimsByClaimant(ctx, uid)
	if err != nil {
		return nil, err
	}

	resp := make([]ClaimResponse, 0, len(claims))
	for i := range claims {
		resp = append(resp, ClaimResponse{
			ID:         claims[i].ID.String(),
			EntityID:   claims[i].EntityID.String(),
			EntityName: claims[i].EntityName,
			EntityType: claims[i].EntityType,
			Status:     claims[i].Status,
			CreatedAt:  claims[i].CreatedAt.Format(time.RFC3339),
			UpdatedAt:  claims[i].UpdatedAt.Format(time.RFC3339),
		})
	}
	return resp, nil
}

func (s *Service) GetClaimForUser(ctx context.Context, userID, claimID string) (*ClaimDetailResponse, error) {
	if err := s.ensureClaimOwner(ctx, userID, claimID); err != nil {
		return nil, err
	}
	return s.getClaimDetail(ctx, claimID)
}

func (s *Service) GetClaimDetail(ctx context.Context, claimID string) (*ClaimDetailResponse, error) {
	return s.getClaimDetail(ctx, claimID)
}

func (s *Service) getClaimDetail(ctx context.Context, claimID string) (*ClaimDetailResponse, error) {
	cid, err := uuid.Parse(claimID)
	if err != nil {
		return nil, ErrClaimNotFound
	}

	claim, err := s.Queries.GetCompanyClaimByID(ctx, cid)
	if err != nil {
		return nil, ErrClaimNotFound
	}

	entityName := ""
	entityType := ""
	if e, err := s.Queries.GetEntityByID(ctx, claim.EntityID); err == nil {
		entityName = e.NamaUtama
		entityType = e.EntityType
	}

	docs, _ := s.Queries.GetClaimDocuments(ctx, cid)
	docResp := make([]DocumentResponse, 0, len(docs))
	for i := range docs {
		docResp = append(docResp, s.docToResponse(&docs[i]))
	}

	auditLog, _ := s.Queries.GetClaimAuditLog(ctx, cid)
	auditResp := make([]AuditLogEntry, 0, len(auditLog))
	for i := range auditLog {
		auditResp = append(auditResp, s.auditToResponse(&auditLog[i]))
	}

	resp := &ClaimDetailResponse{
		ClaimResponse: ClaimResponse{
			ID:         claimID,
			EntityID:   claim.EntityID.String(),
			EntityName: entityName,
			EntityType: entityType,
			Status:     claim.Status,
			CreatedAt:  claim.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  claim.UpdatedAt.Format(time.RFC3339),
		},
		ClaimantUserID:   claim.ClaimantUserID.String(),
		ClaimantEntityID: claim.ClaimantEntityID.String(),
		Documents:        docResp,
		AuditLog:         auditResp,
	}

	if claim.ReviewerID.Valid {
		id := uuid.UUID(claim.ReviewerID.Bytes).String()
		resp.ReviewerID = &id
	}
	if claim.ReviewedAt != nil {
		t := claim.ReviewedAt.Format(time.RFC3339)
		resp.ReviewedAt = &t
	}
	if claim.RejectionReason.Valid {
		resp.RejectionReason = &claim.RejectionReason.String
	}
	if claim.DisputeReason.Valid {
		resp.DisputeReason = &claim.DisputeReason.String
	}
	if claim.Notes.Valid {
		resp.Notes = &claim.Notes.String
	}

	return resp, nil
}

func (s *Service) DisputeClaim(ctx context.Context, userID, claimID string, req *DisputeRequest) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user_id")
	}

	cid, err := uuid.Parse(claimID)
	if err != nil {
		return ErrClaimNotFound
	}

	claim, err := s.Queries.GetCompanyClaimByID(ctx, cid)
	if err != nil {
		return ErrClaimNotFound
	}

	if claim.ClaimantUserID != uid {
		return ErrNotOwner
	}

	if claim.Status != StatusRejected {
		return ErrInvalidStatus
	}

	oldStatus := claim.Status
	now := time.Now()

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.Queries.WithTx(tx)

	if err := qtx.DisputeCompanyClaim(ctx, db.DisputeCompanyClaimParams{
		ID:            cid,
		DisputeReason: pgtype.Text{String: req.Reason, Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to dispute claim: %w", err)
	}

	if err := s.ensureVerificationRecordTx(ctx, qtx, claim.EntityID); err != nil {
		return err
	}

	if err := qtx.UpdateEntityVerificationStatus(ctx, db.UpdateEntityVerificationStatusParams{
		EntityID: claim.EntityID,
		Status:   identity.StatusPending,
	}); err != nil {
		return fmt.Errorf("failed to update verification status: %w", err)
	}

	details, _ := json.Marshal(map[string]string{"reason": req.Reason})
	_ = qtx.CreateClaimAuditEntry(ctx, db.CreateClaimAuditEntryParams{
		ID:        uuid.New(),
		ClaimID:   cid,
		ActorID:   uid,
		ActorType: ActorUser,
		Action:    ActionDisputed,
		OldStatus: pgtype.Text{String: oldStatus, Valid: true},
		NewStatus: pgtype.Text{String: StatusDisputed, Valid: true},
		Details:   details,
		CreatedAt: now,
	})

	return tx.Commit(ctx)
}

// --- Admin Methods ---

func (s *Service) ListClaimsForReview(ctx context.Context, status string, limit, offset int) ([]ClaimListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	claims, err := s.Queries.ListClaimsByStatus(ctx, db.ListClaimsByStatusParams{
		Status: status,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	return s.claimListToResponse(claims)
}

func (s *Service) ListAllClaims(ctx context.Context, limit, offset int) ([]ClaimListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	claims, err := s.Queries.ListAllClaims(ctx, db.ListAllClaimsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	resp := make([]ClaimListResponse, 0, len(claims))
	for i := range claims {
		docCount, _ := s.Queries.CountClaimDocuments(context.Background(), claims[i].ID)
		resp = append(resp, ClaimListResponse{
			ClaimResponse: ClaimResponse{
				ID:         claims[i].ID.String(),
				EntityID:   claims[i].EntityID.String(),
				EntityName: claims[i].EntityName,
				EntityType: claims[i].EntityType,
				Status:     claims[i].Status,
				CreatedAt:  claims[i].CreatedAt.Format(time.RFC3339),
				UpdatedAt:  claims[i].UpdatedAt.Format(time.RFC3339),
			},
			ClaimantName:  strings.TrimSpace(claims[i].ClaimantName.String),
			DocumentCount: int(docCount),
		})
	}
	return resp, nil
}

func (s *Service) AssignClaim(ctx context.Context, adminID, claimID string) error {
	cid, err := uuid.Parse(claimID)
	if err != nil {
		return ErrClaimNotFound
	}

	aid, err := uuid.Parse(adminID)
	if err != nil {
		return fmt.Errorf("invalid admin_id")
	}

	claim, err := s.Queries.GetCompanyClaimByID(ctx, cid)
	if err != nil {
		return ErrClaimNotFound
	}

	if claim.Status != StatusSubmitted && claim.Status != StatusDisputed {
		return ErrInvalidStatus
	}

	now := time.Now()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.Queries.WithTx(tx)

	if err := qtx.AssignClaimReviewer(ctx, db.AssignClaimReviewerParams{
		ID:         cid,
		ReviewerID: pgtype.UUID{Bytes: aid, Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to assign reviewer: %w", err)
	}

	details, _ := json.Marshal(map[string]string{"reviewer_id": adminID})
	_ = qtx.CreateClaimAuditEntry(ctx, db.CreateClaimAuditEntryParams{
		ID:        uuid.New(),
		ClaimID:   cid,
		ActorID:   aid,
		ActorType: ActorAdmin,
		Action:    ActionAssigned,
		OldStatus: pgtype.Text{String: claim.Status, Valid: true},
		NewStatus: pgtype.Text{String: StatusUnderReview, Valid: true},
		Details:   details,
		CreatedAt: now,
	})

	return tx.Commit(ctx)
}

func (s *Service) ApproveClaim(ctx context.Context, adminID, claimID string, req *ReviewRequest) error {
	cid, err := uuid.Parse(claimID)
	if err != nil {
		return ErrClaimNotFound
	}

	aid, err := uuid.Parse(adminID)
	if err != nil {
		return fmt.Errorf("invalid admin_id")
	}

	claim, err := s.Queries.GetCompanyClaimByID(ctx, cid)
	if err != nil {
		return ErrClaimNotFound
	}

	if claim.Status != StatusUnderReview {
		return ErrInvalidStatus
	}

	now := time.Now()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.Queries.WithTx(tx)

	notes := pgtype.Text{}
	if req != nil && req.Notes != nil {
		notes = pgtype.Text{String: *req.Notes, Valid: true}
	}

	if err := qtx.ApproveCompanyClaim(ctx, db.ApproveCompanyClaimParams{
		ID:         cid,
		ReviewerID: pgtype.UUID{Bytes: aid, Valid: true},
		ReviewedAt: &now,
		Notes:      notes,
	}); err != nil {
		return fmt.Errorf("failed to approve claim: %w", err)
	}

	if err := s.linkShadowEntity(ctx, qtx, claim); err != nil {
		return fmt.Errorf("failed to link shadow entity: %w", err)
	}

	entity, err := qtx.GetEntityByID(ctx, claim.EntityID)
	if err != nil {
		return fmt.Errorf("failed to load entity: %w", err)
	}
	if err := s.IdentityService.EnsureNormalizedName(ctx, claim.EntityID, entity.NamaUtama); err != nil {
		return fmt.Errorf("failed to normalize entity name: %w", err)
	}

	if err := qtx.UpdateEntityVerificationStatus(ctx, db.UpdateEntityVerificationStatusParams{
		EntityID:   claim.EntityID,
		Status:     identity.StatusVerified,
		VerifiedBy: pgtype.UUID{Bytes: aid, Valid: true},
		VerifiedAt: &now,
	}); err != nil {
		return fmt.Errorf("failed to update verification: %w", err)
	}

	details, _ := json.Marshal(map[string]string{"action": "approved"})
	_ = qtx.CreateClaimAuditEntry(ctx, db.CreateClaimAuditEntryParams{
		ID:        uuid.New(),
		ClaimID:   cid,
		ActorID:   aid,
		ActorType: ActorAdmin,
		Action:    ActionApproved,
		OldStatus: pgtype.Text{String: StatusUnderReview, Valid: true},
		NewStatus: pgtype.Text{String: StatusApproved, Valid: true},
		Details:   details,
		CreatedAt: now,
	})

	_ = events.EmitEvent(ctx, tx, events.CompanyClaimApproved, map[string]string{
		"claim_id":         claimID,
		"entity_id":        claim.EntityID.String(),
		"claimant_user_id": claim.ClaimantUserID.String(),
	},
		events.WithWorkspace(claim.EntityID.String()),
		events.WithActor(aid.String()),
	)

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	if s.Workspace != nil {
		if err := s.Workspace.EnsureWorkspaceForClaimedEntity(ctx, claim.ClaimantUserID.String(), claim.EntityID); err != nil {
			return fmt.Errorf("failed to bootstrap workspace: %w", err)
		}
	}

	return nil
}

func (s *Service) RejectClaim(ctx context.Context, adminID, claimID string, req *ReviewRequest) error {
	cid, err := uuid.Parse(claimID)
	if err != nil {
		return ErrClaimNotFound
	}

	aid, err := uuid.Parse(adminID)
	if err != nil {
		return fmt.Errorf("invalid admin_id")
	}

	claim, err := s.Queries.GetCompanyClaimByID(ctx, cid)
	if err != nil {
		return ErrClaimNotFound
	}

	if claim.Status != StatusUnderReview {
		return ErrInvalidStatus
	}

	now := time.Now()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.Queries.WithTx(tx)

	rejectionReason := pgtype.Text{}
	if req != nil && req.RejectionReason != nil {
		rejectionReason = pgtype.Text{String: *req.RejectionReason, Valid: true}
	}

	if err := qtx.RejectCompanyClaim(ctx, db.RejectCompanyClaimParams{
		ID:              cid,
		ReviewerID:      pgtype.UUID{Bytes: aid, Valid: true},
		ReviewedAt:      &now,
		RejectionReason: rejectionReason,
	}); err != nil {
		return fmt.Errorf("failed to reject claim: %w", err)
	}

	if err := qtx.UpdateEntityVerificationStatus(ctx, db.UpdateEntityVerificationStatusParams{
		EntityID:        claim.EntityID,
		Status:          identity.StatusRejected,
		VerifiedBy:      pgtype.UUID{Bytes: aid, Valid: true},
		VerifiedAt:      &now,
		RejectionReason: rejectionReason,
	}); err != nil {
		return fmt.Errorf("failed to update verification: %w", err)
	}

	details, _ := json.Marshal(map[string]string{"rejection_reason": rejectionReason.String})
	_ = qtx.CreateClaimAuditEntry(ctx, db.CreateClaimAuditEntryParams{
		ID:        uuid.New(),
		ClaimID:   cid,
		ActorID:   aid,
		ActorType: ActorAdmin,
		Action:    ActionRejected,
		OldStatus: pgtype.Text{String: StatusUnderReview, Valid: true},
		NewStatus: pgtype.Text{String: StatusRejected, Valid: true},
		Details:   details,
		CreatedAt: now,
	})

	_ = events.EmitEvent(ctx, tx, events.CompanyClaimRejected, map[string]string{
		"claim_id":         claimID,
		"entity_id":        claim.EntityID.String(),
		"claimant_user_id": claim.ClaimantUserID.String(),
	},
		events.WithWorkspace(claim.EntityID.String()),
		events.WithActor(aid.String()),
	)

	return tx.Commit(ctx)
}

func (s *Service) GetClaimAuditLog(ctx context.Context, claimID string) ([]AuditLogEntry, error) {
	cid, err := uuid.Parse(claimID)
	if err != nil {
		return nil, ErrClaimNotFound
	}

	logs, err := s.Queries.GetClaimAuditLog(ctx, cid)
	if err != nil {
		return nil, err
	}

	resp := make([]AuditLogEntry, 0, len(logs))
	for i := range logs {
		resp = append(resp, s.auditToResponse(&logs[i]))
	}
	return resp, nil
}

func (s *Service) GetDocumentViewURL(ctx context.Context, documentID string) (string, error) {
	did, err := uuid.Parse(documentID)
	if err != nil {
		return "", ErrDocNotFound
	}

	doc, err := s.Queries.GetClaimDocumentByID(ctx, did)
	if err != nil {
		return "", ErrDocNotFound
	}

	if s.Storage == nil {
		return "", fmt.Errorf("storage not configured")
	}

	return s.Storage.GeneratePresignedGetURL(ctx, doc.FileKey, 1*time.Hour)
}

func (s *Service) CountPendingClaims(ctx context.Context) (int64, error) {
	return s.Queries.CountClaimsByStatus(ctx, StatusSubmitted)
}

// --- Internal ---

func (s *Service) ensureClaimOwner(ctx context.Context, userID, claimID string) error {
	cid, err := uuid.Parse(claimID)
	if err != nil {
		return ErrClaimNotFound
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrNotOwner
	}

	claim, err := s.Queries.GetCompanyClaimByID(ctx, cid)
	if err != nil {
		return ErrClaimNotFound
	}

	if claim.ClaimantUserID != uid {
		return ErrNotOwner
	}

	return nil
}

func (s *Service) linkShadowEntity(ctx context.Context, qtx *db.Queries, claim db.CompanyClaim) error {
	return qtx.LinkShadowEntity(ctx, db.LinkShadowEntityParams{
		ID:     claim.EntityID,
		UserID: pgtype.UUID{Bytes: claim.ClaimantUserID, Valid: true},
	})
}

func (s *Service) ensureVerificationRecordTx(ctx context.Context, qtx *db.Queries, entityID uuid.UUID) error {
	_, err := qtx.GetEntityVerification(ctx, entityID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check verification: %w", err)
	}

	now := time.Now()
	_, err = qtx.CreateEntityVerification(ctx, db.CreateEntityVerificationParams{
		ID:        uuid.New(),
		EntityID:  entityID,
		Status:    identity.StatusUnverified,
		CreatedAt: now,
		UpdatedAt: now,
	})
	return err
}

func (s *Service) createAuditEntry(ctx context.Context, claimID, actorID, actorType, action, oldStatus, newStatus string, extra map[string]string) {
	cid, _ := uuid.Parse(claimID)
	aid, _ := uuid.Parse(actorID)
	now := time.Now()

	details, _ := json.Marshal(extra)

	_ = s.Queries.CreateClaimAuditEntry(ctx, db.CreateClaimAuditEntryParams{
		ID:        uuid.New(),
		ClaimID:   cid,
		ActorID:   aid,
		ActorType: actorType,
		Action:    action,
		OldStatus: pgtype.Text{String: oldStatus, Valid: true},
		NewStatus: pgtype.Text{String: newStatus, Valid: true},
		Details:   details,
		CreatedAt: now,
	})
}

func (s *Service) docToResponse(doc *db.ClaimDocument) DocumentResponse {
	return DocumentResponse{
		ID:           doc.ID.String(),
		ClaimID:      doc.ClaimID.String(),
		DocumentType: doc.DocumentType,
		FileName:     doc.FileName,
		FileSize:     doc.FileSize,
		MimeType:     doc.MimeType,
		UploadStatus: doc.UploadStatus,
		CreatedAt:    doc.CreatedAt.Format(time.RFC3339),
	}
}

func (s *Service) auditToResponse(entry *db.ClaimAuditLog) AuditLogEntry {
	audit := AuditLogEntry{
		ID:        entry.ID.String(),
		ClaimID:   entry.ClaimID.String(),
		ActorID:   entry.ActorID.String(),
		ActorType: entry.ActorType,
		Action:    entry.Action,
		CreatedAt: entry.CreatedAt.Format(time.RFC3339),
	}
	if entry.OldStatus.Valid {
		audit.OldStatus = &entry.OldStatus.String
	}
	if entry.NewStatus.Valid {
		audit.NewStatus = &entry.NewStatus.String
	}
	if entry.Details != nil {
		var details map[string]interface{}
		if json.Unmarshal(entry.Details, &details) == nil {
			audit.Details = details
		}
	}
	return audit
}

func (s *Service) claimListToResponse(claims []db.ListClaimsByStatusRow) ([]ClaimListResponse, error) {
	resp := make([]ClaimListResponse, 0, len(claims))
	for i := range claims {
		docCount, _ := s.Queries.CountClaimDocuments(context.Background(), claims[i].ID)
		resp = append(resp, ClaimListResponse{
			ClaimResponse: ClaimResponse{
				ID:         claims[i].ID.String(),
				EntityID:   claims[i].EntityID.String(),
				EntityName: claims[i].EntityName,
				EntityType: claims[i].EntityType,
				Status:     claims[i].Status,
				CreatedAt:  claims[i].CreatedAt.Format(time.RFC3339),
				UpdatedAt:  claims[i].UpdatedAt.Format(time.RFC3339),
			},
			ClaimantName:  strings.TrimSpace(claims[i].ClaimantName.String),
			DocumentCount: int(docCount),
		})
	}
	return resp, nil
}
