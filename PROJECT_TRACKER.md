# Azzet Backend - Project Tracker

> Implementation tracker for the Azzet Backend. Ordered by dependency chain.
> Each phase builds on the previous one. Do NOT skip phases.

---

## Status Legend

```
[x] = Completed
[ ] = Not started
[~] = In progress
[-] = Skipped/Deferred
```

---

## Phase 0 — Foundation & Infrastructure

> Project setup, tooling, Docker, and base utilities.

- [x] Go project initialization (go.mod, go.sum)
- [x] Docker Compose (PostgreSQL, Redis, NATS JetStream)
- [x] Custom migration tool (cmd/migrate)
- [x] SQLC configuration (sqlc.yaml)
- [x] Chi router setup with middleware
- [x] Configuration management (godotenv)
- [x] Database connection pool (pgx/v5)
- [x] Redis client
- [x] Structured logging (slog)
- [x] HTTP response helpers
- [x] Custom validator
- [x] Password hashing (bcrypt)
- [x] Health check endpoint (db + redis ping)
- [x] Swagger/OpenAPI setup (swaggo)
- [x] Makefile with all commands
- [x] .env.example
- [x] .gitignore
- [x] Request body size limit middleware (1MB)

---

## Phase 1 — Authentication

> User registration, login, session management, and platform admin.

### 1A. User Auth

- [x] JWT service (access + refresh tokens, separate secrets)
- [x] OTP service (crypto/rand, SHA256 hashed storage)
- [x] Zenziva WhatsApp OTP client
- [x] Email OTP sender (SMTP)
- [x] Migration: users, sessions, otp_codes, audit_logs
- [x] Migration: users.name column
- [x] SQLC queries (auth.sql)
- [x] Auth service (register, login email, login OTP, refresh, logout)
- [x] Auth middleware (JWT validation, Redis blacklist)
- [x] Auth handler with input validation
- [x] Refresh token in HttpOnly cookie (Secure, SameSite=Strict)
- [x] Token blacklist via Redis (auto-expire)
- [x] OTP race condition fix (SELECT FOR UPDATE SKIP LOCKED)
- [x] Generic error messages (prevent enumeration)
- [x] Device/IP tracking on sessions
- [x] Unit tests (JWT, OTP, password, DTOs)
- [x] API tests (middleware, validation, cookies)

### 1B. Platform Admin

- [x] Migration: platform_admins table
- [x] SQLC queries (admin.sql)
- [x] Admin service (login, MFA, invite, CRUD)
- [x] MFA (TOTP via pquerna/otp - Google Authenticator)
- [x] Admin middleware (separate JWT, role-based)
- [x] Admin handler with Swagger docs
- [x] Separate CORS for admin subdomain (admin.azzet.com)
- [x] Admin seed CLI tool (cmd/admin-seed)
- [x] Roles: SUPER_ADMIN, SUPPORT, REVIEWER, ENGINEER
- [x] Shorter token expiry (10 min access, 10 hour session)
- [x] Min 12 char password for admins

---

## Phase 2 — Plan System

> Subscription plans with feature-based permissions and quotas.

- [x] Migration: plans, plan_features tables
- [x] SQLC queries (plan.sql)
- [x] Plan service (CRUD plans, set/remove features)
- [x] Plan handler (public list + admin management)
- [x] Public endpoints: GET /api/v1/plans, GET /api/v1/plans/{slug}
- [x] Admin endpoints: CRUD /admin/plans (SUPER_ADMIN + ENGINEER)
- [x] Feature types: boolean, quota (-1 = unlimited), tier
- [x] Swagger docs for all plan endpoints

---

## Phase 3 — Entity & Relations (Tenant System)

> Core identity system. Entity = anyone involved in transactions.
> Entity Relations = multi-tenant isolation + RBAC.
>
> **Why before Subscription?** Entity IS the tenant. Without entities,
> there's nothing to subscribe. A "tenant" in Azzet is an entity (BADAN_USAHA
> or ORANG_PRIBADI) that owns a workspace via entity_relations.

### 3A. Entity Core

- [x] Migration: entities, entity_meta
- [x] SQLC queries for entities
- [x] Entity service (create, get, update, search)
- [x] Entity types: ORANG_PRIBADI, BADAN_USAHA
- [x] Shadow entity creation (id_user = NULL)
- [x] Link entity to user (id_user FK)
- [x] Entity handler + Swagger docs
- [x] Auto-create personal entity on user registration

