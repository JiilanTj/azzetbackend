package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/entity"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

type EntityHandler struct {
	Service *entity.Service
}

func NewEntityHandler(service *entity.Service) *EntityHandler {
	return &EntityHandler{Service: service}
}

// CreateEntity godoc
// @Summary      Create entity
// @Description  Create a new entity (personal or business) linked to the authenticated user.
// @Tags         Entities
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      entity.CreateEntityRequest  true  "Entity data"
// @Success      201   {object}  shared.APIResponse{data=entity.EntityResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Failure      401   {object}  shared.ErrorResponse
// @Router       /entities [post]
func (h *EntityHandler) CreateEntity(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req entity.CreateEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "entity", "invalid request body")
		return
	}

	if req.NamaUtama == "" || req.EntityType == "" {
		shared.BadRequest(w, r, "entity", "nama_utama and entity_type are required")
		return
	}

	resp, err := h.Service.CreateEntity(r.Context(), userID, &req)
	if err != nil {
		shared.BadRequest(w, r, "entity", err.Error())
		return
	}

	shared.Created(w, r, resp)
}

// ListMyEntities godoc
// @Summary      List my entities
// @Description  Returns all entities owned by the authenticated user.
// @Tags         Entities
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  shared.APIResponse{data=[]entity.EntityResponse}
// @Failure      401  {object}  shared.ErrorResponse
// @Router       /entities [get]
func (h *EntityHandler) ListMyEntities(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	entities, err := h.Service.ListMyEntities(r.Context(), userID)
	if err != nil {
		shared.InternalError(w, r, "entity", "failed to list entities")
		return
	}

	shared.OK(w, r, entities)
}

// GetEntity godoc
// @Summary      Get entity by ID
// @Description  Returns entity detail with meta information.
// @Tags         Entities
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Entity ID (UUID)"
// @Success      200  {object}  shared.APIResponse{data=entity.EntityResponse}
// @Failure      404  {object}  shared.ErrorResponse
// @Router       /entities/{id} [get]
func (h *EntityHandler) GetEntity(w http.ResponseWriter, r *http.Request) {
	entityID := chi.URLParam(r, "id")

	resp, err := h.Service.GetEntityByID(r.Context(), entityID)
	if err != nil {
		shared.NotFound(w, r, "entity", "entity not found")
		return
	}

	shared.OK(w, r, resp)
}

// UpdateEntity godoc
// @Summary      Update entity
// @Description  Update an entity's basic information. Only the owner can update.
// @Tags         Entities
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                      true  "Entity ID (UUID)"
// @Param        body  body      entity.UpdateEntityRequest  true  "Update data"
// @Success      200   {object}  shared.APIResponse{data=entity.MessageResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Failure      401   {object}  shared.ErrorResponse
// @Router       /entities/{id} [patch]
func (h *EntityHandler) UpdateEntity(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	entityID := chi.URLParam(r, "id")

	var req entity.UpdateEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "entity", "invalid request body")
		return
	}

	if err := h.Service.UpdateEntity(r.Context(), userID, entityID, &req); err != nil {
		shared.BadRequest(w, r, "entity", err.Error())
		return
	}

	shared.OK(w, r, entity.MessageResponse{Message: "Entity updated"})
}

// UpdateEntityMeta godoc
// @Summary      Update entity meta
// @Description  Update entity administrative/compliance data. Only the owner can update.
// @Tags         Entities
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                          true  "Entity ID (UUID)"
// @Param        body  body      entity.UpdateEntityMetaRequest  true  "Meta data"
// @Success      200   {object}  shared.APIResponse{data=entity.MessageResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Failure      401   {object}  shared.ErrorResponse
// @Router       /entities/{id}/meta [patch]
func (h *EntityHandler) UpdateEntityMeta(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	entityID := chi.URLParam(r, "id")

	var req entity.UpdateEntityMetaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "entity", "invalid request body")
		return
	}

	if err := h.Service.UpdateEntityMeta(r.Context(), userID, entityID, &req); err != nil {
		shared.BadRequest(w, r, "entity", err.Error())
		return
	}

	shared.OK(w, r, entity.MessageResponse{Message: "Entity meta updated"})
}

// SearchEntities godoc
// @Summary      Search entities
// @Description  Search entities by name. Returns active entities matching the query.
// @Tags         Entities
// @Produce      json
// @Security     BearerAuth
// @Param        q       query     string  true   "Search query"
// @Param        limit   query     int     false  "Limit (default 20, max 100)"
// @Param        offset  query     int     false  "Offset (default 0)"
// @Success      200     {object}  shared.APIResponse{data=[]entity.EntityResponse}
// @Failure      400     {object}  shared.ErrorResponse
// @Router       /entities/search [get]
func (h *EntityHandler) SearchEntities(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		shared.BadRequest(w, r, "entity", "query parameter 'q' is required")
		return
	}

	limit := 20
	offset := 0
	// Simple parsing, ignore errors (defaults apply)
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	entities, err := h.Service.SearchEntities(r.Context(), query, limit, offset)
	if err != nil {
		shared.InternalError(w, r, "entity", "search failed")
		return
	}

	shared.OK(w, r, entities)
}
