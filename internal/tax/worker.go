package tax

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/accounting"
	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/events"
)

type Worker struct {
	Queries *db.Queries
	Service *Service
}

func NewWorker(queries *db.Queries, service *Service) *Worker {
	return &Worker{Queries: queries, Service: service}
}

type ledgerPostedPayload struct {
	TransactionID string `json:"transaction_id"`
	WorkspaceID   string `json:"workspace_id"`
	IsReversal    string `json:"is_reversal,omitempty"`
}

type reportPayload struct {
	JobID       string `json:"job_id"`
	WorkspaceID string `json:"workspace_id"`
	ReportType  string `json:"report_type"`
	PeriodFrom  string `json:"period_from"`
	PeriodTo    string `json:"period_to"`
}

func (w *Worker) HandleLedgerPosted(ctx context.Context, event *events.Event) error {
	var payload ledgerPostedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse ledger.posted payload: %w", err)
	}

	txID, err := uuid.Parse(payload.TransactionID)
	if err != nil {
		return fmt.Errorf("invalid transaction_id: %w", err)
	}
	workspaceID, err := uuid.Parse(payload.WorkspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace_id: %w", err)
	}

	if payload.IsReversal == "true" {
		return w.handleReversal(ctx, txID, workspaceID)
	}

	return w.calculateForTransaction(ctx, txID, workspaceID)
}

func (w *Worker) handleReversal(ctx context.Context, reversalTxID, workspaceID uuid.UUID) error {
	reversal, err := w.Queries.GetTransactionByID(ctx, db.GetTransactionByIDParams{
		ID:          reversalTxID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("reversal transaction not found: %w", err)
	}
	if !reversal.ReversedTransactionID.Valid {
		return nil
	}
	originalID := uuid.UUID(reversal.ReversedTransactionID.Bytes)
	return w.Queries.VoidTaxCalculationsByTransaction(ctx, db.VoidTaxCalculationsByTransactionParams{
		TransactionID: originalID,
		WorkspaceID:   workspaceID,
	})
}

func (w *Worker) calculateForTransaction(ctx context.Context, txID, workspaceID uuid.UUID) error {
	transaction, err := w.Queries.GetTransactionByID(ctx, db.GetTransactionByIDParams{
		ID:          txID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("transaction not found: %w", err)
	}

	if transaction.TransactionType == accounting.TxTypeReversal || transaction.TransactionType == accounting.TxTypeJournal {
		return nil
	}

	profileRow, err := w.Queries.GetTaxProfileByWorkspace(ctx, workspaceID)
	if err != nil {
		// No profile yet — use defaults for calculation
		profileRow = db.TaxProfile{
			IsPpnLiable:      false,
			DefaultPpnRate:   floatToNumeric(DefaultPPNRate),
			Pph23Enabled:     false,
			DefaultPph23Rate: floatToNumeric(DefaultPPh23Rate),
		}
	}

	amount := numericToFloat(transaction.Amount)
	category := pgtextToString(transaction.Category)

	calcs := ComputeTaxes(CalcInput{
		TransactionType: transaction.TransactionType,
		Category:        category,
		Amount:          amount,
		IncludesTax:     transaction.IncludesTax,
		PPNRate:         numericToFloat(profileRow.DefaultPpnRate),
		PPh23Rate:       numericToFloat(profileRow.DefaultPph23Rate),
		PPh23Enabled:    profileRow.Pph23Enabled,
		IsPPNLiable:     profileRow.IsPpnLiable,
	})

	if len(calcs) == 0 {
		return nil
	}

	period := transaction.TransactionDate.Time.Format("2006-01")
	now := time.Now()

	for _, calc := range calcs {
		if _, err := w.Queries.GetTaxCalculationByTransactionAndType(ctx, db.GetTaxCalculationByTransactionAndTypeParams{
			TransactionID: txID,
			TaxType:     calc.TaxType,
		}); err == nil {
			continue
		}

		meta, _ := json.Marshal(map[string]string{
			"transaction_type": transaction.TransactionType,
			"category":         category,
		})

		_, err := w.Queries.CreateTaxCalculation(ctx, db.CreateTaxCalculationParams{
			ID:                   uuid.New(),
			WorkspaceID:          workspaceID,
			TransactionID:        txID,
			TaxType:              calc.TaxType,
			Direction:            calc.Direction,
			BaseAmount:           floatToNumeric(calc.Base),
			TaxRate:              floatToNumeric(calc.Rate),
			TaxAmount:            floatToNumeric(calc.Amount),
			Period:               period,
			Status:               CalcStatusActive,
			CounterpartyEntityID: transaction.CounterpartyEntityID,
			FakturNumber:         pgtype.Text{},
			Metadata:             meta,
			CreatedAt:            now,
			UpdatedAt:            now,
		})
		if err != nil {
			return fmt.Errorf("failed to create tax calculation: %w", err)
		}

		slog.Info("tax calculated",
			"transaction_id", txID.String(),
			"tax_type", calc.TaxType,
			"amount", calc.Amount,
		)
	}

	return nil
}

func (w *Worker) HandleReportGeneration(ctx context.Context, event *events.Event) error {
	var payload reportPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse report payload: %w", err)
	}

	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		return fmt.Errorf("invalid job_id: %w", err)
	}
	workspaceID, err := uuid.Parse(payload.WorkspaceID)
	if err != nil {
		return fmt.Errorf("invalid workspace_id: %w", err)
	}

	if err := w.Queries.UpdateTaxReportJobProcessing(ctx, jobID); err != nil {
		return fmt.Errorf("failed to mark job processing: %w", err)
	}

	var result interface{}
	var genErr error

	switch payload.ReportType {
	case ReportTypePPNSummary:
		result, genErr = w.generatePPNReport(ctx, workspaceID, payload.PeriodFrom, payload.PeriodTo)
	case ReportTypePPhSummary:
		result, genErr = w.generatePPhReport(ctx, workspaceID, payload.PeriodFrom, payload.PeriodTo)
	case ReportTypeTaxOverview:
		result, genErr = w.generateOverviewReport(ctx, workspaceID, payload.PeriodFrom, payload.PeriodTo)
	default:
		genErr = ErrInvalidReportType
	}

	if genErr != nil {
		_ = w.Queries.FailTaxReportJob(ctx, db.FailTaxReportJobParams{
			ID:           jobID,
			ErrorMessage: pgtype.Text{String: genErr.Error(), Valid: true},
		})
		return genErr
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal report result: %w", err)
	}

	if err := w.Queries.CompleteTaxReportJob(ctx, db.CompleteTaxReportJobParams{
		ID:     jobID,
		Result: resultJSON,
	}); err != nil {
		return fmt.Errorf("failed to complete report job: %w", err)
	}

	slog.Info("tax report generated", "job_id", jobID.String(), "type", payload.ReportType)
	return nil
}