### 3B. Entity Relations (Tenant Isolation)

- [x] Migration: entity_relations, master_roles (seeded: PEMILIK, AKUNTAN, KASIR, VIEWER)
- [x] SQLC queries for relations
- [x] Relation service (create, list, update status)
- [x] Relation types: PEMILIK, KARYAWAN, PELANGGAN, VENDOR
- [x] nama_alias_kustom (custom naming per relation)
- [x] Tenant context middleware (resolve workspace from X-Workspace-ID header)
- [x] RBAC: master_roles with JSONB permissions (resource:action pattern)
- [x] Privacy boundary enforcement (query scoping by relation)
- [x] Handler + Swagger docs

### 3C. Workspace Management

- [x] Create workspace (entity becomes "tenant")
- [x] Invite members to workspace
- [x] Assign roles to members
- [x] Switch workspace context (X-Workspace-ID header)
- [x] List my workspaces
- [x] Add counterparties (creates shadow entity if needed)
- [x] List counterparties

> **Note:** Entity + workspace creation uses hybrid approach:
> - Synchronous creation during registration (instant, no polling needed for frontend)
> - Event emitted for audit trail, notifications, and future async consumers

---

## Phase 4 — Subscription

> Links a tenant (entity workspace) to a plan.
> Controls what features the tenant can use.
>
> **Why after Entity?** Because subscription belongs to a tenant,
> and tenant = entity with workspace. Can't subscribe without an entity.
>
> **Why before Billing?** Because free plans and trials don't need payment.
> Users can start using the system immediately after subscribing to free/trial.

- [x] Migration: tenant_subscriptions, tenant_usage tables
- [x] SQLC queries for subscriptions
- [x] Subscription service:
  - [x] Subscribe to free plan (instant activation)
  - [x] Start trial (active for plan.trial_days)
  - [x] Get active subscription
  - [x] Check subscription status (active/trial/expired/cancelled)
  - [x] Upgrade/downgrade plan
  - [x] Cancel subscription
- [x] Feature gate: HasFeature() + CheckQuota()
- [x] Quota tracking (tenant_usage table, monthly reset)
- [x] Handler + Swagger docs
- [x] Admin: list subscriptions

---

## Phase 5 — Billing & Payment

> Payment processing for paid plans. Integrates with Xendit.
>
> **Why after Subscription?** Because billing is triggered BY subscription.
> Free/trial users never touch billing. Only when:
> - User subscribes to paid plan
> - Trial expires and user wants to continue
>
> **Why before Business Logic?** Because paid features must be gated.
> If a user is on a paid plan but hasn't paid, they shouldn't access paid features.

- [x] Migration: invoices, payments tables
- [x] Xendit integration (payment gateway client)
- [x] Invoice creation
- [x] Payment initiation (returns Xendit checkout URL)
- [x] Payment webhook handler (Xendit callback)
- [x] Payment status tracking (pending, paid, failed, expired, refunded)
- [x] Auto-activate subscription on payment success
- [x] Auto-expire on payment failure
- [x] Webhook signature verification (x-callback-token)
- [x] Handler + Swagger docs
- [x] Admin: list all invoices

---

## Phase 6 — Event System

> Transactional outbox, NATS JetStream, idempotent consumers.
>
> **Why here?** Business logic (Phase 7+) needs async processing.
> Ledger posting, OCR, notifications all run via events.
> Building event system before business logic ensures all domain
> events are properly captured from day one.

- [x] Migration: outbox_events, inbox_consumed_events + LISTEN/NOTIFY trigger
- [x] Event envelope definition (Go struct with functional options)
- [x] Outbox publisher with PostgreSQL LISTEN/NOTIFY (real-time) + polling fallback
- [x] NATS JetStream client (connect, publish, subscribe, ensure streams)
- [x] Consumer base with idempotency (inbox_consumed_events check)
- [x] Dead letter queue handling (5 retries → DLQ)
- [x] Exponential backoff (1s, 5s, 30s, 2min, 10min)
- [x] cmd/publisher/main.go (outbox → NATS publisher process)
- [x] cmd/consumer/main.go (NATS event consumers: ledger, claim, document, notification, report)
- [x] cmd/worker/main.go updated (Asynq task handlers + scheduled cron jobs)
- [x] EmitEvent helper for services (write to outbox in same TX)
- [x] NATS streams: ACCOUNTING, COMPANY, DOCUMENT, NOTIFICATION, REPORT, WEBHOOK, USER, SUBSCRIPTION
- [x] 14 day event retention
- [x] Refactored auth registration to event-driven (Option C: user.registered → entity creation)

