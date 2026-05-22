package accounting

import "github.com/google/uuid"

// COATemplateEntry represents a single account in the default COA template
type COATemplateEntry struct {
	Code          string
	Name          string
	AccountType   string
	NormalBalance string
	Level         int
	ParentCode    string // empty for top-level
	IsSystem      bool
}

// DefaultCOATemplate returns the SAK EMKM + SAK ETAP compatible Chart of Accounts template.
// Covers: Orang Pribadi, UMKM, and Enterprise (PKP).
func DefaultCOATemplate() []COATemplateEntry {
	return []COATemplateEntry{
		// ============================================================
		// ASET (ASSET)
		// ============================================================
		{Code: "1-0000", Name: "Aset", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 1, ParentCode: "", IsSystem: true},

		// Aset Lancar
		{Code: "1-1000", Name: "Aset Lancar", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 2, ParentCode: "1-0000", IsSystem: true},
		{Code: "1-1001", Name: "Kas", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "1-1000", IsSystem: true},
		{Code: "1-1002", Name: "Bank", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "1-1000", IsSystem: true},
		{Code: "1-1003", Name: "Piutang Usaha", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "1-1000", IsSystem: true},
		{Code: "1-1004", Name: "Persediaan Barang", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "1-1000", IsSystem: true},
		{Code: "1-1005", Name: "Piutang Lain-lain", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "1-1000", IsSystem: true},
		{Code: "1-1006", Name: "Perlengkapan", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "1-1000", IsSystem: true},
		{Code: "1-1007", Name: "Uang Muka Pembelian", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "1-1000", IsSystem: true},
		{Code: "1-1008", Name: "PPN Masukan", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "1-1000", IsSystem: true},
		{Code: "1-1009", Name: "Biaya Dibayar di Muka", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "1-1000", IsSystem: true},

		// Aset Tetap
		{Code: "1-2000", Name: "Aset Tetap", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 2, ParentCode: "1-0000", IsSystem: true},
		{Code: "1-2001", Name: "Peralatan", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "1-2000", IsSystem: true},
		{Code: "1-2002", Name: "Kendaraan", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "1-2000", IsSystem: true},
		{Code: "1-2003", Name: "Bangunan", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "1-2000", IsSystem: true},
		{Code: "1-2004", Name: "Tanah", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "1-2000", IsSystem: true},
		{Code: "1-2099", Name: "Akumulasi Penyusutan", AccountType: AccountTypeAsset, NormalBalance: NormalBalanceCredit, Level: 3, ParentCode: "1-2000", IsSystem: true},

		// ============================================================
		// LIABILITAS (LIABILITY)
		// ============================================================
		{Code: "2-0000", Name: "Liabilitas", AccountType: AccountTypeLiability, NormalBalance: NormalBalanceCredit, Level: 1, ParentCode: "", IsSystem: true},

		// Hutang Lancar
		{Code: "2-1000", Name: "Hutang Lancar", AccountType: AccountTypeLiability, NormalBalance: NormalBalanceCredit, Level: 2, ParentCode: "2-0000", IsSystem: true},
		{Code: "2-1001", Name: "Hutang Usaha", AccountType: AccountTypeLiability, NormalBalance: NormalBalanceCredit, Level: 3, ParentCode: "2-1000", IsSystem: true},
		{Code: "2-1002", Name: "Hutang Gaji", AccountType: AccountTypeLiability, NormalBalance: NormalBalanceCredit, Level: 3, ParentCode: "2-1000", IsSystem: true},
		{Code: "2-1003", Name: "Hutang Pajak", AccountType: AccountTypeLiability, NormalBalance: NormalBalanceCredit, Level: 3, ParentCode: "2-1000", IsSystem: true},
		{Code: "2-1004", Name: "Pendapatan Diterima di Muka", AccountType: AccountTypeLiability, NormalBalance: NormalBalanceCredit, Level: 3, ParentCode: "2-1000", IsSystem: true},
		{Code: "2-1005", Name: "PPN Keluaran", AccountType: AccountTypeLiability, NormalBalance: NormalBalanceCredit, Level: 3, ParentCode: "2-1000", IsSystem: true},
		{Code: "2-1006", Name: "Uang Muka Penjualan", AccountType: AccountTypeLiability, NormalBalance: NormalBalanceCredit, Level: 3, ParentCode: "2-1000", IsSystem: true},
		{Code: "2-1007", Name: "Hutang Lain-lain", AccountType: AccountTypeLiability, NormalBalance: NormalBalanceCredit, Level: 3, ParentCode: "2-1000", IsSystem: true},

		// Hutang Jangka Panjang
		{Code: "2-2000", Name: "Hutang Jangka Panjang", AccountType: AccountTypeLiability, NormalBalance: NormalBalanceCredit, Level: 2, ParentCode: "2-0000", IsSystem: true},
		{Code: "2-2001", Name: "Hutang Bank", AccountType: AccountTypeLiability, NormalBalance: NormalBalanceCredit, Level: 3, ParentCode: "2-2000", IsSystem: true},

		// ============================================================
		// EKUITAS (EQUITY)
		// ============================================================
		{Code: "3-0000", Name: "Ekuitas", AccountType: AccountTypeEquity, NormalBalance: NormalBalanceCredit, Level: 1, ParentCode: "", IsSystem: true},
		{Code: "3-1001", Name: "Modal Pemilik", AccountType: AccountTypeEquity, NormalBalance: NormalBalanceCredit, Level: 2, ParentCode: "3-0000", IsSystem: true},
		{Code: "3-1002", Name: "Prive", AccountType: AccountTypeEquity, NormalBalance: NormalBalanceDebit, Level: 2, ParentCode: "3-0000", IsSystem: true},
		{Code: "3-1003", Name: "Laba Ditahan", AccountType: AccountTypeEquity, NormalBalance: NormalBalanceCredit, Level: 2, ParentCode: "3-0000", IsSystem: true},
		{Code: "3-1004", Name: "Laba Periode Berjalan", AccountType: AccountTypeEquity, NormalBalance: NormalBalanceCredit, Level: 2, ParentCode: "3-0000", IsSystem: true},

		// ============================================================
		// PENDAPATAN (REVENUE)
		// ============================================================
		{Code: "4-0000", Name: "Pendapatan", AccountType: AccountTypeRevenue, NormalBalance: NormalBalanceCredit, Level: 1, ParentCode: "", IsSystem: true},
		{Code: "4-1001", Name: "Pendapatan Usaha", AccountType: AccountTypeRevenue, NormalBalance: NormalBalanceCredit, Level: 2, ParentCode: "4-0000", IsSystem: true},
		{Code: "4-1002", Name: "Pendapatan Jasa", AccountType: AccountTypeRevenue, NormalBalance: NormalBalanceCredit, Level: 2, ParentCode: "4-0000", IsSystem: true},
		{Code: "4-1003", Name: "Diskon Penjualan", AccountType: AccountTypeRevenue, NormalBalance: NormalBalanceDebit, Level: 2, ParentCode: "4-0000", IsSystem: true},
		{Code: "4-1004", Name: "Retur Penjualan", AccountType: AccountTypeRevenue, NormalBalance: NormalBalanceDebit, Level: 2, ParentCode: "4-0000", IsSystem: true},
		{Code: "4-2001", Name: "Pendapatan Bunga", AccountType: AccountTypeRevenue, NormalBalance: NormalBalanceCredit, Level: 2, ParentCode: "4-0000", IsSystem: true},
		{Code: "4-2002", Name: "Pendapatan Lain-lain", AccountType: AccountTypeRevenue, NormalBalance: NormalBalanceCredit, Level: 2, ParentCode: "4-0000", IsSystem: true},

		// ============================================================
		// BEBAN (EXPENSE)
		// ============================================================
		{Code: "5-0000", Name: "Beban", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 1, ParentCode: "", IsSystem: true},

		// Beban Operasional
		{Code: "5-1000", Name: "Beban Operasional", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 2, ParentCode: "5-0000", IsSystem: true},
		{Code: "5-1001", Name: "Beban Gaji & Tunjangan", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-1000", IsSystem: true},
		{Code: "5-1002", Name: "Beban Sewa", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-1000", IsSystem: true},
		{Code: "5-1003", Name: "Beban Listrik & Air", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-1000", IsSystem: true},
		{Code: "5-1004", Name: "Beban Telepon & Internet", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-1000", IsSystem: true},
		{Code: "5-1005", Name: "Beban Transportasi", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-1000", IsSystem: true},
		{Code: "5-1006", Name: "Beban Makan & Minum", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-1000", IsSystem: true},
		{Code: "5-1007", Name: "Beban Perlengkapan", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-1000", IsSystem: true},
		{Code: "5-1008", Name: "Beban Penyusutan", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-1000", IsSystem: true},
		{Code: "5-1009", Name: "Beban Asuransi", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-1000", IsSystem: true},
		{Code: "5-1010", Name: "Beban Administrasi & Umum", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-1000", IsSystem: true},
		{Code: "5-1011", Name: "Beban Biaya Bank", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-1000", IsSystem: true},
		{Code: "5-1012", Name: "Beban Pemasaran & Iklan", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-1000", IsSystem: true},

		// Beban Non-Operasional
		{Code: "5-2000", Name: "Beban Non-Operasional", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 2, ParentCode: "5-0000", IsSystem: true},
		{Code: "5-2001", Name: "Beban Bunga Pinjaman", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-2000", IsSystem: true},
		{Code: "5-2002", Name: "Beban Pajak", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-2000", IsSystem: true},
		{Code: "5-2003", Name: "Beban Denda & Penalti", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-2000", IsSystem: true},
		{Code: "5-2004", Name: "Kerugian Lain-lain", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-2000", IsSystem: true},

		// Harga Pokok
		{Code: "5-3000", Name: "Harga Pokok", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 2, ParentCode: "5-0000", IsSystem: true},
		{Code: "5-3001", Name: "Harga Pokok Penjualan", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 3, ParentCode: "5-3000", IsSystem: true},
		{Code: "5-3002", Name: "Retur Pembelian", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceCredit, Level: 3, ParentCode: "5-3000", IsSystem: true},

		// Beban Lain-lain (catch-all)
		{Code: "5-9001", Name: "Beban Lain-lain", AccountType: AccountTypeExpense, NormalBalance: NormalBalanceDebit, Level: 2, ParentCode: "5-0000", IsSystem: true},
	}
}

// COACodeToID is a helper map built during seeding to resolve parent references
type COACodeToID map[string]uuid.UUID
