package accounting

// --- Account Types ---

const (
	AccountTypeAsset     = "ASSET"
	AccountTypeLiability = "LIABILITY"
	AccountTypeEquity    = "EQUITY"
	AccountTypeRevenue   = "REVENUE"
	AccountTypeExpense   = "EXPENSE"
)

var ValidAccountTypes = []string{
	AccountTypeAsset,
	AccountTypeLiability,
	AccountTypeEquity,
	AccountTypeRevenue,
	AccountTypeExpense,
}

// --- Normal Balance ---

const (
	NormalBalanceDebit  = "DEBIT"
	NormalBalanceCredit = "CREDIT"
)

// --- Transaction Types ---

const (
	TxTypeCashIn   = "CASH_IN"
	TxTypeCashOut  = "CASH_OUT"
	TxTypeSales    = "SALES"
	TxTypePurchase = "PURCHASE"
	TxTypeJournal  = "JOURNAL"
	TxTypeReversal = "REVERSAL"
)

var ValidTransactionTypes = []string{
	TxTypeCashIn,
	TxTypeCashOut,
	TxTypeSales,
	TxTypePurchase,
	TxTypeJournal,
	TxTypeReversal,
}

// --- Input Modes ---

const (
	InputModeSimple   = "SIMPLE"
	InputModeAdvanced = "ADVANCED"
	InputModeOCR      = "OCR"
)

var ValidInputModes = []string{
	InputModeSimple,
	InputModeAdvanced,
	InputModeOCR,
}

// --- Transaction Status ---

const (
	TxStatusDraft  = "DRAFT"
	TxStatusPosted = "POSTED"
	TxStatusVoid   = "VOID"
	TxStatusFailed = "FAILED"
)

// --- Payment Methods ---

const (
	PaymentMethodTunai    = "TUNAI"
	PaymentMethodKredit   = "KREDIT"
	PaymentMethodTransfer = "TRANSFER"
)

var ValidPaymentMethods = []string{
	PaymentMethodTunai,
	PaymentMethodKredit,
	PaymentMethodTransfer,
}

// --- Item Types ---

const (
	ItemTypeBarang      = "BARANG"
	ItemTypeJasa        = "JASA"
	ItemTypeProyek      = "PROYEK"
	ItemTypeAHSPRakitan = "AHSP_RAKITAN"
)

var ValidItemTypes = []string{
	ItemTypeBarang,
	ItemTypeJasa,
	ItemTypeProyek,
	ItemTypeAHSPRakitan,
}

// --- Unit Types ---

var ValidUnits = []string{
	"Pcs", "Kg", "Liter", "Meter", "M2", "M3",
	"Jam", "Hari", "Paket", "Unit", "Box", "Lusin", "Set", "Rim",
}

// --- AI Categories (Strict Whitelist) ---
// AI output MUST be one of these. Backend validates against this list.

// CASH_IN categories
const (
	CatPendapatanUsaha   = "pendapatan_usaha"
	CatPendapatanJasa    = "pendapatan_jasa"
	CatPendapatanBunga   = "pendapatan_bunga"
	CatPiutangDibayar    = "piutang_dibayar"
	CatHutangDiterima    = "hutang_diterima"
	CatModalDisetor      = "modal_disetor"
	CatUangMukaDiterima  = "uang_muka_diterima"
	CatPendapatanLain    = "pendapatan_lain"
)

// CASH_OUT categories
const (
	CatBebanGaji         = "beban_gaji"
	CatBebanSewa         = "beban_sewa"
	CatBebanListrik      = "beban_listrik"
	CatBebanTelepon      = "beban_telepon"
	CatBebanTransport    = "beban_transport"
	CatBebanMakan        = "beban_makan"
	CatBebanPerlengkapan = "beban_perlengkapan"
	CatBebanAsuransi     = "beban_asuransi"
	CatBebanAdmin        = "beban_admin"
	CatBebanBank         = "beban_bank"
	CatBebanPemasaran    = "beban_pemasaran"
	CatBebanBunga        = "beban_bunga"
	CatBebanPajak        = "beban_pajak"
	CatPembelianBarang   = "pembelian_barang"
	CatBayarHutang       = "bayar_hutang"
	CatBayarPajak        = "bayar_pajak"
	CatUangMukaBeli      = "uang_muka_beli"
	CatPrive             = "prive"
	CatBebanLain         = "beban_lain"
)

