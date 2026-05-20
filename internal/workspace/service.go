package workspace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/entity"
)

var ErrWorkspaceNotFound = errors.New("workspace not found")
var ErrNotAuthorized = errors.New("not authorized")
var ErrRelationExists = errors.New("relation already exists")

type Service struct {
	Queries       *db.Queries
	EntityService *entity.Service
}

func NewService(queries *db.Queries, entityService *entity.Service) *Service {
	return &Service{
		Queries:       queries,
		EntityService: entityService,
	}
}

// CreateWorkspace creates a workspace from an entity (user becomes PEMILIK)
func (s *Service) CreateWorkspace(ctx context.Context, userID string, req *CreateWorkspaceRequest) (*WorkspaceResponse, error) {
	entityID, err := uuid.Parse(req.EntityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity_id")
	}

	// Verify entity exists and belongs to user
	e, err := s.Queries.GetEntityByID(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("entity not found")
	}

	uid, _ := uuid.Parse(userID)
	if !e.UserID.Valid || e.UserID.Bytes != uid {
		return nil, ErrNotAuthorized
	}

	// Get user's personal entity (subject in relation)
	personalEntity, err := s.EntityService.GetPersonalEntity(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("personal entity not found, please create one first")
	}

	// Check if workspace relation already exists
	exists, err := s.Queries.ExistsRelation(ctx, db.ExistsRelationParams{
		ObjectID:     entityID,
		SubjectID:    personalEntity.ID,
		RelationType: RelationPemilik,
	})
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("workspace already exists for this entity")
	}

	// Get PEMILIK role
	role, err := s.Queries.GetRoleByName(ctx, "PEMILIK")
	if err != nil {
		return nil, fmt.Errorf("failed to get PEMILIK role")
	}

	// Create relation: entity (object/workspace) ← user's personal entity (subject/member) as PEMILIK
	now := time.Now()
	rel, err := s.Queries.CreateRelation(ctx, db.CreateRelationParams{
		ID:           uuid.New(),
		ObjectID:     entityID,
		SubjectID:    personalEntity.ID,
		RelationType: RelationPemilik,
		RoleID:       pgtype.UUID{Bytes: role.ID, Valid: true},
		Status:       "ACTIVE",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	return &WorkspaceResponse{
		ID:         rel.ID.String(),
		EntityID:   entityID.String(),
		EntityName: e.NamaUtama,
		EntityType: e.EntityType,
		Role:       RelationPemilik,
		CreatedAt:  rel.CreatedAt.Format(time.RFC3339),
	}, nil
}

// CreatePersonalWorkspace creates the personal workspace for a user (called during registration)
func (s *Service) CreatePersonalWorkspace(ctx context.Context, personalEntityID uuid.UUID) error {
	role, err := s.Queries.GetRoleByName(ctx, "PEMILIK")
	if err != nil {
		return fmt.Errorf("failed to get PEMILIK role")
	}

	now := time.Now()
	_, err = s.Queries.CreateRelation(ctx, db.CreateRelationParams{
		ID:           uuid.New(),
		ObjectID:     personalEntityID, // workspace = personal entity
		SubjectID:    personalEntityID, // member = same entity (self-owned)
		RelationType: RelationPemilik,
		RoleID:       pgtype.UUID{Bytes: role.ID, Valid: true},
		Status:       "ACTIVE",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	return err
}

// ListMyWorkspaces returns all workspaces the user has access to
func (s *Service) ListMyWorkspaces(ctx context.Context, userID string) ([]WorkspaceResponse, error) {
	personalEntity, err := s.EntityService.GetPersonalEntity(ctx, userID)
	if err != nil {
		return nil, err
	}

	relations, err := s.Queries.ListWorkspacesBySubject(ctx, personalEntity.ID)
	if err != nil {
		return nil, err
	}

	var resp []WorkspaceResponse
	for _, r := range relations {
		e, err := s.Queries.GetEntityByID(ctx, r.ObjectID)
		if err != nil {
			continue
		}

		roleName := r.RelationType
		if r.RoleID.Valid {
			if role, err := s.Queries.GetRoleByID(ctx, r.RoleID.Bytes); err == nil {
				roleName = role.Name
			}
		}

		resp = append(resp, WorkspaceResponse{
			ID:         r.ID.String(),
			EntityID:   r.ObjectID.String(),
			EntityName: e.NamaUtama,
			EntityType: e.EntityType,
			Role:       roleName,
			CreatedAt:  r.CreatedAt.Format(time.RFC3339),
		})
	}
	if resp == nil {
		resp = []WorkspaceResponse{}
	}
	return resp, nil
}

// InviteMember invites a member to the workspace
func (s *Service) InviteMember(ctx context.Context, workspaceID string, req *InviteMemberRequest) (*MemberResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrWorkspaceNotFound
	}

	memberEntityID, err := uuid.Parse(req.EntityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity_id")
	}

	// Verify member entity exists
	memberEntity, err := s.Queries.GetEntityByID(ctx, memberEntityID)
	if err != nil {
		return nil, fmt.Errorf("member entity not found")
	}

	// Check if relation already exists
	exists, err := s.Queries.ExistsRelation(ctx, db.ExistsRelationParams{
		ObjectID:     wsID,
		SubjectID:    memberEntityID,
		RelationType: RelationKaryawan,
	})
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrRelationExists
	}

	// Get role
	var roleID pgtype.UUID
	var roleName *string
	if req.Role != "" {
		role, err := s.Queries.GetRoleByName(ctx, req.Role)
		if err != nil {
			return nil, fmt.Errorf("invalid role: %s", req.Role)
		}
		roleID = pgtype.UUID{Bytes: role.ID, Valid: true}
		roleName = &role.Name
	}

	now := time.Now()
	rel, err := s.Queries.CreateRelation(ctx, db.CreateRelationParams{
		ID:           uuid.New(),
		ObjectID:     wsID,
		SubjectID:    memberEntityID,
		RelationType: RelationKaryawan,
		CustomAlias:  toPgText(req.CustomAlias),
		RoleID:       roleID,
		Status:       "ACTIVE",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invite member: %w", err)
	}

	resp := RelationToMemberResponse(&rel, &memberEntity, roleName)
	return &resp, nil
}

// ListMembers returns all members (PEMILIK + KARYAWAN) of a workspace
func (s *Service) ListMembers(ctx context.Context, workspaceID string) ([]MemberResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrWorkspaceNotFound
	}

	relations, err := s.Queries.ListRelationsByObject(ctx, wsID)
	if err != nil {
		return nil, err
	}

	var resp []MemberResponse
	for _, r := range relations {
		if r.RelationType != RelationPemilik && r.RelationType != RelationKaryawan {
			continue
		}

		e, err := s.Queries.GetEntityByID(ctx, r.SubjectID)
		if err != nil {
			continue
		}

		var roleName *string
		if r.RoleID.Valid {
			if role, err := s.Queries.GetRoleByID(ctx, r.RoleID.Bytes); err == nil {
				roleName = &role.Name
			}
		}

		resp = append(resp, RelationToMemberResponse(&r, &e, roleName))
	}
	if resp == nil {
		resp = []MemberResponse{}
	}
	return resp, nil
}

// UpdateMember updates a member's role/alias/status
func (s *Service) UpdateMember(ctx context.Context, relationID string, req *UpdateMemberRequest) error {
	relID, err := uuid.Parse(relationID)
	if err != nil {
		return fmt.Errorf("invalid relation_id")
	}

	rel, err := s.Queries.GetRelationByID(ctx, relID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("member not found")
		}
		return err
	}

	customAlias := rel.CustomAlias
	roleID := rel.RoleID
	status := rel.Status

	if req.CustomAlias != nil {
		customAlias = pgtype.Text{String: *req.CustomAlias, Valid: true}
	}
	if req.Role != nil {
		role, err := s.Queries.GetRoleByName(ctx, *req.Role)
		if err != nil {
			return fmt.Errorf("invalid role: %s", *req.Role)
		}
		roleID = pgtype.UUID{Bytes: role.ID, Valid: true}
	}
	if req.Status != nil {
		if *req.Status != "ACTIVE" && *req.Status != "INACTIVE" {
			return fmt.Errorf("invalid status")
		}
		status = *req.Status
	}

	return s.Queries.UpdateRelation(ctx, db.UpdateRelationParams{
		ID:          relID,
		CustomAlias: customAlias,
		RoleID:      roleID,
		Status:      status,
	})
}