---

## Phase 7 — Accounting Core

> The heart of the business logic. Double-entry bookkeeping.
>
> **Stakeholder Vision:** "Hulu ke Hilir" - from the simplest cash recording
> to full double-entry accounting. User doesn't need to understand debit/credit.
>
> **Key Principle:** Every transaction affects at least 2 entities and 4 accounts
> (2 per entity). The system handles this complexity behind the scenes.

### 7A. Chart of Accounts

- [ ] Migration: chart_of_accounts (per entity/tenant)
- [ ] Default COA template (seeded per new workspace)
- [ ] Account types: Asset, Liability, Equity, Revenue, Expense
- [ ] Account hierarchy (parent-child)
- [ ] Account codes (numbering system)
- [ ] SQLC queries + service + handler

### 7B. Items & Products

- [ ] Migration: items table (per tenant, privacy boundary)
- [ ] Item types: BARANG_FISIK, JASA, PROYEK, AHSP_RAKITAN
- [ ] Item CRUD with tenant isolation
- [ ] Price management (harga_satuan)
- [ ] Soft delete (status_aktif)
- [ ] SQLC queries + service + handler

### 7C. Transactions

- [ ] Migration: transactions, journal_entries, ledger_entries
- [ ] Transaction types: cash_in, cash_out, journal
- [ ] Dual-mode input:
  - [ ] Simple mode: "I received 100k" -> auto journal
  - [ ] Advanced mode: manual journal entry
- [ ] Double-entry enforcement (debit = credit)
- [ ] Transaction status: draft, posted, void
- [ ] Counterparty linking (id_entitas_lawan)
- [ ] Shadow entity auto-creation for unknown counterparties
- [ ] SQLC queries + service + handler

### 7D. Ledger Worker (Async)

- [ ] NATS consumer: ledger.posting_requested
- [ ] Validate posting rules
- [ ] Generate ledger entries from journal
- [ ] Update account balances
- [ ] Emit ledger.posted event
- [ ] Accounting equation validation (A = L + E)

### 7E. Reporting (Basic)

- [ ] Balance Sheet (Neraca)
- [ ] Income Statement (Laba Rugi)
- [ ] Cash Flow
- [ ] Trial Balance
- [ ] Entity-wise transaction history
- [ ] Async report generation (Asynq)

---

## Phase 8 — Company Identity & Claim Workflow

> Verification system for entities. Proves ownership of a company.
>
> **Why after Accounting?** Because the claim workflow references
> transactions. "PT Maju Jaya" becomes a shadow entity through
> transactions first, then gets claimed and verified later.

### 8A. Company Identity

- [ ] Verification status on entities (unverified, pending, verified, rejected)
- [ ] Legal identifiers (NPWP, NIB, SIUP)
- [ ] Normalized company names (for matching)
- [ ] Company aliases
- [ ] Duplicate detection (fuzzy matching)

### 8B. Company Claim Workflow

- [ ] Migration: company_claims, claim_documents
- [ ] Claim request (user claims a shadow entity)
- [ ] Document submission (upload legal docs to R2)
- [ ] Admin review queue (REVIEWER role)
- [ ] Approve/reject/dispute flow
- [ ] Link shadow entity to verified entity on approval
- [ ] Audit trail for all claim actions
- [ ] NATS event: company.claim_requested, company.claim_approved
- [ ] Claim verification worker (NATS consumer)

### 8C. Counterparty Management

- [ ] Privacy-safe counterparty lookup
- [ ] Counterparty matching (suggest existing entities)
- [ ] Alias mapping (nama_alias_kustom)
- [ ] Cross-tenant reference (without data leakage)

---

## Phase 9 — Document & OCR

> Document upload, storage, and AI-powered extraction.
>
> **Stakeholder Vision:** User scans receipt -> system extracts vendor,
> amount, date, NPWP -> auto-creates transaction.

- [ ] Migration: documents table
- [ ] Document upload flow (presigned URL -> R2)
- [ ] Document metadata (type, status, entity link)
- [ ] Document worker (NATS consumer)
- [ ] OpenAI OCR integration (extract structured data)
- [ ] Entity extraction from documents (vendor name, NPWP)
- [ ] Auto-create transaction from extracted data
- [ ] Document verification status
- [ ] Access control (per tenant)
- [ ] Handler + Swagger docs

