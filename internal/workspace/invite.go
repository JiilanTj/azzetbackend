package workspace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"codeberg.org/azzet/azzetbe/internal/db"
	smtpclient "codeberg.org/azzet/azzetbe/internal/smtp"
)

const (
	InviteTokenLength = 32 // 32 bytes = 64 hex chars
	InviteExpiry      = 24 * time.Hour
)

var ErrInviteNotFound = errors.New("invite not found")
var ErrInviteExpired = errors.New("invite has expired")
var ErrInviteAlreadyAccepted = errors.New("invite already accepted")
var ErrEmailNotRegistered = errors.New("email is not registered on this platform")
var ErrEmailMismatch = errors.New("logged in user email does not match invite")

// InviteService handles workspace invitation logic
type InviteService struct {
	Queries     *db.Queries
	Mailer      *smtpclient.Mailer
	FrontendURL string
}

func NewInviteService(queries *db.Queries, mailer *smtpclient.Mailer, frontendURL string) *InviteService {
	return &InviteService{
		Queries:     queries,
		Mailer:      mailer,
		FrontendURL: frontendURL,
	}
}

// CreateInvite creates a workspace invite and sends email
func (s *InviteService) CreateInvite(ctx context.Context, workspaceID, inviterID string, req *CreateInviteRequest) (*InviteResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrWorkspaceNotFound
	}
	uid, err := uuid.Parse(inviterID)
	if err != nil {
		return nil, fmt.Errorf("invalid inviter_id")
	}
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		return nil, fmt.Errorf("invalid role_id")
	}

	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}

	// Verify the email is registered
	invitedUser, err := s.Queries.GetUserByEmail(ctx, pgtype.Text{String: req.Email, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEmailNotRegistered
		}
		return nil, err
	}

	// Check if user is already a member of this workspace
	invitedEntities, _ := s.Queries.ListEntitiesByUserID(ctx, pgtype.UUID{Bytes: invitedUser.ID, Valid: true})
	for _, e := range invitedEntities {
		if e.EntityType == "ORANG_PRIBADI" {
			existsKaryawan, _ := s.Queries.ExistsRelation(ctx, db.ExistsRelationParams{
				ObjectID:     wsID,
				SubjectID:    e.ID,
				RelationType: RelationKaryawan,
			})
			existsPemilik, _ := s.Queries.ExistsRelation(ctx, db.ExistsRelationParams{
				ObjectID:     wsID,
				SubjectID:    e.ID,
				RelationType: RelationPemilik,
			})
			if existsKaryawan || existsPemilik {
				return nil, fmt.Errorf("user tersebut sudah menjadi anggota workspace ini")
			}
			break
		}
	}

	// Check if there's already a pending invite for this email + workspace
	existsPending, err := s.Queries.ExistsPendingInvite(ctx, db.ExistsPendingInviteParams{
		WorkspaceID:  wsID,
		InvitedEmail: req.Email,
	})
	if err != nil {
		return nil, err
	}
	if existsPending {
		return nil, fmt.Errorf("undangan untuk email ini sudah dikirim dan masih berlaku")
	}

	// Verify role belongs to this workspace
	role, err := s.Queries.GetWorkspaceRoleByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("role not found")
	}
	if role.WorkspaceID != wsID {
		return nil, fmt.Errorf("role does not belong to this workspace")
	}

	// Generate secure token
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite token")
	}

	now := time.Now()
	expiresAt := now.Add(InviteExpiry)

	invite, err := s.Queries.CreateInvite(ctx, db.CreateInviteParams{
		ID:           uuid.New(),
		WorkspaceID:  wsID,
		InvitedEmail: req.Email,
		RoleID:       roleID,
		Token:        token,
		InvitedBy:    uid,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create invite: %w", err)
	}

	// Get workspace name for email
	wsEntity, _ := s.Queries.GetEntityByID(ctx, wsID)
	workspaceName := "Workspace"
	if wsEntity.NamaUtama != "" {
		workspaceName = wsEntity.NamaUtama
	}

	// Get inviter name
	inviterUser, _ := s.Queries.GetUserByID(ctx, uid)
	inviterName := "Seseorang"
	if inviterUser.Name.Valid {
		inviterName = inviterUser.Name.String
	}

	// Send invite email (non-blocking error)
	inviteURL := fmt.Sprintf("%s/invite/%s", s.FrontendURL, token)
	if err := s.sendInviteEmail(req.Email, workspaceName, inviterName, role.Name, inviteURL); err != nil {
		slog.Error("failed to send invite email", "to", req.Email, "error", err)
		// Don't fail the invite creation — email delivery is best-effort
	}

	return &InviteResponse{
		ID:           invite.ID.String(),
		WorkspaceID:  invite.WorkspaceID.String(),
		InvitedEmail: invite.InvitedEmail,
		RoleName:     role.Name,
		Token:        invite.Token,
		InvitedBy:    invite.InvitedBy.String(),
		ExpiresAt:    invite.ExpiresAt.Format(time.RFC3339),
		CreatedAt:    invite.CreatedAt.Format(time.RFC3339),
	}, nil
}

