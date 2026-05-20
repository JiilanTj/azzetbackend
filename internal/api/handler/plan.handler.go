package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"codeberg.org/azzet/azzetbe/internal/plan"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

type PlanHandler struct {
	Service *plan.Service
}

func NewPlanHandler(service *plan.Service) *PlanHandler {
	return &PlanHandler{Service: service}
}

// --- Public Endpoints ---

// ListPlans godoc
// @Summary      List active plans
// @Description  Returns all active plans sorted by tier. Public endpoint (no auth required).
// @Tags         Plans
// @Produce      json
// @Success      200  {object}  shared.APIResponse{data=[]plan.PlanListResponse}
// @Router       /plans [get]
func (h *PlanHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.Service.ListPlans(r.Context())
	if err != nil {
		shared.InternalError(w, r, "plan", "failed to list plans")
		return
	}
	shared.OK(w, r, plans)
}

// GetPlanBySlug godoc
// @Summary      Get plan by slug
// @Description  Returns a plan with all its features. Public endpoint (no auth required).
// @Tags         Plans
// @Produce      json
// @Param        slug  path      string  true  "Plan slug"
// @Success      200   {object}  shared.APIResponse{data=plan.PlanResponse}
// @Failure      404   {object}  shared.ErrorResponse
// @Router       /plans/{slug} [get]
func (h *PlanHandler) GetPlanBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	p, err := h.Service.GetPlanBySlug(r.Context(), slug)
	if err != nil {
		shared.NotFound(w, r, "plan", "plan not found")
		return
	}

	shared.OK(w, r, p)
}

// --- Admin Endpoints ---

// AdminListPlans godoc
// @Summary      List all plans (admin)
// @Description  Returns all plans including inactive with full features. SUPER_ADMIN/ENGINEER only.
// @Tags         Admin Plans
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  shared.APIResponse{data=[]plan.PlanResponse}
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse
// @Router       /admin/plans [get]
func (h *PlanHandler) AdminListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.Service.ListAllPlans(r.Context())
	if err != nil {
		shared.InternalError(w, r, "plan", "failed to list plans")
		return
	}
	shared.OK(w, r, plans)
}

// AdminGetPlan godoc
// @Summary      Get plan by ID (admin)
// @Description  Returns a plan with all its features by ID. SUPER_ADMIN/ENGINEER only.
// @Tags         Admin Plans
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Plan ID (UUID)"
// @Success      200  {object}  shared.APIResponse{data=plan.PlanResponse}
// @Failure      404  {object}  shared.ErrorResponse
// @Router       /admin/plans/{id} [get]
func (h *PlanHandler) AdminGetPlan(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "id")

	p, err := h.Service.GetPlanByID(r.Context(), planID)
	if err != nil {
		shared.NotFound(w, r, "plan", "plan not found")
		return
	}

	shared.OK(w, r, p)
}

// AdminCreatePlan godoc
// @Summary      Create plan
// @Description  Create a new plan. SUPER_ADMIN/ENGINEER only.
// @Tags         Admin Plans
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      plan.CreatePlanRequest  true  "Plan data"
// @Success      201   {object}  shared.APIResponse{data=plan.PlanResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Router       /admin/plans [post]
func (h *PlanHandler) AdminCreatePlan(w http.ResponseWriter, r *http.Request) {
	var req plan.CreatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "plan", "invalid request body")
		return
	}

	p, err := h.Service.CreatePlan(r.Context(), &req)
	if err != nil {
		shared.BadRequest(w, r, "plan", err.Error())
		return
	}

	shared.Created(w, r, p)
}

// AdminUpdatePlan godoc
// @Summary      Update plan
// @Description  Update an existing plan. SUPER_ADMIN/ENGINEER only.
// @Tags         Admin Plans
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                 true  "Plan ID (UUID)"
// @Param        body  body      plan.UpdatePlanRequest  true  "Update data"
// @Success      200   {object}  shared.APIResponse{data=plan.MessageResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Failure      404   {object}  shared.ErrorResponse
// @Router       /admin/plans/{id} [patch]
func (h *PlanHandler) AdminUpdatePlan(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "id")

	var req plan.UpdatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "plan", "invalid request body")
		return
	}

	if err := h.Service.UpdatePlan(r.Context(), planID, &req); err != nil {
		shared.BadRequest(w, r, "plan", err.Error())
		return
	}

	shared.OK(w, r, plan.MessageResponse{Message: "Plan updated"})
}

// AdminDeletePlan godoc
// @Summary      Delete plan
// @Description  Soft-delete a plan (sets is_active = false). SUPER_ADMIN/ENGINEER only.
// @Tags         Admin Plans
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Plan ID (UUID)"
// @Success      200  {object}  shared.APIResponse{data=plan.MessageResponse}
// @Failure      400  {object}  shared.ErrorResponse
// @Router       /admin/plans/{id} [delete]
func (h *PlanHandler) AdminDeletePlan(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "id")

	if err := h.Service.DeletePlan(r.Context(), planID); err != nil {
		shared.BadRequest(w, r, "plan", err.Error())
		return
	}

	shared.OK(w, r, plan.MessageResponse{Message: "Plan deactivated"})
}

// AdminSetFeature godoc
// @Summary      Set plan feature
// @Description  Add or update a feature on a plan. SUPER_ADMIN/ENGINEER only.
// @Tags         Admin Plans
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                  true  "Plan ID (UUID)"
// @Param        body  body      plan.SetFeatureRequest  true  "Feature data"
// @Success      200   {object}  shared.APIResponse{data=plan.FeatureResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Failure      404   {object}  shared.ErrorResponse
// @Router       /admin/plans/{id}/features [post]
func (h *PlanHandler) AdminSetFeature(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "id")

	var req plan.SetFeatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "plan", "invalid request body")
		return
	}

	f, err := h.Service.SetFeature(r.Context(), planID, &req)
	if err != nil {
		shared.BadRequest(w, r, "plan", err.Error())
		return
	}

	shared.OK(w, r, f)
}

// AdminRemoveFeature godoc
// @Summary      Remove plan feature
// @Description  Remove a feature from a plan. SUPER_ADMIN/ENGINEER only.
// @Tags         Admin Plans
// @Produce      json
// @Security     BearerAuth
// @Param        id           path      string  true  "Plan ID (UUID)"
// @Param        feature_key  path      string  true  "Feature key"
// @Success      200          {object}  shared.APIResponse{data=plan.MessageResponse}
// @Failure      400          {object}  shared.ErrorResponse
// @Router       /admin/plans/{id}/features/{feature_key} [delete]
func (h *PlanHandler) AdminRemoveFeature(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "id")
	featureKey := chi.URLParam(r, "feature_key")

	if err := h.Service.RemoveFeature(r.Context(), planID, featureKey); err != nil {
		shared.BadRequest(w, r, "plan", err.Error())
		return
	}

	shared.OK(w, r, plan.MessageResponse{Message: "Feature removed"})
}
