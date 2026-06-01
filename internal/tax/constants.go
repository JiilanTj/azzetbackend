package tax

const (
	TaxStatusNonPKP        = "NON_PKP"
	TaxStatusPKP           = "PKP"
	TaxStatusNotRegistered = "NOT_REGISTERED"

	TaxTypePPNMasukan  = "PPN_MASUKAN"
	TaxTypePPNKeluaran = "PPN_KELUARAN"
	TaxTypePPh21       = "PPH21"
	TaxTypePPh23       = "PPH23"

	DirectionInput       = "INPUT"
	DirectionOutput      = "OUTPUT"
	DirectionWithholding = "WITHHOLDING"

	CalcStatusActive   = "ACTIVE"
	CalcStatusVoided   = "VOIDED"
	CalcStatusReversed = "REVERSED"

	ReportTypePPNSummary  = "PPN_SUMMARY"
	ReportTypePPhSummary  = "PPH_SUMMARY"
	ReportTypeTaxOverview = "TAX_OVERVIEW"

	ReportStatusPending    = "PENDING"
	ReportStatusProcessing = "PROCESSING"
	ReportStatusCompleted  = "COMPLETED"
	ReportStatusFailed     = "FAILED"

	DocRefFakturPajak  = "FAKTUR_PAJAK"
	DocRefBuktiPotong  = "BUKTI_POTONG"
	DocRefInvoice      = "INVOICE"
	DocRefReceipt      = "RECEIPT"
	DocRefOther        = "OTHER"

	DefaultPPNRate  = 0.11
	DefaultPPh23Rate = 0.02
)
