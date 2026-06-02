package tax

import (
	"codeberg.org/azzet/azzetbe/internal/accounting"
	"codeberg.org/azzet/azzetbe/internal/db"
)

type CalcInput struct {
	TransactionType string
	Category        string
	Amount          float64
	IncludesTax     bool
	PPNRate         float64
	PPh23Rate       float64
	PPh23Enabled    bool
	IsPPNLiable     bool
}

type CalcResult struct {
	TaxType   string
	Direction string
	Base      float64
	Rate      float64
	Amount    float64
}

func ComputeTaxes(input CalcInput) []CalcResult {
	ppnRate := input.PPNRate
	if ppnRate <= 0 {
		ppnRate = DefaultPPNRate
	}

	var results []CalcResult

	if ppn := computePPN(input, ppnRate); ppn != nil {
		results = append(results, *ppn)
	}

	if pph := computePPh23(input); pph != nil {
		results = append(results, *pph)
	}

	return results
}

func computePPN(input CalcInput, rate float64) *CalcResult {
	category := input.Category
	hasPPNSignal := input.IncludesTax ||
		category == accounting.CatPenjualanDenganPPN ||
		category == accounting.CatPembelianDenganPPN
	if !hasPPNSignal {
		return nil
	}
	if !input.IsPPNLiable && !input.IncludesTax {
		return nil
	}

	isSalesPPN := category == accounting.CatPenjualanDenganPPN ||
		(input.IncludesTax && (input.TransactionType == accounting.TxTypeSales || input.TransactionType == accounting.TxTypeCashIn))
	isPurchasePPN := category == accounting.CatPembelianDenganPPN ||
		(input.IncludesTax && input.TransactionType == accounting.TxTypePurchase)

	if !isSalesPPN && !isPurchasePPN {
		return nil
	}

	base, tax := splitTaxAmount(input.Amount, rate, input.IncludesTax || category == accounting.CatPenjualanDenganPPN || category == accounting.CatPembelianDenganPPN)

	if isSalesPPN {
		return &CalcResult{
			TaxType:   TaxTypePPNKeluaran,
			Direction: DirectionOutput,
			Base:      base,
			Rate:      rate,
			Amount:    tax,
		}
	}

	if isPurchasePPN {
		return &CalcResult{
			TaxType:   TaxTypePPNMasukan,
			Direction: DirectionInput,
			Base:      base,
			Rate:      rate,
			Amount:    tax,
		}
	}

	return nil
}

func computePPh23(input CalcInput) *CalcResult {
	if !input.PPh23Enabled {
		return nil
	}

	rate := input.PPh23Rate
	if rate <= 0 {
		rate = DefaultPPh23Rate
	}

	category := input.Category
	isPurchaseJasa := input.TransactionType == accounting.TxTypePurchase &&
		(category == accounting.CatPembelianJasaTunai || category == accounting.CatPembelianJasaKredit)
	isSalesJasa := input.TransactionType == accounting.TxTypeSales &&
		(category == accounting.CatPenjualanJasaTunai || category == accounting.CatPenjualanJasaKredit)

	if !isPurchaseJasa && !isSalesJasa {
		return nil
	}

	ppnRate := input.PPNRate
	if ppnRate <= 0 {
		ppnRate = DefaultPPNRate
	}
	base := input.Amount
	if input.IncludesTax {
		base, _ = splitTaxAmount(input.Amount, ppnRate, true)
	}
	tax := base * rate

	if isPurchaseJasa {
		return &CalcResult{
			TaxType:   TaxTypePPh23,
			Direction: DirectionWithholding,
			Base:      base,
			Rate:      rate,
			Amount:    tax,
		}
	}

	// Sales jasa: informational — customer withholds from payment
	return &CalcResult{
		TaxType:   TaxTypePPh23,
		Direction: DirectionOutput,
		Base:      base,
		Rate:      rate,
		Amount:    tax,
	}
}

func splitTaxAmount(total, rate float64, taxInclusive bool) (base, tax float64) {
	if taxInclusive {
		base = total / (1 + rate)
		tax = total - base
		return base, tax
	}
	base = total
	tax = total * rate
	return base, tax
}

func profileFromRow(p db.TaxProfile) ProfileResponse {
	return ProfileResponse{
		ID:               p.ID.String(),
		WorkspaceID:      p.WorkspaceID.String(),
		EntityID:         p.EntityID.String(),
		NPWP:             pgtextToString(p.Npwp),
		TaxStatus:        p.TaxStatus,
		IsPPNLiable:      p.IsPpnLiable,
		DefaultPPNRate:   numericToFloat(p.DefaultPpnRate),
		PPh23Enabled:     p.Pph23Enabled,
		DefaultPPh23Rate: numericToFloat(p.DefaultPph23Rate),
		PKPNumber:        pgtextToString(p.PkpNumber),
		TaxOfficeCode:    pgtextToString(p.TaxOfficeCode),
		EFakturReady:     p.EfakturReady,
		EBupotReady:      p.EbupotReady,
		Notes:            pgtextToString(p.Notes),
		CreatedAt:        p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func calculationFromRow(row db.ListTaxCalculationsRow) CalculationResponse {
	return CalculationResponse{
		ID:                     row.ID.String(),
		WorkspaceID:            row.WorkspaceID.String(),
		TransactionID:          row.TransactionID.String(),
		TransactionNumber:      row.TransactionNumber,
		TransactionType:        row.TransactionType,
		TransactionDescription: pgtextToString(row.TransactionDescription),
		TaxType:                row.TaxType,
		Direction:              row.Direction,
		BaseAmount:             numericToString(row.BaseAmount),
		TaxRate:                numericToFloat(row.TaxRate),
		TaxAmount:              numericToString(row.TaxAmount),
		Period:                 row.Period,
		Status:                 row.Status,
		CounterpartyEntityID:   pgtypeUUIDToString(row.CounterpartyEntityID),
		FakturNumber:           pgtextToString(row.FakturNumber),
		CreatedAt:              row.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func calculationFromEntity(c db.TaxCalculation) CalculationResponse {
	return CalculationResponse{
		ID:                   c.ID.String(),
		WorkspaceID:          c.WorkspaceID.String(),
		TransactionID:        c.TransactionID.String(),
		TaxType:              c.TaxType,
		Direction:            c.Direction,
		BaseAmount:           numericToString(c.BaseAmount),
		TaxRate:              numericToFloat(c.TaxRate),
		TaxAmount:            numericToString(c.TaxAmount),
		Period:               c.Period,
		Status:               c.Status,
		CounterpartyEntityID: pgtypeUUIDToString(c.CounterpartyEntityID),
		FakturNumber:         pgtextToString(c.FakturNumber),
		CreatedAt:            c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
