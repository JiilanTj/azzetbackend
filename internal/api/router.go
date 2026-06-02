package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	"codeberg.org/azzet/azzetbe/internal/accounting"
	"codeberg.org/azzet/azzetbe/internal/admin"
	"codeberg.org/azzet/azzetbe/internal/ai"
	"codeberg.org/azzet/azzetbe/internal/api/handler"
	"codeberg.org/azzet/azzetbe/internal/api/middleware"
	"codeberg.org/azzet/azzetbe/internal/auth"
	"codeberg.org/azzet/azzetbe/internal/billing"
	"codeberg.org/azzet/azzetbe/internal/claim"
	"codeberg.org/azzet/azzetbe/internal/config"
	"codeberg.org/azzet/azzetbe/internal/database"
	"codeberg.org/azzet/azzetbe/internal/db"
	"codeberg.org/azzet/azzetbe/internal/document"
	"codeberg.org/azzet/azzetbe/internal/entity"
	"codeberg.org/azzet/azzetbe/internal/identity"
	"codeberg.org/azzet/azzetbe/internal/plan"
	rdb "codeberg.org/azzet/azzetbe/internal/redis"
	"codeberg.org/azzet/azzetbe/internal/shared"
	smtpclient "codeberg.org/azzet/azzetbe/internal/smtp"
	"codeberg.org/azzet/azzetbe/internal/storage"
	"codeberg.org/azzet/azzetbe/internal/subscription"
	"codeberg.org/azzet/azzetbe/internal/tax"
	"codeberg.org/azzet/azzetbe/internal/workspace"
)

