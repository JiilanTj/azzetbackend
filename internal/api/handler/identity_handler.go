package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/identity"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

type IdentityHandler struct {
	Service *identity.Service
}

func NewIdentityHandler(service *identity.Service) *IdentityHandler {
	return &IdentityHandler{Service: service}
}

func (h *IdentityHandler) GetVerificationStatus(w http.ResponseWriter, r *http.Request) {
	entityID := chi.URLParam(r, "id")
	resp, err := h.Service.GetVerificationStatus(r.Context(), entityID)
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "identity", "failed to get verification status")
		return
	}
	shared.Success(w, 200, resp)
}

func (h *IdentityHandler) AddLegalID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	entityID := chi.URLParam(r, "id")

	var req identity.AddLegalIDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.Error(w, r, 400, "INVALID_BODY", "identity", "invalid request body")
		return
	}

	if req.IDType == "" || req.IDValue == "" {
		shared.ValidationError(w, r, "identity", "validation failed", []shared.FieldError{
			{Field: "id_type", Message: "is required"},
			{Field: "id_value", Message: "is required"},
		})
		return
	}

	resp, err := h.Service.AddLegalID(r.Context(), userID, entityID, &req)
	if err != nil {
		if err == identity.ErrNotAuthorized {
			shared.Error(w, r, 403, "FORBIDDEN", "identity", "not authorized")
			return
		}
		if err == identity.ErrInvalidLegalIDType {
			shared.Error(w, r, 400, "INVALID_TYPE", "identity", "invalid legal ID type. Must be one of: NPWP, NIB, SIUP, KTP, AKTA")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "identity", "failed to add legal ID")
		return
	}
	shared.Success(w, 201, resp)
}

func (h *IdentityHandler) GetLegalIDs(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	entityID := chi.URLParam(r, "id")

	resp, err := h.Service.GetLegalIDs(r.Context(), userID, entityID)
	if err != nil {
		if err == identity.ErrNotAuthorized {
			shared.Error(w, r, 403, "FORBIDDEN", "identity", "not authorized")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "identity", "failed to get legal IDs")
		return
	}
	shared.Success(w, 200, resp)
}

func (h *IdentityHandler) UpdateLegalID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	entityID := chi.URLParam(r, "id")
	idType := strings.ToUpper(chi.URLParam(r, "type"))

	var req struct {
		IDValue string `json:"id_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.Error(w, r, 400, "INVALID_BODY", "identity", "invalid request body")
		return
	}

	if req.IDValue == "" {
		shared.Error(w, r, 400, "VALIDATION_ERROR", "identity", "id_value is required")
		return
	}

	if err := h.Service.UpdateLegalID(r.Context(), userID, entityID, idType, req.IDValue); err != nil {
		if err == identity.ErrNotAuthorized {
			shared.Error(w, r, 403, "FORBIDDEN", "identity", "not authorized")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "identity", "failed to update legal ID")
		return
	}
	shared.Success(w, 200, map[string]string{"status": "updated"})
}

func (h *IdentityHandler) DeleteLegalID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	entityID := chi.URLParam(r, "id")
	idType := strings.ToUpper(chi.URLParam(r, "type"))

	if err := h.Service.DeleteLegalID(r.Context(), userID, entityID, idType); err != nil {
		if err == identity.ErrNotAuthorized {
			shared.Error(w, r, 403, "FORBIDDEN", "identity", "not authorized")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "identity", "failed to delete legal ID")
		return
	}
	shared.Success(w, 200, map[string]string{"status": "deleted"})
}

func (h *IdentityHandler) AddAlias(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	entityID := chi.URLParam(r, "id")

	var req identity.AddAliasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.Error(w, r, 400, "INVALID_BODY", "identity", "invalid request body")
		return
	}

	if req.Alias == "" {
		shared.Error(w, r, 400, "VALIDATION_ERROR", "identity", "alias is required")
		return
	}

	resp, err := h.Service.AddAlias(r.Context(), userID, entityID, &req)
	if err != nil {
		if err == identity.ErrNotAuthorized {
			shared.Error(w, r, 403, "FORBIDDEN", "identity", "not authorized")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "identity", "failed to add alias")
		return
	}
	shared.Success(w, 201, resp)
}

func (h *IdentityHandler) GetAliases(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	entityID := chi.URLParam(r, "id")

	resp, err := h.Service.GetAliases(r.Context(), userID, entityID)
	if err != nil {
		if err == identity.ErrNotAuthorized {
			shared.Error(w, r, 403, "FORBIDDEN", "identity", "not authorized")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "identity", "failed to get aliases")
		return
	}
	shared.Success(w, 200, resp)
}

func (h *IdentityHandler) DeleteAlias(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	entityID := chi.URLParam(r, "id")
	aliasID := chi.URLParam(r, "alias_id")

	if err := h.Service.DeleteAlias(r.Context(), userID, entityID, aliasID); err != nil {
		if err == identity.ErrNotAuthorized {
			shared.Error(w, r, 403, "FORBIDDEN", "identity", "not authorized")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "identity", "failed to delete alias")
		return
	}
	shared.Success(w, 200, map[string]string{"status": "deleted"})
}

func (h *IdentityHandler) SearchFuzzy(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		shared.Error(w, r, 400, "VALIDATION_ERROR", "identity", "q parameter is required")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	resp, err := h.Service.SearchFuzzy(r.Context(), query, limit, offset)
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "identity", "search failed")
		return
	}
	shared.Success(w, 200, resp)
}

func (h *IdentityHandler) FindDuplicates(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	entityID := chi.URLParam(r, "id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	resp, err := h.Service.FindDuplicates(r.Context(), userID, entityID, limit)
	if err != nil {
		if err == identity.ErrNotAuthorized {
			shared.Error(w, r, 403, "FORBIDDEN", "identity", "not authorized")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "identity", "duplicate search failed")
		return
	}
	shared.Success(w, 200, resp)
}
