package accounting

// JournalRule defines which accounts to debit and credit for a given category
type JournalRule struct {
	DebitCode  string
	CreditCode string
}

// MultiJournalRule defines rules that generate multiple journal entries (e.g., with HPP or PPN)
type MultiJournalRule struct {
	Primary    JournalRule
	Additional []JournalRule // Additional entries (HPP, PPN, etc.)
}

// --- CASH_IN Rules ---
// Direction: money comes IN → Debit Kas/Bank, Credit varies

var CashInRules = map[string]JournalRule{
	CatPendapatanUsaha:  {DebitCode: "1-1001", CreditCode: "4-1001"},
	CatPendapatanJasa:   {DebitCode: "1-1001", CreditCode: "4-1002"},
	CatPendapatanBunga:  {DebitCode: "1-1002", CreditCode: "4-2001"},
	CatPiutangDibayar:   {DebitCode: "1-1001", CreditCode: "1-1003"},
	CatHutangDiterima:   {DebitCode: "1-1001", CreditCode: "2-1001"},
	CatModalDisetor:     {DebitCode: "1-1001", CreditCode: "3-1001"},
	CatUangMukaDiterima: {DebitCode: "1-1001", CreditCode: "2-1006"},
	CatPendapatanLain:   {DebitCode: "1-1001", CreditCode: "4-2002"},
}

// --- CASH_OUT Rules ---
// Direction: money goes OUT → Debit varies, Credit Kas

var CashOutRules = map[string]JournalRule{
	CatBebanGaji:         {DebitCode: "5-1001", CreditCode: "1-1001"},
	CatBebanSewa:         {DebitCode: "5-1002", CreditCode: "1-1001"},
	CatBebanListrik:      {DebitCode: "5-1003", CreditCode: "1-1001"},
	CatBebanTelepon:      {DebitCode: "5-1004", CreditCode: "1-1001"},
	CatBebanTransport:    {DebitCode: "5-1005", CreditCode: "1-1001"},
	CatBebanMakan:        {DebitCode: "5-1006", CreditCode: "1-1001"},
	CatBebanPerlengkapan: {DebitCode: "5-1007", CreditCode: "1-1001"},
	CatBebanAsuransi:     {DebitCode: "5-1009", CreditCode: "1-1001"},
	CatBebanAdmin:        {DebitCode: "5-1010", CreditCode: "1-1001"},
	CatBebanBank:         {DebitCode: "5-1011", CreditCode: "1-1002"},
	CatBebanPemasaran:    {DebitCode: "5-1012", CreditCode: "1-1001"},
	CatBebanBunga:        {DebitCode: "5-2001", CreditCode: "1-1001"},
	CatBebanPajak:        {DebitCode: "5-2002", CreditCode: "1-1001"},
	CatPembelianBarang:   {DebitCode: "1-1004", CreditCode: "1-1001"},
	CatBayarHutang:       {DebitCode: "2-1001", CreditCode: "1-1001"},
	CatBayarPajak:        {DebitCode: "2-1003", CreditCode: "1-1001"},
	CatUangMukaBeli:      {DebitCode: "1-1007", CreditCode: "1-1001"},
	CatPrive:             {DebitCode: "3-1002", CreditCode: "1-1001"},
	CatBebanLain:         {DebitCode: "5-9001", CreditCode: "1-1001"},
}

// --- SALES Rules ---
// Penjualan: Debit Kas(tunai)/Piutang(kredit), Credit Pendapatan
// + HPP entry for barang: Debit HPP, Credit Persediaan

