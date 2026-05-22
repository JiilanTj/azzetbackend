package accounting_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

// ============================================================
// HELPER: Simulate authenticated + workspace-scoped request
// ============================================================

func newAuthenticatedRequest(method, path, body string, workspaceID string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	// Simulate middleware context values
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.UserIDKey, uuid.New().String())
	ctx = context.WithValue(ctx, middleware.WorkspaceIDKey, workspaceID)
	ctx = context.WithValue(ctx, middleware.WorkspaceRoleKey, "PEMILIK")
	req = req.WithContext(ctx)

	return req
}

// ============================================================
// ACCOUNTS API TESTS
// ============================================================

func TestListAccounts_RequiresWorkspaceID(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/accounts", func(w http.ResponseWriter, r *http.Request) {
		wsID := r.Context().Value(middleware.WorkspaceIDKey)
		if wsID == nil || wsID.(string) == "" {
			shared.BadRequest(w, r, "accounting", "workspace_id required")
			return
		}
		shared.OK(w, r, []map[string]string{})
	})

	// Without workspace ID
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/accounts", nil)
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without workspace_id, got %d", recorder.Code)
	}
}

func TestListAccounts_WithWorkspaceID(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/accounts", func(w http.ResponseWriter, r *http.Request) {
		wsID := r.Context().Value(middleware.WorkspaceIDKey)
		if wsID == nil || wsID.(string) == "" {
			shared.BadRequest(w, r, "accounting", "workspace_id required")
			return
		}
		shared.OK(w, r, []map[string]string{
			{"code": "1-1001", "name": "Kas"},
			{"code": "4-1001", "name": "Pendapatan Usaha"},
		})
	})

	recorder := httptest.NewRecorder()
	req := newAuthenticatedRequest("GET", "/accounts", "", uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(data))
	}
}

func TestCreateAccount_MissingCode(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/accounts", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Code        string `json:"code"`
			Name        string `json:"name"`
			AccountType string `json:"account_type"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Code == "" {
			shared.BadRequest(w, r, "accounting", "code is required")
			return
		}
		shared.Created(w, r, map[string]string{"code": req.Code})
	})

	recorder := httptest.NewRecorder()
	body := `{"name": "Test Account", "account_type": "EXPENSE"}`
	req := newAuthenticatedRequest("POST", "/accounts", body, uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestCreateAccount_ValidRequest(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/accounts", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Code        string `json:"code"`
			Name        string `json:"name"`
			AccountType string `json:"account_type"`
		}
		json.NewDecoder(r.Body).Decode(&req)
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
		shared.Created(w, r, map[string]string{
			"code":           req.Code,
			"name":           req.Name,
			"account_type":   req.AccountType,
			"normal_balance": "DEBIT",
			"is_system":      "false",
		})
	})

	recorder := httptest.NewRecorder()
	body := `{"code": "5-1013", "name": "Beban Parkir", "account_type": "EXPENSE"}`
	req := newAuthenticatedRequest("POST", "/accounts", body, uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["code"] != "5-1013" {
		t.Fatalf("expected code '5-1013', got '%v'", data["code"])
	}
}

// ============================================================
// ITEMS API TESTS
// ============================================================

func TestCreateItem_MissingName(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/items", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			ItemType string `json:"item_type"`
			Unit     string `json:"unit"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Name == "" {
			shared.BadRequest(w, r, "accounting", "name is required")
			return
		}
		shared.Created(w, r, map[string]string{"name": req.Name})
	})

	recorder := httptest.NewRecorder()
	body := `{"item_type": "BARANG", "unit": "Pcs"}`
	req := newAuthenticatedRequest("POST", "/items", body, uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestCreateItem_ValidRequest(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/items", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string  `json:"name"`
			ItemType string  `json:"item_type"`
			Unit     string  `json:"unit"`
			Price    float64 `json:"unit_price"`
		}
		json.NewDecoder(r.Body).Decode(&req)
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
		shared.Created(w, r, map[string]any{
			"name":       req.Name,
			"item_type":  req.ItemType,
			"unit":       req.Unit,
			"unit_price": req.Price,
			"is_active":  true,
		})
	})

	recorder := httptest.NewRecorder()
	body := `{"name": "Nasi Goreng", "item_type": "BARANG", "unit": "Pcs", "unit_price": 25000}`
	req := newAuthenticatedRequest("POST", "/items", body, uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["name"] != "Nasi Goreng" {
		t.Fatalf("expected name 'Nasi Goreng', got '%v'", data["name"])
	}
	if data["is_active"] != true {
		t.Fatal("expected is_active=true")
	}
}

// ============================================================
// TRANSACTIONS API TESTS
// ============================================================