func (w *Worker) generatePPNReport(ctx context.Context, workspaceID uuid.UUID, periodFrom, periodTo string) (interface{}, error) {
	periods := periodsBetween(periodFrom, periodTo)
	summaries := make([]PPNSummaryResponse, 0, len(periods))
	for _, p := range periods {
		summary, err := w.Service.GetPPNSummary(ctx, workspaceID, p)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, *summary)
	}
	return map[string]interface{}{
		"period_from": periodFrom,
		"period_to":   periodTo,
		"periods":     summaries,
	}, nil
}

func (w *Worker) generatePPhReport(ctx context.Context, workspaceID uuid.UUID, periodFrom, periodTo string) (interface{}, error) {
	return w.Service.GetPPhSummary(ctx, workspaceID, periodFrom, periodTo)
}

func (w *Worker) generateOverviewReport(ctx context.Context, workspaceID uuid.UUID, periodFrom, periodTo string) (interface{}, error) {
	ppn, err := w.generatePPNReport(ctx, workspaceID, periodFrom, periodTo)
	if err != nil {
		return nil, err
	}
	pph, err := w.generatePPhReport(ctx, workspaceID, periodFrom, periodTo)
	if err != nil {
		return nil, err
	}
	profile, err := w.Service.GetOrCreateProfile(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"profile":     profile,
		"ppn":         ppn,
		"pph":         pph,
		"period_from": periodFrom,
		"period_to":   periodTo,
		"efaktur_ready": profile.EFakturReady,
		"ebupot_ready":  profile.EBupotReady,
	}, nil
}

func periodsBetween(from, to string) []string {
	start, err1 := time.Parse("2006-01", from)
	end, err2 := time.Parse("2006-01", to)
	if err1 != nil || err2 != nil || start.After(end) {
		return []string{from}
	}
	var periods []string
	for !start.After(end) {
		periods = append(periods, start.Format("2006-01"))
		start = start.AddDate(0, 1, 0)
	}
	return periods
}
