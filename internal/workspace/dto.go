package workspace

import (
	"time"

	"codeberg.org/azzet/azzetbe/internal/db"
)

// --- Request DTOs ---

// CreateWorkspaceRequest represents workspace creation
// @Description Create a workspace from an existing entity
type CreateWorkspaceRequest struct {
	EntityID string `json:"entity_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// InviteMemberRequest represents member invitation
// @Description Invite a member to the workspace
type InviteMemberRequest struct {
	EntityID    string  `json:"entity_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Role        string  `json:"role" example:"KASIR" enums:"AKUNTAN,KASIR,VIEWER"`
	CustomAlias *string `json:"custom_alias,omitempty" example:"Andi Accounting"`
}

// AddCounterpartyRequest represents adding a counterparty
// @Description Add a counterparty (customer/vendor) to workspace. Creates shadow entity if needed.
type AddCounterpartyRequest struct {
	EntityID     *string `json:"entity_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	RelationType string  `json:"relation_type" example:"PELANGGAN" enums:"PELANGGAN,VENDOR"`
	CustomAlias  *string `json:"custom_alias,omitempty" example:"Toko Maju"`
	// If entity_id is nil, create shadow entity from these fields:
	NamaUtama  *string `json:"nama_utama,omitempty" example:"Toko Maju"`
	EntityType *string `json:"entity_type,omitempty" example:"BADAN_USAHA" enums:"ORANG_PRIBADI,BADAN_USAHA"`
	NikNpwp    *string `json:"nik_npwp,omitempty" example:"01.234.567.8-901.000"`
	NomorWa    *string `json:"nomor_wa,omitempty" example:"+628123456789"`
}

// UpdateMemberRequest represents member update
// @Description Update member role or status
type UpdateMemberRequest struct {
	Role        *string `json:"role,omitempty" example:"AKUNTAN" enums:"AKUNTAN,KASIR,VIEWER"`
	CustomAlias *string `json:"custom_alias,omitempty" example:"Updated Alias"`
	Status      *string `json:"status,omitempty" example:"INACTIVE" enums:"ACTIVE,INACTIVE"`
}

// --- Response DTOs ---

// WorkspaceResponse represents a workspace
// @Description Workspace information
type WorkspaceResponse struct {
	ID         string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EntityID   string `json:"entity_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EntityName string `json:"entity_name" example:"PT Maju Jaya"`
	EntityType string `json:"entity_type" example:"BADAN_USAHA"`
	Role       string `json:"role" example:"PEMILIK"`
	CreatedAt  string `json:"created_at" example:"2026-05-20T10:00:00Z"`
}

// MemberResponse represents a workspace member
// @Description Workspace member information
type MemberResponse struct {
	ID           string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EntityID     string  `json:"entity_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EntityName   string  `json:"entity_name" example:"Andi"`
	EntityType   string  `json:"entity_type" example:"ORANG_PRIBADI"`
	RelationType string  `json:"relation_type" example:"KARYAWAN"`
	CustomAlias  *string `json:"custom_alias,omitempty" example:"Andi Accounting"`
	Role         *string `json:"role,omitempty" example:"KASIR"`
	Status       string  `json:"status" example:"ACTIVE"`
	CreatedAt    string  `json:"created_at" example:"2026-05-20T10:00:00Z"`
}

// CounterpartyResponse represents a counterparty in workspace
// @Description Counterparty (customer/vendor) information
type CounterpartyResponse struct {
	ID           string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EntityID     string  `json:"entity_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EntityName   string  `json:"entity_name" example:"Toko Maju"`
	EntityType   string  `json:"entity_type" example:"BADAN_USAHA"`
	RelationType string  `json:"relation_type" example:"PELANGGAN"`
	CustomAlias  *string `json:"custom_alias,omitempty" example:"Toko Maju Cabang Utara"`
	IsShadow     bool    `json:"is_shadow" example:"true"`
	Status       string  `json:"status" example:"ACTIVE"`
	CreatedAt    string  `json:"created_at" example:"2026-05-20T10:00:00Z"`
}

// RoleResponse represents a role
// @Description Available role
type RoleResponse struct {
	ID          string   `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string   `json:"name" example:"KASIR"`
	Description *string  `json:"description,omitempty" example:"Transaction entry only"`
	Permissions []string `json:"permissions" example:"[\"transaction:create\",\"transaction:read\"]"`
}

// MessageResponse represents a simple message
// @Description Simple message response
type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// --- Constants ---

const (
	RelationPemilik   = "PEMILIK"
	RelationKaryawan  = "KARYAWAN"
	RelationPelanggan = "PELANGGAN"
	RelationVendor    = "VENDOR"
)

// --- Converters ---

func RelationToMemberResponse(r *db.EntityRelation, e *db.Entity, roleName *string) MemberResponse {
	resp := MemberResponse{
		ID:           r.ID.String(),
		EntityID:     r.SubjectID.String(),
		EntityName:   e.NamaUtama,
		EntityType:   e.EntityType,
		RelationType: r.RelationType,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt.Format(time.RFC3339),
	}
	if r.CustomAlias.Valid {
		resp.CustomAlias = &r.CustomAlias.String
	}
	if roleName != nil {
		resp.Role = roleName
	}
	return resp
}

func RelationToCounterpartyResponse(r *db.EntityRelation, e *db.Entity) CounterpartyResponse {
	resp := CounterpartyResponse{
		ID:           r.ID.String(),
		EntityID:     r.SubjectID.String(),
		EntityName:   e.NamaUtama,
		EntityType:   e.EntityType,
		RelationType: r.RelationType,
		IsShadow:     e.IsShadow,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt.Format(time.RFC3339),
	}
	if r.CustomAlias.Valid {
		resp.CustomAlias = &r.CustomAlias.String
	}
	return resp
}
