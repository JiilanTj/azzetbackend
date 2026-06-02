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
	"codeberg.org/azzet/azzetbe/internal/identity"
)

var ErrEntityNotFound = errors.New("entity not found")
var ErrNotAuthorized = errors.New("not authorized to view this entity")

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

	if req.EntityType == TypeOrangPribadi {
		if _, err := s.Queries.GetEntityByUserID(ctx, pgtype.UUID{Bytes: uid, Valid: true}); err == nil {
			return nil, fmt.Errorf("personal entity already exists for this user")
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to check personal entity: %w", err)
		}
	}

	now := time.Now()
	normalized := identity.NormalizeName(req.NamaUtama)
	e, err := s.Queries.CreateEntity(ctx, db.CreateEntityParams{
		ID:             uuid.New(),
		UserID:         pgtype.UUID{Bytes: uid, Valid: true},
		EntityType:     req.EntityType,
		NamaUtama:      req.NamaUtama,
		NikNpwp:        toPgText(req.NikNpwp),
		NomorWa:        toPgText(req.NomorWa),
		AlamatLengkap:  toPgText(req.AlamatLengkap),
		IsShadow:       false,
		Status:         StatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
		NamaNormalized: pgtype.Text{String: normalized, Valid: normalized != ""},
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
	normalized := identity.NormalizeName(req.NamaUtama)
	e, err := s.Queries.CreateEntity(ctx, db.CreateEntityParams{
		ID:             uuid.New(),
		UserID:         pgtype.UUID{Valid: false}, // NULL = shadow
		EntityType:     entityType,
		NamaUtama:      req.NamaUtama,
		NikNpwp:        toPgText(req.NikNpwp),
		NomorWa:        toPgText(req.NomorWa),
		AlamatLengkap:  toPgText(req.AlamatLengkap),
		IsShadow:       true,
		Status:         StatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
		NamaNormalized: pgtype.Text{String: normalized, Valid: normalized != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create shadow entity: %w", err)
	}

	return &e, nil
}

// CreatePersonalEntity creates a personal entity for a newly registered user
// This is called during registration (Option A - will be refactored to event-driven in Phase 6)
func (s *Service) CreatePersonalEntity(ctx context.Context, userID uuid.UUID, name string) (*db.Entity, error) {
	if existing, err := s.Queries.GetEntityByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true}); err == nil {
		return &existing, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	now := time.Now()
	normalized := identity.NormalizeName(name)
	e, err := s.Queries.CreateEntity(ctx, db.CreateEntityParams{
		ID:             uuid.New(),
		UserID:         pgtype.UUID{Bytes: userID, Valid: true},
		EntityType:     TypeOrangPribadi,
		NamaUtama:      name,
		IsShadow:       false,
		Status:         StatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
		NamaNormalized: pgtype.Text{String: normalized, Valid: normalized != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create personal entity: %w", err)
	}

	return &e, nil
}

// GetEntityByID returns an entity by ID (internal use — no access check).
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

// GetEntityForUser returns entity detail scoped to the caller's access level.
//
// Access rules:
//   - Full (incl. meta, NPWP, alamat): entity owner, workspace member, or claim claimant
//   - Public (name, type, is_shadow, status only): shadow entities discoverable for claim
//   - Limited (public fields): counterparty in one of the user's workspaces
//   - Denied: everyone else
func (s *Service) GetEntityForUser(ctx context.Context, userID, entityID string) (*EntityResponse, error) {
	eid, err := uuid.Parse(entityID)
	if err != nil {
		return nil, ErrEntityNotFound
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrNotAuthorized
	}

	e, err := s.Queries.GetEntityByID(ctx, eid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEntityNotFound
		}
		return nil, err
	}

	level, err := s.resolveEntityViewAccess(ctx, uid, eid, &e)
	if err != nil {
		return nil, err
	}
	if level == entityAccessNone {
		return nil, ErrNotAuthorized
	}

	resp := EntityToResponse(&e)
	if level == entityAccessPublic || level == entityAccessCounterparty {
		public := EntityToPublicResponse(&e)
		return &public, nil
	}

	meta, err := s.Queries.GetEntityMetaByEntityID(ctx, eid)
	if err == nil {
		resp.Meta = EntityMetaToResponse(&meta)
	}

	return &resp, nil
}

type entityViewAccess int

const (
	entityAccessNone entityViewAccess = iota
	entityAccessPublic
	entityAccessCounterparty
	entityAccessFull
)

func (s *Service) resolveEntityViewAccess(ctx context.Context, userID, entityID uuid.UUID, e *db.Entity) (entityViewAccess, error) {
	if e.UserID.Valid && e.UserID.Bytes == userID {
		return entityAccessFull, nil
	}

	isMember, err := s.Queries.UserCanViewEntityAsWorkspaceMember(ctx, db.UserCanViewEntityAsWorkspaceMemberParams{
		ObjectID: entityID,
		UserID:   pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		return entityAccessNone, err
	}
	if isMember {
		return entityAccessFull, nil
	}

	isClaimant, err := s.Queries.UserIsClaimantForEntity(ctx, db.UserIsClaimantForEntityParams{
		EntityID:       entityID,
		ClaimantUserID: userID,
	})
	if err != nil {
		return entityAccessNone, err
	}
	if isClaimant {
		return entityAccessFull, nil
	}

	isCounterparty, err := s.Queries.UserCanViewEntityAsCounterparty(ctx, db.UserCanViewEntityAsCounterpartyParams{
		SubjectID: entityID,
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		return entityAccessNone, err
	}
	if isCounterparty {
		return entityAccessCounterparty, nil
	}

	if e.IsShadow {
		return entityAccessPublic, nil
	}

	return entityAccessNone, nil
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

	normalized := identity.NormalizeName(namaUtama)

	return s.Queries.UpdateEntity(ctx, db.UpdateEntityParams{
		ID:             id,
		NamaUtama:      namaUtama,
		NikNpwp:        nikNpwp,
		NomorWa:        nomorWa,
		AlamatLengkap:  alamat,
		NamaNormalized: pgtype.Text{String: normalized, Valid: normalized != ""},
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

// SearchEntitiesForUser searches entities by name and returns privacy-scoped results.
//
// Search is a discovery endpoint: all matching active entities are returned, but
// sensitive fields (NPWP, alamat, meta) are only included when the caller has
// full access (owner, workspace member, or claim claimant).
func (s *Service) SearchEntitiesForUser(ctx context.Context, userID, query string, limit, offset int) ([]EntityResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id")
	}

	entities, err := s.Queries.SearchEntitiesByName(ctx, db.SearchEntitiesByNameParams{
		Column1: pgtype.Text{String: query, Valid: true},
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, err
	}

	resp := make([]EntityResponse, 0, len(entities))
	for i := range entities {
		level, err := s.resolveEntityViewAccess(ctx, uid, entities[i].ID, &entities[i])
		if err != nil {
			continue
		}

		if level == entityAccessFull {
			er := EntityToResponse(&entities[i])
			if meta, err := s.Queries.GetEntityMetaByEntityID(ctx, entities[i].ID); err == nil {
				er.Meta = EntityMetaToResponse(&meta)
			}
			resp = append(resp, er)
			continue
		}

		resp = append(resp, EntityToPublicResponse(&entities[i]))
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
