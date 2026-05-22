package accounting

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LedgerWorker handles async ledger posting from NATS events
type LedgerWorker struct {
	Queries *db.Queries
	Pool    *pgxpool.Pool
}

// NewLedgerWorker creates a new LedgerWorker
func NewLedgerWorker(queries *db.Queries, pool *pgxpool.Pool) *LedgerWorker {
	return &LedgerWorker{
		Queries: queries,
		Pool:    pool,
	}
}

// ledgerEventPayload is the expected payload from accounting.transaction.created event
type ledgerEventPayload struct {
	TransactionID string `json:"transaction_id"`
	WorkspaceID   string `json:"workspace_id"`
	IsReversal    string `json:"is_reversal,omitempty"`
}

// HandleTransactionCreated processes a transaction for ledger posting.
// Flow:
// 1. Parse payload
// 2. Fetch transaction + journal entries
// 3. Validate sum(debit) == sum(credit)
// 4. For each journal entry: calculate running_balance, insert ledger_entry
// 5. Upsert account_balances (period summary)
// 6. Mark transaction as POSTED
// 7. Validate accounting equation (A = L + E)
// 8. Emit accounting.ledger.posted event
func (w *LedgerWorker) HandleTransactionCreated(ctx context.Context, event *events.Event) error {
	// 1. Parse payload
	var payload ledgerEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	txID, err := uuid.Parse(payload.TransactionID)
	if err != nil {
		return fmt.Errorf("invalid transaction_id: %w", err)
	}

	workspaceID, err := uuid.Parse(payload.WorkspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace_id: %w", err)
	}

	slog.Info("ledger worker: processing transaction",
		"transaction_id", txID.String(),
		"workspace_id", workspaceID.String(),
	)

	// 2. Fetch transaction
	transaction, err := w.Queries.GetTransactionByID(ctx, db.GetTransactionByIDParams{
		ID:          txID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("transaction not found: %w", err)
	}

	// Only process DRAFT transactions
	if transaction.Status != TxStatusDraft {
		slog.Info("ledger worker: skipping non-draft transaction",
			"transaction_id", txID.String(),
			"status", transaction.Status,
		)
		return nil
	}

	// Fetch journal entries
	journals, err := w.Queries.ListJournalEntriesByTransaction(ctx, db.ListJournalEntriesByTransactionParams{
		TransactionID: txID,
		WorkspaceID:   workspaceID,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch journal entries: %w", err)
	}

	if len(journals) == 0 {
		// Mark as failed
		_ = w.Queries.MarkTransactionFailed(ctx, db.MarkTransactionFailedParams{
			ID:          txID,
			WorkspaceID: workspaceID,
		})
		return fmt.Errorf("no journal entries for transaction %s", txID.String())
	}

	// 3. Validate debit = credit
	sums, err := w.Queries.GetJournalSumByTransaction(ctx, db.GetJournalSumByTransactionParams{
		TransactionID: txID,
		WorkspaceID:   workspaceID,
	})
	if err != nil {
		return fmt.Errorf("failed to get journal sums: %w", err)
	}

	totalDebit := interfaceToFloat(sums.TotalDebit)
	totalCredit := interfaceToFloat(sums.TotalCredit)

	if totalDebit != totalCredit {
		_ = w.Queries.MarkTransactionFailed(ctx, db.MarkTransactionFailedParams{
			ID:          txID,
			WorkspaceID: workspaceID,
		})
		return fmt.Errorf("debit/credit mismatch: debit=%.2f credit=%.2f", totalDebit, totalCredit)
	}

	// Begin DB transaction for atomic posting
	dbTx, err := w.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback(ctx)

	qtx := w.Queries.WithTx(dbTx)
	now := time.Now()
	period := transaction.TransactionDate.Time.Format("2006-01")

	// 4. Create ledger entries with running balance
	for _, journal := range journals {
		// Get current running balance for this account
		lastBalance, err := qtx.GetLastLedgerBalance(ctx, db.GetLastLedgerBalanceParams{
			WorkspaceID: workspaceID,
			AccountID:   journal.AccountID,
		})

		var currentBalance float64
		if err == nil {
			currentBalance = numericToFloat(lastBalance)
		}
		// If error (no previous entries), currentBalance stays 0

		// Calculate new running balance based on normal balance
		debit := numericToFloat(journal.Debit)
		credit := numericToFloat(journal.Credit)

		// Running balance: for DEBIT-normal accounts, debit increases balance
		// For CREDIT-normal accounts, credit increases balance
		// We store the absolute effect: debit - credit (positive = debit side)
		newBalance := currentBalance + debit - credit

		_, err = qtx.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			ID:              uuid.New(),
			WorkspaceID:     workspaceID,
			AccountID:       journal.AccountID,
			JournalEntryID:  journal.ID,
			TransactionID:   txID,
			TransactionDate: transaction.TransactionDate,
			Debit:           journal.Debit,
			Credit:          journal.Credit,
			RunningBalance:  floatToNumeric(newBalance),
			PostedAt:        now,
		})
		if err != nil {
			return fmt.Errorf("failed to create ledger entry: %w", err)
		}

		// 5. Upsert account balance for this period
		balanceChange := debit - credit
		_, err = qtx.UpsertAccountBalance(ctx, db.UpsertAccountBalanceParams{
			WorkspaceID:   workspaceID,
			AccountID:     journal.AccountID,
			Period:        period,
			TotalDebit:    floatToNumeric(debit),
			TotalCredit:   floatToNumeric(credit),
			EndingBalance: floatToNumeric(balanceChange),
		})
		if err != nil {
			return fmt.Errorf("failed to upsert account balance: %w", err)
		}
	}

	// 6. Mark transaction as POSTED
	err = qtx.MarkTransactionPosted(ctx, db.MarkTransactionPostedParams{
		ID:          txID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("failed to mark transaction posted: %w", err)
	}

	// 7. Validate accounting equation (A = L + E)
	// This is a soft check - log warning but don't fail the posting
	if err := w.validateAccountingEquation(ctx, qtx, workspaceID); err != nil {
		slog.Warn("accounting equation check failed",
			"workspace_id", workspaceID.String(),
			"error", err,
		)
	}

	// 8. Emit ledger.posted event
	err = events.EmitEvent(ctx, dbTx, events.LedgerPosted, map[string]string{
		"transaction_id": txID.String(),
		"workspace_id":   workspaceID.String(),
	},
		events.WithWorkspace(workspaceID.String()),
	)
	if err != nil {
		return fmt.Errorf("failed to emit ledger.posted event: %w", err)
	}

	// Commit
	if err := dbTx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit ledger posting: %w", err)
	}

	slog.Info("ledger worker: transaction posted successfully",
		"transaction_id", txID.String(),
		"workspace_id", workspaceID.String(),
		"journal_entries", len(journals),
	)

	return nil
}