---

## Phase 10 — Tax

> Tax calculation, profiles, and reporting.
> Indonesian tax system: PPN (VAT), PPh (Income Tax).

- [ ] Migration: tax_profiles, tax_calculations
- [ ] Tax profile per entity (NPWP, tax status)
- [ ] PPN calculation hooks (on transactions)
- [ ] PPh calculation hooks (on income)
- [ ] Tax document references
- [ ] Tax report generation (async)
- [ ] Future: e-Faktur / e-Bupot integration ready
- [ ] Handler + Swagger docs

---

## Phase 11 — Notification & Webhooks

> Multi-channel notifications and outbound webhooks.

### 11A. Notifications

- [ ] Migration: notification_jobs table
- [ ] Notification worker (NATS consumer)
- [ ] Email notifications (Asynq -> SMTP)
- [ ] WhatsApp notifications (Asynq -> Zenziva)
- [ ] In-app notifications (stored in DB, polled by frontend)
- [ ] Notification preferences per user
- [ ] Retry with DLQ

### 11B. Webhooks

- [ ] Migration: webhook_endpoints, webhook_deliveries
- [ ] Webhook registration (per tenant)
- [ ] Webhook delivery worker (NATS consumer)
- [ ] HMAC signature on payloads
- [ ] Retry with exponential backoff (Asynq)
- [ ] Delivery attempt tracking
- [ ] DLQ for failed deliveries
- [ ] Handler + Swagger docs

---

## Phase 12 — Admin Review System

> Admin tools for managing users, claims, and system health.

- [ ] User management (SUPPORT+): list, suspend, activate
- [ ] Company claim review (REVIEWER+): approve, reject
- [ ] Document verification (REVIEWER+)
- [ ] System health dashboard (ENGINEER+)
- [ ] Audit log viewer (all admins)
- [ ] Metrics endpoint (Prometheus-compatible)

---

## Phase 13 — Hardening & Production Readiness

> Security, performance, observability, and deployment.

### 13A. Security

- [ ] Rate limiting middleware (Redis-based, per IP + per user)
- [ ] API key authentication (for service-to-service)
- [ ] Input sanitization audit
- [ ] SQL injection prevention audit
- [ ] Dependency vulnerability scan
- [ ] Penetration testing

### 13B. Performance

- [ ] Database query optimization (EXPLAIN ANALYZE)
- [ ] Index review and optimization
- [ ] Connection pool tuning
- [ ] Redis caching strategy (hot data)
- [ ] Large table partitioning plan (audit_logs, ledger_entries)
- [ ] Load testing (k6 or similar)

### 13C. Observability

- [ ] Structured logging audit (all services)
- [ ] Request correlation IDs (end-to-end)
- [ ] Prometheus metrics export
- [ ] Grafana dashboards
- [ ] Alerting rules
- [ ] Error tracking (Sentry)
- [ ] NATS JetStream monitoring
- [ ] Asynq queue monitoring

### 13D. Deployment

- [ ] Production Dockerfile (multi-stage build)
- [ ] CI/CD pipeline
- [ ] Database backup strategy (WAL archiving)
- [ ] Disaster recovery plan
- [ ] Blue-green deployment setup
- [ ] Secret management (vault or similar)
- [ ] TLS/HTTPS configuration

---

## Dependency Graph (Visual)

```
Phase 0: Foundation
    |
Phase 1: Auth (User + Admin)
    |
Phase 2: Plan System
    |
Phase 3: Entity & Relations  <-- "Tenant" = Entity with workspace
    |
Phase 4: Subscription  <-- Links Tenant to Plan
    |
Phase 5: Billing  <-- Payment for paid plans
    |
Phase 6: Event System  <-- Foundation for async business logic
    |
Phase 7: Accounting Core  <-- The main business logic
    |
Phase 8: Company Identity  <-- Verification layer on top of entities
    |
Phase 9: Document & OCR  <-- Creates transactions from documents
    |
Phase 10: Tax  <-- Hooks into transactions
    |
Phase 11: Notifications & Webhooks  <-- Triggered by all domain events
    |
Phase 12: Admin Review  <-- Manages claims, users, system
    |
Phase 13: Hardening  <-- Production readiness
```

---

## Common Questions

### "Why Entity before Subscription?"

Because in Azzet, **tenant = entity**. There's no separate "tenant" table.
A tenant is an entity (ORANG_PRIBADI or BADAN_USAHA) that has a workspace
via `entity_relations`. You can't subscribe without first having an entity
that acts as the workspace owner.

