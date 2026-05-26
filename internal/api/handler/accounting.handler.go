package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"codeberg.org/azzet/azzetbe/internal/accounting"
	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/shared"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AccountingHandler handles all accounting-related HTTP requests
type AccountingHandler struct {
	Service       *accounting.Service
	COAService    *accounting.COAService
	ItemService   *accounting.ItemService
	ReportService *accounting.ReportService
}

// NewAccountingHandler creates a new AccountingHandler
func NewAccountingHandler(
	service *accounting.Service,
	coaService *accounting.COAService,
	itemService *accounting.ItemService,
	reportService *accounting.ReportService,
) *AccountingHandler {
	return &AccountingHandler{
		Service:       service,
		COAService:    coaService,
		ItemService:   itemService,
		ReportService: reportService,
	}
}

// ============================================================
// CHART OF ACCOUNTS
// ============================================================

// ListAccounts godoc
// @Summary List chart of accounts
// @Description Returns all active accounts for the workspace
// @Tags Accounting
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param type query string false "Filter by account type" Enums(ASSET,LIABILITY,EQUITY,REVENUE,EXPENSE)
// @Success 200 {object} shared.APIResponse{data=[]accounting.AccountResponse}
// @Failure 400 {object} shared.APIResponse
// @Router /api/v1/accounts [get]
func (h *AccountingHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}

	accountType := r.URL.Query().Get("type")
	includeInactive := r.URL.Query().Get("include_inactive") == "true"

	if accountType != "" {
		accounts, err := h.COAService.ListAccountsByType(r.Context(), workspaceID, accountType)
		if err != nil {
			shared.BadRequest(w, r, "accounting", err.Error())
			return
		}
		shared.OK(w, r, accounts)
		return
	}

	var accounts []accounting.AccountResponse
	if includeInactive {
		accounts, err = h.COAService.ListAllAccounts(r.Context(), workspaceID)
	} else {
		accounts, err = h.COAService.ListAccounts(r.Context(), workspaceID)
	}
	if err != nil {
		shared.InternalError(w, r, "accounting", err.Error())
		return
	}
	shared.OK(w, r, accounts)
}

// GetAccount godoc
// @Summary Get account by ID
// @Tags Accounting
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param id path string true "Account ID"
// @Success 200 {object} shared.APIResponse{data=accounting.AccountResponse}
// @Failure 404 {object} shared.APIResponse
// @Router /api/v1/accounts/{id} [get]
func (h *AccountingHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	accountID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid account id")
		return
	}

	account, err := h.COAService.GetAccount(r.Context(), workspaceID, accountID)
	if err != nil {
		shared.NotFound(w, r, "accounting", err.Error())
		return
	}
	shared.OK(w, r, account)
}

// CreateAccount godoc
// @Summary Create a custom account
// @Tags Accounting
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param body body accounting.CreateAccountRequest true "Account data"
// @Success 201 {object} shared.APIResponse{data=accounting.AccountResponse}
// @Failure 400 {object} shared.APIResponse
// @Router /api/v1/accounts [post]
func (h *AccountingHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}

	var req accounting.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "accounting", "invalid request body")
		return
	}

	if req.Code == "" {
		shared.BadRequest(w, r, "accounting", "code is required")
		return
	}
	if req.Name == "" {
		shared.BadRequest(w, r, "accounting", "name is required")
		return
	}
	if req.AccountType == "" {
		shared.BadRequest(w, r, "accounting", "account_type is required")
		return
	}

	account, err := h.COAService.CreateAccount(r.Context(), workspaceID, &req)
	if err != nil {
		shared.BadRequest(w, r, "accounting", err.Error())
		return
	}
	shared.Created(w, r, account)
}

// UpdateAccount godoc
// @Summary Update a custom account
// @Tags Accounting
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param id path string true "Account ID"
// @Param body body accounting.UpdateAccountRequest true "Update data"
// @Success 200 {object} shared.APIResponse
// @Failure 400 {object} shared.APIResponse
// @Router /api/v1/accounts/{id} [patch]
func (h *AccountingHandler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	accountID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid account id")
		return
	}

	var req accounting.UpdateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "accounting", "invalid request body")
		return
	}

	if err := h.COAService.UpdateAccount(r.Context(), workspaceID, accountID, &req); err != nil {
		shared.BadRequest(w, r, "accounting", err.Error())
		return
	}
	shared.OK(w, r, map[string]string{"message": "account updated"})
}

