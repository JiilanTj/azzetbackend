package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"codeberg.org/azzet/azzetbe/internal/claim"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

type ClaimHandler struct {
	Service *claim.Service
}

func NewClaimHandler(service *claim.Service) *ClaimHandler {
	return &ClaimHandler{Service: service}
}

func (h *ClaimHandler) CreateClaim(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	var req claim.CreateClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.Error(w, r, 400, "INVALID_BODY", "claim", "invalid request body")
		return
	}

	if req.EntityID == "" {
		shared.Error(w, r, 400, "VALIDATION_ERROR", "claim", "entity_id is required")
		return
	}

	resp, err := h.Service.CreateClaim(r.Context(), userID, &req)
	if err != nil {
		if err == claim.ErrNotShadow {
			shared.Error(w, r, 400, "NOT_SHADOW", "claim", "entity is not a shadow entity")
			return
		}
		if err == claim.ErrClaimExists {
			shared.Error(w, r, 409, "CLAIM_EXISTS", "claim", "active claim already exists for this entity")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to create claim")
		return
	}
	shared.Success(w, 201, resp)
}

func (h *ClaimHandler) GetMyClaims(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	resp, err := h.Service.GetMyClaims(r.Context(), userID)
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to list claims")
		return
	}
	shared.Success(w, 200, resp)
}

func (h *ClaimHandler) GetClaim(w http.ResponseWriter, r *http.Request) {
	claimID := chi.URLParam(r, "id")

	resp, err := h.Service.GetClaim(r.Context(), claimID)
	if err != nil {
		if err == claim.ErrClaimNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "claim", "claim not found")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to get claim")
		return
	}
	shared.Success(w, 200, resp)
}

func (h *ClaimHandler) SubmitClaim(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	claimID := chi.URLParam(r, "id")

	resp, err := h.Service.SubmitClaim(r.Context(), userID, claimID)
	if err != nil {
		if err == claim.ErrClaimNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "claim", "claim not found")
			return
		}
		if err == claim.ErrNotOwner {
			shared.Error(w, r, 403, "FORBIDDEN", "claim", "not authorized")
			return
		}
		if err == claim.ErrInvalidStatus {
			shared.Error(w, r, 400, "INVALID_STATUS", "claim", "claim cannot be submitted in current status")
			return
		}
		if err == claim.ErrDocumentsMissing {
			shared.Error(w, r, 400, "DOCS_REQUIRED", "claim", "at least one document is required before submission")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to submit claim")
		return
	}
	shared.Success(w, 200, resp)
}

func (h *ClaimHandler) RequestUpload(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	claimID := chi.URLParam(r, "id")

	var req claim.DocumentUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.Error(w, r, 400, "INVALID_BODY", "claim", "invalid request body")
		return
	}

	if req.DocumentType == "" || req.FileName == "" {
		shared.Error(w, r, 400, "VALIDATION_ERROR", "claim", "document_type and file_name are required")
		return
	}

	resp, err := h.Service.RequestDocumentUpload(r.Context(), userID, claimID, &req)
	if err != nil {
		if err == claim.ErrClaimNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "claim", "claim not found")
			return
		}
		if err == claim.ErrNotOwner {
			shared.Error(w, r, 403, "FORBIDDEN", "claim", "not authorized")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to generate upload URL")
		return
	}
	shared.Success(w, 201, resp)
}

func (h *ClaimHandler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	claimID := chi.URLParam(r, "id")
	docID := chi.URLParam(r, "doc_id")

	if err := h.Service.ConfirmDocumentUpload(r.Context(), userID, claimID, docID); err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to confirm upload")
		return
	}
	shared.Success(w, 200, map[string]string{"status": "confirmed"})
}

func (h *ClaimHandler) GetClaimDocuments(w http.ResponseWriter, r *http.Request) {
	claimID := chi.URLParam(r, "id")

	resp, err := h.Service.GetClaimDocuments(r.Context(), claimID)
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to get documents")
		return
	}
	shared.Success(w, 200, resp)
}

func (h *ClaimHandler) DisputeClaim(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	claimID := chi.URLParam(r, "id")

	var req claim.DisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.Error(w, r, 400, "INVALID_BODY", "claim", "invalid request body")
		return
	}

	if req.Reason == "" {
		shared.Error(w, r, 400, "VALIDATION_ERROR", "claim", "reason is required")
		return
	}

	if err := h.Service.DisputeClaim(r.Context(), userID, claimID, &req); err != nil {
		if err == claim.ErrClaimNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "claim", "claim not found")
			return
		}
		if err == claim.ErrNotOwner {
			shared.Error(w, r, 403, "FORBIDDEN", "claim", "not authorized")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to dispute claim")
		return
	}
	shared.Success(w, 200, map[string]string{"status": "disputed"})
}

