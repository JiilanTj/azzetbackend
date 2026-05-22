package accounting_test

import (
	"testing"

	"codeberg.org/azzet/azzetbe/internal/accounting"
)

// ============================================================
// RULE ENGINE TESTS
// ============================================================

func TestGetJournalRule_CashIn(t *testing.T) {
	tests := []struct {
		category    string
		wantDebit   string
		wantCredit  string
	}{
		{"pendapatan_usaha", "1-1001", "4-1001"},
		{"pendapatan_jasa", "1-1001", "4-1002"},
		{"pendapatan_bunga", "1-1002", "4-2001"},
		{"piutang_dibayar", "1-1001", "1-1003"},
		{"hutang_diterima", "1-1001", "2-1001"},
		{"modal_disetor", "1-1001", "3-1001"},
		{"uang_muka_diterima", "1-1001", "2-1006"},
		{"pendapatan_lain", "1-1001", "4-2002"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			rule := accounting.GetJournalRule("CASH_IN", tt.category)
			if rule == nil {
				t.Fatalf("expected rule for CASH_IN/%s, got nil", tt.category)
			}
			if rule.Primary.DebitCode != tt.wantDebit {
				t.Errorf("debit: got %s, want %s", rule.Primary.DebitCode, tt.wantDebit)
			}
			if rule.Primary.CreditCode != tt.wantCredit {
				t.Errorf("credit: got %s, want %s", rule.Primary.CreditCode, tt.wantCredit)
			}
		})
	}
}

func TestGetJournalRule_CashOut(t *testing.T) {
	tests := []struct {
		category    string
		wantDebit   string
		wantCredit  string
	}{
		{"beban_gaji", "5-1001", "1-1001"},
		{"beban_sewa", "5-1002", "1-1001"},
		{"beban_listrik", "5-1003", "1-1001"},
		{"beban_telepon", "5-1004", "1-1001"},
		{"beban_transport", "5-1005", "1-1001"},
		{"beban_makan", "5-1006", "1-1001"},
		{"beban_perlengkapan", "5-1007", "1-1001"},
		{"beban_asuransi", "5-1009", "1-1001"},
		{"beban_admin", "5-1010", "1-1001"},
		{"beban_bank", "5-1011", "1-1002"},
		{"beban_pemasaran", "5-1012", "1-1001"},
		{"beban_bunga", "5-2001", "1-1001"},
		{"beban_pajak", "5-2002", "1-1001"},
		{"pembelian_barang", "1-1004", "1-1001"},
		{"bayar_hutang", "2-1001", "1-1001"},
		{"bayar_pajak", "2-1003", "1-1001"},
		{"uang_muka_beli", "1-1007", "1-1001"},
		{"prive", "3-1002", "1-1001"},
		{"beban_lain", "5-9001", "1-1001"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			rule := accounting.GetJournalRule("CASH_OUT", tt.category)
			if rule == nil {
				t.Fatalf("expected rule for CASH_OUT/%s, got nil", tt.category)
			}
			if rule.Primary.DebitCode != tt.wantDebit {
				t.Errorf("debit: got %s, want %s", rule.Primary.DebitCode, tt.wantDebit)
			}
			if rule.Primary.CreditCode != tt.wantCredit {
				t.Errorf("credit: got %s, want %s", rule.Primary.CreditCode, tt.wantCredit)
			}
		})
	}
}

func TestGetJournalRule_Sales_WithHPP(t *testing.T) {
	rule := accounting.GetJournalRule("SALES", "penjualan_barang_tunai")
	if rule == nil {
		t.Fatal("expected rule for SALES/penjualan_barang_tunai")
	}

	// Primary: Debit Kas, Credit Pendapatan Usaha
	if rule.Primary.DebitCode != "1-1001" {
		t.Errorf("primary debit: got %s, want 1-1001", rule.Primary.DebitCode)
	}
	if rule.Primary.CreditCode != "4-1001" {
		t.Errorf("primary credit: got %s, want 4-1001", rule.Primary.CreditCode)
	}

	// Additional: HPP entry (Debit HPP, Credit Persediaan)
	if len(rule.Additional) != 1 {
		t.Fatalf("expected 1 additional entry (HPP), got %d", len(rule.Additional))
	}
	if rule.Additional[0].DebitCode != "5-3001" {
		t.Errorf("HPP debit: got %s, want 5-3001", rule.Additional[0].DebitCode)
	}
	if rule.Additional[0].CreditCode != "1-1004" {
		t.Errorf("HPP credit: got %s, want 1-1004", rule.Additional[0].CreditCode)
	}
}

func TestGetJournalRule_Sales_Kredit(t *testing.T) {
	rule := accounting.GetJournalRule("SALES", "penjualan_barang_kredit")
	if rule == nil {
		t.Fatal("expected rule for SALES/penjualan_barang_kredit")
	}

	// Primary: Debit Piutang (not Kas), Credit Pendapatan
	if rule.Primary.DebitCode != "1-1003" {
		t.Errorf("primary debit: got %s, want 1-1003 (Piutang)", rule.Primary.DebitCode)
	}
}

func TestGetJournalRule_Sales_Jasa_NoHPP(t *testing.T) {
	rule := accounting.GetJournalRule("SALES", "penjualan_jasa_tunai")
	if rule == nil {
		t.Fatal("expected rule for SALES/penjualan_jasa_tunai")
	}

	// Jasa should NOT have HPP entry
	if len(rule.Additional) != 0 {
		t.Errorf("jasa should have no additional entries, got %d", len(rule.Additional))
	}
}

func TestGetJournalRule_Purchase_WithPPN(t *testing.T) {
	rule := accounting.GetJournalRule("PURCHASE", "pembelian_dengan_ppn")
	if rule == nil {
		t.Fatal("expected rule for PURCHASE/pembelian_dengan_ppn")
	}

	// Additional: PPN Masukan entry
	if len(rule.Additional) != 1 {
		t.Fatalf("expected 1 additional entry (PPN), got %d", len(rule.Additional))
	}
	if rule.Additional[0].DebitCode != "1-1008" {
		t.Errorf("PPN debit: got %s, want 1-1008 (PPN Masukan)", rule.Additional[0].DebitCode)
	}
}

func TestGetJournalRule_InvalidCombination(t *testing.T) {
	// Cash out category used with cash in type
	rule := accounting.GetJournalRule("CASH_IN", "beban_gaji")
	if rule != nil {
		t.Error("expected nil for invalid combination CASH_IN/beban_gaji")
	}

	// Completely invalid
	rule = accounting.GetJournalRule("INVALID", "invalid")
	if rule != nil {
		t.Error("expected nil for invalid type/category")
	}
}

func TestGetFallbackCategory(t *testing.T) {
	tests := []struct {
		txType   string
		expected string
	}{
		{"CASH_IN", "pendapatan_lain"},
		{"CASH_OUT", "beban_lain"},
		{"SALES", "penjualan_barang_tunai"},
		{"PURCHASE", "pembelian_barang_tunai"},
		{"INVALID", "beban_lain"},
	}

	for _, tt := range tests {
		t.Run(tt.txType, func(t *testing.T) {
			got := accounting.GetFallbackCategory(tt.txType)
			if got != tt.expected {
				t.Errorf("got %s, want %s", got, tt.expected)
			}
		})
	}
}