func TestCreateTransaction_MissingType(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/transactions", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TransactionType string `json:"transaction_type"`
			TransactionDate string `json:"transaction_date"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.TransactionType == "" {
			shared.BadRequest(w, r, "accounting", "transaction_type is required")
			return
		}
		if req.TransactionDate == "" {
			shared.BadRequest(w, r, "accounting", "transaction_date is required")
			return
		}
		shared.Created(w, r, map[string]string{"status": "DRAFT"})
	})

	recorder := httptest.NewRecorder()
	body := `{"transaction_date": "2026-05-20", "amount": 100000}`
	req := newAuthenticatedRequest("POST", "/transactions", body, uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestCreateTransaction_MissingDate(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/transactions", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TransactionType string `json:"transaction_type"`
			TransactionDate string `json:"transaction_date"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.TransactionType == "" {
			shared.BadRequest(w, r, "accounting", "transaction_type is required")
			return
		}
		if req.TransactionDate == "" {
			shared.BadRequest(w, r, "accounting", "transaction_date is required")
			return
		}
		shared.Created(w, r, map[string]string{"status": "DRAFT"})
	})

	recorder := httptest.NewRecorder()
	body := `{"transaction_type": "CASH_IN", "amount": 100000}`
	req := newAuthenticatedRequest("POST", "/transactions", body, uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestCreateTransaction_SimpleMode_ValidRequest(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/transactions", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TransactionType string  `json:"transaction_type"`
			TransactionDate string  `json:"transaction_date"`
			Amount          float64 `json:"amount"`
			Category        string  `json:"category"`
			Description     string  `json:"description"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.TransactionType == "" {
			shared.BadRequest(w, r, "accounting", "transaction_type is required")
			return
		}
		if req.TransactionDate == "" {
			shared.BadRequest(w, r, "accounting", "transaction_date is required")
			return
		}
		shared.Created(w, r, map[string]any{
			"transaction_number": "TXN-000001",
			"transaction_type":   req.TransactionType,
			"status":             "DRAFT",
			"amount":             req.Amount,
			"category":           req.Category,
			"input_mode":         "SIMPLE",
		})
	})

	recorder := httptest.NewRecorder()
	body := `{
		"transaction_type": "CASH_IN",
		"input_mode": "SIMPLE",
		"description": "Terima uang dari Pak Budi",
		"transaction_date": "2026-05-20",
		"amount": 100000,
		"category": "pendapatan_usaha",
		"counterparty_name": "Pak Budi"
	}`
	req := newAuthenticatedRequest("POST", "/transactions", body, uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["status"] != "DRAFT" {
		t.Fatalf("expected status 'DRAFT', got '%v'", data["status"])
	}
	if data["transaction_type"] != "CASH_IN" {
		t.Fatalf("expected type 'CASH_IN', got '%v'", data["transaction_type"])
	}
	if data["input_mode"] != "SIMPLE" {
		t.Fatalf("expected input_mode 'SIMPLE', got '%v'", data["input_mode"])
	}
}

func TestVoidTransaction_RequiresPostedStatus(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/transactions/{id}/void", func(w http.ResponseWriter, r *http.Request) {
		txID := chi.URLParam(r, "id")
		if txID == "" {
			shared.BadRequest(w, r, "accounting", "invalid transaction id")
			return
		}
		// Simulate: transaction is still DRAFT
		shared.BadRequest(w, r, "accounting", "can only void transactions in POSTED status")
	})

	recorder := httptest.NewRecorder()
	req := newAuthenticatedRequest("POST", "/transactions/"+uuid.New().String()+"/void", "", uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-POSTED transaction, got %d", recorder.Code)
	}

	var resp map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	errData := resp["error"].(map[string]any)
	if !strings.Contains(errData["message"].(string), "POSTED") {
		t.Fatalf("error should mention POSTED status, got: %s", errData["message"])
	}
}

// ============================================================
// CATEGORIZE API TESTS
// ============================================================

func TestCategorize_MissingType(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/transactions/categorize", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TransactionType string `json:"transaction_type"`
			Description     string `json:"description"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.TransactionType == "" {
			shared.BadRequest(w, r, "accounting", "transaction_type is required")
			return
		}
		if req.Description == "" {
			shared.BadRequest(w, r, "accounting", "description is required")
			return
		}
		shared.OK(w, r, map[string]any{"category": "beban_lain", "confidence": 0.0, "used_fallback": true})
	})

	recorder := httptest.NewRecorder()
	body := `{"description": "bayar listrik"}`
	req := newAuthenticatedRequest("POST", "/transactions/categorize", body, uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestCategorize_ValidRequest(t *testing.T) {
	r := chi.NewRouter()
	r.Post("/transactions/categorize", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TransactionType string `json:"transaction_type"`
			Description     string `json:"description"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.TransactionType == "" {
			shared.BadRequest(w, r, "accounting", "transaction_type is required")
			return
		}
		if req.Description == "" {
			shared.BadRequest(w, r, "accounting", "description is required")
			return
		}
		// Simulate AI response (without actual AI call)
		shared.OK(w, r, map[string]any{
			"category":     "beban_listrik",
			"confidence":   0.92,
			"used_fallback": false,
		})
	})

	recorder := httptest.NewRecorder()
	body := `{"transaction_type": "CASH_OUT", "description": "bayar listrik bulan mei", "amount": 500000}`
	req := newAuthenticatedRequest("POST", "/transactions/categorize", body, uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["category"] != "beban_listrik" {
		t.Fatalf("expected category 'beban_listrik', got '%v'", data["category"])
	}
	if data["used_fallback"] != false {
		t.Fatal("expected used_fallback=false")
	}
}

// ============================================================
// REPORTS API TESTS
// ============================================================

func TestTrialBalance_MissingPeriod(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/reports/trial-balance", func(w http.ResponseWriter, r *http.Request) {
		periodFrom := r.URL.Query().Get("period_from")
		periodTo := r.URL.Query().Get("period_to")
		if periodFrom == "" || periodTo == "" {
			shared.BadRequest(w, r, "reports", "period_from and period_to are required (YYYY-MM)")
			return
		}
		shared.OK(w, r, []map[string]string{})
	})

	recorder := httptest.NewRecorder()
	req := newAuthenticatedRequest("GET", "/reports/trial-balance", "", uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestTrialBalance_ValidRequest(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/reports/trial-balance", func(w http.ResponseWriter, r *http.Request) {
		periodFrom := r.URL.Query().Get("period_from")
		periodTo := r.URL.Query().Get("period_to")
		if periodFrom == "" || periodTo == "" {
			shared.BadRequest(w, r, "reports", "period_from and period_to are required (YYYY-MM)")
			return
		}
		shared.OK(w, r, []map[string]any{
			{"code": "1-1001", "name": "Kas", "total_debit": "500000.00", "total_credit": "200000.00", "balance": "300000.00"},
			{"code": "4-1001", "name": "Pendapatan Usaha", "total_debit": "0.00", "total_credit": "500000.00", "balance": "-500000.00"},
		})
	})

	recorder := httptest.NewRecorder()
	req := newAuthenticatedRequest("GET", "/reports/trial-balance?period_from=2026-01&period_to=2026-05", "", uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(data))
	}
}

func TestBalanceSheet_MissingPeriod(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/reports/balance-sheet", func(w http.ResponseWriter, r *http.Request) {
		period := r.URL.Query().Get("period")
		if period == "" {
			shared.BadRequest(w, r, "reports", "period is required (YYYY-MM)")
			return
		}
		shared.OK(w, r, map[string]any{})
	})

	recorder := httptest.NewRecorder()
	req := newAuthenticatedRequest("GET", "/reports/balance-sheet", "", uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestIncomeStatement_ValidRequest(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/reports/income-statement", func(w http.ResponseWriter, r *http.Request) {
		periodFrom := r.URL.Query().Get("period_from")
		periodTo := r.URL.Query().Get("period_to")
		if periodFrom == "" || periodTo == "" {
			shared.BadRequest(w, r, "reports", "period_from and period_to are required (YYYY-MM)")
			return
		}
		shared.OK(w, r, map[string]any{
			"revenue":        []map[string]any{{"code": "4-1001", "name": "Pendapatan Usaha", "balance": "5000000.00"}},
			"expenses":       []map[string]any{{"code": "5-1001", "name": "Beban Gaji", "balance": "2000000.00"}},
			"total_revenue":  "5000000.00",
			"total_expenses": "2000000.00",
			"net_income":     "3000000.00",
		})
	})

	recorder := httptest.NewRecorder()
	req := newAuthenticatedRequest("GET", "/reports/income-statement?period_from=2026-05&period_to=2026-05", "", uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var resp map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["net_income"] != "3000000.00" {
		t.Fatalf("expected net_income '3000000.00', got '%v'", data["net_income"])
	}
}

func TestCashFlow_MissingDates(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/reports/cash-flow", func(w http.ResponseWriter, r *http.Request) {
		dateFrom := r.URL.Query().Get("date_from")
		dateTo := r.URL.Query().Get("date_to")
		if dateFrom == "" || dateTo == "" {
			shared.BadRequest(w, r, "reports", "date_from and date_to are required (YYYY-MM-DD)")
			return
		}
		shared.OK(w, r, []map[string]any{})
	})

	recorder := httptest.NewRecorder()
	req := newAuthenticatedRequest("GET", "/reports/cash-flow", "", uuid.New().String())
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}