### "Why Subscription before Billing?"

Because **free plans and trials don't need payment**. A user can:
1. Register -> create entity -> create workspace
2. Subscribe to free plan -> immediately active
3. Start using features (gated by plan)

Billing only kicks in when:
- User subscribes to a **paid** plan
- User's **trial expires** and they want to continue

This means the system is usable from day one without Xendit integration.

### "Why Billing before Business Logic?"

Because paid features must be **gated**. If a user is on a paid plan but
hasn't paid (invoice overdue), they shouldn't access paid features.
The billing system provides the "is this subscription actually paid?" check
that the feature gate middleware needs.

However, **free plan users** can use business logic without billing.
The gate is: `has active subscription?` not `has paid?`.

### "Why Event System before Accounting?"

Because accounting operations are **async by design**:
- Ledger posting -> event-driven (NATS consumer)
- Report generation -> background task (Asynq)
- OCR extraction -> background task (Asynq)
- Notification on transaction -> event-driven

Without the event system, all business logic would be synchronous,
which violates the core principle: "No heavy workload inside request-response handlers."

### "Why Company Identity after Accounting?"

Because shadow entities are **created through transactions**. The flow is:
1. User records transaction to "PT Maju Jaya" (unknown entity)
2. System creates shadow entity automatically
3. Later, PT Maju Jaya registers and claims the shadow entity
4. Admin verifies documents and approves

You need transactions (Phase 7) to generate shadow entities that can
later be claimed (Phase 8).

### "Can I work on multiple phases in parallel?"

Some phases can overlap:
- Phase 6 (Event System) can start alongside Phase 4-5
- Phase 11 (Notifications) can start alongside Phase 8-9
- Phase 13 (Hardening) should be ongoing throughout

But the core dependency chain (Entity -> Subscription -> Billing -> Accounting)
must be sequential.

---

## Current Progress

```
Phase 0:  ████████████████████ 100%
Phase 1:  ████████████████████ 100%
Phase 2:  ████████████████████ 100%
Phase 3:  ████████████████████ 100%
Phase 4:  ████████████████████ 100%
Phase 5:  ████████████████████ 100%
Phase 6:  ████████████████████ 100%
Phase 7:  ░░░░░░░░░░░░░░░░░░░░   0% <-- NEXT
Phase 8:  ░░░░░░░░░░░░░░░░░░░░   0%
Phase 9:  ░░░░░░░░░░░░░░░░░░░░   0%
Phase 10: ░░░░░░░░░░░░░░░░░░░░   0%
Phase 11: ░░░░░░░░░░░░░░░░░░░░   0%
Phase 12: ░░░░░░░░░░░░░░░░░░░░   0%
Phase 13: ░░░░░░░░░░░░░░░░░░░░   0%
```

**Next up:** Phase 7 - Accounting Core

---

## Schema Version History

| Version | Migration | Description |
|---------|-----------|-------------|
| 1 | 001_auth.sql | users, sessions, otp_codes, audit_logs |
| 2 | 002_add_user_name.sql | users.name column |
| 3 | 003_platform_admins.sql | platform_admins table |
| 4 | 004_plans.sql | plans, plan_features tables |
| 5 | 005_entities.sql | entities, entity_meta tables |
| 6 | 006_entity_relations.sql | entity_relations, master_roles (seeded) |
| 7 | 007_subscriptions.sql | tenant_subscriptions, tenant_usage |
| 8 | 008_billing.sql | invoices, payments |
| 9 | 009_events.sql | outbox_events, inbox_consumed_events, LISTEN/NOTIFY trigger |

---

## Tech Stack Reference

| Component | Technology |
|-----------|-----------|
| Language | Go 1.26.2 |
| HTTP Router | Chi v5 |
| Database | PostgreSQL 16 + pgx/v5 |
| SQL Generator | SQLC |
| Cache/Queue | Redis 7 |
| Event Streaming | NATS JetStream |
| Background Tasks | Asynq |
| Object Storage | Cloudflare R2 |
| WhatsApp OTP | Zenziva |
| Payment Gateway | Xendit |
| AI/OCR | OpenAI |
| Auth | JWT (HS256) + bcrypt |
| Admin MFA | TOTP (Google Authenticator) |
| Docs | Swagger/OpenAPI (swaggo) |
| Logging | slog (structured) |

---

**Last Updated:** 2026-05-20