func NewRouter(cfg *config.Config, database *database.Database, redis *rdb.Redis) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))
	r.Use(middleware.MaxBodySize(1 << 20)) // 1MB request body limit

	// Swagger documentation (development only)
	if cfg.AppEnv == "development" {
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
	}

	secureCookie := cfg.AppEnv != "development"
	queries := db.New(database.Pool)
	otpService := shared.NewOTPService(6)
	zenzivaClient := shared.NewZenzivaClient(cfg.ZenzivaURL, cfg.ZenzivaUserKey, cfg.ZenzivaPassKey, cfg.ZenzivaBrand, cfg.AppEnv)
	emailSender := shared.NewEmailOTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom, cfg.AppEnv)

	// --- Entity & Workspace ---
	entityService := entity.NewService(queries)
	workspaceService := workspace.NewService(queries, entityService)
	workspaceService.Pool = database.Pool

	// --- User Auth ---
	userAccessExpiry := time.Duration(cfg.AccessTokenExpiryMinutes) * time.Minute
	userRefreshExpiry := time.Duration(cfg.RefreshTokenExpiryDays) * 24 * time.Hour
	userJWT := shared.NewJWTService(cfg.AppSecret, cfg.RefreshTokenSecret, userAccessExpiry, userRefreshExpiry)

	authService := auth.NewService(queries, database.Pool, redis, userJWT, otpService, zenzivaClient, emailSender, &auth.ServiceConfig{
		AccessTokenExpiry:  userAccessExpiry,
		RefreshTokenExpiry: userRefreshExpiry,
		OTPExpiry:          5 * time.Minute,
		OTPMaxAttempts:     3,
	})
	authService.EntityService = entityService
	authService.WorkspaceService = workspaceService

	userIsBlacklisted := func(ctx context.Context, jti string) (bool, error) {
		return authService.IsTokenBlacklisted(ctx, jti)
	}
	getUserStatus := func(ctx context.Context, userID string) (string, error) {
		u, err := authService.GetMe(ctx, userID)
		if err != nil {
			return "", err
		}
		return u.Status, nil
	}
	authMiddleware := middleware.NewAuthMiddleware(userJWT, userIsBlacklisted, getUserStatus)

	authHandler := handler.NewAuthHandler(authService, userRefreshExpiry, secureCookie)

	// --- Admin Auth ---
	adminJWT := shared.NewJWTService(cfg.AdminJWTSecret, cfg.AdminRefreshTokenSecret, admin.AdminAccessTokenExpiry, admin.AdminRefreshTokenExpiry)
	adminService := admin.NewService(queries, redis, adminJWT)

	adminIsBlacklisted := func(ctx context.Context, jti string) (bool, error) {
		return adminService.IsTokenBlacklisted(ctx, jti)
	}
	getAdminRole := func(ctx context.Context, adminID string) (string, error) {
		a, err := adminService.GetMe(ctx, adminID)
		if err != nil {
			return "", err
		}
		if a.Status != admin.AdminStatusActive {
			return "", fmt.Errorf("admin account is not active")
		}
		return a.Role, nil
	}
	adminMiddleware := middleware.NewAdminMiddleware(adminJWT, adminIsBlacklisted, getAdminRole)

	adminHandler := handler.NewAdminHandler(adminService, admin.AdminRefreshTokenExpiry, secureCookie)

	// --- Plan ---
	planService := plan.NewService(queries)
	planHandler := handler.NewPlanHandler(planService)

	// --- Subscription ---
	subscriptionService := subscription.NewService(queries)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService)

	// --- Billing ---
	xenditClient := billing.NewXenditClient(cfg.XenditAPIKey, cfg.XenditWebhookSecret, cfg.XenditCallbackURL, cfg.XenditSuccessURL, cfg.XenditFailureURL)
	billingService := billing.NewService(queries, xenditClient)
	billingHandler := handler.NewBillingHandler(billingService)

	// Wire billing into subscription for paid plan flow
	subscriptionService.BillingService = billingService

	// Wire subscription into auth for free plan auto-assign on register
	authService.SubscriptionService = subscriptionService

	// --- Entity & Workspace Handlers ---
	entityHandler := handler.NewEntityHandler(entityService)
	workspaceHandler := handler.NewWorkspaceHandler(workspaceService)

	// --- Invite ---
	mailer := smtpclient.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	inviteService := workspace.NewInviteService(queries, mailer, cfg.FrontendURL)
	inviteHandler := handler.NewInviteHandler(inviteService)

	// --- Workspace Middleware ---
	workspaceMiddleware := middleware.NewWorkspaceMiddleware(workspaceService.VerifyWorkspaceAccess)

	// --- Accounting ---
	aiClient := ai.NewFromEnv(cfg.OpenAIApiKey, cfg.OpenAIModel)
	coaService := accounting.NewCOAService(queries, database.Pool)
	itemService := accounting.NewItemService(queries)
	categorizer := accounting.NewCategorizer(aiClient)
	accountingService := accounting.NewService(queries, database.Pool, coaService, itemService, categorizer)
	reportService := accounting.NewReportService(queries)
	accountingHandler := handler.NewAccountingHandler(accountingService, coaService, itemService, reportService)

	// --- Identity & Claim (Phase 8) ---
	identityService := identity.NewService(queries, database.Pool)
	var r2Client *storage.R2Client
	r2Client, err := storage.NewR2Client(cfg)
	if err != nil {
		slog.Warn("R2 storage not configured; claim document upload will be unavailable", "error", err)
	}
	claimService := claim.NewService(queries, database.Pool, r2Client, identityService, workspaceService)
	identityHandler := handler.NewIdentityHandler(identityService)
	claimHandler := handler.NewClaimHandler(claimService)
	claimAdminHandler := handler.NewClaimAdminHandler(claimService)

	// --- Document & OCR (Phase 9) ---
	documentService := document.NewService(queries, database.Pool, r2Client, subscriptionService)
	documentHandler := handler.NewDocumentHandler(documentService)

	// --- Tax (Phase 10) ---
	taxService := tax.NewService(queries, database.Pool)
	taxHandler := handler.NewTaxHandler(taxService)

	// ═══════════════════════════════════════════════════════════════
	// USER API ROUTES (/api/v1)
	// ═══════════════════════════════════════════════════════════════
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.CORSAllowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key", "X-Device-Name", "X-Workspace-ID"},
			AllowCredentials: true,
			MaxAge:           300,
		}))

		r.Get("/health", HealthCheck(database, redis))

		// Public plans (no auth required)
		r.Route("/plans", func(r chi.Router) {
			r.Get("/", planHandler.ListPlans)
			r.Get("/{slug}", planHandler.GetPlanBySlug)
		})

		// Public auth routes (rate-limited)
		authRateLimit := middleware.RateLimit(redis, "auth", 20, time.Minute)
		r.Route("/auth", func(r chi.Router) {
			r.With(authRateLimit).Post("/register", authHandler.Register)
			r.With(authRateLimit).Post("/login/email", authHandler.LoginEmail)
			r.With(authRateLimit).Post("/login/otp", authHandler.LoginOTP)
			r.With(authRateLimit).Post("/otp/request", authHandler.RequestOTP)
			r.Post("/refresh", authHandler.RefreshToken)
			r.With(authRateLimit).Post("/verify", authHandler.VerifyOTP)
			r.With(authRateLimit).Post("/password/reset", authHandler.ResetPassword)

			// Protected auth routes
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.Authenticate)
				r.Get("/me", authHandler.Me)
				r.Post("/logout", authHandler.Logout)
				r.Post("/logout-all", authHandler.LogoutAll)
				r.Post("/password/change", authHandler.ChangePassword)
				r.Get("/sessions", authHandler.GetSessions)
				r.Delete("/sessions/{id}", authHandler.RevokeSession)
			})
		})

		// Entity routes (authenticated)
		r.Route("/entities", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Post("/", entityHandler.CreateEntity)
			r.Get("/", entityHandler.ListMyEntities)
			r.Get("/search", entityHandler.SearchEntities)
			r.Get("/match", identityHandler.SearchFuzzy)
			r.Get("/{id}", entityHandler.GetEntity)
			r.Patch("/{id}", entityHandler.UpdateEntity)
			r.Patch("/{id}/meta", entityHandler.UpdateEntityMeta)
			r.Get("/{id}/verification", identityHandler.GetVerificationStatus)
			r.Post("/{id}/legal-ids", identityHandler.AddLegalID)
			r.Get("/{id}/legal-ids", identityHandler.GetLegalIDs)
			r.Patch("/{id}/legal-ids/{type}", identityHandler.UpdateLegalID)
			r.Delete("/{id}/legal-ids/{type}", identityHandler.DeleteLegalID)
			r.Post("/{id}/aliases", identityHandler.AddAlias)
			r.Get("/{id}/aliases", identityHandler.GetAliases)
			r.Delete("/{id}/aliases/{alias_id}", identityHandler.DeleteAlias)
			r.Get("/{id}/duplicates", identityHandler.FindDuplicates)
		})

		// Claims (authenticated)
		r.Route("/claims", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Post("/", claimHandler.CreateClaim)
			r.Get("/", claimHandler.GetMyClaims)
			r.Get("/{id}", claimHandler.GetClaim)
			r.Post("/{id}/submit", claimHandler.SubmitClaim)
			r.Post("/{id}/documents", claimHandler.RequestUpload)
			r.Post("/{id}/documents/{doc_id}/confirm", claimHandler.ConfirmUpload)
			r.Get("/{id}/documents", claimHandler.GetClaimDocuments)
			r.Post("/{id}/dispute", claimHandler.DisputeClaim)
		})

		// Workspace routes (authenticated)
		r.Route("/workspaces", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Post("/", workspaceHandler.CreateWorkspace)
			r.Get("/", workspaceHandler.ListMyWorkspaces)

			// Accept invite (auth-only, no workspace scope needed)
			r.Post("/invites/accept", inviteHandler.AcceptInvite)

			// Workspace-scoped routes (requires X-Workspace-ID header)
			r.Group(func(r chi.Router) {
				r.Use(workspaceMiddleware.RequireWorkspace)
				r.Use(workspaceMiddleware.RequireAssignedRole)

				r.Route("/members", func(r chi.Router) {
					r.With(workspaceMiddleware.RequirePermission(workspace.PermMemberInvite)).Get("/", workspaceHandler.ListMembers)
					r.With(workspaceMiddleware.RequirePermission("member:manage")).Patch("/{id}", workspaceHandler.UpdateMember)
					r.With(workspaceMiddleware.RequirePermission("member:remove")).Delete("/{id}", workspaceHandler.RemoveMember)
				})

				r.Route("/roles", func(r chi.Router) {
					r.With(workspaceMiddleware.RequirePermission(workspace.PermRoleAssign)).Get("/", workspaceHandler.ListRoles)
					r.With(workspaceMiddleware.RequirePermission("role:create")).Post("/", workspaceHandler.CreateRole)
					r.With(workspaceMiddleware.RequirePermission("role:update")).Patch("/{id}", workspaceHandler.UpdateRole)
					r.With(workspaceMiddleware.RequirePermission("role:delete")).Delete("/{id}", workspaceHandler.DeleteRole)
					r.With(workspaceMiddleware.RequirePermission("role:assign")).Post("/assign", workspaceHandler.AssignRole)
					r.With(workspaceMiddleware.RequirePermission("role:assign")).Post("/unassign", workspaceHandler.UnassignRole)
				})

				r.Route("/invites", func(r chi.Router) {
					r.With(workspaceMiddleware.RequirePermission("member:invite")).Post("/", inviteHandler.CreateInvite)
					r.With(workspaceMiddleware.RequirePermission("member:invite")).Get("/", inviteHandler.ListInvites)
					r.With(workspaceMiddleware.RequirePermission("member:invite")).Delete("/{id}", inviteHandler.RevokeInvite)
				})

				r.Route("/counterparties", func(r chi.Router) {
					r.With(workspaceMiddleware.RequirePermission(workspace.PermTransactionCreate)).Post("/", workspaceHandler.AddCounterparty)
					r.With(workspaceMiddleware.RequirePermission(workspace.PermTransactionRead)).Get("/", workspaceHandler.ListCounterparties)
					r.With(workspaceMiddleware.RequirePermission(workspace.PermTransactionRead)).Get("/search", workspaceHandler.SearchCounterparties)
					r.With(workspaceMiddleware.RequirePermission(workspace.PermTransactionUpdate)).Post("/aliases", workspaceHandler.SetCounterpartyAlias)
					r.With(workspaceMiddleware.RequirePermission(workspace.PermTransactionRead)).Get("/aliases", workspaceHandler.ListCounterpartyAliases)
					r.With(workspaceMiddleware.RequirePermission(workspace.PermTransactionUpdate)).Delete("/aliases/{entity_id}", workspaceHandler.DeleteCounterpartyAlias)
				})
			})
		})

		// Subscription routes (workspace-scoped)
		r.Route("/subscription", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(workspaceMiddleware.RequireWorkspace)
			r.With(workspaceMiddleware.RequirePermission(workspace.PermBillingManage)).Post("/", subscriptionHandler.Subscribe)
			r.With(workspaceMiddleware.RequirePermission(workspace.PermBillingRead)).Get("/", subscriptionHandler.GetActive)
			r.With(workspaceMiddleware.RequirePermission(workspace.PermBillingRead)).Get("/history", subscriptionHandler.ListSubscriptions)
			r.With(workspaceMiddleware.RequirePermission(workspace.PermBillingManage)).Post("/cancel", subscriptionHandler.Cancel)
			r.With(workspaceMiddleware.RequirePermission(workspace.PermBillingManage)).Post("/change", subscriptionHandler.ChangePlan)
			r.With(workspaceMiddleware.RequirePermission(workspace.PermBillingRead)).Get("/usage", subscriptionHandler.GetUsage)
		})

		// Billing routes (workspace-scoped)
		r.Route("/billing", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(workspaceMiddleware.RequireWorkspace)
			r.With(workspaceMiddleware.RequirePermission(workspace.PermBillingRead)).Get("/invoices", billingHandler.ListInvoices)
			r.With(workspaceMiddleware.RequirePermission(workspace.PermBillingRead)).Get("/invoices/{id}", billingHandler.GetInvoice)
			r.With(workspaceMiddleware.RequirePermission(workspace.PermBillingManage)).Post("/pay", billingHandler.PayInvoice)
			r.With(workspaceMiddleware.RequirePermission(workspace.PermBillingRead)).Get("/payments", billingHandler.ListPayments)
		})

		// Xendit webhook (public, verified by x-callback-token)
		r.Post("/webhooks/xendit", billingHandler.XenditWebhook)

		// Accounting routes (workspace-scoped)
		r.Route("/accounts", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(workspaceMiddleware.RequireWorkspace)
			r.With(workspaceMiddleware.RequirePermission("transaction:read")).Get("/", accountingHandler.ListAccounts)
			r.With(workspaceMiddleware.RequirePermission("transaction:create")).Post("/", accountingHandler.CreateAccount)
			r.With(workspaceMiddleware.RequirePermission("transaction:read")).Get("/{id}", accountingHandler.GetAccount)
			r.With(workspaceMiddleware.RequirePermission("transaction:create")).Patch("/{id}", accountingHandler.UpdateAccount)
		})

		r.Route("/items", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(workspaceMiddleware.RequireWorkspace)
			r.With(workspaceMiddleware.RequirePermission("transaction:read")).Get("/", accountingHandler.ListItems)
			r.With(workspaceMiddleware.RequirePermission("transaction:create")).Post("/", accountingHandler.CreateItem)
			r.With(workspaceMiddleware.RequirePermission("transaction:read")).Get("/{id}", accountingHandler.GetItem)
			r.With(workspaceMiddleware.RequirePermission("transaction:create")).Patch("/{id}", accountingHandler.UpdateItem)
			r.With(workspaceMiddleware.RequirePermission("transaction:create")).Delete("/{id}", accountingHandler.DeleteItem)
			r.With(workspaceMiddleware.RequirePermission("transaction:create")).Post("/{id}/reactivate", accountingHandler.ReactivateItem)
		})

		r.Route("/transactions", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(workspaceMiddleware.RequireWorkspace)
			r.With(workspaceMiddleware.RequirePermission("transaction:create")).Post("/", accountingHandler.CreateTransaction)
			r.With(workspaceMiddleware.RequirePermission("transaction:read")).Get("/", accountingHandler.ListTransactions)
			r.With(workspaceMiddleware.RequirePermission("transaction:create")).Post("/categorize", accountingHandler.CategorizeTransaction)
			r.With(workspaceMiddleware.RequirePermission("transaction:read")).Get("/{id}", accountingHandler.GetTransaction)
			r.With(workspaceMiddleware.RequirePermission("transaction:void")).Post("/{id}/void", accountingHandler.VoidTransaction)
		})

		r.Route("/documents", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(workspaceMiddleware.RequireWorkspace)
			r.With(workspaceMiddleware.RequirePermission("transaction:create")).Post("/", documentHandler.RequestUpload)
			r.With(workspaceMiddleware.RequirePermission("transaction:read")).Get("/", documentHandler.ListDocuments)
			r.With(workspaceMiddleware.RequirePermission("transaction:read")).Get("/{id}", documentHandler.GetDocument)
			r.With(workspaceMiddleware.RequirePermission("transaction:create")).Post("/{id}/confirm", documentHandler.ConfirmUpload)
		})

		r.Route("/reports", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(workspaceMiddleware.RequireWorkspace)
			r.Use(workspaceMiddleware.RequirePermission("report:read"))
			r.Get("/trial-balance", accountingHandler.GetTrialBalance)
			r.Get("/balance-sheet", accountingHandler.GetBalanceSheet)
			r.Get("/income-statement", accountingHandler.GetIncomeStatement)
			r.Get("/cash-flow", accountingHandler.GetCashFlow)
			r.Get("/ledger/{account_id}", accountingHandler.GetGeneralLedger)
		})

		r.Route("/tax", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(workspaceMiddleware.RequireWorkspace)
			r.With(workspaceMiddleware.RequirePermission("report:read")).Get("/profile", taxHandler.GetProfile)
			r.With(workspaceMiddleware.RequirePermission("workspace:settings")).Put("/profile", taxHandler.UpdateProfile)
			r.With(workspaceMiddleware.RequirePermission("report:read")).Get("/calculations", taxHandler.ListCalculations)
			r.With(workspaceMiddleware.RequirePermission("report:read")).Get("/calculations/{id}", taxHandler.GetCalculation)
			r.With(workspaceMiddleware.RequirePermission("report:read")).Get("/calculations/{id}/documents", taxHandler.ListDocumentRefs)
			r.With(workspaceMiddleware.RequirePermission("transaction:create")).Post("/calculations/{id}/documents", taxHandler.LinkDocument)
			r.With(workspaceMiddleware.RequirePermission("report:read")).Get("/summary/ppn", taxHandler.GetPPNSummary)
			r.With(workspaceMiddleware.RequirePermission("report:read")).Get("/summary/pph", taxHandler.GetPPhSummary)
			r.With(workspaceMiddleware.RequirePermission("report:read")).Get("/reports", taxHandler.ListReportJobs)
			r.With(workspaceMiddleware.RequirePermission("report:export")).Post("/reports", taxHandler.RequestReport)
			r.With(workspaceMiddleware.RequirePermission("report:read")).Get("/reports/{id}", taxHandler.GetReportJob)
		})
	})

	// ═══════════════════════════════════════════════════════════════
	// ADMIN API ROUTES (/api/v1/admin)
	// ═══════════════════════════════════════════════════════════════
	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.AdminCORSAllowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: true,
			MaxAge:           300,
		}))

		// Public admin auth (rate-limited)
		adminAuthRateLimit := middleware.RateLimit(redis, "admin-auth", 10, time.Minute)
		r.Route("/auth", func(r chi.Router) {
			r.With(adminAuthRateLimit).Post("/login", adminHandler.Login)
			r.With(adminAuthRateLimit).Post("/mfa/verify", adminHandler.VerifyMFA)
			r.Post("/refresh", adminHandler.RefreshToken)

			r.Group(func(r chi.Router) {
				r.Use(adminMiddleware.Authenticate)
				r.Use(adminMiddleware.RequireMFASetupScope)
				r.Post("/mfa/setup", adminHandler.SetupMFA)
				r.Post("/mfa/confirm", adminHandler.ConfirmMFASetup)
			})

			r.Group(func(r chi.Router) {
				r.Use(adminMiddleware.Authenticate)
				r.Use(adminMiddleware.RequireFullAuth)
				r.Post("/logout", adminHandler.Logout)
				r.Get("/me", adminHandler.Me)
			})
		})

		// Admin management (SUPER_ADMIN only)
		r.Route("/admins", func(r chi.Router) {
			r.Use(adminMiddleware.Authenticate)
			r.Use(adminMiddleware.RequireFullAuth)
			r.Use(adminMiddleware.RequireRole(admin.RoleSuperAdmin))
			r.Post("/", adminHandler.InviteAdmin)
			r.Get("/", adminHandler.ListAdmins)
			r.Patch("/{id}", adminHandler.UpdateAdmin)
			r.Delete("/{id}", adminHandler.DeleteAdmin)
		})

		// Plan management (SUPER_ADMIN + ENGINEER)
		r.Route("/plans", func(r chi.Router) {
			r.Use(adminMiddleware.Authenticate)
			r.Use(adminMiddleware.RequireFullAuth)
			r.Use(adminMiddleware.RequireRole(admin.RoleSuperAdmin, admin.RoleEngineer))
			r.Get("/", planHandler.AdminListPlans)
			r.Post("/", planHandler.AdminCreatePlan)
			r.Get("/{id}", planHandler.AdminGetPlan)
			r.Patch("/{id}", planHandler.AdminUpdatePlan)
			r.Delete("/{id}", planHandler.AdminDeletePlan)
			r.Post("/{id}/features", planHandler.AdminSetFeature)
			r.Delete("/{id}/features/{feature_key}", planHandler.AdminRemoveFeature)
		})

		// Subscription management (SUPER_ADMIN + ENGINEER)
		r.Route("/subscriptions", func(r chi.Router) {
			r.Use(adminMiddleware.Authenticate)
			r.Use(adminMiddleware.RequireFullAuth)
			r.Use(adminMiddleware.RequireRole(admin.RoleSuperAdmin, admin.RoleEngineer))
			r.Get("/", subscriptionHandler.AdminListSubscriptions)
		})

		// Billing management (SUPER_ADMIN + ENGINEER)
		r.Route("/billing", func(r chi.Router) {
			r.Use(adminMiddleware.Authenticate)
			r.Use(adminMiddleware.RequireFullAuth)
			r.Use(adminMiddleware.RequireRole(admin.RoleSuperAdmin, admin.RoleEngineer))
			r.Get("/invoices", billingHandler.AdminListInvoices)
		})

		// Company claims review (REVIEWER+)
		r.Route("/claims", func(r chi.Router) {
			r.Use(adminMiddleware.Authenticate)
			r.Use(adminMiddleware.RequireFullAuth)
			r.Use(adminMiddleware.RequireRole(admin.RoleSuperAdmin, admin.RoleEngineer, admin.RoleReviewer))
			r.Get("/", claimAdminHandler.ListClaims)
			r.Get("/stats", claimAdminHandler.CountPendingClaims)
			r.Get("/{id}", claimAdminHandler.GetClaim)
			r.Post("/{id}/assign", claimAdminHandler.AssignClaim)
			r.Post("/{id}/approve", claimAdminHandler.ApproveClaim)
			r.Post("/{id}/reject", claimAdminHandler.RejectClaim)
			r.Get("/{id}/audit", claimAdminHandler.GetClaimAuditLog)
			r.Get("/{id}/documents/{doc_id}/view", claimAdminHandler.GetDocumentViewURL)
		})

		// TODO: User management (SUPPORT+)
		// TODO: System monitoring (ENGINEER+)
	})

	return r
}
