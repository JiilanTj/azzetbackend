package workspace

import (
	"context"
	"encoding/json"
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

	// Create relation: entity (object/workspace) ← user's personal entity (subject/member) as PEMILIK
	now := time.Now()
	rel, err := s.Queries.CreateRelation(ctx, db.CreateRelationParams{
		ID:           uuid.New(),
		ObjectID:     entityID,
		SubjectID:    personalEntity.ID,
		RelationType: RelationPemilik,
		Status:       "ACTIVE",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Bootstrap system "Owner" role with wildcard permissions
	_ = s.bootstrapOwnerRole(ctx, entityID, uid, now)

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
func (s *Service) CreatePersonalWorkspace(ctx context.Context, personalEntityID uuid.UUID, userID uuid.UUID) error {
	now := time.Now()
	_, err := s.Queries.CreateRelation(ctx, db.CreateRelationParams{
		ID:           uuid.New(),
		ObjectID:     personalEntityID, // workspace = personal entity
		SubjectID:    personalEntityID, // member = same entity (self-owned)
		RelationType: RelationPemilik,
		Status:       "ACTIVE",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return err
	}

	// Bootstrap system "Owner" role with wildcard permissions
	_ = s.bootstrapOwnerRole(ctx, personalEntityID, userID, now)

	return nil
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

		ws := WorkspaceResponse{
			ID:         r.ID.String(),
			EntityID:   r.ObjectID.String(),
			EntityName: e.NamaUtama,
			EntityType: e.EntityType,
			Role:       r.RelationType,
			CreatedAt:  r.CreatedAt.Format(time.RFC3339),
		}

		// Fetch subscription status for this workspace
		sub, err := s.Queries.GetActiveSubscription(ctx, r.ObjectID)
		if err == nil {
			ws.SubscriptionStatus = &sub.Status
			// Fetch plan name
			plan, err := s.Queries.GetPlanByID(ctx, sub.PlanID)
			if err == nil {
				ws.PlanName = &plan.Name
			}
		}

		resp = append(resp, ws)
	}
	if resp == nil {
		resp = []WorkspaceResponse{}
	}
	return resp, nil
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

		// Get role name from workspace_role_assignments
		var roleName *string
		assignments, err := s.Queries.ListRoleAssignmentsByMember(ctx, db.ListRoleAssignmentsByMemberParams{
			WorkspaceID:    r.ObjectID,
			MemberEntityID: r.SubjectID,
		})
		if err == nil && len(assignments) > 0 {
			roleName = &assignments[0].RoleName
		}

		resp = append(resp, RelationToMemberResponse(&r, &e, roleName))
	}
	if resp == nil {
		resp = []MemberResponse{}
	}
	return resp, nil
}

// UpdateMember updates a member's alias/status
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
	status := rel.Status

	if req.CustomAlias != nil {
		customAlias = pgtype.Text{String: *req.CustomAlias, Valid: true}
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

// ListWorkspaceRoles returns all custom roles for a workspace
func (s *Service) ListWorkspaceRoles(ctx context.Context, workspaceID string) ([]RoleResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrWorkspaceNotFound
	}

	roles, err := s.Queries.ListWorkspaceRoles(ctx, wsID)
	if err != nil {
		return nil, err
	}

	var resp []RoleResponse
	for _, r := range roles {
		var desc *string
		if r.Description.Valid {
			desc = &r.Description.String
		}

		resp = append(resp, RoleResponse{
			ID:          r.ID.String(),
			Name:        r.Name,
			Description: desc,
			Permissions: r.Permissions,
		})
	}
	if resp == nil {
		resp = []RoleResponse{}
	}
	return resp, nil
}

// CreateWorkspaceRole creates a custom role for a workspace
func (s *Service) CreateWorkspaceRole(ctx context.Context, workspaceID, userID string, req *CreateRoleRequest) (*RoleResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrWorkspaceNotFound
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(req.Permissions) == 0 {
		return nil, fmt.Errorf("at least one permission is required")
	}

	// Validate permissions
	for _, p := range req.Permissions {
		if !IsValidPermission(p) {
			return nil, fmt.Errorf("invalid permission: %s", p)
		}
		// Non-owner cannot create roles with wildcard
		if p == PermAll {
			return nil, fmt.Errorf("cannot assign wildcard permission to custom roles")
		}
	}

	now := time.Now()
	role, err := s.Queries.CreateWorkspaceRole(ctx, db.CreateWorkspaceRoleParams{
		ID:          uuid.New(),
		WorkspaceID: wsID,
		Name:        req.Name,
		Description: toPgText(req.Description),
		Permissions: req.Permissions,
		IsSystem:    false,
		CreatedBy:   uid,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	var desc *string
	if role.Description.Valid {
		desc = &role.Description.String
	}

	return &RoleResponse{
		ID:          role.ID.String(),
		Name:        role.Name,
		Description: desc,
		Permissions: role.Permissions,
	}, nil
}

// UpdateWorkspaceRole updates a custom role
func (s *Service) UpdateWorkspaceRole(ctx context.Context, roleID string, req *UpdateRoleRequest) error {
	rID, err := uuid.Parse(roleID)
	if err != nil {
		return fmt.Errorf("invalid role_id")
	}

	role, err := s.Queries.GetWorkspaceRoleByID(ctx, rID)
	if err != nil {
		return fmt.Errorf("role not found")
	}

	if role.IsSystem {
		return fmt.Errorf("cannot modify system roles")
	}

	name := role.Name
	description := role.Description
	permissions := role.Permissions

	if req.Name != nil {
		name = *req.Name
	}
	if req.Description != nil {
		description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.Permissions != nil {
		for _, p := range req.Permissions {
			if !IsValidPermission(p) {
				return fmt.Errorf("invalid permission: %s", p)
			}
			if p == PermAll {
				return fmt.Errorf("cannot assign wildcard permission to custom roles")
			}
		}
		permissions = req.Permissions
	}

	return s.Queries.UpdateWorkspaceRole(ctx, db.UpdateWorkspaceRoleParams{
		ID:          rID,
		Name:        name,
		Description: description,
		Permissions: permissions,
	})
}

// DeleteWorkspaceRole deletes a custom role (system roles cannot be deleted)
func (s *Service) DeleteWorkspaceRole(ctx context.Context, roleID string) error {
	rID, err := uuid.Parse(roleID)
	if err != nil {
		return fmt.Errorf("invalid role_id")
	}

	role, err := s.Queries.GetWorkspaceRoleByID(ctx, rID)
	if err != nil {
		return fmt.Errorf("role not found")
	}

	if role.IsSystem {
		return fmt.Errorf("cannot delete system roles")
	}

	return s.Queries.DeleteWorkspaceRole(ctx, rID)
}

// AssignRole assigns a role to a workspace member
func (s *Service) AssignRole(ctx context.Context, workspaceID, userID string, req *AssignRoleRequest) (*RoleAssignmentResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrWorkspaceNotFound
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id")
	}
	memberEntityID, err := uuid.Parse(req.MemberEntityID)
	if err != nil {
		return nil, fmt.Errorf("invalid member_entity_id")
	}
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		return nil, fmt.Errorf("invalid role_id")
	}

	// Verify role belongs to this workspace
	role, err := s.Queries.GetWorkspaceRoleByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("role not found")
	}
	if role.WorkspaceID != wsID {
		return nil, fmt.Errorf("role does not belong to this workspace")
	}

	// Verify member exists in workspace
	exists, err := s.Queries.ExistsRelation(ctx, db.ExistsRelationParams{
		ObjectID:     wsID,
		SubjectID:    memberEntityID,
		RelationType: RelationKaryawan,
	})
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("member not found in workspace")
	}

	now := time.Now()
	assignment, err := s.Queries.CreateRoleAssignment(ctx, db.CreateRoleAssignmentParams{
		ID:             uuid.New(),
		WorkspaceID:    wsID,
		MemberEntityID: memberEntityID,
		RoleID:         roleID,
		AssignedBy:     uid,
		CreatedAt:      now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to assign role: %w", err)
	}

	return &RoleAssignmentResponse{
		ID:             assignment.ID.String(),
		WorkspaceID:    assignment.WorkspaceID.String(),
		MemberEntityID: assignment.MemberEntityID.String(),
		RoleID:         assignment.RoleID.String(),
		RoleName:       role.Name,
		AssignedBy:     assignment.AssignedBy.String(),
		CreatedAt:      assignment.CreatedAt.Format(time.RFC3339),
	}, nil
}

// UnassignRole removes a role assignment from a member
func (s *Service) UnassignRole(ctx context.Context, workspaceID string, req *AssignRoleRequest) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return ErrWorkspaceNotFound
	}
	memberEntityID, err := uuid.Parse(req.MemberEntityID)
	if err != nil {
		return fmt.Errorf("invalid member_entity_id")
	}
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		return fmt.Errorf("invalid role_id")
	}

	return s.Queries.DeleteRoleAssignment(ctx, db.DeleteRoleAssignmentParams{
		WorkspaceID:    wsID,
		MemberEntityID: memberEntityID,
		RoleID:         roleID,
	})
}

