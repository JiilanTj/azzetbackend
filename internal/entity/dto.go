package entity

import (
	"time"

	"github.com/google/uuid"

	"codeberg.org/azzet/azzetbe/internal/db"
)

// --- Request DTOs ---

// CreateEntityRequest represents entity creation payload
// @Description Create a new entity (personal or business)
type CreateEntityRequest struct {
	EntityType    string  `json:"entity_type" example:"BADAN_USAHA" enums:"ORANG_PRIBADI,BADAN_USAHA"`
	NamaUtama     string  `json:"nama_utama" example:"PT Maju Jaya"`
	NikNpwp       *string `json:"nik_npwp,omitempty" example:"01.234.567.8-901.000"`
	NomorWa       *string `json:"nomor_wa,omitempty" example:"+628123456789"`
	AlamatLengkap *string `json:"alamat_lengkap,omitempty" example:"Jl. Sudirman No. 1, Jakarta"`
}

// UpdateEntityRequest represents entity update payload
// @Description Update an existing entity
type UpdateEntityRequest struct {
	NamaUtama     *string `json:"nama_utama,omitempty" example:"PT Maju Jaya Sejahtera"`
	NikNpwp       *string `json:"nik_npwp,omitempty" example:"01.234.567.8-901.000"`
	NomorWa       *string `json:"nomor_wa,omitempty" example:"+628123456789"`
	AlamatLengkap *string `json:"alamat_lengkap,omitempty" example:"Jl. Sudirman No. 1, Jakarta"`
}

// UpdateEntityMetaRequest represents entity meta update
// @Description Update entity administrative data
type UpdateEntityMetaRequest struct {
	BidangUsaha *string `json:"bidang_usaha,omitempty" example:"Konstruksi"`
	LogoURL     *string `json:"logo_url,omitempty" example:"https://r2.azzet.com/logos/abc.png"`
	Website     *string `json:"website,omitempty" example:"https://majujaya.co.id"`
	Email       *string `json:"email,omitempty" example:"info@majujaya.co.id"`
	Description *string `json:"description,omitempty" example:"Perusahaan konstruksi terkemuka"`
}

// --- Response DTOs ---

// EntityResponse represents an entity
// @Description Entity information
type EntityResponse struct {
	ID            string              `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID        *string             `json:"user_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	EntityType    string              `json:"entity_type" example:"BADAN_USAHA"`
	NamaUtama     string              `json:"nama_utama" example:"PT Maju Jaya"`
	NikNpwp       *string             `json:"nik_npwp,omitempty" example:"01.234.567.8-901.000"`
	NomorWa       *string             `json:"nomor_wa,omitempty" example:"+628123456789"`
	AlamatLengkap *string             `json:"alamat_lengkap,omitempty" example:"Jl. Sudirman No. 1"`
	IsShadow      bool                `json:"is_shadow" example:"false"`
	Status        string              `json:"status" example:"ACTIVE"`
	Meta          *EntityMetaResponse `json:"meta,omitempty"`
	CreatedAt     string              `json:"created_at" example:"2026-05-20T10:00:00Z"`
	UpdatedAt     string              `json:"updated_at" example:"2026-05-20T10:00:00Z"`
}

// EntityMetaResponse represents entity administrative data
// @Description Entity meta/administrative information
type EntityMetaResponse struct {
	BidangUsaha *string `json:"bidang_usaha,omitempty" example:"Konstruksi"`
	LogoURL     *string `json:"logo_url,omitempty" example:"https://r2.azzet.com/logos/abc.png"`
	Website     *string `json:"website,omitempty" example:"https://majujaya.co.id"`
	Email       *string `json:"email,omitempty" example:"info@majujaya.co.id"`
	Description *string `json:"description,omitempty" example:"Perusahaan konstruksi terkemuka"`
}

// MessageResponse represents a simple message
// @Description Simple message response
type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// --- Constants ---

const (
	TypeOrangPribadi = "ORANG_PRIBADI"
	TypeBadanUsaha   = "BADAN_USAHA"

	StatusActive   = "ACTIVE"
	StatusInactive = "INACTIVE"
	StatusClaimed  = "CLAIMED"
)

// --- Converters ---

func EntityToResponse(e *db.Entity) EntityResponse {
	resp := EntityResponse{
		ID:         e.ID.String(),
		EntityType: e.EntityType,
		NamaUtama:  e.NamaUtama,
		IsShadow:   e.IsShadow,
		Status:     e.Status,
		CreatedAt:  e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  e.UpdatedAt.Format(time.RFC3339),
	}

	if e.UserID.Valid {
		uid := uuid.UUID(e.UserID.Bytes)
		s := uid.String()
		resp.UserID = &s
	}
	if e.NikNpwp.Valid {
		resp.NikNpwp = &e.NikNpwp.String
	}
	if e.NomorWa.Valid {
		resp.NomorWa = &e.NomorWa.String
	}
	if e.AlamatLengkap.Valid {
		resp.AlamatLengkap = &e.AlamatLengkap.String
	}

	return resp
}

func EntityMetaToResponse(m *db.EntityMetum) *EntityMetaResponse {
	if m == nil {
		return nil
	}
	resp := &EntityMetaResponse{}
	if m.BidangUsaha.Valid {
		resp.BidangUsaha = &m.BidangUsaha.String
	}
	if m.LogoUrl.Valid {
		resp.LogoURL = &m.LogoUrl.String
	}
	if m.Website.Valid {
		resp.Website = &m.Website.String
	}
	if m.Email.Valid {
		resp.Email = &m.Email.String
	}
	if m.Description.Valid {
		resp.Description = &m.Description.String
	}
	return resp
}
