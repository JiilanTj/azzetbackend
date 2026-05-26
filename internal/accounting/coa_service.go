package accounting

import (
	"context"
	"fmt"
	"time"

	"codeberg.org/azzet/azzetbe/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// COAService handles Chart of Accounts operations
type COAService struct {
	Queries *db.Queries
	Pool    *pgxpool.Pool
}

// NewCOAService creates a new COAService
func NewCOAService(queries *db.Queries, pool *pgxpool.Pool) *COAService {
	return &COAService{
		Queries: queries,
		Pool:    pool,
	}
}

// SeedDefaultCOA seeds the default Chart of Accounts for a workspace.
// Called when a workspace is created (via workspace.created event).
func (s *COAService) SeedDefaultCOA(ctx context.Context, workspaceID uuid.UUID) error {
	// Check if COA already exists for this workspace
	count, err := s.Queries.CountAccountsByWorkspace(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to check existing accounts: %w", err)
	}
	if count > 0 {
		return ErrCOAAlreadySeeded
	}

	template := DefaultCOATemplate()
	codeToID := make(COACodeToID)
	now := time.Now()

	// Use a transaction to seed all accounts atomically
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.Queries.WithTx(tx)

	for _, entry := range template {
		id := uuid.New()

		var parentID pgtype.UUID
		if entry.ParentCode != "" {
			pid, ok := codeToID[entry.ParentCode]
			if !ok {
				return fmt.Errorf("parent code %s not found for account %s", entry.ParentCode, entry.Code)
			}
			parentID = pgtype.UUID{Bytes: pid, Valid: true}
		}

		_, err := qtx.CreateAccount(ctx, db.CreateAccountParams{
			ID:            id,
			WorkspaceID:   workspaceID,
			ParentID:      parentID,
			Code:          entry.Code,
			Name:          entry.Name,
			AccountType:   entry.AccountType,
			NormalBalance: entry.NormalBalance,
			Level:         int32(entry.Level),
			IsSystem:      entry.IsSystem,
			IsActive:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
		if err != nil {
			return fmt.Errorf("failed to create account %s: %w", entry.Code, err)
		}

		codeToID[entry.Code] = id
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit COA seed: %w", err)
	}

	return nil
}

// ListAccounts returns all active accounts for a workspace
func (s *COAService) ListAccounts(ctx context.Context, workspaceID uuid.UUID) ([]AccountResponse, error) {
	accounts, err := s.Queries.ListAccountsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	resp := make([]AccountResponse, 0, len(accounts))
	for _, a := range accounts {
		resp = append(resp, AccountToResponse(a))
	}
	return resp, nil
}

// ListAllAccounts returns all accounts (including inactive) for a workspace
func (s *COAService) ListAllAccounts(ctx context.Context, workspaceID uuid.UUID) ([]AccountResponse, error) {
	accounts, err := s.Queries.ListAllAccountsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list all accounts: %w", err)
	}

	resp := make([]AccountResponse, 0, len(accounts))
	for _, a := range accounts {
		resp = append(resp, AccountToResponse(a))
	}
	return resp, nil
}

// ListAccountsByType returns accounts filtered by type
func (s *COAService) ListAccountsByType(ctx context.Context, workspaceID uuid.UUID, accountType string) ([]AccountResponse, error) {
	// Validate account type
	valid := false
	for _, t := range ValidAccountTypes {
		if t == accountType {
			valid = true
			break
		}
	}
	if !valid {
		return nil, ErrInvalidAccountType
	}

	accounts, err := s.Queries.ListAccountsByType(ctx, db.ListAccountsByTypeParams{
		WorkspaceID: workspaceID,
		AccountType: accountType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts by type: %w", err)
	}

	resp := make([]AccountResponse, 0, len(accounts))
	for _, a := range accounts {
		resp = append(resp, AccountToResponse(a))
	}
	return resp, nil
}

// GetAccount returns a single account by ID
func (s *COAService) GetAccount(ctx context.Context, workspaceID, accountID uuid.UUID) (*AccountResponse, error) {
	account, err := s.Queries.GetAccountByID(ctx, db.GetAccountByIDParams{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, ErrAccountNotFound
	}

	resp := AccountToResponse(account)
	return &resp, nil
}

// CreateAccount creates a custom account in the workspace COA
func (s *COAService) CreateAccount(ctx context.Context, workspaceID uuid.UUID, req *CreateAccountRequest) (*AccountResponse, error) {
	// Validate account type
	valid := false
	for _, t := range ValidAccountTypes {
		if t == req.AccountType {
			valid = true
			break
		}
	}
	if !valid {
		return nil, ErrInvalidAccountType
	}

	// Determine normal balance from account type
	normalBalance := NormalBalanceDebit
	if req.AccountType == AccountTypeLiability || req.AccountType == AccountTypeEquity || req.AccountType == AccountTypeRevenue {
		normalBalance = NormalBalanceCredit
	}

	// Resolve parent
	var parentID pgtype.UUID
	level := int32(1)
	if req.ParentID != "" {
		pid, err := uuid.Parse(req.ParentID)
		if err != nil {
			return nil, ErrInvalidParentAccount
		}
		parent, err := s.Queries.GetAccountByID(ctx, db.GetAccountByIDParams{
			ID:          pid,
			WorkspaceID: workspaceID,
		})
		if err != nil {
			return nil, ErrInvalidParentAccount
		}
		parentID = pgtype.UUID{Bytes: parent.ID, Valid: true}
		level = parent.Level + 1
	}

	now := time.Now()
	account, err := s.Queries.CreateAccount(ctx, db.CreateAccountParams{
		ID:            uuid.New(),
		WorkspaceID:   workspaceID,
		ParentID:      parentID,
		Code:          req.Code,
		Name:          req.Name,
		AccountType:   req.AccountType,
		NormalBalance: normalBalance,
		Level:         level,
		IsSystem:      false,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return nil, ErrAccountCodeExists
	}

	resp := AccountToResponse(account)
	return &resp, nil
}

// UpdateAccount updates a non-system account
func (s *COAService) UpdateAccount(ctx context.Context, workspaceID, accountID uuid.UUID, req *UpdateAccountRequest) error {
	// Check account exists and is not system
	account, err := s.Queries.GetAccountByID(ctx, db.GetAccountByIDParams{
		ID:          accountID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return ErrAccountNotFound
	}
	if account.IsSystem {
		return ErrAccountIsSystem
	}

	// Resolve parent
	var parentID pgtype.UUID
	if req.ParentID != "" {
		pid, err := uuid.Parse(req.ParentID)
		if err != nil {
			return ErrInvalidParentAccount
		}
		parentID = pgtype.UUID{Bytes: pid, Valid: true}
	}

	name := account.Name
	if req.Name != "" {
		name = req.Name
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	return s.Queries.UpdateAccount(ctx, db.UpdateAccountParams{
		ID:          accountID,
		WorkspaceID: workspaceID,
		Name:        name,
		ParentID:    parentID,
		IsActive:    isActive,
	})
}