var SalesRules = map[string]MultiJournalRule{
	CatPenjualanBarangTunai: {
		Primary:    JournalRule{DebitCode: "1-1001", CreditCode: "4-1001"},
		Additional: []JournalRule{{DebitCode: "5-3001", CreditCode: "1-1004"}}, // HPP
	},
	CatPenjualanBarangKredit: {
		Primary:    JournalRule{DebitCode: "1-1003", CreditCode: "4-1001"},
		Additional: []JournalRule{{DebitCode: "5-3001", CreditCode: "1-1004"}}, // HPP
	},
	CatPenjualanJasaTunai: {
		Primary:    JournalRule{DebitCode: "1-1001", CreditCode: "4-1002"},
		Additional: nil,
	},
	CatPenjualanJasaKredit: {
		Primary:    JournalRule{DebitCode: "1-1003", CreditCode: "4-1002"},
		Additional: nil,
	},
	CatPenjualanDenganPPN: {
		Primary:    JournalRule{DebitCode: "1-1001", CreditCode: "4-1001"},
		Additional: []JournalRule{{DebitCode: "1-1001", CreditCode: "2-1005"}}, // PPN Keluaran
	},
}

// --- PURCHASE Rules ---
// Pembelian: Debit Persediaan/Beban, Credit Kas(tunai)/Hutang(kredit)

var PurchaseRules = map[string]MultiJournalRule{
	CatPembelianBarangTunai: {
		Primary:    JournalRule{DebitCode: "1-1004", CreditCode: "1-1001"},
		Additional: nil,
	},
	CatPembelianBarangKredit: {
		Primary:    JournalRule{DebitCode: "1-1004", CreditCode: "2-1001"},
		Additional: nil,
	},
	CatPembelianJasaTunai: {
		Primary:    JournalRule{DebitCode: "5-9001", CreditCode: "1-1001"}, // Default to beban lain, overridden by item.account_id
		Additional: nil,
	},
	CatPembelianJasaKredit: {
		Primary:    JournalRule{DebitCode: "5-9001", CreditCode: "2-1001"}, // Default to beban lain, overridden by item.account_id
		Additional: nil,
	},
	CatPembelianDenganPPN: {
		Primary:    JournalRule{DebitCode: "1-1004", CreditCode: "1-1001"},
		Additional: []JournalRule{{DebitCode: "1-1008", CreditCode: "1-1001"}}, // PPN Masukan
	},
}

// --- SPECIAL Rules ---

var SpecialRules = map[string]MultiJournalRule{
	CatDiskonPenjualan: {
		Primary:    JournalRule{DebitCode: "4-1003", CreditCode: "1-1003"}, // Diskon Penjualan (contra) → reduce Piutang
		Additional: nil,
	},
	CatReturPenjualan: {
		Primary:    JournalRule{DebitCode: "4-1004", CreditCode: "1-1003"},          // Retur Penjualan → reduce Piutang
		Additional: []JournalRule{{DebitCode: "1-1004", CreditCode: "5-3001"}}, // Restock: Persediaan naik, HPP turun
	},
	CatReturPembelian: {
		Primary:    JournalRule{DebitCode: "2-1001", CreditCode: "5-3002"}, // Reduce Hutang, Credit Retur Pembelian (contra-expense)
		Additional: nil,
	},
}

// GetJournalRule returns the journal rule(s) for a given transaction type and category.
// Returns nil if no rule found (invalid combination).
func GetJournalRule(txType, category string) *MultiJournalRule {
	switch txType {
	case TxTypeCashIn:
		if rule, ok := CashInRules[category]; ok {
			return &MultiJournalRule{Primary: rule}
		}
	case TxTypeCashOut:
		if rule, ok := CashOutRules[category]; ok {
			return &MultiJournalRule{Primary: rule}
		}
	case TxTypeSales:
		if rule, ok := SalesRules[category]; ok {
			return &rule
		}
	case TxTypePurchase:
		if rule, ok := PurchaseRules[category]; ok {
			return &rule
		}
	}

	// Check special rules (applicable to any type context)
	if rule, ok := SpecialRules[category]; ok {
		return &rule
	}

	return nil
}

// GetFallbackCategory returns the default fallback category for a transaction type
// Used when AI confidence is too low or AI returns invalid category
func GetFallbackCategory(txType string) string {
	switch txType {
	case TxTypeCashIn:
		return CatPendapatanLain
	case TxTypeCashOut:
		return CatBebanLain
	case TxTypeSales:
		return CatPenjualanBarangTunai
	case TxTypePurchase:
		return CatPembelianBarangTunai
	default:
		return CatBebanLain
	}
}