// ============================================================
// ITEMS
// ============================================================

// ListItems godoc
// @Summary List items
// @Tags Accounting
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param type query string false "Filter by item type" Enums(BARANG,JASA,PROYEK,AHSP_RAKITAN)
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} shared.APIResponse{data=[]accounting.ItemResponse}
// @Router /api/v1/items [get]
func (h *AccountingHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	limit, offset := parsePagination(r)

	itemType := r.URL.Query().Get("type")
	if itemType != "" {
		items, err := h.ItemService.ListItemsByType(r.Context(), workspaceID, itemType, limit, offset)
		if err != nil {
			shared.BadRequest(w, r, "accounting", err.Error())
			return
		}
		shared.OK(w, r, items)
		return
	}

	items, err := h.ItemService.ListItems(r.Context(), workspaceID, limit, offset)
	if err != nil {
		shared.InternalError(w, r, "accounting", err.Error())
		return
	}
	shared.OK(w, r, items)
}

// GetItem godoc
// @Summary Get item by ID
// @Tags Accounting
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param id path string true "Item ID"
// @Success 200 {object} shared.APIResponse{data=accounting.ItemResponse}
// @Failure 404 {object} shared.APIResponse
// @Router /api/v1/items/{id} [get]
func (h *AccountingHandler) GetItem(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid item id")
		return
	}

	item, err := h.ItemService.GetItem(r.Context(), workspaceID, itemID)
	if err != nil {
		shared.NotFound(w, r, "accounting", err.Error())
		return
	}
	shared.OK(w, r, item)
}

// CreateItem godoc
// @Summary Create an item
// @Tags Accounting
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param body body accounting.CreateItemRequest true "Item data"
// @Success 201 {object} shared.APIResponse{data=accounting.ItemResponse}
// @Failure 400 {object} shared.APIResponse
// @Router /api/v1/items [post]
func (h *AccountingHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}

	var req accounting.CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "accounting", "invalid request body")
		return
	}

	if req.Name == "" {
		shared.BadRequest(w, r, "accounting", "name is required")
		return
	}
	if req.ItemType == "" {
		shared.BadRequest(w, r, "accounting", "item_type is required")
		return
	}
	if req.Unit == "" {
		shared.BadRequest(w, r, "accounting", "unit is required")
		return
	}

	item, err := h.ItemService.CreateItem(r.Context(), workspaceID, &req)
	if err != nil {
		shared.BadRequest(w, r, "accounting", err.Error())
		return
	}
	shared.Created(w, r, item)
}

// UpdateItem godoc
// @Summary Update an item
// @Tags Accounting
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param id path string true "Item ID"
// @Param body body accounting.UpdateItemRequest true "Update data"
// @Success 200 {object} shared.APIResponse
// @Failure 400 {object} shared.APIResponse
// @Router /api/v1/items/{id} [patch]
func (h *AccountingHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid item id")
		return
	}

	var req accounting.UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "accounting", "invalid request body")
		return
	}

	if err := h.ItemService.UpdateItem(r.Context(), workspaceID, itemID, &req); err != nil {
		shared.BadRequest(w, r, "accounting", err.Error())
		return
	}
	shared.OK(w, r, map[string]string{"message": "item updated"})
}

// DeleteItem godoc
// @Summary Soft-delete an item
// @Tags Accounting
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param id path string true "Item ID"
// @Success 200 {object} shared.APIResponse
// @Failure 404 {object} shared.APIResponse
// @Router /api/v1/items/{id} [delete]
func (h *AccountingHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid item id")
		return
	}

	if err := h.ItemService.SoftDeleteItem(r.Context(), workspaceID, itemID); err != nil {
		shared.NotFound(w, r, "accounting", err.Error())
		return
	}
	shared.OK(w, r, map[string]string{"message": "item deleted"})
}

// ============================================================
// TRANSACTIONS
// ============================================================

