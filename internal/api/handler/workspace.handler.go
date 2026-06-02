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
	workspaceID := middleware.GetWorkspaceID(r.Context())
	relationID := chi.URLParam(r, "id")

	var req workspace.UpdateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "workspace", "invalid request body")
		return
	}

	if err := h.Service.UpdateMember(r.Context(), workspaceID, relationID, &req); err != nil {
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
	workspaceID := middleware.GetWorkspaceID(r.Context())
	relationID := chi.URLParam(r, "id")

	if err := h.Service.RemoveMember(r.Context(), workspaceID, relationID); err != nil {
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
// @Summary      List workspace roles
// @Description  Returns all custom roles for the current workspace.
// @Tags         Workspaces
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true  "Workspace ID"
// @Success      200  {object}  shared.APIResponse{data=[]workspace.RoleResponse}
// @Router       /workspaces/roles [get]
func (h *WorkspaceHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	roles, err := h.Service.ListWorkspaceRoles(r.Context(), workspaceID)
	if err != nil {
		shared.InternalError(w, r, "workspace", "failed to list roles")
		return
	}

	shared.OK(w, r, roles)
}

// CreateRole godoc
// @Summary      Create a custom workspace role
// @Description  Create a new custom role with specific permissions for the workspace.
// @Tags         Workspace Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string                          true  "Workspace ID"
// @Param        body            body      workspace.CreateRoleRequest     true  "Role data"
// @Success      201             {object}  shared.APIResponse{data=workspace.RoleResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Router       /workspaces/roles [post]
func (h *WorkspaceHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var req workspace.CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "workspace", "invalid request body")
		return
	}

	role, err := h.Service.CreateWorkspaceRole(r.Context(), workspaceID, userID, &req)
	if err != nil {
		shared.BadRequest(w, r, "workspace", err.Error())
		return
	}

	shared.Created(w, r, role)
}

// UpdateRole godoc
// @Summary      Update a workspace role
// @Description  Update name, description, or permissions of a custom role.
// @Tags         Workspace Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string                          true  "Workspace ID"
// @Param        id              path      string                          true  "Role ID"
// @Param        body            body      workspace.UpdateRoleRequest     true  "Role data"
// @Success      200             {object}  shared.APIResponse{data=workspace.MessageResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Router       /workspaces/roles/{id} [patch]
func (h *WorkspaceHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())
	roleID := chi.URLParam(r, "id")

	var req workspace.UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "workspace", "invalid request body")
		return
	}

	if err := h.Service.UpdateWorkspaceRole(r.Context(), workspaceID, roleID, &req); err != nil {
		shared.BadRequest(w, r, "workspace", err.Error())
		return
	}

	shared.OK(w, r, workspace.MessageResponse{Message: "Role updated"})
}

// DeleteRole godoc
// @Summary      Delete a workspace role
// @Description  Delete a custom role. System roles cannot be deleted.
// @Tags         Workspace Roles
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true  "Workspace ID"
// @Param        id              path      string  true  "Role ID"
// @Success      200             {object}  shared.APIResponse{data=workspace.MessageResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Router       /workspaces/roles/{id} [delete]
func (h *WorkspaceHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())
	roleID := chi.URLParam(r, "id")

	if err := h.Service.DeleteWorkspaceRole(r.Context(), workspaceID, roleID); err != nil {
		shared.BadRequest(w, r, "workspace", err.Error())
		return
	}

	shared.OK(w, r, workspace.MessageResponse{Message: "Role deleted"})
}

// AssignRole godoc
// @Summary      Assign a role to a member
// @Description  Assign a workspace role to a member entity.
// @Tags         Workspace Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string                          true  "Workspace ID"
// @Param        body            body      workspace.AssignRoleRequest     true  "Assignment data"
// @Success      201             {object}  shared.APIResponse{data=workspace.RoleAssignmentResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Router       /workspaces/roles/assign [post]
func (h *WorkspaceHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var req workspace.AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "workspace", "invalid request body")
		return
	}

	assignment, err := h.Service.AssignRole(r.Context(), workspaceID, userID, &req)
	if err != nil {
		shared.BadRequest(w, r, "workspace", err.Error())
		return
	}

	shared.Created(w, r, assignment)
}

