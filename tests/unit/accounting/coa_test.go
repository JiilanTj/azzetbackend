package accounting_test

import (
	"testing"

	"codeberg.org/azzet/azzetbe/internal/accounting"
)

// ============================================================
// COA TEMPLATE TESTS
// ============================================================

func TestDefaultCOATemplate_NotEmpty(t *testing.T) {
	template := accounting.DefaultCOATemplate()
	if len(template) == 0 {
		t.Fatal("COA template should not be empty")
	}
}

func TestDefaultCOATemplate_HasAllAccountTypes(t *testing.T) {
	template := accounting.DefaultCOATemplate()

	typeFound := map[string]bool{
		"ASSET":     false,
		"LIABILITY": false,
		"EQUITY":    false,
		"REVENUE":   false,
		"EXPENSE":   false,
	}

	for _, entry := range template {
		typeFound[entry.AccountType] = true
	}

	for typ, found := range typeFound {
		if !found {
			t.Errorf("COA template missing account type: %s", typ)
		}
	}
}

func TestDefaultCOATemplate_TopLevelAccounts(t *testing.T) {
	template := accounting.DefaultCOATemplate()

	expectedTopLevel := []string{"1-0000", "2-0000", "3-0000", "4-0000", "5-0000"}
	topLevelCodes := make(map[string]bool)

	for _, entry := range template {
		if entry.Level == 1 {
			topLevelCodes[entry.Code] = true
		}
	}

	for _, code := range expectedTopLevel {
		if !topLevelCodes[code] {
			t.Errorf("missing top-level account: %s", code)
		}
	}
}

func TestDefaultCOATemplate_AllSystemAccounts(t *testing.T) {
	template := accounting.DefaultCOATemplate()

	for _, entry := range template {
		if !entry.IsSystem {
			t.Errorf("template account %s (%s) should be is_system=true", entry.Code, entry.Name)
		}
	}
}

func TestDefaultCOATemplate_ParentCodesExist(t *testing.T) {
	template := accounting.DefaultCOATemplate()

	codes := make(map[string]bool)
	for _, entry := range template {
		codes[entry.Code] = true
	}

	for _, entry := range template {
		if entry.ParentCode != "" && !codes[entry.ParentCode] {
			t.Errorf("account %s references non-existent parent %s", entry.Code, entry.ParentCode)
		}
	}
}

func TestDefaultCOATemplate_UniqueAccountCodes(t *testing.T) {
	template := accounting.DefaultCOATemplate()

	seen := make(map[string]bool)
	for _, entry := range template {
		if seen[entry.Code] {
			t.Errorf("duplicate account code: %s", entry.Code)
		}
		seen[entry.Code] = true
	}
}

func TestDefaultCOATemplate_NormalBalanceCorrect(t *testing.T) {
	template := accounting.DefaultCOATemplate()

	// Exceptions: contra accounts have opposite normal balance
	contraAccounts := map[string]bool{
		"1-2099": true, // Akumulasi Penyusutan (ASSET but CREDIT normal)
		"3-1002": true, // Prive (EQUITY but DEBIT normal)
		"4-1003": true, // Diskon Penjualan (REVENUE but DEBIT normal)
		"4-1004": true, // Retur Penjualan (REVENUE but DEBIT normal)
		"5-3002": true, // Retur Pembelian (EXPENSE but CREDIT normal)
	}

	for _, entry := range template {
		if contraAccounts[entry.Code] {
			continue // Skip contra accounts
		}

		expectedNormal := ""
		switch entry.AccountType {
		case "ASSET", "EXPENSE":
			expectedNormal = "DEBIT"
		case "LIABILITY", "EQUITY", "REVENUE":
			expectedNormal = "CREDIT"
		}

		if entry.NormalBalance != expectedNormal {
			t.Errorf("account %s (%s) type=%s: expected normal_balance=%s, got %s",
				entry.Code, entry.Name, entry.AccountType, expectedNormal, entry.NormalBalance)
		}
	}
}

func TestDefaultCOATemplate_HasKeyAccounts(t *testing.T) {
	template := accounting.DefaultCOATemplate()

	// These accounts are referenced by the rule engine and MUST exist
	requiredCodes := []string{
		"1-1001", // Kas
		"1-1002", // Bank
		"1-1003", // Piutang Usaha
		"1-1004", // Persediaan
		"1-1007", // Uang Muka Pembelian
		"1-1008", // PPN Masukan
		"2-1001", // Hutang Usaha
		"2-1003", // Hutang Pajak
		"2-1005", // PPN Keluaran
		"2-1006", // Uang Muka Penjualan
		"3-1001", // Modal Pemilik
		"3-1002", // Prive
		"4-1001", // Pendapatan Usaha
		"4-1002", // Pendapatan Jasa
		"4-1003", // Diskon Penjualan
		"4-1004", // Retur Penjualan
		"4-2001", // Pendapatan Bunga
		"4-2002", // Pendapatan Lain-lain
		"5-1001", // Beban Gaji
		"5-1002", // Beban Sewa
		"5-1003", // Beban Listrik
		"5-1004", // Beban Telepon
		"5-1005", // Beban Transportasi
		"5-1006", // Beban Makan
		"5-1007", // Beban Perlengkapan
		"5-1009", // Beban Asuransi
		"5-1010", // Beban Administrasi
		"5-1011", // Beban Biaya Bank
		"5-1012", // Beban Pemasaran
		"5-2001", // Beban Bunga
		"5-2002", // Beban Pajak
		"5-3001", // HPP
		"5-3002", // Retur Pembelian
		"5-9001", // Beban Lain-lain
	}

	codes := make(map[string]bool)
	for _, entry := range template {
		codes[entry.Code] = true
	}

	for _, code := range requiredCodes {
		if !codes[code] {
			t.Errorf("missing required account code: %s (referenced by rule engine)", code)
		}
	}
}

// ============================================================
// CATEGORIZER TESTS (without AI client)
// ============================================================

func TestCategorizer_NilClient_ReturnsFallback(t *testing.T) {
	categorizer := accounting.NewCategorizer(nil)

	result, err := categorizer.Categorize(nil, "CASH_IN", "terima uang dari pak budi", 100000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.UsedFallback {
		t.Error("expected UsedFallback=true when AI client is nil")
	}
	if result.Category != "pendapatan_lain" {
		t.Errorf("expected fallback category 'pendapatan_lain', got %s", result.Category)
	}
	if result.Confidence != 0.0 {
		t.Errorf("expected confidence 0.0, got %f", result.Confidence)
	}
}

func TestCategorizer_EmptyDescription_ReturnsFallback(t *testing.T) {
	categorizer := accounting.NewCategorizer(nil)

	result, err := categorizer.Categorize(nil, "CASH_OUT", "", 50000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.UsedFallback {
		t.Error("expected UsedFallback=true for empty description")
	}
	if result.Category != "beban_lain" {
		t.Errorf("expected fallback 'beban_lain', got %s", result.Category)
	}
}
