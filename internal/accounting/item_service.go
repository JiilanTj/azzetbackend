package accounting

import (
	"context"
	"fmt"
	"time"

	"codeberg.org/azzet/azzetbe/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ItemService handles item/product CRUD operations
type ItemService struct {
	Queries *db.Queries
}

// NewItemService creates a new ItemService
func NewItemService(queries *db.Queries) *ItemService {
	return &ItemService{
		Queries: queries,
	}
}

// CreateItem creates a new item in the workspace
func (s *ItemService) CreateItem(ctx context.Context, workspaceID uuid.UUID, req *CreateItemRequest) (*ItemResponse, error) {
	// Validate item type
	if !isValidItemType(req.ItemType) {
		return nil, ErrInvalidItemType
	}

	// Validate unit
	if !isValidUnit(req.Unit) {
		return nil, ErrInvalidUnit
	}

	// Resolve account_id if provided
	var accountID pgtype.UUID
	if req.AccountID != "" {
		aid, err := uuid.Parse(req.AccountID)
		if err != nil {
			return nil, fmt.Errorf("invalid account_id: %w", err)
		}
		accountID = pgtype.UUID{Bytes: aid, Valid: true}
	}

	// Description
	var description pgtype.Text
	if req.Description != "" {
		description = pgtype.Text{String: req.Description, Valid: true}
	}

	now := time.Now()
	item, err := s.Queries.CreateItem(ctx, db.CreateItemParams{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        req.Name,
		ItemType:    req.ItemType,
		Unit:        req.Unit,
		UnitPrice:   req.UnitPrice,
		AccountID:   accountID,
		Description: description,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	resp := ItemToResponse(item)
	return &resp, nil
}

// GetItem returns a single item by ID
func (s *ItemService) GetItem(ctx context.Context, workspaceID, itemID uuid.UUID) (*ItemResponse, error) {
	item, err := s.Queries.GetItemByID(ctx, db.GetItemByIDParams{
		ID:          itemID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, ErrItemNotFound
	}

	resp := ItemToResponse(item)
	return &resp, nil
}

// ListItems returns all active items for a workspace with pagination
func (s *ItemService) ListItems(ctx context.Context, workspaceID uuid.UUID, limit, offset int32) ([]ItemResponse, error) {
	items, err := s.Queries.ListItemsByWorkspace(ctx, db.ListItemsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	resp := make([]ItemResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, ItemToResponse(item))
	}
	return resp, nil
}

// ListItemsByType returns items filtered by type
func (s *ItemService) ListItemsByType(ctx context.Context, workspaceID uuid.UUID, itemType string, limit, offset int32) ([]ItemResponse, error) {
	if !isValidItemType(itemType) {
		return nil, ErrInvalidItemType
	}

	items, err := s.Queries.ListItemsByType(ctx, db.ListItemsByTypeParams{
		WorkspaceID: workspaceID,
		ItemType:    itemType,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list items by type: %w", err)
	}

	resp := make([]ItemResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, ItemToResponse(item))
	}
	return resp, nil
}

// UpdateItem updates an existing item
func (s *ItemService) UpdateItem(ctx context.Context, workspaceID, itemID uuid.UUID, req *UpdateItemRequest) error {
	// Check item exists
	existing, err := s.Queries.GetItemByID(ctx, db.GetItemByIDParams{
		ID:          itemID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return ErrItemNotFound
	}

	// Apply updates with fallback to existing values
	name := existing.Name
	if req.Name != "" {
		name = req.Name
	}

	itemType := existing.ItemType
	if req.ItemType != "" {
		if !isValidItemType(req.ItemType) {
			return ErrInvalidItemType
		}
		itemType = req.ItemType
	}

	unit := existing.Unit
	if req.Unit != "" {
		if !isValidUnit(req.Unit) {
			return ErrInvalidUnit
		}
		unit = req.Unit
	}

	unitPrice := existing.UnitPrice
	if req.UnitPrice != nil {
		unitPrice = *req.UnitPrice
	}

	accountID := existing.AccountID
	if req.AccountID != nil {
		if *req.AccountID == "" {
			accountID = pgtype.UUID{}
		} else {
			aid, err := uuid.Parse(*req.AccountID)
			if err != nil {
				return fmt.Errorf("invalid account_id: %w", err)
			}
			accountID = pgtype.UUID{Bytes: aid, Valid: true}
		}
	}

	description := existing.Description
	if req.Description != nil {
		if *req.Description == "" {
			description = pgtype.Text{}
		} else {
			description = pgtype.Text{String: *req.Description, Valid: true}
		}
	}

	return s.Queries.UpdateItem(ctx, db.UpdateItemParams{
		ID:          itemID,
		WorkspaceID: workspaceID,
		Name:        name,
		ItemType:    itemType,
		Unit:        unit,
		UnitPrice:   unitPrice,
		AccountID:   accountID,
		Description: description,
	})
}

// SoftDeleteItem marks an item as inactive
func (s *ItemService) SoftDeleteItem(ctx context.Context, workspaceID, itemID uuid.UUID) error {
	// Check item exists
	_, err := s.Queries.GetItemByID(ctx, db.GetItemByIDParams{
		ID:          itemID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return ErrItemNotFound
	}

	return s.Queries.SoftDeleteItem(ctx, db.SoftDeleteItemParams{
		ID:          itemID,
		WorkspaceID: workspaceID,
	})
}

// --- Helpers ---

func isValidItemType(t string) bool {
	for _, v := range ValidItemTypes {
		if v == t {
			return true
		}
	}
	return false
}

func isValidUnit(u string) bool {
	for _, v := range ValidUnits {
		if v == u {
			return true
		}
	}
	return false
}
