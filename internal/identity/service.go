package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"codeberg.org/azzet/azzetbe/internal/db"
)

var (
	ErrEntityNotFound       = errors.New("entity not found")
	ErrVerificationExists   = errors.New("verification record already exists")
	ErrLegalIDExists        = errors.New("legal ID already exists for this type")
	ErrInvalidLegalIDType   = errors.New("invalid legal ID type")
	ErrInvalidAliasSource   = errors.New("invalid alias source")
)

var validLegalIDTypes = map[string]bool{
	"NPWP": true, "NIB": true, "SIUP": true, "KTP": true, "AKTA": true,
}

var validAliasSources = map[string]bool{
	"MANUAL": true, "CLAIM": true, "COUNTERPARTY": true, "SYSTEM": true,
}

type Service struct {
	Queries *db.Queries
	Pool    *pgxpool.Pool
}

func NewService(queries *db.Queries, pool *pgxpool.Pool) *Service {
	return &Service{Queries: queries, Pool: pool}
}

func (s *Service) EnsureVerificationRecord(ctx context.Context, entityID uuid.UUID) error {
	_, err := s.Queries.GetEntityVerification(ctx, entityID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check verification: %w", err)
	}

	now := time.Now()
	_, err = s.Queries.CreateEntityVerification(ctx, db.CreateEntityVerificationParams{
		ID:        uuid.New(),
		EntityID:  entityID,
		Status:    StatusUnverified,
		CreatedAt: now,
		UpdatedAt: now,
	})
	return err
}

func (s *Service) GetVerificationStatus(ctx context.Context, entityID string) (*VerificationResponse, error) {
	eid, err := uuid.Parse(entityID)
	if err != nil {
		return nil, ErrEntityNotFound
	}

	v, err := s.Queries.GetEntityVerification(ctx, eid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &VerificationResponse{
				EntityID: entityID,
				Status:   StatusUnverified,
			}, nil
		}
		return nil, err
	}

	resp := &VerificationResponse{
		EntityID: entityID,
		Status:   v.Status,
	}
	if v.VerifiedBy.Valid {
		id := uuidToString(v.VerifiedBy)
		resp.VerifiedBy = &id
	}
	if v.VerifiedAt != nil {
		t := v.VerifiedAt.Format(time.RFC3339)
		resp.VerifiedAt = &t
	}
	if v.RejectionReason.Valid {
		resp.RejectionReason = &v.RejectionReason.String
	}
	if v.Notes.Valid {
		resp.Notes = &v.Notes.String
	}
	return resp, nil
}

func (s *Service) SetVerificationStatus(ctx context.Context, entityID string, status string, verifiedBy *string, reason *string, notes *string) error {
	eid, err := uuid.Parse(entityID)
	if err != nil {
		return ErrEntityNotFound
	}

	if err := s.EnsureVerificationRecord(ctx, eid); err != nil {
		return err
	}

	var verifiedByUUID pgtype.UUID
	if verifiedBy != nil {
		uid, err := uuid.Parse(*verifiedBy)
		if err == nil {
			verifiedByUUID = pgtype.UUID{Bytes: uid, Valid: true}
		}
	}

	var verifiedAt *time.Time
	if status == StatusVerified {
		now := time.Now()
		verifiedAt = &now
	}

	rejectionReason := pgtype.Text{}
	if reason != nil {
		rejectionReason = pgtype.Text{String: *reason, Valid: true}
	}

	notesText := pgtype.Text{}
	if notes != nil {
		notesText = pgtype.Text{String: *notes, Valid: true}
	}

	return s.Queries.UpdateEntityVerificationStatus(ctx, db.UpdateEntityVerificationStatusParams{
		EntityID:        eid,
		Status:          status,
		VerifiedBy:      verifiedByUUID,
		VerifiedAt:      verifiedAt,
		RejectionReason: rejectionReason,
		Notes:           notesText,
	})
}

// --- Legal IDs ---

