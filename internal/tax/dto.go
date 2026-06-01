package tax

type UpsertProfileRequest struct {
	NPWP             string  `json:"npwp,omitempty" example:"01.234.567.8-901.000"`
	TaxStatus        string  `json:"tax_status" example:"PKP"`
	IsPPNLiable      bool    `json:"is_ppn_liable" example:"true"`
	DefaultPPNRate   float64 `json:"default_ppn_rate,omitempty" example:"0.11"`
	PPh23Enabled     bool    `json:"pph23_enabled" example:"false"`
	DefaultPPh23Rate float64 `json:"default_pph23_rate,omitempty" example:"0.02"`
	PKPNumber        string  `json:"pkp_number,omitempty" example:"123.456-78.901234"`
	TaxOfficeCode    string  `json:"tax_office_code,omitempty" example:"061"`
	EFakturReady     bool    `json:"efaktur_ready" example:"false"`
	EBupotReady      bool    `json:"ebupot_ready" example:"false"`
	Notes            string  `json:"notes,omitempty"`
}

type ProfileResponse struct {
	ID               string  `json:"id"`
	WorkspaceID      string  `json:"workspace_id"`
	EntityID         string  `json:"entity_id"`
	NPWP             string  `json:"npwp,omitempty"`
	TaxStatus        string  `json:"tax_status"`
	IsPPNLiable      bool    `json:"is_ppn_liable"`
	DefaultPPNRate   float64 `json:"default_ppn_rate"`
	PPh23Enabled     bool    `json:"pph23_enabled"`
	DefaultPPh23Rate float64 `json:"default_pph23_rate"`
	PKPNumber        string  `json:"pkp_number,omitempty"`
	TaxOfficeCode    string  `json:"tax_office_code,omitempty"`
	EFakturReady     bool    `json:"efaktur_ready"`
	EBupotReady      bool    `json:"ebupot_ready"`
	Notes            string  `json:"notes,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

type CalculationResponse struct {
	ID                     string  `json:"id"`
	WorkspaceID            string  `json:"workspace_id"`
	TransactionID          string  `json:"transaction_id"`
	TransactionNumber      string  `json:"transaction_number,omitempty"`
	TransactionType        string  `json:"transaction_type,omitempty"`
	TransactionDescription string  `json:"transaction_description,omitempty"`
	TaxType                string  `json:"tax_type"`
	Direction              string  `json:"direction"`
	BaseAmount             string  `json:"base_amount"`
	TaxRate                float64 `json:"tax_rate"`
	TaxAmount              string  `json:"tax_amount"`
	Period                 string  `json:"period"`
	Status                 string  `json:"status"`
	CounterpartyEntityID   string  `json:"counterparty_entity_id,omitempty"`
	FakturNumber           string  `json:"faktur_number,omitempty"`
	CreatedAt              string  `json:"created_at"`
}

type PPNSummaryResponse struct {
	Period           string `json:"period"`
	PPNMasukan       string `json:"ppn_masukan"`
	PPNKeluaran      string `json:"ppn_keluaran"`
	NetPPN           string `json:"net_ppn"`
	DPPMasukan       string `json:"dpp_masukan"`
	DPPKeluaran      string `json:"dpp_keluaran"`
	TransactionCount int64  `json:"transaction_count"`
}

type PPhSummaryRow struct {
	TaxType    string `json:"tax_type"`
	Direction  string `json:"direction"`
	TotalBase  string `json:"total_base"`
	TotalTax   string `json:"total_tax"`
	Count      int64  `json:"count"`
}

type PPhSummaryResponse struct {
	PeriodFrom string          `json:"period_from"`
	PeriodTo   string          `json:"period_to"`
	Rows       []PPhSummaryRow `json:"rows"`
}

type LinkDocumentRequest struct {
	DocumentID string `json:"document_id"`
	RefType    string `json:"ref_type" example:"FAKTUR_PAJAK"`
}

type DocumentRefResponse struct {
	ID           string `json:"id"`
	DocumentID   string `json:"document_id"`
	RefType      string `json:"ref_type"`
	FileName     string `json:"file_name"`
	DocumentType string `json:"document_type"`
	CreatedAt    string `json:"created_at"`
}

type RequestReportRequest struct {
	ReportType string `json:"report_type" example:"TAX_OVERVIEW"`
	PeriodFrom string `json:"period_from" example:"2026-01"`
	PeriodTo   string `json:"period_to" example:"2026-03"`
}

type ReportJobResponse struct {
	ID          string      `json:"id"`
	ReportType  string      `json:"report_type"`
	PeriodFrom  string      `json:"period_from"`
	PeriodTo    string      `json:"period_to"`
	Status      string      `json:"status"`
	Result      interface{} `json:"result,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt   string      `json:"created_at"`
	CompletedAt string      `json:"completed_at,omitempty"`
}
