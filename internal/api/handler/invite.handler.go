package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/shared"
	"codeberg.org/azzet/azzetbe/internal/workspace"
)

type InviteHandler struct {
	Service *workspace.InviteService
}

func NewInviteHandler(service *workspace.InviteService) *InviteHandler {
	return &InviteHandler{Service: service}
}

// CreateInvite godoc
// @Summary      Invite a user to workspace
// @Description  Send an email invitation to join the workspace. User must be registered.
// @Tags         Workspace Invites
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string                          true  "Workspace ID"
// @Param        body            body      workspace.CreateInviteRequest   true  "Invite data"
// @Success      201             {object}  shared.APIResponse{data=workspace.InviteResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Router       /workspaces/invites [post]
func (h *InviteHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var req workspace.CreateInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "invite", "invalid request body")
		return
	}

	resp, err := h.Service.CreateInvite(r.Context(), workspaceID, userID, &req)
	if err != nil {
		switch err {
		case workspace.ErrEmailNotRegistered:
			shared.BadRequest(w, r, "invite", "Email tersebut belum terdaftar di platform Azzet. Pastikan penerima sudah memiliki akun sebelum diundang.")
		default:
			shared.BadRequest(w, r, "invite", err.Error())
		}
		return
	}

	shared.Created(w, r, resp)
}

// ListInvites godoc
// @Summary      List pending invites
// @Description  Returns all pending (not yet accepted, not expired) invites for the workspace.
// @Tags         Workspace Invites
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true  "Workspace ID"
// @Success      200             {object}  shared.APIResponse{data=[]workspace.InviteResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Router       /workspaces/invites [get]
func (h *InviteHandler) ListInvites(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())

	resp, err := h.Service.ListPendingInvites(r.Context(), workspaceID)
	if err != nil {
		shared.InternalError(w, r, "invite", "failed to list invites")
		return
	}

	shared.OK(w, r, resp)
}

// RevokeInvite godoc
// @Summary      Revoke an invite
// @Description  Delete a pending invite before it's accepted.
// @Tags         Workspace Invites
// @Produce      json
// @Security     BearerAuth
// @Param        X-Workspace-ID  header    string  true  "Workspace ID"
// @Param        id              path      string  true  "Invite ID"
// @Success      200             {object}  shared.APIResponse{data=workspace.MessageResponse}
// @Failure      400             {object}  shared.ErrorResponse
// @Router       /workspaces/invites/{id} [delete]
func (h *InviteHandler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	workspaceID := middleware.GetWorkspaceID(r.Context())
	inviteID := chi.URLParam(r, "id")

	if err := h.Service.RevokeInvite(r.Context(), workspaceID, inviteID); err != nil {
		shared.BadRequest(w, r, "invite", err.Error())
		return
	}

	shared.OK(w, r, workspace.MessageResponse{Message: "Invite revoked"})
}

// AcceptInvite godoc
// @Summary      Accept a workspace invite
// @Description  Accept an invite using the token. User must be logged in and email must match.
// @Tags         Workspace Invites
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      workspace.AcceptInviteRequest  true  "Token"
// @Success      200   {object}  shared.APIResponse{data=workspace.MessageResponse}
// @Failure      400   {object}  shared.ErrorResponse
// @Router       /workspaces/invites/accept [post]
func (h *InviteHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req workspace.AcceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.BadRequest(w, r, "invite", "invalid request body")
		return
	}

	if req.Token == "" {
		shared.BadRequest(w, r, "invite", "token is required")
		return
	}

	if err := h.Service.AcceptInvite(r.Context(), req.Token, userID); err != nil {
		switch err {
		case workspace.ErrInviteNotFound:
			shared.NotFound(w, r, "invite", "Undangan tidak ditemukan atau sudah tidak berlaku.")
		case workspace.ErrInviteExpired:
			shared.BadRequest(w, r, "invite", "Undangan sudah kedaluwarsa. Minta pengirim untuk mengirim ulang undangan.")
		case workspace.ErrInviteAlreadyAccepted:
			shared.BadRequest(w, r, "invite", "Undangan sudah diterima sebelumnya.")
		case workspace.ErrEmailMismatch:
			shared.Forbidden(w, r, "invite", "Email akun Anda tidak sesuai dengan email yang diundang.")
		default:
			shared.BadRequest(w, r, "invite", err.Error())
		}
		return
	}

	shared.OK(w, r, workspace.MessageResponse{Message: "Undangan berhasil diterima. Workspace baru telah ditambahkan."})
}