// CreateTransaction godoc
// @Summary Create a transaction
// @Description Creates a transaction with auto-generated journal entries (SIMPLE mode) or manual entries (ADVANCED mode)
// @Tags Accounting
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param body body accounting.CreateTransactionRequest true "Transaction data"
// @Success 201 {object} shared.APIResponse{data=accounting.TransactionResponse}
// @Failure 400 {object} shared.APIResponse
// @Router /api/v1/transactions [post]
func (h *AccountingHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	userID, err := parseUserID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid user id")
		return
	}

	var req accounting.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "accounting", "invalid request body")
		return
	}

	if req.TransactionType == "" {
		shared.BadRequest(w, r, "accounting", "transaction_type is required")
		return
	}
	if req.TransactionDate == "" {
		shared.BadRequest(w, r, "accounting", "transaction_date is required")
		return
	}

	resp, err := h.Service.CreateTransaction(r.Context(), workspaceID, userID, &req)
	if err != nil {
		shared.BadRequest(w, r, "accounting", err.Error())
		return
	}
	shared.Created(w, r, resp)
}

// GetTransaction godoc
// @Summary Get transaction by ID
// @Description Returns transaction with journal entries and line items
// @Tags Accounting
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param id path string true "Transaction ID"
// @Success 200 {object} shared.APIResponse{data=accounting.TransactionResponse}
// @Failure 404 {object} shared.APIResponse
// @Router /api/v1/transactions/{id} [get]
func (h *AccountingHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	txID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid transaction id")
		return
	}

	resp, err := h.Service.GetTransaction(r.Context(), workspaceID, txID)
	if err != nil {
		shared.NotFound(w, r, "accounting", err.Error())
		return
	}
	shared.OK(w, r, resp)
}

// ListTransactions godoc
// @Summary List transactions
// @Tags Accounting
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} shared.APIResponse{data=[]accounting.TransactionResponse}
// @Router /api/v1/transactions [get]
func (h *AccountingHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	limit, offset := parsePagination(r)

	txs, err := h.Service.ListTransactions(r.Context(), workspaceID, limit, offset)
	if err != nil {
		shared.InternalError(w, r, "accounting", err.Error())
		return
	}
	shared.OK(w, r, txs)
}

// VoidTransaction godoc
// @Summary Void a posted transaction (Jurnal Pembalik)
// @Description Creates a reversal transaction that swaps debit/credit. Original is marked VOID.
// @Tags Accounting
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param id path string true "Transaction ID"
// @Success 201 {object} shared.APIResponse{data=accounting.TransactionResponse}
// @Failure 400 {object} shared.APIResponse
// @Router /api/v1/transactions/{id}/void [post]
func (h *AccountingHandler) VoidTransaction(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	userID, err := parseUserID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid user id")
		return
	}
	txID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid transaction id")
		return
	}

	resp, err := h.Service.VoidTransaction(r.Context(), workspaceID, txID, userID)
	if err != nil {
		shared.BadRequest(w, r, "accounting", err.Error())
		return
	}
	shared.Created(w, r, resp)
}

// CategorizeTransaction godoc
// @Summary AI-powered transaction categorization
// @Description Suggests a category for a transaction based on description. Strict whitelist enforced.
// @Tags Accounting
// @Accept json
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param body body accounting.CategorizationRequest true "Categorization input"
// @Success 200 {object} shared.APIResponse{data=accounting.CategorizationResult}
// @Failure 400 {object} shared.APIResponse
// @Router /api/v1/transactions/categorize [post]
func (h *AccountingHandler) CategorizeTransaction(w http.ResponseWriter, r *http.Request) {
	var req accounting.CategorizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "accounting", "invalid request body")
		return
	}

	if req.TransactionType == "" {
		shared.BadRequest(w, r, "accounting", "transaction_type is required")
		return
	}
	if req.Description == "" {
		shared.BadRequest(w, r, "accounting", "description is required")
		return
	}

	result, err := h.Service.Categorize(r.Context(), &req)
	if err != nil {
		shared.InternalError(w, r, "accounting", err.Error())
		return
	}
	shared.OK(w, r, result)
}

// ============================================================
// REPORTS
// ============================================================

// GetTrialBalance godoc
// @Summary Get Trial Balance (Neraca Saldo)
// @Tags Reports
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param period_from query string true "Period from (YYYY-MM)" example("2026-01")
// @Param period_to query string true "Period to (YYYY-MM)" example("2026-05")
// @Success 200 {object} shared.APIResponse{data=[]accounting.TrialBalanceEntry}
// @Router /api/v1/reports/trial-balance [get]
func (h *AccountingHandler) GetTrialBalance(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	periodFrom := r.URL.Query().Get("period_from")
	periodTo := r.URL.Query().Get("period_to")

	if periodFrom == "" || periodTo == "" {
		shared.BadRequest(w, r, "reports", "period_from and period_to are required (YYYY-MM)")
		return
	}

	entries, err := h.ReportService.GetTrialBalance(r.Context(), workspaceID, periodFrom, periodTo)
	if err != nil {
		shared.InternalError(w, r, "reports", err.Error())
		return
	}
	shared.OK(w, r, entries)
}

