package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/document"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

type DocumentHandler struct {
	Service *document.Service
}

func NewDocumentHandler(service *document.Service) *DocumentHandler {
	return &DocumentHandler{Service: service}
}

// RequestUpload godoc
// @Summary      Request document upload URL
// @Description  Get a presigned URL to upload a receipt/invoice for OCR processing
// @Tags         Document
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        body body document.UploadRequest true "Upload request"
// @Success      201 {object} shared.APIResponse{data=document.PresignedUploadResponse}
// @Router       /documents [post]
func (h *DocumentHandler) RequestUpload(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var req document.UploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.Error(w, r, 400, "INVALID_BODY", "document", "invalid request body")
		return
	}

	if req.DocumentType == "" || req.FileName == "" {
		shared.Error(w, r, 400, "VALIDATION_ERROR", "document", "document_type and file_name are required")
		return
	}

	resp, err := h.Service.RequestUpload(r.Context(), workspaceID, userID, &req)
	if err != nil {
		if err == document.ErrOCRNotEnabled {
			shared.Error(w, r, 403, "FEATURE_DISABLED", "document", "OCR feature not enabled on current plan")
			return
		}
		if err == document.ErrInvalidDocumentType {
			shared.Error(w, r, 400, "VALIDATION_ERROR", "document", "invalid document_type")
			return
		}
		if err == document.ErrStorageNotConfigured {
			shared.Error(w, r, 503, "STORAGE_UNAVAILABLE", "document", "document storage not configured")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "document", "failed to request upload")
		return
	}
	shared.Success(w, 201, resp)
}

// ConfirmUpload godoc
// @Summary      Confirm document upload
// @Description  Confirm that the file was uploaded to R2 and trigger OCR processing
// @Tags         Document
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        id path string true "Document ID"
// @Success      200 {object} shared.APIResponse{data=document.DocumentResponse}
// @Router       /documents/{id}/confirm [post]
func (h *DocumentHandler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())
	userID := middleware.GetUserID(r.Context())
	documentID := chi.URLParam(r, "id")

	resp, err := h.Service.ConfirmUpload(r.Context(), workspaceID, userID, documentID)
	if err != nil {
		if err == document.ErrDocumentNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "document", "document not found")
			return
		}
		if err == document.ErrUploadNotConfirmed {
			shared.Error(w, r, 400, "UPLOAD_MISSING", "document", "document not found in storage")
			return
		}
		if err == document.ErrInvalidStatus {
			shared.Error(w, r, 400, "INVALID_STATUS", "document", "document already confirmed")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "document", "failed to confirm upload")
		return
	}
	shared.Success(w, 200, resp)
}

// GetDocument godoc
// @Summary      Get document detail
// @Tags         Document
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        id path string true "Document ID"
// @Success      200 {object} shared.APIResponse{data=document.DocumentResponse}
// @Router       /documents/{id} [get]
func (h *DocumentHandler) GetDocument(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())
	documentID := chi.URLParam(r, "id")

	resp, err := h.Service.GetDocument(r.Context(), workspaceID, documentID)
	if err != nil {
		if err == document.ErrDocumentNotFound {
			shared.Error(w, r, 404, "NOT_FOUND", "document", "document not found")
			return
		}
		shared.Error(w, r, 500, "INTERNAL_ERROR", "document", "failed to get document")
		return
	}
	shared.Success(w, 200, resp)
}

// ListDocuments godoc
// @Summary      List workspace documents
// @Tags         Document
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        limit query int false "Limit"
// @Param        offset query int false "Offset"
// @Success      200 {object} shared.APIResponse{data=document.DocumentListResponse}
// @Router       /documents [get]
func (h *DocumentHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	resp, err := h.Service.ListDocuments(r.Context(), workspaceID, int32(limit), int32(offset))
	if err != nil {
		shared.Error(w, r, 500, "INTERNAL_ERROR", "document", "failed to list documents")
		return
	}
	shared.Success(w, 200, resp)
}
