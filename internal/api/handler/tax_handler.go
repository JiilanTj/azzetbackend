package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/shared"
	"codeberg.org/azzet/azzetbe/internal/tax"
)

type TaxHandler struct {
	Service *tax.Service
}

func NewTaxHandler(service *tax.Service) *TaxHandler {
	return &TaxHandler{Service: service}
}

// GetProfile godoc
// @Summary      Get workspace tax profile
// @Tags         Tax
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Success      200 {object} shared.APIResponse{data=tax.ProfileResponse}
// @Router       /tax/profile [get]
func (h *TaxHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_WORKSPACE", "tax", "invalid workspace id")
		return
	}

	resp, err := h.Service.GetOrCreateProfile(r.Context(), workspaceID)
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "tax", "failed to get tax profile")
		return
	}
	shared.Success(w, 200, resp)
}

// UpdateProfile godoc
// @Summary      Update workspace tax profile
// @Tags         Tax
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        body body tax.UpsertProfileRequest true "Tax profile"
// @Success      200 {object} shared.APIResponse{data=tax.ProfileResponse}
// @Router       /tax/profile [put]
func (h *TaxHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_WORKSPACE", "tax", "invalid workspace id")
		return
	}

	var req tax.UpsertProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.Error(w, r, 400, "INVALID_BODY", "tax", "invalid request body")
		return
	}

	resp, err := h.Service.UpdateProfile(r.Context(), workspaceID, &req)
	if err != nil {
		if err == tax.ErrInvalidTaxStatus {
			shared.Error(w, r, 400, "VALIDATION_ERROR", "tax", err.Error())
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "tax", "failed to update tax profile")
		return
	}
	shared.Success(w, 200, resp)
}

// ListCalculations godoc
// @Summary      List tax calculations
// @Tags         Tax
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        period query string false "Period YYYY-MM"
// @Param        tax_type query string false "Tax type filter"
// @Param        status query string false "Status filter"
// @Param        limit query int false "Limit"
// @Param        offset query int false "Offset"
// @Success      200 {object} shared.APIResponse{data=[]tax.CalculationResponse,meta=shared.PaginationMeta}
// @Router       /tax/calculations [get]
func (h *TaxHandler) ListCalculations(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_WORKSPACE", "tax", "invalid workspace id")
		return
	}

	pagination := shared.ParsePagination(r)
	if r.URL.Query().Get("page") == "" && r.URL.Query().Get("offset") != "" {
		offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 32)
		limit := int64(pagination.PerPage)
		if limit > 0 && offset%limit == 0 {
			pagination.Page = int(offset/limit) + 1
		}
	}
	if r.URL.Query().Get("limit") != "" {
		if lim, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && lim > 0 {
			pagination.PerPage = lim
		}
	}

	items, total, err := h.Service.ListCalculations(r.Context(), workspaceID,
		r.URL.Query().Get("period"),
		r.URL.Query().Get("tax_type"),
		r.URL.Query().Get("status"),
		int32(pagination.Limit()), int32(pagination.Offset()),
	)
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "tax", "failed to list calculations")
		return
	}
	shared.Paginated(w, 200, items, shared.NewPaginationMeta(pagination, total))
}

// GetCalculation godoc
// @Summary      Get tax calculation detail
// @Tags         Tax
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        id path string true "Calculation ID"
// @Success      200 {object} shared.APIResponse{data=tax.CalculationResponse}
// @Router       /tax/calculations/{id} [get]
func (h *TaxHandler) GetCalculation(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_WORKSPACE", "tax", "invalid workspace id")
		return
	}
	calcID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_ID", "tax", "invalid calculation id")
		return
	}

	resp, err := h.Service.GetCalculation(r.Context(), workspaceID, calcID)
	if err != nil {
		if err == tax.ErrCalculationNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "tax", "calculation not found")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "tax", "failed to get calculation")
		return
	}
	shared.Success(w, 200, resp)
}

// GetPPNSummary godoc
// @Summary      Get PPN summary for a period
// @Tags         Tax
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        period query string true "Period YYYY-MM"
// @Success      200 {object} shared.APIResponse{data=tax.PPNSummaryResponse}
// @Router       /tax/summary/ppn [get]
func (h *TaxHandler) GetPPNSummary(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_WORKSPACE", "tax", "invalid workspace id")
		return
	}
	period := r.URL.Query().Get("period")
	if period == "" {
		shared.Error(w, r, 400, "VALIDATION_ERROR", "tax", "period is required (YYYY-MM)")
		return
	}

	resp, err := h.Service.GetPPNSummary(r.Context(), workspaceID, period)
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "tax", "failed to get PPN summary")
		return
	}
	shared.Success(w, 200, resp)
}