// GetBalanceSheet godoc
// @Summary Get Balance Sheet (Neraca)
// @Tags Reports
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param period query string true "Period up to (YYYY-MM)" example("2026-05")
// @Success 200 {object} shared.APIResponse{data=accounting.BalanceSheetReport}
// @Router /api/v1/reports/balance-sheet [get]
func (h *AccountingHandler) GetBalanceSheet(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	period := r.URL.Query().Get("period")

	if period == "" {
		shared.BadRequest(w, r, "reports", "period is required (YYYY-MM)")
		return
	}

	report, err := h.ReportService.GetBalanceSheet(r.Context(), workspaceID, period)
	if err != nil {
		shared.InternalError(w, r, "reports", err.Error())
		return
	}
	shared.OK(w, r, report)
}

// GetIncomeStatement godoc
// @Summary Get Income Statement (Laba Rugi)
// @Tags Reports
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param period_from query string true "Period from (YYYY-MM)"
// @Param period_to query string true "Period to (YYYY-MM)"
// @Success 200 {object} shared.APIResponse{data=accounting.IncomeStatementReport}
// @Router /api/v1/reports/income-statement [get]
func (h *AccountingHandler) GetIncomeStatement(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	periodFrom := r.URL.Query().Get("period_from")
	periodTo := r.URL.Query().Get("period_to")

	if periodFrom == "" || periodTo == "" {
		shared.BadRequest(w, r, "reports", "period_from and period_to are required (YYYY-MM)")
		return
	}

	report, err := h.ReportService.GetIncomeStatement(r.Context(), workspaceID, periodFrom, periodTo)
	if err != nil {
		shared.InternalError(w, r, "reports", err.Error())
		return
	}
	shared.OK(w, r, report)
}

// GetCashFlow godoc
// @Summary Get Cash Flow (Arus Kas)
// @Tags Reports
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param date_from query string true "Date from (YYYY-MM-DD)"
// @Param date_to query string true "Date to (YYYY-MM-DD)"
// @Success 200 {object} shared.APIResponse{data=[]accounting.CashFlowEntry}
// @Router /api/v1/reports/cash-flow [get]
func (h *AccountingHandler) GetCashFlow(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	if dateFrom == "" || dateTo == "" {
		shared.BadRequest(w, r, "reports", "date_from and date_to are required (YYYY-MM-DD)")
		return
	}

	entries, err := h.ReportService.GetCashFlow(r.Context(), workspaceID, dateFrom, dateTo)
	if err != nil {
		shared.InternalError(w, r, "reports", err.Error())
		return
	}
	shared.OK(w, r, entries)
}

// GetGeneralLedger godoc
// @Summary Get General Ledger (Buku Besar) for an account
// @Tags Reports
// @Produce json
// @Param X-Workspace-ID header string true "Workspace ID"
// @Param account_id path string true "Account ID"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} shared.APIResponse{data=[]accounting.LedgerEntryResponse}
// @Router /api/v1/reports/ledger/{account_id} [get]
func (h *AccountingHandler) GetGeneralLedger(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := parseWorkspaceID(r)
	if err != nil {
		shared.BadRequest(w, r, "accounting", "invalid workspace id")
		return
	}
	accountID, err := uuid.Parse(chi.URLParam(r, "account_id"))
	if err != nil {
		shared.BadRequest(w, r, "reports", "invalid account_id")
		return
	}

	limit, offset := parsePagination(r)

	entries, err := h.ReportService.GetGeneralLedger(r.Context(), workspaceID, accountID, limit, offset)
	if err != nil {
		shared.InternalError(w, r, "reports", err.Error())
		return
	}
	shared.OK(w, r, entries)
}

// ============================================================
// HELPERS
// ============================================================

func parsePagination(r *http.Request) (int32, int32) {
	limit := int32(50)
	offset := int32(0)

	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	return limit, offset
}

func parseWorkspaceID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(middleware.GetWorkspaceID(r.Context()))
}

func parseUserID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(middleware.GetUserID(r.Context()))
}
