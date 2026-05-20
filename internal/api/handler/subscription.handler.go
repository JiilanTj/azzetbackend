package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/shared"
	"codeberg.org/azzet/azzetbe/internal/subscription"
)

type SubscriptionHandler struct {
	Service *subscription.Service
}

func NewSubscriptionHandler(service *subscription.Service) *SubscriptionHandler {
	return &SubscriptionHandler{Service: service}
}

// Subscribe godoc
// @Summary      Subscribe to a plan
// @Description  Subscribe the current workspace to a plan. Free plans activate instantly. Paid plans with is_trial start a trial.
// @Tags         Subscription
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string                          true  "Workspace ID"
// @Param        body            body      subscription.SubscribeRequest   true  "Subscription data"
// @Success      201             {object}  shared.APIResponse{data=subscription.SubscriptionResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Failure      401             {object}  shared.ErrorResponse
// @Failure      403             {object}  shared.ErrorResponse
// @Router       /subscription [post]
func (h *SubscriptionHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	var req subscription.SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "subscription", "invalid request body")
		return
	}

	if req.PlanID == "" {
		shared.BadRequest(w, r, "subscription", "plan_id is required")
		return
	}

	resp, err := h.Service.Subscribe(r.Context(), workspaceID, &req)
	if err != nil {
		shared.BadRequest(w, r, "subscription", err.Error())
		return
	}

	shared.Created(w, r, resp)
}

// GetActive godoc
// @Summary      Get active subscription
// @Description  Returns the current active subscription for the workspace.
// @Tags         Subscription
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true  "Workspace ID"
// @Success      200             {object}  shared.APIResponse{data=subscription.SubscriptionResponse}
// @Failure      401             {object}  shared.ErrorResponse
// @Failure      404             {object}  shared.ErrorResponse
// @Router       /subscription [get]
func (h *SubscriptionHandler) GetActive(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	resp, err := h.Service.GetActiveSubscription(r.Context(), workspaceID)
	if err != nil {
		shared.NotFound(w, r, "subscription", "no active subscription")
		return
	}

	shared.OK(w, r, resp)
}

// ListSubscriptions godoc
// @Summary      List subscription history
// @Description  Returns all subscriptions (active, expired, cancelled) for the workspace.
// @Tags         Subscription
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true  "Workspace ID"
// @Success      200             {object}  shared.APIResponse{data=[]subscription.SubscriptionResponse}
// @Failure      401             {object}  shared.ErrorResponse
// @Router       /subscription/history [get]
func (h *SubscriptionHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	resp, err := h.Service.ListSubscriptions(r.Context(), workspaceID)
	if err != nil {
		shared.InternalError(w, r, "subscription", "failed to list subscriptions")
		return
	}

	shared.OK(w, r, resp)
}

// Cancel godoc
// @Summary      Cancel subscription
// @Description  Cancel the active subscription for the workspace.
// @Tags         Subscription
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true  "Workspace ID"
// @Success      200             {object}  shared.APIResponse{data=subscription.MessageResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Failure      401             {object}  shared.ErrorResponse
// @Router       /subscription/cancel [post]
func (h *SubscriptionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	if err := h.Service.Cancel(r.Context(), workspaceID); err != nil {
		shared.BadRequest(w, r, "subscription", err.Error())
		return
	}

	shared.OK(w, r, subscription.MessageResponse{Message: "Subscription cancelled"})
}

// ChangePlan godoc
// @Summary      Change plan (upgrade/downgrade)
// @Description  Cancel current subscription and subscribe to a new plan.
// @Tags         Subscription
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string                          true  "Workspace ID"
// @Param        body            body      subscription.SubscribeRequest   true  "New plan data"
// @Success      200             {object}  shared.APIResponse{data=subscription.SubscriptionResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Failure      401             {object}  shared.ErrorResponse
// @Router       /subscription/change [post]
func (h *SubscriptionHandler) ChangePlan(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	var req subscription.SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "subscription", "invalid request body")
		return
	}

	if req.PlanID == "" {
		shared.BadRequest(w, r, "subscription", "plan_id is required")
		return
	}

	resp, err := h.Service.ChangePlan(r.Context(), workspaceID, &req)
	if err != nil {
		shared.BadRequest(w, r, "subscription", err.Error())
		return
	}

	shared.OK(w, r, resp)
}

// GetUsage godoc
// @Summary      Get usage summary
// @Description  Returns quota usage for all tracked features in the current billing period.
// @Tags         Subscription
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true  "Workspace ID"
// @Success      200             {object}  shared.APIResponse{data=[]subscription.UsageResponse}
// @Failure      401             {object}  shared.ErrorResponse
// @Failure      404             {object}  shared.ErrorResponse
// @Router       /subscription/usage [get]
func (h *SubscriptionHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	resp, err := h.Service.GetUsageSummary(r.Context(), workspaceID)
	if err != nil {
		shared.NotFound(w, r, "subscription", err.Error())
		return
	}

	shared.OK(w, r, resp)
}

// AdminListSubscriptions godoc
// @Summary      List all subscriptions (admin)
// @Description  Returns all subscriptions across all workspaces. SUPER_ADMIN/ENGINEER only.
// @Tags         Admin Subscriptions
// @Produce      json
// @Security     BearerAuth
// @Param        limit   query     int  false  "Limit (default 50)"
// @Param        offset  query     int  false  "Offset (default 0)"
// @Success      200     {object}  shared.APIResponse{data=[]subscription.SubscriptionResponse}
// @Failure      401     {object}  shared.ErrorResponse
// @Failure      403     {object}  shared.ErrorResponse
// @Router       /admin/subscriptions [get]
func (h *SubscriptionHandler) AdminListSubscriptions(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	resp, err := h.Service.ListAllSubscriptions(r.Context(), limit, offset)
	if err != nil {
		shared.InternalError(w, r, "subscription", "failed to list subscriptions")
		return
	}

	shared.OK(w, r, resp)
}