// GetPPhSummary godoc
// @Summary      Get PPh summary for a period range
// @Tags         Tax
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        period_from query string true "Period from YYYY-MM"
// @Param        period_to query string true "Period to YYYY-MM"
// @Success      200 {object} shared.APIResponse{data=tax.PPhSummaryResponse}
// @Router       /tax/summary/pph [get]
func (h *TaxHandler) GetPPhSummary(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_WORKSPACE", "tax", "invalid workspace id")
		return
	}
	periodFrom := r.URL.Query().Get("period_from")
	periodTo := r.URL.Query().Get("period_to")
	if periodFrom == "" || periodTo == "" {
		shared.Error(w, r, 400, "VALIDATION_ERROR", "tax", "period_from and period_to are required")
		return
	}

	resp, err := h.Service.GetPPhSummary(r.Context(), workspaceID, periodFrom, periodTo)
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "tax", "failed to get PPh summary")
		return
	}
	shared.Success(w, 200, resp)
}

// LinkDocument godoc
// @Summary      Link a document to a tax calculation
// @Tags         Tax
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        id path string true "Calculation ID"
// @Param        body body tax.LinkDocumentRequest true "Document link"
// @Success      201 {object} shared.APIResponse{data=tax.DocumentRefResponse}
// @Router       /tax/calculations/{id}/documents [post]
func (h *TaxHandler) LinkDocument(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_WORKSPACE", "tax", "invalid workspace id")
		return
	}
	calcID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_ID", "tax", "invalid calculation id")
		return
	}

	var req tax.LinkDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.Error(w, r, 400, "INVALID_BODY", "tax", "invalid request body")
		return
	}

	resp, err := h.Service.LinkDocument(r.Context(), workspaceID, calcID, &req)
	if err != nil {
		if err == tax.ErrCalculationNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "tax", "calculation not found")
			return
		}
		if err == tax.ErrDocumentNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "tax", "document not found")
			return
		}
		if err == tax.ErrInvalidDocRefType {
			shared.Error(w, r, 400, "VALIDATION_ERROR", "tax", err.Error())
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "tax", "failed to link document")
		return
	}
	shared.Success(w, 201, resp)
}

// ListDocumentRefs godoc
// @Summary      List documents linked to a tax calculation
// @Tags         Tax
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        id path string true "Calculation ID"
// @Success      200 {object} shared.APIResponse{data=[]tax.DocumentRefResponse}
// @Router       /tax/calculations/{id}/documents [get]
func (h *TaxHandler) ListDocumentRefs(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_WORKSPACE", "tax", "invalid workspace id")
		return
	}
	calcID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_ID", "tax", "invalid calculation id")
		return
	}

	if _, err := h.Service.GetCalculation(r.Context(), workspaceID, calcID); err != nil {
		if err == tax.ErrCalculationNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "tax", "calculation not found")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "tax", "failed to verify calculation")
		return
	}

	items, err := h.Service.ListDocumentRefs(r.Context(), calcID)
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "tax", "failed to list document refs")
		return
	}
	shared.Success(w, 200, items)
}

// RequestReport godoc
// @Summary      Request async tax report generation
// @Tags         Tax
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        body body tax.RequestReportRequest true "Report request"
// @Success      202 {object} shared.APIResponse{data=tax.ReportJobResponse}
// @Router       /tax/reports [post]
func (h *TaxHandler) RequestReport(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_WORKSPACE", "tax", "invalid workspace id")
		return
	}
	userID, err := uuid.Parse(middleware.GetUserID(r.Context()))
	if err != nil {
		shared.Error(w, r, 401, "UNAUTHORIZED", "tax", "invalid user")
		return
	}

	var req tax.RequestReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.Error(w, r, 400, "INVALID_BODY", "tax", "invalid request body")
		return
	}

	resp, err := h.Service.RequestReport(r.Context(), workspaceID, userID, &req)
	if err != nil {
		if err == tax.ErrInvalidReportType {
			shared.Error(w, r, 400, "VALIDATION_ERROR", "tax", err.Error())
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "tax", "failed to request report")
		return
	}
	shared.Success(w, 202, resp)
}

// GetReportJob godoc
// @Summary      Get tax report job status
// @Tags         Tax
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        id path string true "Report job ID"
// @Success      200 {object} shared.APIResponse{data=tax.ReportJobResponse}
// @Router       /tax/reports/{id} [get]
func (h *TaxHandler) GetReportJob(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_WORKSPACE", "tax", "invalid workspace id")
		return
	}
	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_ID", "tax", "invalid job id")
		return
	}

	resp, err := h.Service.GetReportJob(r.Context(), workspaceID, jobID)
	if err != nil {
		if err == tax.ErrReportJobNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "tax", "report job not found")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "tax", "failed to get report job")
		return
	}
	shared.Success(w, 200, resp)
}

// ListReportJobs godoc
// @Summary      List tax report jobs
// @Tags         Tax
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Success      200 {object} shared.APIResponse{data=[]tax.ReportJobResponse}
// @Router       /tax/reports [get]
func (h *TaxHandler) ListReportJobs(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(middleware.GetWorkspaceID(r.Context()))
	if err != nil {
		shared.Error(w, r, 400, "INVALID_WORKSPACE", "tax", "invalid workspace id")
		return
	}

	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 32)

	items, err := h.Service.ListReportJobs(r.Context(), workspaceID, int32(limit), int32(offset))
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "tax", "failed to list report jobs")
		return
	}
	shared.Success(w, 200, items)
}