func (s *Service) AddLegalID(ctx context.Context, entityID string, req *AddLegalIDRequest) (*LegalIDResponse, error) {
	if !validLegalIDTypes[req.IDType] {
		return nil, ErrInvalidLegalIDType
	}

	eid, err := uuid.Parse(entityID)
	if err != nil {
		return nil, ErrEntityNotFound
	}

	now := time.Now()
	lid, err := s.Queries.CreateEntityLegalID(ctx, db.CreateEntityLegalIDParams{
		ID:        uuid.New(),
		EntityID:  eid,
		IDType:    req.IDType,
		IDValue:   req.IDValue,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create legal ID: %w", err)
	}

	return legalIDToResponse(&lid), nil
}

func (s *Service) GetLegalIDs(ctx context.Context, entityID string) ([]LegalIDResponse, error) {
	eid, err := uuid.Parse(entityID)
	if err != nil {
		return nil, ErrEntityNotFound
	}

	ids, err := s.Queries.GetEntityLegalIDs(ctx, eid)
	if err != nil {
		return nil, err
	}

	resp := make([]LegalIDResponse, 0, len(ids))
	for i := range ids {
		resp = append(resp, *legalIDToResponse(&ids[i]))
	}
	return resp, nil
}

func (s *Service) UpdateLegalID(ctx context.Context, entityID, idType, idValue string) error {
	if !validLegalIDTypes[idType] {
		return ErrInvalidLegalIDType
	}

	eid, err := uuid.Parse(entityID)
	if err != nil {
		return ErrEntityNotFound
	}

	return s.Queries.UpdateEntityLegalID(ctx, db.UpdateEntityLegalIDParams{
		EntityID: eid,
		IDType:   idType,
		IDValue:  idValue,
	})
}

func (s *Service) DeleteLegalID(ctx context.Context, entityID, idType string) error {
	eid, err := uuid.Parse(entityID)
	if err != nil {
		return ErrEntityNotFound
	}

	return s.Queries.DeleteEntityLegalID(ctx, db.DeleteEntityLegalIDParams{
		EntityID: eid,
		IDType:   idType,
	})
}

// --- Aliases ---

func (s *Service) AddAlias(ctx context.Context, entityID string, req *AddAliasRequest) (*AliasResponse, error) {
	source := req.Source
	if source == "" {
		source = "MANUAL"
	}
	if !validAliasSources[source] {
		return nil, ErrInvalidAliasSource
	}

	eid, err := uuid.Parse(entityID)
	if err != nil {
		return nil, ErrEntityNotFound
	}

	normalized := NormalizeName(req.Alias)
	now := time.Now()

	a, err := s.Queries.CreateEntityAlias(ctx, db.CreateEntityAliasParams{
		ID:              uuid.New(),
		EntityID:        eid,
		Alias:           req.Alias,
		AliasNormalized: normalized,
		Source:          source,
		CreatedAt:       now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create alias: %w", err)
	}

	return aliasToResponse(&a), nil
}

func (s *Service) GetAliases(ctx context.Context, entityID string) ([]AliasResponse, error) {
	eid, err := uuid.Parse(entityID)
	if err != nil {
		return nil, ErrEntityNotFound
	}

	aliases, err := s.Queries.GetEntityAliases(ctx, eid)
	if err != nil {
		return nil, err
	}

	resp := make([]AliasResponse, 0, len(aliases))
	for i := range aliases {
		resp = append(resp, *aliasToResponse(&aliases[i]))
	}
	return resp, nil
}

func (s *Service) DeleteAlias(ctx context.Context, entityID, aliasID string) error {
	eid, err := uuid.Parse(entityID)
	if err != nil {
		return ErrEntityNotFound
	}
	aid, err := uuid.Parse(aliasID)
	if err != nil {
		return fmt.Errorf("invalid alias_id")
	}

	return s.Queries.DeleteEntityAlias(ctx, db.DeleteEntityAliasParams{
		ID:       aid,
		EntityID: eid,
	})
}

// --- Fuzzy Search ---

func (s *Service) SearchFuzzy(ctx context.Context, query string, limit, offset int) ([]FuzzyMatchResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	normalized := NormalizeName(query)
	if normalized == "" {
		return []FuzzyMatchResponse{}, nil
	}

	results, err := s.Queries.SearchEntitiesFuzzy(ctx, db.SearchEntitiesFuzzyParams{
		Similarity: normalized,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, err
	}

	resp := make([]FuzzyMatchResponse, 0, len(results))
	for i := range results {
		resp = append(resp, FuzzyMatchResponse{
			ID:         results[i].ID.String(),
			NamaUtama:  results[i].NamaUtama,
			EntityType: results[i].EntityType,
			IsShadow:   results[i].IsShadow,
			MatchScore: float64(results[i].MatchScore),
		})
	}
	return resp, nil
}

func (s *Service) FindDuplicates(ctx context.Context, entityID string, limit int) ([]FuzzyMatchResponse, error) {
	eid, err := uuid.Parse(entityID)
	if err != nil {
		return nil, ErrEntityNotFound
	}

	e, err := s.Queries.GetEntityByID(ctx, eid)
	if err != nil {
		return nil, ErrEntityNotFound
	}

	normalized := NormalizeName(e.NamaUtama)
	if normalized == "" {
		return []FuzzyMatchResponse{}, nil
	}

	if limit <= 0 {
		limit = 10
	}

	results, err := s.Queries.FindDuplicateEntities(ctx, db.FindDuplicateEntitiesParams{
		Similarity: normalized,
		ID:         eid,
		Limit:      int32(limit),
	})
	if err != nil {
		return nil, err
	}

	resp := make([]FuzzyMatchResponse, 0, len(results))
	for i := range results {
		resp = append(resp, FuzzyMatchResponse{
			ID:         results[i].ID.String(),
			NamaUtama:  results[i].NamaUtama,
			EntityType: results[i].EntityType,
			IsShadow:   results[i].IsShadow,
			MatchScore: float64(results[i].MatchScore),
		})
	}
	return resp, nil
}

// --- Normalized Name ---

func (s *Service) EnsureNormalizedName(ctx context.Context, entityID uuid.UUID, name string) error {
	normalized := NormalizeName(name)
	return s.Queries.UpdateEntityNormalizedName(ctx, db.UpdateEntityNormalizedNameParams{
		ID:             entityID,
		NamaNormalized: pgtype.Text{String: normalized, Valid: true},
	})
}

// --- Helpers ---

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

func legalIDToResponse(lid *db.EntityLegalID) *LegalIDResponse {
	resp := &LegalIDResponse{
		ID:         lid.ID.String(),
		EntityID:   lid.EntityID.String(),
		IDType:     lid.IDType,
		IDValue:    lid.IDValue,
		IsVerified: lid.IsVerified,
		CreatedAt:  lid.CreatedAt.Format(time.RFC3339),
	}
	if lid.VerifiedAt != nil {
		t := lid.VerifiedAt.Format(time.RFC3339)
		resp.VerifiedAt = &t
	}
	return resp
}

func aliasToResponse(a *db.EntityAlias) *AliasResponse {
	return &AliasResponse{
		ID:        a.ID.String(),
		EntityID:  a.EntityID.String(),
		Alias:     a.Alias,
		Source:    a.Source,
		CreatedAt: a.CreatedAt.Format(time.RFC3339),
	}
}