// VerifyWorkspaceAccess checks if a user has access to a workspace and returns their relation type + permissions
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

	// Check if user has workspace access via entity_relations
	rel, err := s.Queries.GetUserWorkspaceAccess(ctx, db.GetUserWorkspaceAccessParams{
		ObjectID:  wsID,
		SubjectID: personalEntity.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, ErrNotAuthorized
		}
		return "", nil, err
	}

	// PEMILIK always has wildcard permissions
	if rel.RelationType == RelationPemilik {
		return RelationPemilik, []byte(`["*"]`), nil
	}

	// For KARYAWAN, get permissions from workspace_role_assignments
	assignments, err := s.Queries.ListRoleAssignmentsByMember(ctx, db.ListRoleAssignmentsByMemberParams{
		WorkspaceID:    wsID,
		MemberEntityID: personalEntity.ID,
	})
	if err != nil || len(assignments) == 0 {
		// Member exists but has no role assigned — minimal access
		return RelationKaryawan, []byte(`[]`), nil
	}

	// Merge all permissions from all assigned roles
	permSet := make(map[string]bool)
	for _, a := range assignments {
		for _, p := range a.RolePermissions {
			permSet[p] = true
		}
	}

	// Build JSON array
	perms := make([]string, 0, len(permSet))
	for p := range permSet {
		perms = append(perms, p)
	}

	// Serialize as JSON array
	permJSON, err := json.Marshal(perms)
	if err != nil {
		return RelationKaryawan, []byte(`[]`), nil
	}

	return RelationKaryawan, permJSON, nil
}

// --- Helpers ---

// bootstrapOwnerRole creates the system "Owner" role with wildcard permissions for a new workspace
func (s *Service) bootstrapOwnerRole(ctx context.Context, workspaceID, userID uuid.UUID, now time.Time) error {
	_, err := s.Queries.CreateWorkspaceRole(ctx, db.CreateWorkspaceRoleParams{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        "Owner",
		Description: pgtype.Text{String: "Full access to all workspace features", Valid: true},
		Permissions: []string{PermAll},
		IsSystem:    true,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	return err
}

func toPgText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}