// RemoveMember removes a member from workspace
func (s *Service) RemoveMember(ctx context.Context, relationID string) error {
	relID, err := uuid.Parse(relationID)
	if err != nil {
		return fmt.Errorf("invalid relation_id")
	}

	rel, err := s.Queries.GetRelationByID(ctx, relID)
	if err != nil {
		return fmt.Errorf("member not found")
	}

	// Cannot remove PEMILIK
	if rel.RelationType == RelationPemilik {
		return fmt.Errorf("cannot remove workspace owner")
	}

	return s.Queries.DeleteRelation(ctx, relID)
}

// AddCounterparty adds a counterparty (PELANGGAN/VENDOR) to workspace
func (s *Service) AddCounterparty(ctx context.Context, workspaceID string, req *AddCounterpartyRequest) (*CounterpartyResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrWorkspaceNotFound
	}

	if req.RelationType != RelationPelanggan && req.RelationType != RelationVendor {
		return nil, fmt.Errorf("relation_type must be PELANGGAN or VENDOR")
	}

	var counterpartyEntity db.Entity

	if req.EntityID != nil {
		// Link to existing entity
		eid, err := uuid.Parse(*req.EntityID)
		if err != nil {
			return nil, fmt.Errorf("invalid entity_id")
		}
		e, err := s.Queries.GetEntityByID(ctx, eid)
		if err != nil {
			return nil, fmt.Errorf("entity not found")
		}
		counterpartyEntity = e
	} else {
		// Create shadow entity
		if req.NamaUtama == nil || *req.NamaUtama == "" {
			return nil, fmt.Errorf("nama_utama is required when creating new counterparty")
		}

		entityType := "BADAN_USAHA"
		if req.EntityType != nil {
			entityType = *req.EntityType
		}

		shadow, err := s.EntityService.CreateShadowEntity(ctx, &entity.CreateEntityRequest{
			EntityType: entityType,
			NamaUtama:  *req.NamaUtama,
			NikNpwp:    req.NikNpwp,
			NomorWa:    req.NomorWa,
		})
		if err != nil {
			return nil, err
		}
		counterpartyEntity = *shadow
	}

	// Check if relation already exists
	exists, err := s.Queries.ExistsRelation(ctx, db.ExistsRelationParams{
		ObjectID:     wsID,
		SubjectID:    counterpartyEntity.ID,
		RelationType: req.RelationType,
	})
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrRelationExists
	}

	// Determine alias
	alias := req.CustomAlias
	if alias == nil {
		alias = &counterpartyEntity.NamaUtama
	}

	now := time.Now()
	rel, err := s.Queries.CreateRelation(ctx, db.CreateRelationParams{
		ID:           uuid.New(),
		ObjectID:     wsID,
		SubjectID:    counterpartyEntity.ID,
		RelationType: req.RelationType,
		CustomAlias:  toPgText(alias),
		Status:       "ACTIVE",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add counterparty: %w", err)
	}

	resp := RelationToCounterpartyResponse(&rel, &counterpartyEntity)
	return &resp, nil
}

