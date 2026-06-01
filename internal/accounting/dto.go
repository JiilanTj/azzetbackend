package accounting

import (
	"fmt"

	"codeberg.org/azzet/azzetbe/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ============================================================
// ACCOUNT DTOs
// ============================================================

type CreateAccountRequest struct {
	Code        string `json:"code" example:"1-3001"`
	Name        string `json:"name" example:"Deposito"`
	AccountType string `json:"account_type" example:"ASSET" enums:"ASSET,LIABILITY,EQUITY,REVENUE,EXPENSE"`
	ParentID    string `json:"parent_id,omitempty" example:"550e8400-..."`
}

type UpdateAccountRequest struct {
	Name     string `json:"name,omitempty" example:"Deposito Berjangka"`
	ParentID string `json:"parent_id,omitempty" example:"550e8400-..."`
	IsActive *bool  `json:"is_active,omitempty" example:"true"`
}

type AccountResponse struct {
	ID            string `json:"id" example:"550e8400-..."`
	WorkspaceID   string `json:"workspace_id" example:"550e8400-..."`
	ParentID      string `json:"parent_id,omitempty" example:"550e8400-..."`
	Code          string `json:"code" example:"1-1001"`
	Name          string `json:"name" example:"Kas"`
	AccountType   string `json:"account_type" example:"ASSET"`
	NormalBalance string `json:"normal_balance" example:"DEBIT"`
	Level         int    `json:"level" example:"3"`
	IsSystem      bool   `json:"is_system" example:"true"`
	IsActive      bool   `json:"is_active" example:"true"`
	CreatedAt     string `json:"created_at" example:"2026-05-20T10:00:00Z"`
	UpdatedAt     string `json:"updated_at" example:"2026-05-20T10:00:00Z"`
}

func AccountToResponse(a db.Account) AccountResponse {
	resp := AccountResponse{
		ID:            a.ID.String(),
		WorkspaceID:   a.WorkspaceID.String(),
		Code:          a.Code,
		Name:          a.Name,
		AccountType:   a.AccountType,
		NormalBalance: a.NormalBalance,
		Level:         int(a.Level),
		IsSystem:      a.IsSystem,
		IsActive:      a.IsActive,
		CreatedAt:     a.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     a.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if a.ParentID.Valid {
		id := uuid.UUID(a.ParentID.Bytes)
		resp.ParentID = id.String()
	}
	return resp
}

// ============================================================
// ITEM DTOs
// ============================================================

type CreateItemRequest struct {
	Name        string             `json:"name" example:"Nasi Goreng"`
	ItemType    string             `json:"item_type" example:"BARANG" enums:"BARANG,JASA,PROYEK,AHSP_RAKITAN"`
	Unit        string             `json:"unit" example:"Pcs" enums:"Pcs,Kg,Liter,Meter,M2,M3,Jam,Hari,Paket,Unit,Box,Lusin,Set,Rim"`
	UnitPrice   pgtype.Numeric     `json:"unit_price"`
	AccountID   string             `json:"account_id,omitempty" example:"550e8400-..."`
	Description string             `json:"description,omitempty" example:"Nasi goreng spesial"`
}

type UpdateItemRequest struct {
	Name        string          `json:"name,omitempty" example:"Nasi Goreng Spesial"`
	ItemType    string          `json:"item_type,omitempty" example:"BARANG"`
	Unit        string          `json:"unit,omitempty" example:"Pcs"`
	UnitPrice   *pgtype.Numeric `json:"unit_price,omitempty"`
	AccountID   *string         `json:"account_id,omitempty" example:"550e8400-..."`
	Description *string         `json:"description,omitempty" example:"Updated description"`
}

type ItemResponse struct {
	ID          string `json:"id" example:"550e8400-..."`
	WorkspaceID string `json:"workspace_id" example:"550e8400-..."`
	Name        string `json:"name" example:"Nasi Goreng"`
	ItemType    string `json:"item_type" example:"BARANG"`
	Unit        string `json:"unit" example:"Pcs"`
	UnitPrice   string `json:"unit_price" example:"15000.00"`
	AccountID   string `json:"account_id,omitempty" example:"550e8400-..."`
	Description string `json:"description,omitempty" example:"Nasi goreng spesial"`
	IsActive    bool   `json:"is_active" example:"true"`
	CreatedAt   string `json:"created_at" example:"2026-05-20T10:00:00Z"`
	UpdatedAt   string `json:"updated_at" example:"2026-05-20T10:00:00Z"`
}

func ItemToResponse(i db.Item) ItemResponse {
	resp := ItemResponse{
		ID:          i.ID.String(),
		WorkspaceID: i.WorkspaceID.String(),
		Name:        i.Name,
		ItemType:    i.ItemType,
		Unit:        i.Unit,
		UnitPrice:   numericToString(i.UnitPrice),
		IsActive:    i.IsActive,
		CreatedAt:   i.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   i.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if i.Description.Valid {
		resp.Description = i.Description.String
	}
	if i.AccountID.Valid {
		id := uuid.UUID(i.AccountID.Bytes)
		resp.AccountID = id.String()
	}
	return resp
}

// ============================================================
// TRANSACTION DTOs
// ============================================================

type CreateTransactionRequest struct {
	TransactionType string              `json:"transaction_type" example:"CASH_IN" enums:"CASH_IN,CASH_OUT,SALES,PURCHASE,JOURNAL"`
	InputMode       string              `json:"input_mode,omitempty" example:"SIMPLE" enums:"SIMPLE,ADVANCED,OCR"`
	Description     string              `json:"description,omitempty" example:"Terima pembayaran dari Pak Budi"`
	TransactionDate string              `json:"transaction_date" example:"2026-05-20"`
	Amount          pgtype.Numeric      `json:"amount"`
	Category        string              `json:"category,omitempty" example:"pendapatan_usaha"`
	PaymentMethod   string              `json:"payment_method,omitempty" example:"TUNAI" enums:"TUNAI,KREDIT,TRANSFER"`
	IncludesTax     bool                `json:"includes_tax,omitempty" example:"false"`

	// Counterparty
	CounterpartyEntityID string `json:"counterparty_entity_id,omitempty" example:"550e8400-..."`
	CounterpartyName     string `json:"counterparty_name,omitempty" example:"Pak Budi"`

	// Line items (for SALES/PURCHASE)
	LineItems []CreateLineItemRequest `json:"line_items,omitempty"`

	// Advanced mode: manual journal entries
	JournalEntries []CreateJournalEntryRequest `json:"journal_entries,omitempty"`
}

type CreateLineItemRequest struct {
	ItemID         string         `json:"item_id,omitempty" example:"550e8400-..."`
	Description    string         `json:"description" example:"Nasi Goreng"`
	Quantity       float64        `json:"quantity" example:"5"`
	Unit           string         `json:"unit,omitempty" example:"Pcs"`
	UnitPrice      pgtype.Numeric `json:"unit_price"`
	DiscountAmount pgtype.Numeric `json:"discount_amount,omitempty"`
}

type CreateJournalEntryRequest struct {
	AccountCode string         `json:"account_code" example:"1-1001"`
	Description string         `json:"description,omitempty" example:"Kas masuk"`
	Debit       pgtype.Numeric `json:"debit"`
	Credit      pgtype.Numeric `json:"credit"`
}

type UpdateTransactionRequest struct {
	Description          string         `json:"description,omitempty" example:"Updated description"`
	TransactionDate      string         `json:"transaction_date,omitempty" example:"2026-05-21"`
	Amount               *pgtype.Numeric `json:"amount,omitempty"`
	Category             string         `json:"category,omitempty" example:"pendapatan_jasa"`
	CounterpartyEntityID string         `json:"counterparty_entity_id,omitempty"`
	CounterpartyName     string         `json:"counterparty_name,omitempty"`
	PaymentMethod        string         `json:"payment_method,omitempty"`
	IncludesTax          *bool          `json:"includes_tax,omitempty"`
	TaxAmount            *pgtype.Numeric `json:"tax_amount,omitempty"`
}

type CategorizationRequest struct {
	TransactionType string  `json:"transaction_type" example:"CASH_OUT" enums:"CASH_IN,CASH_OUT,SALES,PURCHASE"`
	Description     string  `json:"description" example:"Bayar listrik bulan Mei"`
	Amount          float64 `json:"amount" example:"500000"`
}

type TransactionResponse struct {
	ID                   string               `json:"id" example:"550e8400-..."`
	WorkspaceID          string               `json:"workspace_id" example:"550e8400-..."`
	TransactionNumber    string               `json:"transaction_number" example:"TXN-000001"`
	TransactionType      string               `json:"transaction_type" example:"CASH_IN"`
	InputMode            string               `json:"input_mode" example:"SIMPLE"`
	Status               string               `json:"status" example:"DRAFT"`
	CounterpartyEntityID string               `json:"counterparty_entity_id,omitempty"`
	CounterpartyName     string               `json:"counterparty_name,omitempty"`
	Description          string               `json:"description,omitempty"`
	TransactionDate      string               `json:"transaction_date" example:"2026-05-20"`
	Amount               string               `json:"amount" example:"100000.00"`
	Currency             string               `json:"currency" example:"IDR"`
	Category             string               `json:"category,omitempty" example:"pendapatan_usaha"`
	AIConfidence         *float64             `json:"ai_confidence,omitempty" example:"0.92"`
	PaymentMethod        string               `json:"payment_method,omitempty" example:"TUNAI"`
	IncludesTax          bool                 `json:"includes_tax" example:"false"`
	TaxAmount            string               `json:"tax_amount,omitempty" example:"0.00"`
	ReversedTxID         string               `json:"reversed_transaction_id,omitempty"`
	CreatedBy            string               `json:"created_by" example:"550e8400-..."`
	PostedAt             string               `json:"posted_at,omitempty"`
	VoidedAt             string               `json:"voided_at,omitempty"`
	CreatedAt            string               `json:"created_at" example:"2026-05-20T10:00:00Z"`
	UpdatedAt            string               `json:"updated_at" example:"2026-05-20T10:00:00Z"`
	LineItems            []LineItemResponse    `json:"line_items,omitempty"`
	JournalEntries       []JournalEntryResponse `json:"journal_entries,omitempty"`
}

type LineItemResponse struct {
	ID             string `json:"id" example:"550e8400-..."`
	ItemID         string `json:"item_id,omitempty"`
	Description    string `json:"description" example:"Nasi Goreng"`
	Quantity       string `json:"quantity" example:"5.0000"`
	Unit           string `json:"unit" example:"Pcs"`
	UnitPrice      string `json:"unit_price" example:"15000.00"`
	DiscountAmount string `json:"discount_amount" example:"0.00"`
	TaxAmount      string `json:"tax_amount" example:"0.00"`
	LineTotal      string `json:"line_total" example:"75000.00"`
	SortOrder      int    `json:"sort_order" example:"0"`
}

type JournalEntryResponse struct {
	ID          string `json:"id" example:"550e8400-..."`
	AccountID   string `json:"account_id" example:"550e8400-..."`
	AccountCode string `json:"account_code" example:"1-1001"`
	AccountName string `json:"account_name" example:"Kas"`
	Description string `json:"description,omitempty"`
	Debit       string `json:"debit" example:"100000.00"`
	Credit      string `json:"credit" example:"0.00"`
	SortOrder   int    `json:"sort_order" example:"0"`
}

// ============================================================
// REPORT DTOs
// ============================================================

type TrialBalanceEntry struct {
	AccountID     string `json:"account_id"`
	Code          string `json:"code" example:"1-1001"`
	Name          string `json:"name" example:"Kas"`
	AccountType   string `json:"account_type" example:"ASSET"`
	NormalBalance string `json:"normal_balance" example:"DEBIT"`
	TotalDebit    string `json:"total_debit" example:"500000.00"`
	TotalCredit   string `json:"total_credit" example:"200000.00"`
	Balance       string `json:"balance" example:"300000.00"`
}

type BalanceSheetEntry struct {
	AccountID     string `json:"account_id"`
	Code          string `json:"code" example:"1-1001"`
	Name          string `json:"name" example:"Kas"`
	AccountType   string `json:"account_type" example:"ASSET"`
	NormalBalance string `json:"normal_balance" example:"DEBIT"`
	Balance       string `json:"balance" example:"300000.00"`
}

type BalanceSheetReport struct {
	Assets      []BalanceSheetEntry `json:"assets"`
	Liabilities []BalanceSheetEntry `json:"liabilities"`
	Equity      []BalanceSheetEntry `json:"equity"`
	TotalAssets string              `json:"total_assets" example:"1000000.00"`
	TotalLiab   string              `json:"total_liabilities" example:"300000.00"`
	TotalEquity string              `json:"total_equity" example:"700000.00"`
	IsBalanced  bool                `json:"is_balanced" example:"true"`
}

type IncomeStatementEntry struct {
	AccountID     string `json:"account_id"`
	Code          string `json:"code" example:"4-1001"`
	Name          string `json:"name" example:"Pendapatan Usaha"`
	AccountType   string `json:"account_type" example:"REVENUE"`
	NormalBalance string `json:"normal_balance" example:"CREDIT"`
	TotalDebit    string `json:"total_debit" example:"0.00"`
	TotalCredit   string `json:"total_credit" example:"500000.00"`
	Balance       string `json:"balance" example:"500000.00"`
}

type IncomeStatementReport struct {
	Revenue      []IncomeStatementEntry `json:"revenue"`
	Expenses     []IncomeStatementEntry `json:"expenses"`
	TotalRevenue string                 `json:"total_revenue" example:"500000.00"`
	TotalExpense string                 `json:"total_expenses" example:"200000.00"`
	NetIncome    string                 `json:"net_income" example:"300000.00"`
}

type CashFlowEntry struct {
	Date        string `json:"date" example:"2026-05-20"`
	TotalDebit  string `json:"total_debit" example:"100000.00"`
	TotalCredit string `json:"total_credit" example:"50000.00"`
	NetFlow     string `json:"net_flow" example:"50000.00"`
}

type LedgerEntryResponse struct {
	ID                string `json:"id"`
	TransactionNumber string `json:"transaction_number" example:"TXN-000001"`
	TxDescription     string `json:"tx_description,omitempty"`
	TransactionDate   string `json:"transaction_date" example:"2026-05-20"`
	Debit             string `json:"debit" example:"100000.00"`
	Credit            string `json:"credit" example:"0.00"`
	RunningBalance    string `json:"running_balance" example:"100000.00"`
	PostedAt          string `json:"posted_at" example:"2026-05-20T10:00:00Z"`
}

// ============================================================
// HELPERS
// ============================================================

func numericToString(n pgtype.Numeric) string {
	if !n.Valid {
		return "0.00"
	}
	f, _ := n.Float64Value()
	if f.Valid {
		return fmt.Sprintf("%.2f", f.Float64)
	}
	return "0.00"
}

func stringVal(s string) string {
	return s
}