// validateAccountingEquation checks that Assets = Liabilities + Equity
// Uses account_balances table for efficiency
func (w *LedgerWorker) validateAccountingEquation(ctx context.Context, qtx *db.Queries, workspaceID uuid.UUID) error {
	// Get all account balances (all periods summed)
	accounts, err := qtx.ListAccountsByWorkspace(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	// We need to sum all ledger entries by account type
	// For a quick check, we sum the running balances of the latest ledger entries per account
	var totalAssets, totalLiabilities, totalEquity float64

	for _, acc := range accounts {
		lastBalance, err := qtx.GetLastLedgerBalance(ctx, db.GetLastLedgerBalanceParams{
			WorkspaceID: workspaceID,
			AccountID:   acc.ID,
		})
		if err != nil {
			continue // No entries for this account
		}

		balance := numericToFloat(lastBalance)

		switch acc.AccountType {
		case AccountTypeAsset:
			totalAssets += balance
		case AccountTypeLiability:
			totalLiabilities += balance
		case AccountTypeEquity:
			totalEquity += balance
		case AccountTypeRevenue:
			// Revenue increases equity (credit normal)
			totalEquity -= balance // balance is debit-credit, revenue has negative balance (credit > debit)
		case AccountTypeExpense:
			// Expense decreases equity (debit normal)
			totalEquity -= balance // balance is positive for expenses (debit > credit), reduces equity
		}
	}

	// A = L + E (with tolerance for floating point)
	diff := totalAssets - (totalLiabilities + totalEquity)
	if diff > 0.01 || diff < -0.01 {
		return fmt.Errorf("A=%.2f, L+E=%.2f, diff=%.2f", totalAssets, totalLiabilities+totalEquity, diff)
	}

	return nil
}