// SALES categories
const (
	CatPenjualanBarangTunai  = "penjualan_barang_tunai"
	CatPenjualanBarangKredit = "penjualan_barang_kredit"
	CatPenjualanJasaTunai    = "penjualan_jasa_tunai"
	CatPenjualanJasaKredit   = "penjualan_jasa_kredit"
	CatPenjualanDenganPPN    = "penjualan_dengan_ppn"
)

// PURCHASE categories
const (
	CatPembelianBarangTunai  = "pembelian_barang_tunai"
	CatPembelianBarangKredit = "pembelian_barang_kredit"
	CatPembelianJasaTunai    = "pembelian_jasa_tunai"
	CatPembelianJasaKredit   = "pembelian_jasa_kredit"
	CatPembelianDenganPPN    = "pembelian_dengan_ppn"
)

// SPECIAL categories
const (
	CatDiskonPenjualan = "diskon_penjualan"
	CatReturPenjualan  = "retur_penjualan"
	CatReturPembelian  = "retur_pembelian"
)

// ValidCashInCategories - all valid categories for CASH_IN transactions
var ValidCashInCategories = []string{
	CatPendapatanUsaha, CatPendapatanJasa, CatPendapatanBunga,
	CatPiutangDibayar, CatHutangDiterima, CatModalDisetor,
	CatUangMukaDiterima, CatPendapatanLain,
}

// ValidCashOutCategories - all valid categories for CASH_OUT transactions
var ValidCashOutCategories = []string{
	CatBebanGaji, CatBebanSewa, CatBebanListrik, CatBebanTelepon,
	CatBebanTransport, CatBebanMakan, CatBebanPerlengkapan,
	CatBebanAsuransi, CatBebanAdmin, CatBebanBank, CatBebanPemasaran,
	CatBebanBunga, CatBebanPajak, CatPembelianBarang, CatBayarHutang,
	CatBayarPajak, CatUangMukaBeli, CatPrive, CatBebanLain,
}

// ValidSalesCategories - all valid categories for SALES transactions
var ValidSalesCategories = []string{
	CatPenjualanBarangTunai, CatPenjualanBarangKredit,
	CatPenjualanJasaTunai, CatPenjualanJasaKredit,
	CatPenjualanDenganPPN,
}

// ValidPurchaseCategories - all valid categories for PURCHASE transactions
var ValidPurchaseCategories = []string{
	CatPembelianBarangTunai, CatPembelianBarangKredit,
	CatPembelianJasaTunai, CatPembelianJasaKredit,
	CatPembelianDenganPPN,
}

// ValidSpecialCategories - special transaction categories
var ValidSpecialCategories = []string{
	CatDiskonPenjualan, CatReturPenjualan, CatReturPembelian,
}

// AllValidCategories - complete whitelist for AI validation
var AllValidCategories = func() []string {
	all := make([]string, 0, 40)
	all = append(all, ValidCashInCategories...)
	all = append(all, ValidCashOutCategories...)
	all = append(all, ValidSalesCategories...)
	all = append(all, ValidPurchaseCategories...)
	all = append(all, ValidSpecialCategories...)
	return all
}()

// IsValidCategory checks if a category string is in the whitelist
func IsValidCategory(category string) bool {
	for _, c := range AllValidCategories {
		if c == category {
			return true
		}
	}
	return false
}

// IsValidCategoryForType checks if a category is valid for a given transaction type
func IsValidCategoryForType(category, txType string) bool {
	var valid []string
	switch txType {
	case TxTypeCashIn:
		valid = ValidCashInCategories
	case TxTypeCashOut:
		valid = ValidCashOutCategories
	case TxTypeSales:
		valid = ValidSalesCategories
	case TxTypePurchase:
		valid = ValidPurchaseCategories
	default:
		return false
	}
	for _, c := range valid {
		if c == category {
			return true
		}
	}
	return false
}

// --- Permission Keys ---

const (
	PermTransactionCreate = "transaction:create"
	PermTransactionRead   = "transaction:read"
	PermTransactionVoid   = "transaction:void"
	PermReportRead        = "report:read"
	PermReportExport      = "report:export"
)

// --- PPN Rate ---

const PPNRate = 0.11 // 11% PPN Indonesia (per 2022)

// IsValidTxType checks if a transaction type is valid.
func IsValidTxType(t string) bool {
	for _, v := range ValidTransactionTypes {
		if v == t {
			return true
		}
	}
	return false
}

// IsValidPaymentMethod checks if a payment method is valid.
func IsValidPaymentMethod(m string) bool {
	for _, v := range ValidPaymentMethods {
		if v == m {
			return true
		}
	}
	return false
}
