package claim

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/events"
	"codeberg.org/azzet/azzetbe/internal/identity"
)

type ClaimWorker struct {
	Queries         *db.Queries
	Pool            *pgxpool.Pool
	IdentityService *identity.Service
}

func NewClaimWorker(queries *db.Queries, pool *pgxpool.Pool, identityService *identity.Service) *ClaimWorker {
	return &ClaimWorker{
		Queries:         queries,
		Pool:            pool,
		IdentityService: identityService,
	}
}

func (w *ClaimWorker) HandleClaimRequested(ctx context.Context, event *events.Event) error {
	var payload struct {
		ClaimID  string `json:"claim_id"`
		EntityID string `json:"entity_id"`
		UserID   string `json:"user_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse claim_requested payload: %w", err)
	}

	duplicates, err := w.IdentityService.FindDuplicates(ctx, payload.EntityID, 5)
	if err != nil {
		return fmt.Errorf("failed to check duplicates: %w", err)
	}

	for _, dup := range duplicates {
		if dup.MatchScore >= 0.8 {
			cid, err := uuid.Parse(payload.ClaimID)
			if err != nil {
				continue
			}
			claim, err := w.Queries.GetCompanyClaimByID(ctx, cid)
			if err != nil {
				continue
			}
			if claim.Status == StatusSubmitted {
				_ = w.Queries.UpdateClaimStatus(ctx, db.UpdateClaimStatusParams{
					ID:     claim.ID,
					Status: StatusUnderReview,
				})
			}
			break
		}
	}

	return nil
}

func (w *ClaimWorker) HandleClaimApproved(ctx context.Context, event *events.Event) error {
	var payload struct {
		ClaimID        string `json:"claim_id"`
		EntityID       string `json:"entity_id"`
		ClaimantUserID string `json:"claimant_user_id"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse claim_approved payload: %w", err)
	}

	eid, err := uuid.Parse(payload.EntityID)
	if err != nil {
		return fmt.Errorf("invalid entity_id: %w", err)
	}

	e, err := w.Queries.GetEntityByID(ctx, eid)
	if err != nil {
		return fmt.Errorf("entity not found: %w", err)
	}

	if err := w.IdentityService.EnsureNormalizedName(ctx, eid, e.NamaUtama); err != nil {
		return fmt.Errorf("failed to normalize name: %w", err)
	}

	return nil
}
