package entity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/db"
)

var ErrEntityNotFound = errors.New("entity not found")

type Service struct {
	Queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{Queries: queries}
}

// CreateEntity creates a new entity linked to the authenticated user
func (s *Service) CreateEntity(ctx context.Context, userID string, req *CreateEntityRequest) (*EntityResponse, error) {
	if req.EntityType != TypeOrangPribadi && req.EntityType != TypeBadanUsaha {
		return nil, fmt.Errorf("invalid entity_type")
	}
	if req.NamaUtama == "" {
		return nil, fmt.Errorf("nama_utama is required")
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id")
	}

	now := time.Now()
	e, err := s.Queries.CreateEntity(ctx, db.CreateEntityParams{
		ID:            uuid.New(),
		UserID:        pgtype.UUID{Bytes: uid, Valid: true},
		EntityType:    req.EntityType,
		NamaUtama:     req.NamaUtama,
		NikNpwp:       toPgText(req.NikNpwp),
		NomorWa:       toPgText(req.NomorWa),
		AlamatLengkap: toPgText(req.AlamatLengkap),
		IsShadow:      false,
		Status:        StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create entity: %w", err)
	}

	resp := EntityToResponse(&e)
	return &resp, nil
}

// CreateShadowEntity creates a shadow entity (no user_id, created by others)
func (s *Service) CreateShadowEntity(ctx context.Context, req *CreateEntityRequest) (*db.Entity, error) {
	if req.NamaUtama == "" {
		return nil, fmt.Errorf("nama_utama is required")
	}

	entityType := req.EntityType
	if entityType == "" {
		entityType = TypeBadanUsaha // Default shadow entities to BADAN_USAHA
	}

	now := time.Now()
	e, err := s.Queries.CreateEntity(ctx, db.CreateEntityParams{
		ID:            uuid.New(),
		UserID:        pgtype.UUID{Valid: false}, // NULL = shadow
		EntityType:    entityType,
		NamaUtama:     req.NamaUtama,
		NikNpwp:       toPgText(req.NikNpwp),
		NomorWa:       toPgText(req.NomorWa),
		AlamatLengkap: toPgText(req.AlamatLengkap),
		IsShadow:      true,
		Status:        StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create shadow entity: %w", err)
	}

	return &e, nil
}

// CreatePersonalEntity creates a personal entity for a newly registered user
// This is called during registration (Option A - will be refactored to event-driven in Phase 6)
func (s *Service) CreatePersonalEntity(ctx context.Context, userID uuid.UUID, name string) (*db.Entity, error) {
	now := time.Now()
	e, err := s.Queries.CreateEntity(ctx, db.CreateEntityParams{
		ID:            uuid.New(),
		UserID:        pgtype.UUID{Bytes: userID, Valid: true},
		EntityType:    TypeOrangPribadi,
		NamaUtama:     name,
		IsShadow:      false,
		Status:        StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create personal entity: %w", err)
	}

	return &e, nil
}

// GetEntityByID returns an entity by ID
func (s *Service) GetEntityByID(ctx context.Context, entityID string) (*EntityResponse, error) {
	id, err := uuid.Parse(entityID)
	if err != nil {
		return nil, ErrEntityNotFound
	}

	e, err := s.Queries.GetEntityByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEntityNotFound
		}
		return nil, err
	}

	resp := EntityToResponse(&e)

	// Load meta
	meta, err := s.Queries.GetEntityMetaByEntityID(ctx, id)
	if err == nil {
		resp.Meta = EntityMetaToResponse(&meta)
	}

	return &resp, nil
}

// GetPersonalEntity returns the user's personal entity
func (s *Service) GetPersonalEntity(ctx context.Context, userID string) (*db.Entity, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrEntityNotFound
	}

	e, err := s.Queries.GetEntityByUserID(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEntityNotFound
		}
		return nil, err
	}

	return &e, nil
}

// ListMyEntities returns all entities owned by the user
func (s *Service) ListMyEntities(ctx context.Context, userID string) ([]EntityResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id")
	}

	entities, err := s.Queries.ListEntitiesByUserID(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		return nil, err
	}

	var resp []EntityResponse
	for i := range entities {
		resp = append(resp, EntityToResponse(&entities[i]))
	}
	if resp == nil {
		resp = []EntityResponse{}
	}
	return resp, nil
}

// UpdateEntity updates an entity
func (s *Service) UpdateEntity(ctx context.Context, userID, entityID string, req *UpdateEntityRequest) error {
	id, err := uuid.Parse(entityID)
	if err != nil {
		return ErrEntityNotFound
	}

	// Verify ownership
	e, err := s.Queries.GetEntityByID(ctx, id)
	if err != nil {
		return ErrEntityNotFound
	}

	uid, _ := uuid.Parse(userID)
	if !e.UserID.Valid || e.UserID.Bytes != uid {
		return fmt.Errorf("not authorized to update this entity")
	}

	namaUtama := e.NamaUtama
	nikNpwp := e.NikNpwp
	nomorWa := e.NomorWa
	alamat := e.AlamatLengkap

	if req.NamaUtama != nil {
		namaUtama = *req.NamaUtama
	}
	if req.NikNpwp != nil {
		nikNpwp = pgtype.Text{String: *req.NikNpwp, Valid: true}
	}
	if req.NomorWa != nil {
		nomorWa = pgtype.Text{String: *req.NomorWa, Valid: true}
	}
	if req.AlamatLengkap != nil {
		alamat = pgtype.Text{String: *req.AlamatLengkap, Valid: true}
	}

	return s.Queries.UpdateEntity(ctx, db.UpdateEntityParams{
		ID:            id,
		NamaUtama:     namaUtama,
		NikNpwp:       nikNpwp,
		NomorWa:       nomorWa,
		AlamatLengkap: alamat,
	})
}

// UpdateEntityMeta updates or creates entity meta
func (s *Service) UpdateEntityMeta(ctx context.Context, userID, entityID string, req *UpdateEntityMetaRequest) error {
	id, err := uuid.Parse(entityID)
	if err != nil {
		return ErrEntityNotFound
	}

	// Verify ownership
	e, err := s.Queries.GetEntityByID(ctx, id)
	if err != nil {
		return ErrEntityNotFound
	}

	uid, _ := uuid.Parse(userID)
	if !e.UserID.Valid || e.UserID.Bytes != uid {
		return fmt.Errorf("not authorized to update this entity")
	}

	now := time.Now()
	_, err = s.Queries.UpsertEntityMeta(ctx, db.UpsertEntityMetaParams{
		ID:          uuid.New(),
		EntityID:    id,
		BidangUsaha: toPgText(req.BidangUsaha),
		LogoUrl:     toPgText(req.LogoURL),
		Website:     toPgText(req.Website),
		Email:       toPgText(req.Email),
		Description: toPgText(req.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	return err
}

// SearchEntities searches entities by name
func (s *Service) SearchEntities(ctx context.Context, query string, limit, offset int) ([]EntityResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	entities, err := s.Queries.SearchEntitiesByName(ctx, db.SearchEntitiesByNameParams{
		Column1: pgtype.Text{String: query, Valid: true},
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, err
	}

	var resp []EntityResponse
	for i := range entities {
		resp = append(resp, EntityToResponse(&entities[i]))
	}
	if resp == nil {
		resp = []EntityResponse{}
	}
	return resp, nil
}

// --- Helpers ---

func toPgText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}