// AcceptInvite accepts a workspace invite
func (s *InviteService) AcceptInvite(ctx context.Context, token, userID string) error {
	invite, err := s.Queries.GetInviteByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInviteNotFound
		}
		return err
	}

	// Check if already accepted
	if invite.AcceptedAt != nil {
		return ErrInviteAlreadyAccepted
	}

	// Check expiry
	if time.Now().After(invite.ExpiresAt) {
		return ErrInviteExpired
	}

	// Get user and verify email matches
	uid, _ := uuid.Parse(userID)
	user, err := s.Queries.GetUserByID(ctx, uid)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	userEmail := ""
	if user.Email.Valid {
		userEmail = user.Email.String
	}
	if userEmail == "" || userEmail != invite.InvitedEmail {
		return ErrEmailMismatch
	}

	// Get user's personal entity
	personalEntities, err := s.Queries.ListEntitiesByUserID(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil || len(personalEntities) == 0 {
		return fmt.Errorf("user has no personal entity")
	}

	// Find personal entity (ORANG_PRIBADI)
	var personalEntityID uuid.UUID
	for _, e := range personalEntities {
		if e.EntityType == "ORANG_PRIBADI" {
			personalEntityID = e.ID
			break
		}
	}
	if personalEntityID == uuid.Nil {
		return fmt.Errorf("user has no personal entity")
	}

	// Check if already a member (KARYAWAN or PEMILIK)
	existsKaryawan, err := s.Queries.ExistsRelation(ctx, db.ExistsRelationParams{
		ObjectID:     invite.WorkspaceID,
		SubjectID:    personalEntityID,
		RelationType: RelationKaryawan,
	})
	if err != nil {
		return err
	}
	existsPemilik, err := s.Queries.ExistsRelation(ctx, db.ExistsRelationParams{
		ObjectID:     invite.WorkspaceID,
		SubjectID:    personalEntityID,
		RelationType: RelationPemilik,
	})
	if err != nil {
		return err
	}
	if existsKaryawan || existsPemilik {
		// Already a member — just mark invite as accepted
		return s.Queries.AcceptInvite(ctx, invite.ID)
	}

	// Create entity relation (KARYAWAN)
	now := time.Now()
	_, err = s.Queries.CreateRelation(ctx, db.CreateRelationParams{
		ID:           uuid.New(),
		ObjectID:     invite.WorkspaceID,
		SubjectID:    personalEntityID,
		RelationType: RelationKaryawan,
		Status:       "ACTIVE",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return fmt.Errorf("failed to create membership: %w", err)
	}

	// Assign role
	_, err = s.Queries.CreateRoleAssignment(ctx, db.CreateRoleAssignmentParams{
		ID:             uuid.New(),
		WorkspaceID:    invite.WorkspaceID,
		MemberEntityID: personalEntityID,
		RoleID:         invite.RoleID,
		AssignedBy:     invite.InvitedBy,
		CreatedAt:      now,
	})
	if err != nil {
		slog.Error("failed to assign role after invite accept", "error", err)
		// Non-fatal — membership is created, role can be assigned later
	}

	// Mark invite as accepted
	return s.Queries.AcceptInvite(ctx, invite.ID)
}

// ListPendingInvites returns all pending invites for a workspace
func (s *InviteService) ListPendingInvites(ctx context.Context, workspaceID string) ([]InviteResponse, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrWorkspaceNotFound
	}

	invites, err := s.Queries.ListPendingInvitesByWorkspace(ctx, wsID)
	if err != nil {
		return nil, err
	}

	var resp []InviteResponse
	for _, inv := range invites {
		// Get role name
		roleName := ""
		role, err := s.Queries.GetWorkspaceRoleByID(ctx, inv.RoleID)
		if err == nil {
			roleName = role.Name
		}

		resp = append(resp, InviteResponse{
			ID:           inv.ID.String(),
			WorkspaceID:  inv.WorkspaceID.String(),
			InvitedEmail: inv.InvitedEmail,
			RoleName:     roleName,
			InvitedBy:    inv.InvitedBy.String(),
			ExpiresAt:    inv.ExpiresAt.Format(time.RFC3339),
			CreatedAt:    inv.CreatedAt.Format(time.RFC3339),
		})
	}
	if resp == nil {
		resp = []InviteResponse{}
	}
	return resp, nil
}

