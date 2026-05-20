package events

// --- Domain Event Types ---

// Accounting
const (
	TransactionCreated       = "accounting.transaction.created"
	LedgerPostingRequested   = "accounting.ledger.posting_requested"
	LedgerPosted             = "accounting.ledger.posted"
	JournalEntryCreated      = "accounting.journal_entry.created"
)

// Company & Identity
const (
	CompanyRegistered        = "company.registered"
	CompanyCandidateCreated  = "company.candidate_created"
	CompanyClaimRequested    = "company.claim_requested"
	CompanyClaimApproved     = "company.claim_approved"
	CompanyClaimRejected     = "company.claim_rejected"
)

// Counterparty
const (
	CounterpartyCreated      = "counterparty.reference_created"
	CounterpartyMatched      = "counterparty.matched"
)

// Document
const (
	DocumentUploaded         = "document.uploaded"
	DocumentExtractionReq    = "document.extraction_requested"
	DocumentExtracted        = "document.extracted"
	DocumentVerified         = "document.verified"
)

// User & Auth
const (
	UserRegistered           = "user.registered"
	UserVerified             = "user.verified"
)

// Tenant & Workspace
const (
	WorkspaceCreated         = "workspace.created"
	WorkspaceMemberInvited   = "workspace.member_invited"
)

// Subscription
const (
	SubscriptionCreated      = "subscription.created"
	SubscriptionCancelled    = "subscription.cancelled"
	SubscriptionExpired      = "subscription.expired"
	SubscriptionChanged      = "subscription.changed"
)

// Notification
const (
	NotificationRequested    = "notification.requested"
	NotificationSent         = "notification.sent"
	NotificationFailed       = "notification.failed"
)

// Report
const (
	ReportGenerationReq      = "report.generation_requested"
	ReportGenerated          = "report.generated"
)

// Webhook
const (
	WebhookDeliveryRequested = "webhook.delivery_requested"
	WebhookDelivered         = "webhook.delivered"
	WebhookFailed            = "webhook.failed"
)

// --- NATS Stream Names ---

const (
	StreamAccounting    = "ACCOUNTING"
	StreamCompany       = "COMPANY"
	StreamDocument      = "DOCUMENT"
	StreamNotification  = "NOTIFICATION"
	StreamReport        = "REPORT"
	StreamWebhook       = "WEBHOOK"
	StreamUser          = "USER"
	StreamSubscription  = "SUBSCRIPTION"
)

// StreamConfig maps stream names to their subject filters
var StreamConfig = map[string][]string{
	StreamAccounting:   {"accounting.>"},
	StreamCompany:      {"company.>", "counterparty.>"},
	StreamDocument:     {"document.>"},
	StreamNotification: {"notification.>"},
	StreamReport:       {"report.>"},
	StreamWebhook:      {"webhook.>"},
	StreamUser:         {"user.>", "workspace.>"},
	StreamSubscription: {"subscription.>"},
}

// --- Asynq Task Types ---

const (
	TaskEmailSend          = "email:send"
	TaskEmailVerification  = "email:verification"
	TaskEmailPasswordReset = "email:password_reset"
	TaskEmailInvoice       = "email:invoice"

	TaskImageResize        = "image:resize"
	TaskImageOCR           = "image:ocr"

	TaskWebhookDeliver     = "webhook:deliver"
	TaskWebhookRetry       = "webhook:retry"

	TaskInvoiceGenerate    = "invoice:generate"
	TaskInvoiceReminder    = "invoice:reminder"

	TaskReportGenerate     = "report:generate"

	TaskCleanupSessions    = "cleanup:sessions"
	TaskCleanupTokens      = "cleanup:tokens"
	TaskCleanupOutbox      = "cleanup:outbox"
	TaskSubscriptionCheck  = "subscription:check_expiry"
)
