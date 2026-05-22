package accounting_test

import (
	"testing"

	"codeberg.org/azzet/azzetbe/internal/accounting"
)

// ============================================================
// CONSTANTS & CATEGORY VALIDATION TESTS
// ============================================================

func TestIsValidCategory_ValidCashIn(t *testing.T) {
	validCategories := []string{
		"pendapatan_usaha", "pendapatan_jasa", "pendapatan_bunga",
		"piutang_dibayar", "hutang_diterima", "modal_disetor",
		"uang_muka_diterima", "pendapatan_lain",
	}

	for _, cat := range validCategories {
		if !accounting.IsValidCategory(cat) {
			t.Errorf("expected %q to be valid category", cat)
		}
	}
}

func TestIsValidCategory_ValidCashOut(t *testing.T) {
	validCategories := []string{
		"beban_gaji", "beban_sewa", "beban_listrik", "beban_telepon",
		"beban_transport", "beban_makan", "beban_perlengkapan",
		"beban_asuransi", "beban_admin", "beban_bank", "beban_pemasaran",
		"beban_bunga", "beban_pajak", "pembelian_barang", "bayar_hutang",
		"bayar_pajak", "uang_muka_beli", "prive", "beban_lain",
	}

	for _, cat := range validCategories {
		if !accounting.IsValidCategory(cat) {
			t.Errorf("expected %q to be valid category", cat)
		}
	}
}

func TestIsValidCategory_ValidSales(t *testing.T) {
	validCategories := []string{
		"penjualan_barang_tunai", "penjualan_barang_kredit",
		"penjualan_jasa_tunai", "penjualan_jasa_kredit",
		"penjualan_dengan_ppn",
	}

	for _, cat := range validCategories {
		if !accounting.IsValidCategory(cat) {
			t.Errorf("expected %q to be valid category", cat)
		}
	}
}

func TestIsValidCategory_ValidPurchase(t *testing.T) {
	validCategories := []string{
		"pembelian_barang_tunai", "pembelian_barang_kredit",
		"pembelian_jasa_tunai", "pembelian_jasa_kredit",
		"pembelian_dengan_ppn",
	}

	for _, cat := range validCategories {
		if !accounting.IsValidCategory(cat) {
			t.Errorf("expected %q to be valid category", cat)
		}
	}
}

func TestIsValidCategory_Invalid(t *testing.T) {
	invalidCategories := []string{
		"", "invalid", "BEBAN_GAJI", "random_category",
		"pendapatan", "beban", "sql_injection; DROP TABLE",
	}

	for _, cat := range invalidCategories {
		if accounting.IsValidCategory(cat) {
			t.Errorf("expected %q to be invalid category", cat)
		}
	}
}

func TestIsValidCategoryForType_CashIn(t *testing.T) {
	// Valid
	if !accounting.IsValidCategoryForType("pendapatan_usaha", "CASH_IN") {
		t.Error("pendapatan_usaha should be valid for CASH_IN")
	}

	// Invalid - cash_out category used for cash_in
	if accounting.IsValidCategoryForType("beban_gaji", "CASH_IN") {
		t.Error("beban_gaji should NOT be valid for CASH_IN")
	}
}

func TestIsValidCategoryForType_CashOut(t *testing.T) {
	if !accounting.IsValidCategoryForType("beban_gaji", "CASH_OUT") {
		t.Error("beban_gaji should be valid for CASH_OUT")
	}

	if accounting.IsValidCategoryForType("pendapatan_usaha", "CASH_OUT") {
		t.Error("pendapatan_usaha should NOT be valid for CASH_OUT")
	}
}

func TestIsValidCategoryForType_Sales(t *testing.T) {
	if !accounting.IsValidCategoryForType("penjualan_barang_tunai", "SALES") {
		t.Error("penjualan_barang_tunai should be valid for SALES")
	}

	if accounting.IsValidCategoryForType("beban_gaji", "SALES") {
		t.Error("beban_gaji should NOT be valid for SALES")
	}
}

func TestIsValidCategoryForType_InvalidType(t *testing.T) {
	if accounting.IsValidCategoryForType("pendapatan_usaha", "INVALID") {
		t.Error("should return false for invalid transaction type")
	}
}
