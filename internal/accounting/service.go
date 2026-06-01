package accounting

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles transaction operations
type Service struct {
	Queries     *db.Queries
	Pool        *pgxpool.Pool
	COAService  *COAService
	ItemService *ItemService
	Categorizer *Categorizer
}

// NewService creates a new accounting Service
func NewService(queries *db.Queries, pool *pgxpool.Pool, coaService *COAService, itemService *ItemService, categorizer *Categorizer) *Service {
	return &Service{
		Queries:     queries,
		Pool:        pool,
		COAService:  coaService,
		ItemService: itemService,
		Categorizer: categorizer,
	}
}

// CreateTransaction creates a new transaction with journal entries.
// For SIMPLE mode: uses rule engine to auto-generate journal entries.
// For ADVANCED mode: uses user-provided journal entries.
func (s *Service) CreateTransaction(ctx context.Context, workspaceID, userID uuid.UUID, req *CreateTransactionRequest) (*TransactionResponse, error) {
	// Validate transaction type
	if !isValidTxType(req.TransactionType) {
		return nil, ErrInvalidTransactionType
	}

	// Validate input mode
	inputMode := req.InputMode
	if inputMode == "" {
		inputMode = InputModeSimple
	}
	if !isValidInputMode(inputMode) {
		return nil, ErrInvalidInputMode
	}

	// Validate payment method if provided
	if req.PaymentMethod != "" && !isValidPaymentMethod(req.PaymentMethod) {
		return nil, ErrInvalidPaymentMethod
	}

	// Parse transaction date
	txDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction_date format, use YYYY-MM-DD: %w", err)
	}

	// Validate amount
	amountFloat := numericToFloat(req.Amount)
	if amountFloat <= 0 {
		return nil, ErrAmountMustBePositive
	}

	// Begin transaction
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.Queries.WithTx(tx)

	// Generate transaction number
	nextNum, err := qtx.GetNextTransactionNumber(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next transaction number: %w", err)
	}
	txNumber := fmt.Sprintf("TXN-%06d", nextNum)

	// Resolve counterparty
	var counterpartyEntityID uuid.UUID
	if req.CounterpartyEntityID != "" {
		cid, err := uuid.Parse(req.CounterpartyEntityID)
		if err != nil {
			return nil, fmt.Errorf("invalid counterparty_entity_id: %w", err)
		}
		counterpartyEntityID = cid
	}

	// Resolve category (AI or user-provided)
	category := req.Category
	var aiConfidence pgtype.Numeric
	if inputMode == InputModeSimple && category == "" && req.TransactionType != TxTypeJournal {
		// Use AI categorization
		result, err := s.Categorizer.Categorize(ctx, req.TransactionType, req.Description, amountFloat)
		if err != nil {
			slog.Warn("categorization error, using fallback", "error", err)
			category = GetFallbackCategory(req.TransactionType)
		} else {
			category = result.Category
			aiConfidence = floatToNumeric(result.Confidence)
		}
	}

	// Validate category for type (if not JOURNAL or ADVANCED mode)
	if inputMode == InputModeSimple && req.TransactionType != TxTypeJournal {
		if !IsValidCategoryForType(category, req.TransactionType) {
			return nil, ErrInvalidCategory
		}
	}

	// Calculate tax if applicable
	taxAmount := pgtype.Numeric{Valid: true, Int: nil}
	if req.IncludesTax {
		taxFloat := amountFloat * PPNRate
		taxAmount = floatToNumeric(taxFloat)
	}

	now := time.Now()

	// Create transaction record
	dbTx, err := qtx.CreateTransaction(ctx, db.CreateTransactionParams{
		ID:                     uuid.New(),
		WorkspaceID:            workspaceID,
		TransactionNumber:      txNumber,
		TransactionType:        req.TransactionType,
		InputMode:              inputMode,
		Status:                 TxStatusDraft,
		CounterpartyEntityID:   uuidToPgtype(counterpartyEntityID),
		CounterpartyName:       stringToPgtext(req.CounterpartyName),
		Description:            stringToPgtext(req.Description),
		TransactionDate:        pgtype.Date{Time: txDate, Valid: true},
		Amount:                 req.Amount,
		Currency:               "IDR",
		Category:               stringToPgtext(category),
		AiConfidence:           aiConfidence,
		PaymentMethod:          stringToPgtext(req.PaymentMethod),
		IncludesTax:            req.IncludesTax,
		TaxAmount:              taxAmount,
		ReversedTransactionID:  pgtype.UUID{},
		CreatedBy:              userID,
		CreatedAt:              now,
		UpdatedAt:              now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Create line items (for SALES/PURCHASE)
	if len(req.LineItems) > 0 {
		for i, li := range req.LineItems {
			var itemID pgtype.UUID
			if li.ItemID != "" {
				parsed, _ := uuid.Parse(li.ItemID)
				itemID = pgtype.UUID{Bytes: parsed, Valid: true}
			}

			unit := li.Unit
			if unit == "" {
				unit = "Pcs"
			}

			lineTotal := li.Quantity * numericToFloat(li.UnitPrice) - numericToFloat(li.DiscountAmount)

			_, err := qtx.CreateTransactionLineItem(ctx, db.CreateTransactionLineItemParams{
				ID:             uuid.New(),
				TransactionID:  dbTx.ID,
				WorkspaceID:    workspaceID,
				ItemID:         itemID,
				Description:    li.Description,
				Quantity:       floatToNumeric4(li.Quantity),
				Unit:           unit,
				UnitPrice:      li.UnitPrice,
				DiscountAmount: li.DiscountAmount,
				TaxAmount:      floatToNumeric(0),
				LineTotal:      floatToNumeric(lineTotal),
				SortOrder:      int32(i),
				CreatedAt:      now,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create line item: %w", err)
			}
		}
	}

	// Create journal entries
	if inputMode == InputModeAdvanced && len(req.JournalEntries) > 0 {
		// ADVANCED mode: user provides journal entries directly
		if err := s.createManualJournalEntries(ctx, qtx, dbTx.ID, workspaceID, req.JournalEntries, now); err != nil {
			return nil, err
		}
	} else if req.TransactionType != TxTypeJournal {
		// SIMPLE mode: auto-generate from rule engine
		if err := s.createAutoJournalEntries(ctx, qtx, dbTx.ID, workspaceID, req.TransactionType, category, req.Amount, req.IncludesTax, now); err != nil {
			return nil, err
		}
	}

	// Validate journal entries balance
	sums, err := qtx.GetJournalSumByTransaction(ctx, db.GetJournalSumByTransactionParams{
		TransactionID: dbTx.ID,
		WorkspaceID:   workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to validate journal: %w", err)
	}
	if interfaceToFloat(sums.TotalDebit) != interfaceToFloat(sums.TotalCredit) {
		return nil, ErrDebitCreditMismatch
	}

	// Emit event for async ledger posting
	err = events.EmitEvent(ctx, tx, events.TransactionCreated, map[string]string{
		"transaction_id": dbTx.ID.String(),
		"workspace_id":   workspaceID.String(),
	},
		events.WithWorkspace(workspaceID.String()),
		events.WithActor(userID.String()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to emit event: %w", err)
	}

	// Commit
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	// Build response
	resp := transactionToResponse(dbTx)
	return &resp, nil
}

// GetTransaction returns a transaction with its journal entries and line items
func (s *Service) GetTransaction(ctx context.Context, workspaceID, txID uuid.UUID) (*TransactionResponse, error) {
	dbTx, err := s.Queries.GetTransactionByID(ctx, db.GetTransactionByIDParams{
		ID:          txID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, ErrTransactionNotFound
	}

	resp := transactionToResponse(dbTx)

	// Load journal entries
	journals, err := s.Queries.ListJournalEntriesByTransaction(ctx, db.ListJournalEntriesByTransactionParams{
		TransactionID: txID,
		WorkspaceID:   workspaceID,
	})
	if err == nil {
		resp.JournalEntries = make([]JournalEntryResponse, 0, len(journals))
		for _, j := range journals {
			resp.JournalEntries = append(resp.JournalEntries, JournalEntryResponse{
				ID:          j.ID.String(),
				AccountID:   j.AccountID.String(),
				AccountCode: j.AccountCode,
				AccountName: j.AccountName,
				Description: pgtextToString(j.Description),
				Debit:       numericToString(j.Debit),
				Credit:      numericToString(j.Credit),
				SortOrder:   int(j.SortOrder),
			})
		}
	}

	// Load line items
	lineItems, err := s.Queries.ListLineItemsByTransaction(ctx, db.ListLineItemsByTransactionParams{
		TransactionID: txID,
		WorkspaceID:   workspaceID,
	})
	if err == nil && len(lineItems) > 0 {
		resp.LineItems = make([]LineItemResponse, 0, len(lineItems))
		for _, li := range lineItems {
			itemID := ""
			if li.ItemID.Valid {
				itemID = pgtypeUUIDToString(li.ItemID)
			}
			resp.LineItems = append(resp.LineItems, LineItemResponse{
				ID:             li.ID.String(),
				ItemID:         itemID,
				Description:    li.Description,
				Quantity:       numericToString4(li.Quantity),
				Unit:           li.Unit,
				UnitPrice:      numericToString(li.UnitPrice),
				DiscountAmount: numericToString(li.DiscountAmount),
				TaxAmount:      numericToString(li.TaxAmount),
				LineTotal:      numericToString(li.LineTotal),
				SortOrder:      int(li.SortOrder),
			})
		}
	}

	return &resp, nil
}

// ListTransactions returns transactions for a workspace with pagination
func (s *Service) ListTransactions(ctx context.Context, workspaceID uuid.UUID, limit, offset int32) ([]TransactionResponse, error) {
	txs, err := s.Queries.ListTransactionsByWorkspace(ctx, db.ListTransactionsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	resp := make([]TransactionResponse, 0, len(txs))
	for _, t := range txs {
		resp = append(resp, transactionToResponse(t))
	}
	return resp, nil
}

// VoidTransaction creates a reversal (jurnal pembalik) for a posted transaction.
// The original transaction is marked VOID, and a new REVERSAL transaction is created
// with swapped debit/credit entries.
func (s *Service) VoidTransaction(ctx context.Context, workspaceID, txID, userID uuid.UUID) (*TransactionResponse, error) {
	// Get original transaction
	original, err := s.Queries.GetTransactionByID(ctx, db.GetTransactionByIDParams{
		ID:          txID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, ErrTransactionNotFound
	}

	if original.Status == TxStatusVoid {
		return nil, ErrTransactionAlreadyVoid
	}
	if original.Status != TxStatusPosted {
		return nil, ErrTransactionNotPosted
	}

	// Get original journal entries
	journals, err := s.Queries.ListJournalEntriesByTransaction(ctx, db.ListJournalEntriesByTransactionParams{
		TransactionID: txID,
		WorkspaceID:   workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get journal entries: %w", err)
	}

	// Begin transaction
	dbTxn, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTxn.Rollback(ctx)

	qtx := s.Queries.WithTx(dbTxn)

	// Generate new transaction number
	nextNum, err := qtx.GetNextTransactionNumber(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next number: %w", err)
	}
	txNumber := fmt.Sprintf("TXN-%06d", nextNum)

	now := time.Now()

	// Create reversal transaction
	reversal, err := qtx.CreateTransaction(ctx, db.CreateTransactionParams{
		ID:                    uuid.New(),
		WorkspaceID:           workspaceID,
		TransactionNumber:     txNumber,
		TransactionType:       TxTypeReversal,
		InputMode:             original.InputMode,
		Status:                TxStatusDraft,
		CounterpartyEntityID:  original.CounterpartyEntityID,
		CounterpartyName:      original.CounterpartyName,
		Description:           stringToPgtext(fmt.Sprintf("Jurnal Pembalik: %s", original.TransactionNumber)),
		TransactionDate:       pgtype.Date{Time: now, Valid: true},
		Amount:                original.Amount,
		Currency:              "IDR",
		Category:              original.Category,
		AiConfidence:          pgtype.Numeric{},
		PaymentMethod:         original.PaymentMethod,
		IncludesTax:           original.IncludesTax,
		TaxAmount:             original.TaxAmount,
		ReversedTransactionID: pgtype.UUID{Bytes: original.ID, Valid: true},
		CreatedBy:             userID,
		CreatedAt:             now,
		UpdatedAt:             now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create reversal: %w", err)
	}

	// Create reversed journal entries (swap debit ↔ credit)
	for i, j := range journals {
		_, err := qtx.CreateJournalEntry(ctx, db.CreateJournalEntryParams{
			ID:            uuid.New(),
			TransactionID: reversal.ID,
			WorkspaceID:   workspaceID,
			AccountID:     j.AccountID,
			Description:   stringToPgtext(fmt.Sprintf("Pembalik: %s", pgtextToString(j.Description))),
			Debit:         j.Credit, // SWAP: original credit becomes debit
			Credit:        j.Debit,  // SWAP: original debit becomes credit
			SortOrder:     int32(i),
			CreatedAt:     now,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create reversal journal entry: %w", err)
		}
	}

	// Mark original as VOID
	err = qtx.MarkTransactionVoid(ctx, db.MarkTransactionVoidParams{
		ID:          txID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to void original: %w", err)
	}

	// Emit event for reversal posting
	err = events.EmitEvent(ctx, dbTxn, events.TransactionCreated, map[string]string{
		"transaction_id": reversal.ID.String(),
		"workspace_id":   workspaceID.String(),
		"is_reversal":    "true",
	},
		events.WithWorkspace(workspaceID.String()),
		events.WithActor(userID.String()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to emit event: %w", err)
	}

	if err := dbTxn.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	resp := transactionToResponse(reversal)
	return &resp, nil
}

// Categorize exposes AI categorization as a standalone endpoint
func (s *Service) Categorize(ctx context.Context, req *CategorizationRequest) (*CategorizationResult, error) {
	return s.Categorizer.Categorize(ctx, req.TransactionType, req.Description, req.Amount)
}

// --- Internal helpers ---

func (s *Service) createAutoJournalEntries(ctx context.Context, qtx *db.Queries, txID, workspaceID uuid.UUID, txType, category string, amount pgtype.Numeric, includesTax bool, now time.Time) error {
	rule := GetJournalRule(txType, category)
	if rule == nil {
		return ErrInvalidCategory
	}

	amountFloat := numericToFloat(amount)
	sortOrder := int32(0)

	// Primary journal entry
	debitAccount, err := qtx.GetAccountByCode(ctx, db.GetAccountByCodeParams{
		WorkspaceID: workspaceID,
		Code:        rule.Primary.DebitCode,
	})
	if err != nil {
		return fmt.Errorf("debit account %s not found: %w", rule.Primary.DebitCode, err)
	}

	creditAccount, err := qtx.GetAccountByCode(ctx, db.GetAccountByCodeParams{
		WorkspaceID: workspaceID,
		Code:        rule.Primary.CreditCode,
	})
	if err != nil {
		return fmt.Errorf("credit account %s not found: %w", rule.Primary.CreditCode, err)
	}

	// If includes tax, the base amount is amount / (1 + PPNRate)
	baseAmount := amountFloat
	taxAmount := 0.0
	if includesTax {
		baseAmount = amountFloat / (1 + PPNRate)
		taxAmount = amountFloat - baseAmount
	}

	_, err = qtx.CreateJournalEntry(ctx, db.CreateJournalEntryParams{
		ID:            uuid.New(),
		TransactionID: txID,
		WorkspaceID:   workspaceID,
		AccountID:     debitAccount.ID,
		Description:   stringToPgtext(debitAccount.Name),
		Debit:         floatToNumeric(baseAmount),
		Credit:        floatToNumeric(0),
		SortOrder:     sortOrder,
		CreatedAt:     now,
	})
	if err != nil {
		return fmt.Errorf("failed to create debit entry: %w", err)
	}
	sortOrder++

	_, err = qtx.CreateJournalEntry(ctx, db.CreateJournalEntryParams{
		ID:            uuid.New(),
		TransactionID: txID,
		WorkspaceID:   workspaceID,
		AccountID:     creditAccount.ID,
		Description:   stringToPgtext(creditAccount.Name),
		Debit:         floatToNumeric(0),
		Credit:        floatToNumeric(baseAmount),
		SortOrder:     sortOrder,
		CreatedAt:     now,
	})
	if err != nil {
		return fmt.Errorf("failed to create credit entry: %w", err)
	}
	sortOrder++

	// Additional entries (HPP, PPN, etc.)
	for _, additional := range rule.Additional {
		addDebitAcc, err := qtx.GetAccountByCode(ctx, db.GetAccountByCodeParams{
			WorkspaceID: workspaceID,
			Code:        additional.DebitCode,
		})
		if err != nil {
			return fmt.Errorf("additional debit account %s not found: %w", additional.DebitCode, err)
		}

		addCreditAcc, err := qtx.GetAccountByCode(ctx, db.GetAccountByCodeParams{
			WorkspaceID: workspaceID,
			Code:        additional.CreditCode,
		})
		if err != nil {
			return fmt.Errorf("additional credit account %s not found: %w", additional.CreditCode, err)
		}

		// Determine amount for additional entry
		addAmount := baseAmount // HPP uses same base amount
		if includesTax && (additional.DebitCode == "1-1008" || additional.CreditCode == "2-1005") {
			addAmount = taxAmount // PPN entries use tax amount
		}

		_, err = qtx.CreateJournalEntry(ctx, db.CreateJournalEntryParams{
			ID:            uuid.New(),
			TransactionID: txID,
			WorkspaceID:   workspaceID,
			AccountID:     addDebitAcc.ID,
			Description:   stringToPgtext(addDebitAcc.Name),
			Debit:         floatToNumeric(addAmount),
			Credit:        floatToNumeric(0),
			SortOrder:     sortOrder,
			CreatedAt:     now,
		})
		if err != nil {
			return fmt.Errorf("failed to create additional debit: %w", err)
		}
		sortOrder++

		_, err = qtx.CreateJournalEntry(ctx, db.CreateJournalEntryParams{
			ID:            uuid.New(),
			TransactionID: txID,
			WorkspaceID:   workspaceID,
			AccountID:     addCreditAcc.ID,
			Description:   stringToPgtext(addCreditAcc.Name),
			Debit:         floatToNumeric(0),
			Credit:        floatToNumeric(addAmount),
			SortOrder:     sortOrder,
			CreatedAt:     now,
		})
		if err != nil {
			return fmt.Errorf("failed to create additional credit: %w", err)
		}
		sortOrder++
	}

	return nil
}

func (s *Service) createManualJournalEntries(ctx context.Context, qtx *db.Queries, txID, workspaceID uuid.UUID, entries []CreateJournalEntryRequest, now time.Time) error {
	if len(entries) == 0 {
		return ErrNoJournalEntries
	}

	for i, entry := range entries {
		account, err := qtx.GetAccountByCode(ctx, db.GetAccountByCodeParams{
			WorkspaceID: workspaceID,
			Code:        entry.AccountCode,
		})
		if err != nil {
			return fmt.Errorf("account %s not found: %w", entry.AccountCode, err)
		}

		_, err = qtx.CreateJournalEntry(ctx, db.CreateJournalEntryParams{
			ID:            uuid.New(),
			TransactionID: txID,
			WorkspaceID:   workspaceID,
			AccountID:     account.ID,
			Description:   stringToPgtext(entry.Description),
			Debit:         entry.Debit,
			Credit:        entry.Credit,
			SortOrder:     int32(i),
			CreatedAt:     now,
		})
		if err != nil {
			return fmt.Errorf("failed to create journal entry: %w", err)
		}
	}

	return nil
}

// --- Utility functions ---

func transactionToResponse(t db.Transaction) TransactionResponse {
	resp := TransactionResponse{
		ID:                t.ID.String(),
		WorkspaceID:       t.WorkspaceID.String(),
		TransactionNumber: t.TransactionNumber,
		TransactionType:   t.TransactionType,
		InputMode:         t.InputMode,
		Status:            t.Status,
		Description:       pgtextToString(t.Description),
		TransactionDate:   t.TransactionDate.Time.Format("2006-01-02"),
		Amount:            numericToString(t.Amount),
		Currency:          t.Currency,
		Category:          pgtextToString(t.Category),
		PaymentMethod:     pgtextToString(t.PaymentMethod),
		IncludesTax:       t.IncludesTax,
		TaxAmount:         numericToString(t.TaxAmount),
		CreatedBy:         t.CreatedBy.String(),
		CreatedAt:         t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:         t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if t.CounterpartyEntityID.Valid {
		resp.CounterpartyEntityID = pgtypeUUIDToString(t.CounterpartyEntityID)
	}
	if t.CounterpartyName.Valid {
		resp.CounterpartyName = t.CounterpartyName.String
	}
	if t.ReversedTransactionID.Valid {
		resp.ReversedTxID = pgtypeUUIDToString(t.ReversedTransactionID)
	}
	if t.PostedAt != nil {
		resp.PostedAt = t.PostedAt.Format("2006-01-02T15:04:05Z")
	}
	if t.VoidedAt != nil {
		resp.VoidedAt = t.VoidedAt.Format("2006-01-02T15:04:05Z")
	}
	if t.AiConfidence.Valid {
		f := numericToFloat(t.AiConfidence)
		resp.AIConfidence = &f
	}

	return resp
}

func isValidTxType(t string) bool {
	for _, v := range ValidTransactionTypes {
		if v == t {
			return true
		}
	}
	return false
}

func isValidInputMode(m string) bool {
	for _, v := range ValidInputModes {
		if v == m {
			return true
		}
	}
	return false
}

func isValidPaymentMethod(m string) bool {
	for _, v := range ValidPaymentMethods {
		if v == m {
			return true
		}
	}
	return false
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	if f.Valid {
		return f.Float64
	}
	return 0
}

// FloatToNumeric converts a float64 to pgtype.Numeric for transaction amounts.
func FloatToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

func floatToNumeric(f float64) pgtype.Numeric {
	return FloatToNumeric(f)
}

func floatToNumeric4(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	n.Scan(fmt.Sprintf("%.4f", f))
	return n
}

func numericToString4(n pgtype.Numeric) string {
	if !n.Valid {
		return "0.0000"
	}
	f, _ := n.Float64Value()
	if f.Valid {
		return fmt.Sprintf("%.4f", f.Float64)
	}
	return "0.0000"
}

func stringToNullable(s string) string {
	return s
}
