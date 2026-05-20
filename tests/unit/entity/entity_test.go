package entity_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/entity"
)

func TestEntityToResponse_OrangPribadi(t *testing.T) {
	uid := uuid.New()
	now := time.Now()

	e := &db.Entity{
		ID:            uuid.New(),
		UserID:        pgtype.UUID{Bytes: uid, Valid: true},
		EntityType:    entity.TypeOrangPribadi,
		NamaUtama:     "Jiilan Nashrulloh",
		NikNpwp:       pgtype.Text{String: "3201234567890001", Valid: true},
		NomorWa:       pgtype.Text{String: "+628123456789", Valid: true},
		AlamatLengkap: pgtype.Text{String: "Jl. Sudirman No. 1", Valid: true},
		IsShadow:      false,
		Status:        entity.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	resp := entity.EntityToResponse(e)

	if resp.ID != e.ID.String() {
		t.Fatalf("expected ID '%s', got '%s'", e.ID.String(), resp.ID)
	}
	if resp.UserID == nil || *resp.UserID != uid.String() {
		t.Fatalf("expected user_id '%s', got %v", uid.String(), resp.UserID)
	}
	if resp.EntityType != entity.TypeOrangPribadi {
		t.Fatalf("expected entity_type '%s', got '%s'", entity.TypeOrangPribadi, resp.EntityType)
	}
	if resp.NamaUtama != "Jiilan Nashrulloh" {
		t.Fatalf("expected nama_utama 'Jiilan Nashrulloh', got '%s'", resp.NamaUtama)
	}
	if resp.NikNpwp == nil || *resp.NikNpwp != "3201234567890001" {
		t.Fatalf("expected nik_npwp '3201234567890001', got %v", resp.NikNpwp)
	}
	if resp.NomorWa == nil || *resp.NomorWa != "+628123456789" {
		t.Fatalf("expected nomor_wa '+628123456789', got %v", resp.NomorWa)
	}
	if resp.AlamatLengkap == nil || *resp.AlamatLengkap != "Jl. Sudirman No. 1" {
		t.Fatalf("expected alamat_lengkap, got %v", resp.AlamatLengkap)
	}
	if resp.IsShadow {
		t.Fatal("expected is_shadow false")
	}
	if resp.Status != entity.StatusActive {
		t.Fatalf("expected status '%s', got '%s'", entity.StatusActive, resp.Status)
	}
}

func TestEntityToResponse_ShadowEntity(t *testing.T) {
	now := time.Now()

	e := &db.Entity{
		ID:         uuid.New(),
		UserID:     pgtype.UUID{Valid: false}, // Shadow = no user
		EntityType: entity.TypeBadanUsaha,
		NamaUtama:  "Toko Maju",
		NikNpwp:    pgtype.Text{Valid: false},
		NomorWa:    pgtype.Text{Valid: false},
		IsShadow:   true,
		Status:     entity.StatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	resp := entity.EntityToResponse(e)

	if resp.UserID != nil {
		t.Fatalf("expected nil user_id for shadow entity, got %v", resp.UserID)
	}
	if !resp.IsShadow {
		t.Fatal("expected is_shadow true")
	}
	if resp.NikNpwp != nil {
		t.Fatalf("expected nil nik_npwp, got %v", resp.NikNpwp)
	}
	if resp.NomorWa != nil {
		t.Fatalf("expected nil nomor_wa, got %v", resp.NomorWa)
	}
}

func TestEntityMetaToResponse(t *testing.T) {
	now := time.Now()

	meta := &db.EntityMetum{
		ID:          uuid.New(),
		EntityID:    uuid.New(),
		BidangUsaha: pgtype.Text{String: "Konstruksi", Valid: true},
		LogoUrl:     pgtype.Text{String: "https://r2.azzet.com/logo.png", Valid: true},
		Website:     pgtype.Text{String: "https://majujaya.co.id", Valid: true},
		Email:       pgtype.Text{String: "info@majujaya.co.id", Valid: true},
		Description: pgtype.Text{String: "Perusahaan konstruksi", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resp := entity.EntityMetaToResponse(meta)

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.BidangUsaha == nil || *resp.BidangUsaha != "Konstruksi" {
		t.Fatalf("expected bidang_usaha 'Konstruksi', got %v", resp.BidangUsaha)
	}
	if resp.LogoURL == nil || *resp.LogoURL != "https://r2.azzet.com/logo.png" {
		t.Fatalf("expected logo_url, got %v", resp.LogoURL)
	}
	if resp.Website == nil || *resp.Website != "https://majujaya.co.id" {
		t.Fatalf("expected website, got %v", resp.Website)
	}
	if resp.Email == nil || *resp.Email != "info@majujaya.co.id" {
		t.Fatalf("expected email, got %v", resp.Email)
	}
	if resp.Description == nil || *resp.Description != "Perusahaan konstruksi" {
		t.Fatalf("expected description, got %v", resp.Description)
	}
}

func TestEntityMetaToResponse_Nil(t *testing.T) {
	resp := entity.EntityMetaToResponse(nil)
	if resp != nil {
		t.Fatal("expected nil response for nil input")
	}
}

func TestEntityMetaToResponse_EmptyFields(t *testing.T) {
	meta := &db.EntityMetum{
		ID:          uuid.New(),
		EntityID:    uuid.New(),
		BidangUsaha: pgtype.Text{Valid: false},
		LogoUrl:     pgtype.Text{Valid: false},
		Website:     pgtype.Text{Valid: false},
		Email:       pgtype.Text{Valid: false},
		Description: pgtype.Text{Valid: false},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	resp := entity.EntityMetaToResponse(meta)

	if resp.BidangUsaha != nil {
		t.Fatalf("expected nil bidang_usaha, got %v", resp.BidangUsaha)
	}
	if resp.LogoURL != nil {
		t.Fatalf("expected nil logo_url, got %v", resp.LogoURL)
	}
}

func TestEntityConstants(t *testing.T) {
	if entity.TypeOrangPribadi != "ORANG_PRIBADI" {
		t.Fatalf("expected 'ORANG_PRIBADI', got '%s'", entity.TypeOrangPribadi)
	}
	if entity.TypeBadanUsaha != "BADAN_USAHA" {
		t.Fatalf("expected 'BADAN_USAHA', got '%s'", entity.TypeBadanUsaha)
	}
	if entity.StatusActive != "ACTIVE" {
		t.Fatalf("expected 'ACTIVE', got '%s'", entity.StatusActive)
	}
	if entity.StatusInactive != "INACTIVE" {
		t.Fatalf("expected 'INACTIVE', got '%s'", entity.StatusInactive)
	}
	if entity.StatusClaimed != "CLAIMED" {
		t.Fatalf("expected 'CLAIMED', got '%s'", entity.StatusClaimed)
	}
}
