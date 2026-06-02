package accounting

import (
	"context"
	"fmt"
	"time"

	"codeberg.org/azzet/azzetbe/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ReportService handles financial report generation
type ReportService struct {
	Queries *db.Queries
}

// NewReportService creates a new ReportService
func NewReportService(queries *db.Queries) *ReportService {
	return &ReportService{
		Queries: queries,
	}
}

// GetTrialBalance returns the trial balance for a workspace within a date range.
// Period format: "YYYY-MM"
func (s *ReportService) GetTrialBalance(ctx context.Context, workspaceID uuid.UUID, periodFrom, periodTo string) ([]TrialBalanceEntry, error) {
	rows, err := s.Queries.GetTrialBalance(ctx, db.GetTrialBalanceParams{
		WorkspaceID: workspaceID,
		Period:      periodFrom,
		Period_2:    periodTo,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get trial balance: %w", err)
	}

	entries := make([]TrialBalanceEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, TrialBalanceEntry{
			AccountID:     r.ID.String(),
			Code:          r.Code,
			Name:          r.Name,
			AccountType:   r.AccountType,
			NormalBalance: r.NormalBalance,
			TotalDebit:    interfaceToString(r.TotalDebit),
			TotalCredit:   interfaceToString(r.TotalCredit),
			Balance:       interfaceToString(r.Balance),
		})
	}
	return entries, nil
}

// GetBalanceSheet returns the balance sheet (Neraca) at a point in time.
// Shows all ASSET, LIABILITY, EQUITY accounts with cumulative balances up to the given period.
func (s *ReportService) GetBalanceSheet(ctx context.Context, workspaceID uuid.UUID, period string) (*BalanceSheetReport, error) {
	rows, err := s.Queries.GetBalanceSheet(ctx, db.GetBalanceSheetParams{
		WorkspaceID: workspaceID,
		Period:      period,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get balance sheet: %w", err)
	}

	report := &BalanceSheetReport{
		Assets:      make([]BalanceSheetEntry, 0),
		Liabilities: make([]BalanceSheetEntry, 0),
		Equity:      make([]BalanceSheetEntry, 0),
	}

	var totalAssets, totalLiab, totalEquity float64

	for _, r := range rows {
		balance := interfaceToFloat(r.Balance)
		entry := BalanceSheetEntry{
			AccountID:     r.ID.String(),
			Code:          r.Code,
			Name:          r.Name,
			AccountType:   r.AccountType,
			NormalBalance: r.NormalBalance,
			Balance:       interfaceToString(r.Balance),
		}

		switch r.AccountType {
		case AccountTypeAsset:
			report.Assets = append(report.Assets, entry)
			totalAssets += balance
		case AccountTypeLiability:
			report.Liabilities = append(report.Liabilities, entry)
			totalLiab += balance
		case AccountTypeEquity:
			report.Equity = append(report.Equity, entry)
			totalEquity += balance
		}
	}

	report.TotalAssets = fmt.Sprintf("%.2f", totalAssets)
	report.TotalLiab = fmt.Sprintf("%.2f", totalLiab)
	report.TotalEquity = fmt.Sprintf("%.2f", totalEquity)

	// Check if balanced (A = L + E)
	diff := totalAssets - (totalLiab + totalEquity)
	report.IsBalanced = diff > -0.01 && diff < 0.01

	return report, nil
}

// GetIncomeStatement returns the income statement (Laba Rugi) for a period range.
// Shows REVENUE and EXPENSE accounts with their totals.
func (s *ReportService) GetIncomeStatement(ctx context.Context, workspaceID uuid.UUID, periodFrom, periodTo string) (*IncomeStatementReport, error) {
	rows, err := s.Queries.GetIncomeStatement(ctx, db.GetIncomeStatementParams{
		WorkspaceID: workspaceID,
		Period:      periodFrom,
		Period_2:    periodTo,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get income statement: %w", err)
	}

	report := &IncomeStatementReport{
		Revenue:  make([]IncomeStatementEntry, 0),
		Expenses: make([]IncomeStatementEntry, 0),
	}

	var totalRevenue, totalExpense float64

	for _, r := range rows {
		balance := interfaceToFloat(r.Balance)
		entry := IncomeStatementEntry{
			AccountID:     r.ID.String(),
			Code:          r.Code,
			Name:          r.Name,
			AccountType:   r.AccountType,
			NormalBalance: r.NormalBalance,
			TotalDebit:    interfaceToString(r.TotalDebit),
			TotalCredit:   interfaceToString(r.TotalCredit),
			Balance:       interfaceToString(r.Balance),
		}

		switch r.AccountType {
		case AccountTypeRevenue:
			report.Revenue = append(report.Revenue, entry)
			// Revenue normal balance is CREDIT, so balance is negative (credit - debit)
			// We want positive number for display
			totalRevenue += -balance // negate because balance = debit - credit
		case AccountTypeExpense:
			report.Expenses = append(report.Expenses, entry)
			totalExpense += balance // expense balance is positive (debit - credit)
		}
	}

	report.TotalRevenue = fmt.Sprintf("%.2f", totalRevenue)
	report.TotalExpense = fmt.Sprintf("%.2f", totalExpense)
	report.NetIncome = fmt.Sprintf("%.2f", totalRevenue-totalExpense)

	return report, nil
}

// GetCashFlow returns cash flow (Arus Kas) for a date range.
// Tracks movements in Kas (1-1001) and Bank (1-1002) accounts.
func (s *ReportService) GetCashFlow(ctx context.Context, workspaceID uuid.UUID, dateFrom, dateTo string) ([]CashFlowEntry, error) {
	fromDate, err := parseDateToPgtype(dateFrom)
	if err != nil {
		return nil, fmt.Errorf("invalid date_from: %w", err)
	}
	toDate, err := parseDateToPgtype(dateTo)
	if err != nil {
		return nil, fmt.Errorf("invalid date_to: %w", err)
	}

	rows, err := s.Queries.GetCashFlow(ctx, db.GetCashFlowParams{
		WorkspaceID:     workspaceID,
		TransactionDate: fromDate,
		TransactionDate_2: toDate,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get cash flow: %w", err)
	}

	entries := make([]CashFlowEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, CashFlowEntry{
			Date:        r.TransactionDate.Time.Format("2006-01-02"),
			TotalDebit:  fmt.Sprintf("%.2f", r.TotalDebit),
			TotalCredit: fmt.Sprintf("%.2f", r.TotalCredit),
			NetFlow:     fmt.Sprintf("%.2f", r.NetFlow),
		})
	}
	return entries, nil
}

// GetGeneralLedger returns all ledger entries for a specific account (Buku Besar).
func (s *ReportService) GetGeneralLedger(ctx context.Context, workspaceID, accountID uuid.UUID, limit, offset int32) ([]LedgerEntryResponse, error) {
	entries, err := s.Queries.ListLedgerEntriesByAccount(ctx, db.ListLedgerEntriesByAccountParams{
		WorkspaceID: workspaceID,
		AccountID:   accountID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get general ledger: %w", err)
	}

	resp := make([]LedgerEntryResponse, 0, len(entries))
	for _, e := range entries {
		resp = append(resp, LedgerEntryResponse{
			ID:                e.ID.String(),
			TransactionNumber: e.TransactionNumber,
			TxDescription:     pgtextToString(e.TxDescription),
			TransactionDate:   e.TransactionDate.Time.Format("2006-01-02"),
			Debit:             numericToString(e.Debit),
			Credit:            numericToString(e.Credit),
			RunningBalance:    numericToString(e.RunningBalance),
			PostedAt:          e.PostedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	return resp, nil
}

// --- Helpers ---

func parseDateToPgtype(dateStr string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}
