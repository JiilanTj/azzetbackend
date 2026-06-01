package tax

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/events"
)

type Service struct {
	Queries *db.Queries
	Pool    *pgxpool.Pool
}

func NewService(queries *db.Queries, pool *pgxpool.Pool) *Service {
	return &Service{Queries: queries, Pool: pool}
}

func (s *Service) GetOrCreateProfile(ctx context.Context, workspaceID uuid.UUID) (*ProfileResponse, error) {
	profile, err := s.Queries.GetTaxProfileByWorkspace(ctx, workspaceID)
	if err == nil {
		resp := profileFromRow(profile)
		return &resp, nil
	}

	now := time.Now()
	created, err := s.Queries.CreateTaxProfile(ctx, db.CreateTaxProfileParams{
		ID:               uuid.New(),
		WorkspaceID:      workspaceID,
		EntityID:         workspaceID,
		Npwp:             pgtype.Text{},
		TaxStatus:        TaxStatusNonPKP,
		IsPpnLiable:      false,
		DefaultPpnRate:   floatToNumeric(DefaultPPNRate),
		Pph23Enabled:     false,
		DefaultPph23Rate: floatToNumeric(DefaultPPh23Rate),
		PkpNumber:        pgtype.Text{},
		TaxOfficeCode:    pgtype.Text{},
		EfakturReady:     false,
		EbupotReady:      false,
		Notes:            pgtype.Text{},
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tax profile: %w", err)
	}
	resp := profileFromRow(created)
	return &resp, nil
}

func (s *Service) UpdateProfile(ctx context.Context, workspaceID uuid.UUID, req *UpsertProfileRequest) (*ProfileResponse, error) {
	if _, err := s.GetOrCreateProfile(ctx, workspaceID); err != nil {
		return nil, err
	}

	taxStatus := req.TaxStatus
	if taxStatus == "" {
		taxStatus = TaxStatusNonPKP
	}
	if taxStatus != TaxStatusNonPKP && taxStatus != TaxStatusPKP && taxStatus != TaxStatusNotRegistered {
		return nil, ErrInvalidTaxStatus
	}

	ppnRate := req.DefaultPPNRate
	if ppnRate <= 0 {
		ppnRate = DefaultPPNRate
	}
	pph23Rate := req.DefaultPPh23Rate
	if pph23Rate <= 0 {
		pph23Rate = DefaultPPh23Rate
	}

	isPPNLiable := req.IsPPNLiable || taxStatus == TaxStatusPKP

	updated, err := s.Queries.UpdateTaxProfile(ctx, db.UpdateTaxProfileParams{
		WorkspaceID:      workspaceID,
		Npwp:             stringToPgtext(req.NPWP),
		TaxStatus:        taxStatus,
		IsPpnLiable:      isPPNLiable,
		DefaultPpnRate:   floatToNumeric(ppnRate),
		Pph23Enabled:     req.PPh23Enabled,
		DefaultPph23Rate: floatToNumeric(pph23Rate),
		PkpNumber:        stringToPgtext(req.PKPNumber),
		TaxOfficeCode:    stringToPgtext(req.TaxOfficeCode),
		EfakturReady:     req.EFakturReady,
		EbupotReady:      req.EBupotReady,
		Notes:            stringToPgtext(req.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update tax profile: %w", err)
	}
	resp := profileFromRow(updated)
	return &resp, nil
}

func (s *Service) ListCalculations(ctx context.Context, workspaceID uuid.UUID, period, taxType, status string, limit, offset int32) ([]CalculationResponse, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.Queries.ListTaxCalculations(ctx, db.ListTaxCalculationsParams{
		WorkspaceID: workspaceID,
		Column2:     period,
		Column3:     taxType,
		Column4:     status,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tax calculations: %w", err)
	}

	count, err := s.Queries.CountTaxCalculations(ctx, db.CountTaxCalculationsParams{
		WorkspaceID: workspaceID,
		Column2:     period,
		Column3:     taxType,
		Column4:     status,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tax calculations: %w", err)
	}

	items := make([]CalculationResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, calculationFromRow(row))
	}
	return items, count, nil
}

func (s *Service) GetCalculation(ctx context.Context, workspaceID, calcID uuid.UUID) (*CalculationResponse, error) {
	row, err := s.Queries.GetTaxCalculationByID(ctx, db.GetTaxCalculationByIDParams{
		ID:          calcID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, ErrCalculationNotFound
	}
	resp := calculationFromEntity(row)
	return &resp, nil
}

func (s *Service) GetPPNSummary(ctx context.Context, workspaceID uuid.UUID, period string) (*PPNSummaryResponse, error) {
	row, err := s.Queries.GetPPNSummaryByPeriod(ctx, db.GetPPNSummaryByPeriodParams{
		WorkspaceID: workspaceID,
		Period:      period,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get PPN summary: %w", err)
	}

	masukan := interfaceToFloat(row.PpnMasukan)
	keluaran := interfaceToFloat(row.PpnKeluaran)

	return &PPNSummaryResponse{
		Period:           period,
		PPNMasukan:       fmt.Sprintf("%.2f", masukan),
		PPNKeluaran:      fmt.Sprintf("%.2f", keluaran),
		NetPPN:           fmt.Sprintf("%.2f", keluaran-masukan),
		DPPMasukan:       fmt.Sprintf("%.2f", interfaceToFloat(row.DppMasukan)),
		DPPKeluaran:      fmt.Sprintf("%.2f", interfaceToFloat(row.DppKeluaran)),
		TransactionCount: row.TransactionCount,
	}, nil
}

func (s *Service) GetPPhSummary(ctx context.Context, workspaceID uuid.UUID, periodFrom, periodTo string) (*PPhSummaryResponse, error) {
	rows, err := s.Queries.GetPPhSummaryByPeriod(ctx, db.GetPPhSummaryByPeriodParams{
		WorkspaceID: workspaceID,
		Period:      periodFrom,
		Period_2:    periodTo,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get PPh summary: %w", err)
	}

	result := &PPhSummaryResponse{
		PeriodFrom: periodFrom,
		PeriodTo:   periodTo,
		Rows:       make([]PPhSummaryRow, 0, len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, PPhSummaryRow{
			TaxType:   r.TaxType,
			Direction: r.Direction,
			TotalBase: fmt.Sprintf("%.2f", interfaceToFloat(r.TotalBase)),
			TotalTax:  fmt.Sprintf("%.2f", interfaceToFloat(r.TotalTax)),
			Count:     r.Count,
		})
	}
	return result, nil
}

func (s *Service) LinkDocument(ctx context.Context, workspaceID, calcID uuid.UUID, req *LinkDocumentRequest) (*DocumentRefResponse, error) {
	if _, err := s.GetCalculation(ctx, workspaceID, calcID); err != nil {
		return nil, err
	}

	docID, err := uuid.Parse(req.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("invalid document_id")
	}

	if !validDocRefType(req.RefType) {
		return nil, ErrInvalidDocRefType
	}

	if _, err := s.Queries.GetDocumentByID(ctx, db.GetDocumentByIDParams{
		ID:          docID,
		WorkspaceID: workspaceID,
	}); err != nil {
		return nil, ErrDocumentNotFound
	}

	now := time.Now()
	ref, err := s.Queries.CreateTaxDocumentRef(ctx, db.CreateTaxDocumentRefParams{
		ID:               uuid.New(),
		TaxCalculationID: calcID,
		DocumentID:       docID,
		RefType:          req.RefType,
		CreatedAt:        now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to link document: %w", err)
	}

	return &DocumentRefResponse{
		ID:         ref.ID.String(),
		DocumentID: ref.DocumentID.String(),
		RefType:    ref.RefType,
		CreatedAt:  ref.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *Service) ListDocumentRefs(ctx context.Context, calcID uuid.UUID) ([]DocumentRefResponse, error) {
	rows, err := s.Queries.ListTaxDocumentRefs(ctx, calcID)
	if err != nil {
		return nil, fmt.Errorf("failed to list document refs: %w", err)
	}
	items := make([]DocumentRefResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, DocumentRefResponse{
			ID:           r.ID.String(),
			DocumentID:   r.DocumentID.String(),
			RefType:      r.RefType,
			FileName:     r.FileName,
			DocumentType: r.DocumentType,
			CreatedAt:    r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return items, nil
}

func (s *Service) RequestReport(ctx context.Context, workspaceID, userID uuid.UUID, req *RequestReportRequest) (*ReportJobResponse, error) {
	if !validReportType(req.ReportType) {
		return nil, ErrInvalidReportType
	}
	if req.PeriodFrom == "" || req.PeriodTo == "" {
		return nil, fmt.Errorf("period_from and period_to are required")
	}

	now := time.Now()
	jobID := uuid.New()
	job, err := s.Queries.CreateTaxReportJob(ctx, db.CreateTaxReportJobParams{
		ID:          jobID,
		WorkspaceID: workspaceID,
		ReportType:  req.ReportType,
		PeriodFrom:  req.PeriodFrom,
		PeriodTo:    req.PeriodTo,
		Status:      ReportStatusPending,
		RequestedBy: userID,
		CreatedAt:   now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create report job: %w", err)
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	err = events.EmitEvent(ctx, tx, events.ReportGenerationReq, map[string]string{
		"job_id":       jobID.String(),
		"workspace_id": workspaceID.String(),
		"report_type":  req.ReportType,
		"period_from":  req.PeriodFrom,
		"period_to":    req.PeriodTo,
	}, events.WithWorkspace(workspaceID.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to emit report event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit report event: %w", err)
	}

	resp := reportJobFromEntity(job)
	return &resp, nil
}

func (s *Service) GetReportJob(ctx context.Context, workspaceID, jobID uuid.UUID) (*ReportJobResponse, error) {
	job, err := s.Queries.GetTaxReportJob(ctx, db.GetTaxReportJobParams{
		ID:          jobID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, ErrReportJobNotFound
	}
	resp := reportJobFromEntity(job)
	return &resp, nil
}

func (s *Service) ListReportJobs(ctx context.Context, workspaceID uuid.UUID, limit, offset int32) ([]ReportJobResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.Queries.ListTaxReportJobs(ctx, db.ListTaxReportJobsParams{
		WorkspaceID: workspaceID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list report jobs: %w", err)
	}
	items := make([]ReportJobResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, reportJobFromEntity(row))
	}
	return items, nil
}

func validDocRefType(t string) bool {
	switch t {
	case DocRefFakturPajak, DocRefBuktiPotong, DocRefInvoice, DocRefReceipt, DocRefOther:
		return true
	default:
		return false
	}
}

func validReportType(t string) bool {
	switch t {
	case ReportTypePPNSummary, ReportTypePPhSummary, ReportTypeTaxOverview:
		return true
	default:
		return false
	}
}

func reportJobFromEntity(job db.TaxReportJob) ReportJobResponse {
	resp := ReportJobResponse{
		ID:         job.ID.String(),
		ReportType: job.ReportType,
		PeriodFrom: job.PeriodFrom,
		PeriodTo:   job.PeriodTo,
		Status:     job.Status,
		CreatedAt:  job.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if job.ErrorMessage.Valid {
		resp.ErrorMessage = job.ErrorMessage.String
	}
	if job.CompletedAt != nil {
		resp.CompletedAt = job.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if len(job.Result) > 0 {
		var result interface{}
		if err := json.Unmarshal(job.Result, &result); err == nil {
			resp.Result = result
		}
	}
	return resp
}