// --- Admin Handler ---

type ClaimAdminHandler struct {
	Service *claim.Service
}

func NewClaimAdminHandler(service *claim.Service) *ClaimAdminHandler {
	return &ClaimAdminHandler{Service: service}
}

func (h *ClaimAdminHandler) ListClaims(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if status != "" {
		resp, err := h.Service.ListClaimsForReview(r.Context(), status, limit, offset)
		if err != nil {
			shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to list claims")
			return
		}
		shared.Success(w, 200, resp)
		return
	}

	resp, err := h.Service.ListAllClaims(r.Context(), limit, offset)
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to list claims")
		return
	}
	shared.Success(w, 200, resp)
}

func (h *ClaimAdminHandler) GetClaim(w http.ResponseWriter, r *http.Request) {
	claimID := chi.URLParam(r, "id")

	resp, err := h.Service.GetClaim(r.Context(), claimID)
	if err != nil {
		if err == claim.ErrClaimNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "claim", "claim not found")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to get claim")
		return
	}
	shared.Success(w, 200, resp)
}

func (h *ClaimAdminHandler) AssignClaim(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value("admin_id").(string)
	claimID := chi.URLParam(r, "id")

	if err := h.Service.AssignClaim(r.Context(), adminID, claimID); err != nil {
		if err == claim.ErrClaimNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "claim", "claim not found")
			return
		}
		if err == claim.ErrInvalidStatus {
			shared.Error(w, r, 400, "INVALID_STATUS", "claim", "claim cannot be assigned in current status")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to assign claim")
		return
	}
	shared.Success(w, 200, map[string]string{"status": "assigned"})
}

func (h *ClaimAdminHandler) ApproveClaim(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value("admin_id").(string)
	claimID := chi.URLParam(r, "id")

	var req claim.ReviewRequest
	json.NewDecoder(r.Body).Decode(&req)

	if err := h.Service.ApproveClaim(r.Context(), adminID, claimID, &req); err != nil {
		if err == claim.ErrClaimNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "claim", "claim not found")
			return
		}
		if err == claim.ErrInvalidStatus {
			shared.Error(w, r, 400, "INVALID_STATUS", "claim", "claim cannot be approved in current status")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to approve claim")
		return
	}
	shared.Success(w, 200, map[string]string{"status": "approved"})
}

func (h *ClaimAdminHandler) RejectClaim(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value("admin_id").(string)
	claimID := chi.URLParam(r, "id")

	var req claim.ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.Error(w, r, 400, "INVALID_BODY", "claim", "invalid request body")
		return
	}

	if req.RejectionReason == nil || *req.RejectionReason == "" {
		shared.Error(w, r, 400, "VALIDATION_ERROR", "claim", "rejection_reason is required")
		return
	}

	if err := h.Service.RejectClaim(r.Context(), adminID, claimID, &req); err != nil {
		if err == claim.ErrClaimNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "claim", "claim not found")
			return
		}
		if err == claim.ErrInvalidStatus {
			shared.Error(w, r, 400, "INVALID_STATUS", "claim", "claim cannot be rejected in current status")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to reject claim")
		return
	}
	shared.Success(w, 200, map[string]string{"status": "rejected"})
}

func (h *ClaimAdminHandler) GetClaimAuditLog(w http.ResponseWriter, r *http.Request) {
	claimID := chi.URLParam(r, "id")

	resp, err := h.Service.GetClaimAuditLog(r.Context(), claimID)
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to get audit log")
		return
	}
	shared.Success(w, 200, resp)
}

func (h *ClaimAdminHandler) GetDocumentViewURL(w http.ResponseWriter, r *http.Request) {
	docID := chi.URLParam(r, "doc_id")

	url, err := h.Service.GetDocumentViewURL(r.Context(), docID)
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to generate view URL")
		return
	}
	shared.Success(w, 200, map[string]string{"view_url": url})
}

func (h *ClaimAdminHandler) CountPendingClaims(w http.ResponseWriter, r *http.Request) {
	count, err := h.Service.CountPendingClaims(r.Context())
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "claim", "failed to count pending claims")
		return
	}
	shared.Success(w, 200, map[string]int64{"pending": count})
}
