package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/shared"
	"codeberg.org/azzet/azzetbe/internal/workspace"
)

type WorkspaceHandler struct {
	Service *workspace.Service
}

func NewWorkspaceHandler(service *workspace.Service) *WorkspaceHandler {
	return &WorkspaceHandler{Service: service}
}

// CreateWorkspace godoc
// @Summary      Create workspace
// @Description  Create a workspace from an existing entity. The user becomes PEMILIK (owner).
// @Tags         Workspaces
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      workspace.CreateWorkspaceRequest  true  "Workspace data"
// @Success      201   {object}  shared.APIResponse{data=workspace.WorkspaceResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Failure      401   {object}  shared.ErrorResponse
// @Router       /workspaces [post]
func (h *WorkspaceHandler) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req workspace.CreateWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "workspace", "invalid request body")
		return
	}

	if req.EntityID == "" {
		shared.BadRequest(w, r, "workspace", "entity_id is required")
		return
	}

	resp, err := h.Service.CreateWorkspace(r.Context(), userID, &req)
	if err != nil {
		shared.BadRequest(w, r, "workspace", err.Error())
		return
	}

	shared.Created(w, r, resp)
}

// ListMyWorkspaces godoc
// @Summary      List my workspaces
// @Description  Returns all workspaces the authenticated user has access to.
// @Tags         Workspaces
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  shared.APIResponse{data=[]workspace.WorkspaceResponse}
// @Failure      401  {object}  shared.ErrorResponse
// @Router       /workspaces [get]
func (h *WorkspaceHandler) ListMyWorkspaces(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	workspaces, err := h.Service.ListMyWorkspaces(r.Context(), userID)
	if err != nil {
		shared.InternalError(w, r, "workspace", "failed to list workspaces")
		return
	}

	shared.OK(w, r, workspaces)
}

// InviteMember godoc
// @Summary      Invite member to workspace
// @Description  Invite a member (entity) to the current workspace with a specific role.
// @Tags         Workspace Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string                          true  "Workspace ID"
// @Param        body            body      workspace.InviteMemberRequest   true  "Member data"
// @Success      201             {object}  shared.APIResponse{data=workspace.MemberResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Failure      401             {object}  shared.ErrorResponse
// @Failure      403             {object}  shared.ErrorResponse
// @Router       /workspaces/members [post]
func (h *WorkspaceHandler) InviteMember(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	var req workspace.InviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "workspace", "invalid request body")
		return
	}

	if req.EntityID == "" || req.Role == "" {
		shared.BadRequest(w, r, "workspace", "entity_id and role are required")
		return
	}

	resp, err := h.Service.InviteMember(r.Context(), workspaceID, &req)
	if err != nil {
		shared.BadRequest(w, r, "workspace", err.Error())
		return
	}

	shared.Created(w, r, resp)
}

// ListMembers godoc
// @Summary      List workspace members
// @Description  Returns all members (PEMILIK + KARYAWAN) of the current workspace.
// @Tags         Workspace Members
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true  "Workspace ID"
// @Success      200             {object}  shared.APIResponse{data=[]workspace.MemberResponse}
// @Failure      401             {object}  shared.ErrorResponse
// @Failure      403             {object}  shared.ErrorResponse
// @Router       /workspaces/members [get]
func (h *WorkspaceHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	members, err := h.Service.ListMembers(r.Context(), workspaceID)
	if err != nil {
		shared.InternalError(w, r, "workspace", "failed to list members")
		return
	}

	shared.OK(w, r, members)
}

// UpdateMember godoc
// @Summary      Update workspace member
// @Description  Update a member's role, alias, or status.
// @Tags         Workspace Members
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string                          true  "Workspace ID"
// @Param        id              path      string                          true  "Relation ID (UUID)"
// @Param        body            body      workspace.UpdateMemberRequest   true  "Update data"
// @Success      200             {object}  shared.APIResponse{data=workspace.MessageResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Failure      401             {object}  shared.ErrorResponse
// @Failure      403             {object}  shared.ErrorResponse
// @Router       /workspaces/members/{id} [patch]
func (h *WorkspaceHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	relationID := chi.URLParam(r, "id")

	var req workspace.UpdateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "workspace", "invalid request body")
		return
	}

	if err := h.Service.UpdateMember(r.Context(), relationID, &req); err != nil {
		shared.BadRequest(w, r, "workspace", err.Error())
		return
	}

	shared.OK(w, r, workspace.MessageResponse{Message: "Member updated"})
}

// RemoveMember godoc
// @Summary      Remove workspace member
// @Description  Remove a member from the workspace. Cannot remove the owner.
// @Tags         Workspace Members
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true  "Workspace ID"
// @Param        id              path      string  true  "Relation ID (UUID)"
// @Success      200             {object}  shared.APIResponse{data=workspace.MessageResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Failure      401             {object}  shared.ErrorResponse
// @Failure      403             {object}  shared.ErrorResponse
// @Router       /workspaces/members/{id} [delete]
func (h *WorkspaceHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	relationID := chi.URLParam(r, "id")

	if err := h.Service.RemoveMember(r.Context(), relationID); err != nil {
		shared.BadRequest(w, r, "workspace", err.Error())
		return
	}

	shared.OK(w, r, workspace.MessageResponse{Message: "Member removed"})
}

// AddCounterparty godoc
// @Summary      Add counterparty
// @Description  Add a counterparty (customer/vendor) to workspace. Creates shadow entity if entity_id is not provided.
// @Tags         Workspace Counterparties
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string                              true  "Workspace ID"
// @Param        body            body      workspace.AddCounterpartyRequest    true  "Counterparty data"
// @Success      201             {object}  shared.APIResponse{data=workspace.CounterpartyResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Failure      401             {object}  shared.ErrorResponse
// @Failure      403             {object}  shared.ErrorResponse
// @Router       /workspaces/counterparties [post]
func (h *WorkspaceHandler) AddCounterparty(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	var req workspace.AddCounterpartyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "workspace", "invalid request body")
		return
	}

	if req.RelationType == "" {
		shared.BadRequest(w, r, "workspace", "relation_type is required")
		return
	}

	resp, err := h.Service.AddCounterparty(r.Context(), workspaceID, &req)
	if err != nil {
		shared.BadRequest(w, r, "workspace", err.Error())
		return
	}

	shared.Created(w, r, resp)
}

// ListCounterparties godoc
// @Summary      List counterparties
// @Description  Returns all counterparties (PELANGGAN + VENDOR) of the current workspace.
// @Tags         Workspace Counterparties
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true  "Workspace ID"
// @Success      200             {object}  shared.APIResponse{data=[]workspace.CounterpartyResponse}
// @Failure      401             {object}  shared.ErrorResponse
// @Failure      403             {object}  shared.ErrorResponse
// @Router       /workspaces/counterparties [get]
func (h *WorkspaceHandler) ListCounterparties(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	counterparties, err := h.Service.ListCounterparties(r.Context(), workspaceID)
	if err != nil {
		shared.InternalError(w, r, "workspace", "failed to list counterparties")
		return
	}

	shared.OK(w, r, counterparties)
}

// ListRoles godoc
// @Summary      List available roles
// @Description  Returns all available roles that can be assigned to workspace members.
// @Tags         Workspaces
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  shared.APIResponse{data=[]workspace.RoleResponse}
// @Router       /roles [get]
func (h *WorkspaceHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.Service.ListRoles(r.Context())
	if err != nil {
		shared.InternalError(w, r, "workspace", "failed to list roles")
		return
	}

	shared.OK(w, r, roles)
}