// UnassignRole godoc
// @Summary      Remove a role from a member
// @Description  Remove a role assignment from a workspace member.
// @Tags         Workspace Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string                          true  "Workspace ID"
// @Param        body            body      workspace.AssignRoleRequest     true  "Assignment data"
// @Success      200             {object}  shared.APIResponse{data=workspace.MessageResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Router       /workspaces/roles/unassign [post]
func (h *WorkspaceHandler) UnassignRole(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	var req workspace.AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "workspace", "invalid request body")
		return
	}

	if err := h.Service.UnassignRole(r.Context(), workspaceID, &req); err != nil {
		shared.BadRequest(w, r, "workspace", err.Error())
		return
	}

	shared.OK(w, r, workspace.MessageResponse{Message: "Role unassigned"})
}

// --- Counterparty Alias & Search (Phase 8C) ---

// SetCounterpartyAlias godoc
// @Summary      Set custom alias for counterparty
// @Tags         Workspace
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        body body workspace.SetCounterpartyAliasRequest true "Alias request"
// @Success      200 {object} shared.APIResponse
// @Router       /workspaces/counterparties/aliases [post]
func (h *WorkspaceHandler) SetCounterpartyAlias(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	var req workspace.SetCounterpartyAliasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "workspace", "invalid request body")
		return
	}

	if req.EntityID == "" || req.CustomAlias == "" {
		shared.BadRequest(w, r, "workspace", "entity_id and custom_alias are required")
		return
	}

	resp, err := h.Service.SetCounterpartyAlias(r.Context(), workspaceID, &req)
	if err != nil {
		shared.BadRequest(w, r, "workspace", err.Error())
		return
	}

	shared.OK(w, r, resp)
}

// ListCounterpartyAliases godoc
// @Summary      List counterparty aliases
// @Tags         Workspace
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Success      200 {object} shared.APIResponse
// @Router       /workspaces/counterparties/aliases [get]
func (h *WorkspaceHandler) ListCounterpartyAliases(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	resp, err := h.Service.ListCounterpartyAliases(r.Context(), workspaceID)
	if err != nil {
		shared.InternalError(w, r, "workspace", "failed to list counterparty aliases")
		return
	}

	shared.OK(w, r, resp)
}

// DeleteCounterpartyAlias godoc
// @Summary      Delete counterparty alias
// @Tags         Workspace
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        entity_id path string true "Counterparty entity ID"
// @Success      200 {object} shared.APIResponse
// @Router       /workspaces/counterparties/aliases/{entity_id} [delete]
func (h *WorkspaceHandler) DeleteCounterpartyAlias(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())
	entityID := chi.URLParam(r, "entity_id")

	if err := h.Service.DeleteCounterpartyAlias(r.Context(), workspaceID, entityID); err != nil {
		shared.BadRequest(w, r, "workspace", err.Error())
		return
	}

	shared.OK(w, r, workspace.MessageResponse{Message: "Counterparty alias deleted"})
}

// SearchCounterparties godoc
// @Summary      Search counterparties by name
// @Tags         Workspace
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID header string true "Workspace entity ID"
// @Param        q query string true "Search query"
// @Success      200 {object} shared.APIResponse
// @Router       /workspaces/counterparties/search [get]
func (h *WorkspaceHandler) SearchCounterparties(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		shared.BadRequest(w, r, "workspace", "q parameter is required")
		return
	}

	workspaceID := middleware.GetWorkspaceID(r.Context())
	resp, err := h.Service.SearchCounterparties(r.Context(), workspaceID, query, 10)
	if err != nil {
		shared.InternalError(w, r, "workspace", "search failed")
		return
	}

	shared.OK(w, r, resp)
}