// RevokeInvite deletes a pending invite
func (s *InviteService) RevokeInvite(ctx context.Context, workspaceID, inviteID string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return ErrWorkspaceNotFound
	}
	invID, err := uuid.Parse(inviteID)
	if err != nil {
		return ErrInviteNotFound
	}

	invite, err := s.Queries.GetInviteByID(ctx, invID)
	if err != nil {
		return ErrInviteNotFound
	}

	if invite.WorkspaceID != wsID {
		return ErrInviteNotFound
	}

	if invite.AcceptedAt != nil {
		return fmt.Errorf("cannot revoke an already accepted invite")
	}

	return s.Queries.DeleteInvite(ctx, invID)
}

// --- Email ---

func (s *InviteService) sendInviteEmail(to, workspaceName, inviterName, roleName, inviteURL string) error {
	if s.Mailer == nil {
		slog.Warn("invite email: mailer not configured, skipping", "to", to)
		return nil
	}

	html := buildInviteEmailHTML(workspaceName, inviterName, roleName, inviteURL)

	return s.Mailer.Send(smtpclient.Email{
		To:       []string{to},
		Subject:  fmt.Sprintf("Undangan bergabung ke workspace \"%s\" di Azzet", workspaceName),
		HTMLBody: html,
	})
}

func buildInviteEmailHTML(workspaceName, inviterName, roleName, inviteURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Undangan Workspace Azzet</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f6f9;font-family:'Segoe UI',Arial,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f6f9;padding:40px 0;">
    <tr>
      <td align="center">
        <table width="560" cellpadding="0" cellspacing="0"
               style="background:#ffffff;border-radius:12px;overflow:hidden;
                      box-shadow:0 2px 8px rgba(0,0,0,.08);">

          <!-- Header -->
          <tr>
            <td align="center"
                style="background:linear-gradient(135deg,#2563eb 0%%,#1d4ed8 100%%);
                       padding:36px 40px;">
              <span style="font-size:26px;font-weight:700;color:#ffffff;
                           letter-spacing:-0.5px;">Azzet</span>
            </td>
          </tr>

          <!-- Body -->
          <tr>
            <td style="padding:40px 48px 32px;">
              <p style="margin:0 0 8px;font-size:22px;font-weight:600;
                        color:#111827;">Undangan Workspace</p>
              <p style="margin:0 0 28px;font-size:15px;color:#6b7280;
                        line-height:1.6;">
                <strong>%s</strong> mengundang Anda untuk bergabung ke workspace
                <strong>"%s"</strong> sebagai <strong>%s</strong> di platform Azzet.
              </p>

              <!-- CTA Button -->
              <table width="100%%" cellpadding="0" cellspacing="0" style="margin-bottom:28px;">
                <tr>
                  <td align="center">
                    <a href="%s"
                       style="display:inline-block;padding:14px 32px;
                              background:linear-gradient(135deg,#2563eb 0%%,#1d4ed8 100%%);
                              color:#ffffff;font-size:15px;font-weight:600;
                              text-decoration:none;border-radius:8px;">
                      Terima Undangan
                    </a>
                  </td>
                </tr>
              </table>

              <p style="margin:0 0 12px;font-size:13px;color:#9ca3af;line-height:1.6;word-break:break-all;">
                Atau salin link berikut: <a href="%s" style="color:#2563eb;text-decoration:underline;">%s</a>
              </p>

              <p style="margin:0 0 12px;font-size:14px;color:#9ca3af;line-height:1.6;">
                Link ini berlaku selama <strong>24 jam</strong>. Setelah itu, undangan akan kedaluwarsa
                dan Anda perlu meminta undangan baru.
              </p>

              <p style="margin:0 0 24px;font-size:13px;color:#d1d5db;line-height:1.6;">
                Jika Anda tidak mengenal pengirim undangan ini, abaikan email ini.
              </p>

              <hr style="border:none;border-top:1px solid #e5e7eb;margin:0 0 24px;" />

              <p style="margin:0;font-size:13px;color:#d1d5db;text-align:center;">
                &copy; Azzet &mdash; Platform Akuntansi &amp; Keuangan Enterprise
              </p>
            </td>
          </tr>

        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, inviterName, workspaceName, roleName, inviteURL, inviteURL, inviteURL)
}

// --- Helpers ---

func generateToken() (string, error) {
	b := make([]byte, InviteTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