// ListCounterparties returns all counterparties (PELANGGAN + VENDOR) of a workspace
func (s *Service) ListCounterparties(ctx context.Context, workspaceID string) ([]CounterpartyResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrWorkspaceNotFound
	}

	relations, err := s.Queries.ListRelationsByObject(ctx, wsID)
	if err != nil {
		return nil, err
	}

	var resp []CounterpartyResponse
	for _, r := range relations {
		if r.RelationType != RelationPelanggan && r.RelationType != RelationVendor {
			continue
		}

		e, err := s.Queries.GetEntityByID(ctx, r.SubjectID)
		if err != nil {
			continue
		}

		resp = append(resp, RelationToCounterpartyResponse(&r, &e))
	}
	if resp == nil {
		resp = []CounterpartyResponse{}
	}
	return resp, nil
}

// ListRoles returns all available roles
func (s *Service) ListRoles(ctx context.Context) ([]RoleResponse, error) {
	roles, err := s.Queries.ListRoles(ctx)
	if err != nil {
		return nil, err
	}

	var resp []RoleResponse
	for _, r := range roles {
		var perms []string
		if r.Permissions != nil {
			// Parse JSONB permissions
			_ = r.Permissions // TODO: parse JSON array
		}

		var desc *string
		if r.Description.Valid {
			desc = &r.Description.String
		}

		resp = append(resp, RoleResponse{
			ID:          r.ID.String(),
			Name:        r.Name,
			Description: desc,
			Permissions: perms,
		})
	}
	if resp == nil {
		resp = []RoleResponse{}
	}
	return resp, nil
}

// VerifyWorkspaceAccess checks if a user has access to a workspace and returns their role
func (s *Service) VerifyWorkspaceAccess(ctx context.Context, workspaceID, userID string) (string, []byte, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return "", nil, ErrWorkspaceNotFound
	}

	// Get user's personal entity
	personalEntity, err := s.EntityService.GetPersonalEntity(ctx, userID)
	if err != nil {
		return "", nil, ErrNotAuthorized
	}

	// Check if user has workspace access
	row, err := s.Queries.GetUserWorkspaceRole(ctx, db.GetUserWorkspaceRoleParams{
		ObjectID:  wsID,
		SubjectID: personalEntity.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, ErrNotAuthorized
		}
		return "", nil, err
	}

	roleName := ""
	if row.RoleName.Valid {
		roleName = row.RoleName.String
	}

	var permissions []byte
	if row.RolePermissions != nil {
		permissions = row.RolePermissions
	}

	return roleName, permissions, nil
}

// --- Helpers ---

func toPgText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}
