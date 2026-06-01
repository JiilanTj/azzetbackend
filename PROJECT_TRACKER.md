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

- [x] Migration: entity_relations ~~master_roles (seeded: PEMILIK, AKUNTAN, KASIR, VIEWER)~~
- [x] SQLC queries for relations
- [x] Relation service (create, list, update status)
- [x] Relation types: PEMILIK, KARYAWAN, PELANGGAN, VENDOR
- [x] nama_alias_kustom (custom naming per relation)
- [x] Tenant context middleware (resolve workspace from X-Workspace-ID header)
- [x] ~~RBAC: master_roles with JSONB permissions (resource:action pattern)~~ → Replaced by ABAC (Phase 3D)
- [x] Privacy boundary enforcement (query scoping by relation)
- [x] Handler + Swagger docs

### 3C. Workspace Management

- [x] Create workspace (entity becomes "tenant")
- [x] ~~Invite members to workspace (instant)~~ → Replaced by email invite flow (Phase 3E)
- [x] ~~Assign roles to members (via master_roles)~~ → Replaced by ABAC role assignments (Phase 3D)
- [x] Switch workspace context (X-Workspace-ID header)
- [x] List my workspaces (includes subscription_status + plan_name)
- [x] Add counterparties (creates shadow entity if needed)
- [x] List counterparties
- [x] Auto-assign free plan to personal workspace on registration
- [x] Bootstrap "Owner" system role on workspace creation

> **Note:** Entity + workspace creation uses hybrid approach:
> - Synchronous creation during registration (instant, no polling needed for frontend)
> - Event emitted for audit trail, notifications, and future async consumers

### 3D. ABAC Permission System (NEW)

> Replaced master_roles with per-workspace custom roles.
> Owner (PEMILIK) always has wildcard `["*"]` permissions.

- [x] Migration 011: workspace_roles, workspace_role_assignments tables
- [x] Migration 011: Drop master_roles, drop role_id from entity_relations
- [x] Permission keys defined (transaction:*, report:*, member:invite, role:*, billing:*, etc.)
- [x] HasPermission() utility with wildcard + resource wildcard support
- [x] RequirePermission middleware wired to routes (member:manage, member:invite, role:*, etc.)
- [x] CRUD endpoints: POST/GET/PATCH/DELETE /workspaces/roles
- [x] Assign/unassign role: POST /workspaces/roles/assign, /workspaces/roles/unassign
- [x] System "Owner" role auto-created on workspace creation (is_system=true, permissions=["*"])
- [x] Swagger docs updated

### 3E. Workspace Invite Flow (NEW)

> Email-based invitations with 24h expiry. Replaces old instant InviteMember.

- [x] Migration 012: workspace_invites table
- [x] Invite service: create, accept, list pending, revoke
- [x] Validation: email must be registered, no duplicate pending invites, no invite to existing members
- [x] Secure token generation (32 bytes / 64 hex chars)
- [x] Email template with styled HTML (invite link to frontend)
- [x] Accept invite: validate token, check expiry (24h), verify email match, create relation + assign role
- [x] Endpoints: POST/GET/DELETE /workspaces/invites, POST /workspaces/invites/accept
- [x] RequirePermission("member:invite") enforced on create/revoke
- [x] Config: FRONTEND_URL for invite link generation
- [x] Swagger docs updated

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
  - [x] Subscribe to paid plan (status: pending_payment → invoice → Xendit payment → webhook activates)
  - [x] Get active subscription
  - [x] Check subscription status (active/trial/expired/cancelled/pending_payment)
  - [x] Upgrade/downgrade plan
  - [x] Cancel subscription
- [x] Feature gate: HasFeature() + CheckQuota()
- [x] Quota tracking (tenant_usage table, monthly reset)
- [x] Handler + Swagger docs
- [x] Admin: list subscriptions
- [x] Migration 013: Add pending_payment to check_sub_status constraint

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
- [x] Subscription → Billing integration (paid plan auto-creates invoice + payment)
- [x] Payment URL returned in subscription response for frontend redirect

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

### Architecture: "Simple Input to Proper Accounting"

```
┌─────────────────────────────────────────────────────────────────┐
│                      USER INPUT MODES                             │
│                                                                   │
│  Mode A: Simple Cash     Mode B: Sales/Purchase    Mode C: OCR   │
│  "Terima 100rb"          "Jual 5 nasi @15rb"      [scan struk]  │
│  dari Pak Budi           ke Pak Budi               (Phase 9)     │
└──────────────┬────────────────────┬────────────────────┬─────────┘
               │                    │                    │
               v                    v                    v
┌─────────────────────────────────────────────────────────────────┐
│         AI CATEGORIZATION LAYER (Sandboxed, Strict)              │
│                                                                   │
│  • Input: user text/description                                  │
│  • Output: ONLY valid category enum from whitelist               │
│  • Double-check: AI output validated against DB enum             │
│  • Security: no prompt injection, no data leakage                │
│  • Fallback: if confidence < 0.7 → "lain_lain" category         │
│  • Token-efficient: structured prompt, minimal context           │
└──────────────────────────────┬───────────────────────────────────┘
                               │
                               v
┌─────────────────────────────────────────────────────────────────┐
│              TRANSACTION SERVICE (Synchronous)                    │
│                                                                   │
│  1. Validate input + AI category against strict enum whitelist   │
│  2. Resolve/create counterparty (shadow entity if unknown)       │
│  3. Rule Engine: category → COA account codes (deterministic)    │
│  4. Build journal entries (enforce sum(debit) = sum(credit))     │
│  5. Support multi-line items (1 transaksi = N line items)        │
│  6. Save transaction + journal_entries as DRAFT                  │
│  7. Emit event: accounting.transaction.created                   │
└──────────────────────────────┬───────────────────────────────────┘
                               │
                               v (async via NATS)
┌─────────────────────────────────────────────────────────────────┐
│              LEDGER WORKER (Async Consumer)                       │
│                                                                   │
│  1. Receive accounting.transaction.created                       │
│  2. Validate: sum(debit) == sum(credit) per transaction          │
│  3. Generate ledger_entries from journal_entries                  │
│  4. Calculate running_balance per account                        │
│  5. Upsert account_balances (period summary: YYYY-MM)            │
│  6. Validate accounting equation: A = L + E                      │
│  7. UPDATE transaction status → POSTED                           │
│  8. Emit: accounting.ledger.posted                               │
└─────────────────────────────────────────────────────────────────┘
```

### Architecture Decisions

| Decision | Rationale |
|----------|-----------|
| **Void = Jurnal Pembalik (Reverse Entry)** | Standar akuntansi Indonesia. Jurnal asli TIDAK dihapus. Sistem buat jurnal baru yang membalik debit↔credit. Audit trail tetap utuh. |
| **Multi-line items per transaction** | Standar SAK EMKM & industri (Accurate, Jurnal.id). Satu faktur/struk bisa berisi banyak barang/jasa. |
| **IDR only (NUMERIC 15,2)** | Fokus pasar Indonesia. Max ~9.99 triliun. Cukup untuk UMKM-enterprise. |
| **AI categorization with strict whitelist** | AI hanya output category enum yang valid. Double-check di backend. Tidak bisa inject/hallucinate akun baru. |
| **COA auto-seed via workspace.created event** | Consumer listen event, seed template COA standar SAK EMKM otomatis. |
| **account_balances table terpisah** | Avoid full ledger scan untuk report. Ledger worker upsert per posting. |
| **Status DRAFT → POSTED async** | User dapat response cepat. Ledger worker posting di background via NATS. |
| **Rule engine hardcoded** | Aturan akuntansi jarang berubah. Performa tinggi. Bisa extend ke DB-based nanti. |
| **Shadow entity via existing counterparty flow** | Reuse Phase 3 logic. Tidak perlu table baru. |

### AI Categorization Security Design

```
┌─────────────────────────────────────────────────────────────────┐
│                    AI CATEGORIZATION SERVICE                      │
│                                                                   │
│  STRICT RULES:                                                   │
│  1. System prompt is HARDCODED, not user-configurable            │
│  2. User input is SANITIZED before sending to OpenAI             │
│  3. AI output MUST be one of the valid category enums            │
│  4. Backend VALIDATES AI output against whitelist (double-check) │
│  5. If AI returns invalid category → fallback to "lain_lain"    │
│  6. NO platform data (tenant info, financials) sent to AI        │
│  7. Only send: transaction description + amount + direction      │
│  8. Token budget: max ~200 input tokens, ~50 output tokens       │
│  9. Prompt injection defense: input wrapped in delimiters        │
│ 10. Response format: JSON with category + confidence only        │
│                                                                   │
│  VALID CATEGORIES (strict enum, no others accepted):             │
│                                                                   │
│  CASH_IN categories:                                             │
│    pendapatan_usaha, pendapatan_jasa, pendapatan_bunga,          │
│    piutang_dibayar, hutang_diterima, modal_disetor,              │
│    uang_muka_diterima, pendapatan_lain                           │
│                                                                   │
│  CASH_OUT categories:                                            │
│    beban_gaji, beban_sewa, beban_listrik, beban_telepon,         │
│    beban_transport, beban_makan, beban_perlengkapan,             │
│    beban_asuransi, beban_admin, beban_bank, beban_pemasaran,     │
│    beban_bunga, beban_pajak, pembelian_barang, bayar_hutang,     │
│    bayar_pajak, uang_muka_beli, prive, beban_lain               │
│                                                                   │
│  SALES categories:                                               │
│    penjualan_barang_tunai, penjualan_barang_kredit,              │
│    penjualan_jasa_tunai, penjualan_jasa_kredit,                  │
│    penjualan_dengan_ppn                                          │
│                                                                   │
│  PURCHASE categories:                                            │
│    pembelian_barang_tunai, pembelian_barang_kredit,              │
│    pembelian_jasa_tunai, pembelian_jasa_kredit,                  │
│    pembelian_dengan_ppn                                          │
│                                                                   │
│  SPECIAL categories:                                             │
│    diskon_penjualan, retur_penjualan, retur_pembelian            │
└─────────────────────────────────────────────────────────────────┘
```

### COA Template (SAK EMKM + SAK ETAP Compatible)

> Covers: Orang Pribadi, UMKM, dan Enterprise (PKP).
> User bisa tambah akun custom via API. Template ini adalah starting point.

```
1-0000  ASET (normal_balance: DEBIT)
├── 1-1000  Aset Lancar
│   ├── 1-1001  Kas
│   ├── 1-1002  Bank
│   ├── 1-1003  Piutang Usaha
│   ├── 1-1004  Persediaan Barang
│   ├── 1-1005  Piutang Lain-lain
│   ├── 1-1006  Perlengkapan
│   ├── 1-1007  Uang Muka Pembelian
│   ├── 1-1008  PPN Masukan (normal: DEBIT)
│   └── 1-1009  Biaya Dibayar di Muka
├── 1-2000  Aset Tetap
│   ├── 1-2001  Peralatan
│   ├── 1-2002  Kendaraan
│   ├── 1-2003  Bangunan
│   ├── 1-2004  Tanah
│   └── 1-2099  Akumulasi Penyusutan (normal: CREDIT)

2-0000  LIABILITAS (normal_balance: CREDIT)
├── 2-1000  Hutang Lancar
│   ├── 2-1001  Hutang Usaha
│   ├── 2-1002  Hutang Gaji
│   ├── 2-1003  Hutang Pajak (PPh 21/23/25/29)
│   ├── 2-1004  Pendapatan Diterima di Muka
│   ├── 2-1005  PPN Keluaran (normal: CREDIT)
│   ├── 2-1006  Uang Muka Penjualan
│   └── 2-1007  Hutang Lain-lain
├── 2-2000  Hutang Jangka Panjang
│   └── 2-2001  Hutang Bank

3-0000  EKUITAS (normal_balance: CREDIT)
├── 3-1001  Modal Pemilik
├── 3-1002  Prive (normal: DEBIT)
├── 3-1003  Laba Ditahan
└── 3-1004  Laba Periode Berjalan

4-0000  PENDAPATAN (normal_balance: CREDIT)
├── 4-1001  Pendapatan Usaha
├── 4-1002  Pendapatan Jasa
├── 4-1003  Diskon Penjualan (normal: DEBIT, contra-revenue)
├── 4-1004  Retur Penjualan (normal: DEBIT, contra-revenue)
├── 4-2001  Pendapatan Bunga
├── 4-2002  Pendapatan Lain-lain

5-0000  BEBAN (normal_balance: DEBIT)
├── 5-1000  Beban Operasional
│   ├── 5-1001  Beban Gaji & Tunjangan
│   ├── 5-1002  Beban Sewa
│   ├── 5-1003  Beban Listrik & Air
│   ├── 5-1004  Beban Telepon & Internet
│   ├── 5-1005  Beban Transportasi
│   ├── 5-1006  Beban Makan & Minum
│   ├── 5-1007  Beban Perlengkapan
│   ├── 5-1008  Beban Penyusutan
│   ├── 5-1009  Beban Asuransi (BPJS, asuransi aset)
│   ├── 5-1010  Beban Administrasi & Umum
│   ├── 5-1011  Beban Biaya Bank (transfer fee, admin)
│   └── 5-1012  Beban Pemasaran & Iklan
├── 5-2000  Beban Non-Operasional
│   ├── 5-2001  Beban Bunga Pinjaman
│   ├── 5-2002  Beban Pajak
│   ├── 5-2003  Beban Denda & Penalti
│   └── 5-2004  Kerugian Lain-lain
├── 5-3000  Harga Pokok
│   ├── 5-3001  Harga Pokok Penjualan (HPP)
│   └── 5-3002  Retur Pembelian (normal: CREDIT, contra-expense)
└── 5-9001  Beban Lain-lain
```

### Rule Engine: Category → Account Mapping (Deterministic)

```
CASH_IN rules (Debit: Kas/Bank, Credit: varies):
  pendapatan_usaha   → D:1-1001  C:4-1001
  pendapatan_jasa    → D:1-1001  C:4-1002
  pendapatan_bunga   → D:1-1002  C:4-2001
  piutang_dibayar    → D:1-1001  C:1-1003
  hutang_diterima    → D:1-1001  C:2-1001
  modal_disetor      → D:1-1001  C:3-1001
  uang_muka_diterima → D:1-1001  C:2-1006
  pendapatan_lain    → D:1-1001  C:4-2002

CASH_OUT rules (Debit: varies, Credit: Kas):
  beban_gaji         → D:5-1001  C:1-1001
  beban_sewa         → D:5-1002  C:1-1001
  beban_listrik      → D:5-1003  C:1-1001
  beban_telepon      → D:5-1004  C:1-1001
  beban_transport    → D:5-1005  C:1-1001
  beban_makan        → D:5-1006  C:1-1001
  beban_perlengkapan → D:5-1007  C:1-1001
  beban_asuransi     → D:5-1009  C:1-1001
  beban_admin        → D:5-1010  C:1-1001
  beban_bank         → D:5-1011  C:1-1002
  beban_pemasaran    → D:5-1012  C:1-1001
  beban_bunga        → D:5-2001  C:1-1001
  beban_pajak        → D:5-2002  C:1-1001
  pembelian_barang   → D:1-1004  C:1-1001
  bayar_hutang       → D:2-1001  C:1-1001
  bayar_pajak        → D:2-1003  C:1-1001
  uang_muka_beli     → D:1-1007  C:1-1001
  prive              → D:3-1002  C:1-1001
  beban_lain         → D:5-9001  C:1-1001

SALES rules (multi-line items, payment_method determines debit):
  penjualan_barang_tunai    → D:1-1001  C:4-1001  (+ D:5-3001 C:1-1004 for HPP)
  penjualan_barang_kredit   → D:1-1003  C:4-1001  (+ D:5-3001 C:1-1004 for HPP)
  penjualan_jasa_tunai      → D:1-1001  C:4-1002
  penjualan_jasa_kredit     → D:1-1003  C:4-1002
  penjualan_dengan_ppn      → adds D:1-1001 C:2-1005 (PPN Keluaran 11%)

PURCHASE rules (multi-line items, payment_method determines credit):
  pembelian_barang_tunai    → D:1-1004  C:1-1001
  pembelian_barang_kredit   → D:1-1004  C:2-1001
  pembelian_jasa_tunai      → D:5-xxxx  C:1-1001 (account from item.account_id)
  pembelian_jasa_kredit     → D:5-xxxx  C:2-1001
  pembelian_dengan_ppn      → adds D:1-1008 C:1-1001 (PPN Masukan 11%)

SPECIAL rules:
  diskon_penjualan   → D:4-1003  C:1-1003/1-1001 (reduce receivable/cash)
  retur_penjualan    → D:4-1004  C:1-1003/1-1001 (+ D:1-1004 C:5-3001 restock)
  retur_pembelian    → D:2-1001/1-1001  C:5-3002 (reduce payable/get cash back)
```

### Void Transaction Flow (Jurnal Pembalik)

```
Original Transaction (POSTED):
  ID: tx-001
  Debit:  1-1001 Kas        100,000
  Credit: 4-1001 Pendapatan 100,000

Void Request:
  1. Create NEW transaction (type: REVERSAL, references: tx-001)
  2. Reverse all journal entries (swap debit↔credit):
     Debit:  4-1001 Pendapatan 100,000
     Credit: 1-1001 Kas        100,000
  3. Mark original tx-001 status → VOID
  4. New reversal transaction → DRAFT → POSTED (via ledger worker)
  5. Net effect on all accounts = 0
  6. Both transactions remain in audit trail forever
```

### Database Schema (Migration 014)

```sql
-- Tables: accounts, items, transaction_line_items,
--         transactions, journal_entries, ledger_entries, account_balances

-- accounts: Chart of Accounts per workspace (SAK EMKM template)
-- items: Products/services per workspace with multi-type support
-- transactions: Header with status lifecycle (DRAFT→POSTED→VOID)
-- transaction_line_items: Multi-item support per transaction
-- journal_entries: Double-entry lines (debit XOR credit per line)
-- ledger_entries: Posted entries with running_balance (async)
-- account_balances: Period summary per account (YYYY-MM, upserted by worker)
```

### API Endpoints

```
# Chart of Accounts (workspace-scoped)
GET    /api/v1/accounts                    — List COA [transaction:read]
POST   /api/v1/accounts                    — Create custom account [transaction:create]
GET    /api/v1/accounts/{id}               — Get account detail
PATCH  /api/v1/accounts/{id}               — Update account [transaction:create]

# Items (workspace-scoped)
GET    /api/v1/items                       — List items [transaction:read]
POST   /api/v1/items                       — Create item [transaction:create]
GET    /api/v1/items/{id}                  — Get item detail
PATCH  /api/v1/items/{id}                  — Update item [transaction:create]
DELETE /api/v1/items/{id}                  — Soft-delete item [transaction:create]

# Transactions (workspace-scoped)
POST   /api/v1/transactions                — Create transaction [transaction:create]
GET    /api/v1/transactions                — List transactions [transaction:read]
GET    /api/v1/transactions/{id}           — Get transaction + journal entries
PATCH  /api/v1/transactions/{id}           — Update DRAFT transaction [transaction:create]
POST   /api/v1/transactions/{id}/void      — Void (jurnal pembalik) [transaction:void]

# AI Categorization (workspace-scoped, internal helper)
POST   /api/v1/transactions/categorize     — AI suggest category [transaction:create]

# Reports (workspace-scoped)
GET    /api/v1/reports/trial-balance       — Trial Balance [report:read]
GET    /api/v1/reports/balance-sheet       — Neraca [report:read]
GET    /api/v1/reports/income-statement    — Laba Rugi [report:read]
GET    /api/v1/reports/cash-flow           — Arus Kas [report:read]
GET    /api/v1/reports/ledger/{account_id} — Buku Besar per akun [report:read]
```

### Permission Keys (New)

```
transaction:create   — Create/edit transactions
transaction:read     — View transactions & accounts
transaction:void     — Void posted transactions (jurnal pembalik)
report:read          — View financial reports
report:export        — Export reports (future)
```

### File Structure

```
migrations/014_accounting.sql           — All accounting tables
queries/accounting.sql                  — SQLC queries (accounts, items, transactions, journal, ledger)
internal/accounting/
├── service.go                          — Transaction service (create, list, get, void)
├── coa_service.go                      — Chart of Accounts CRUD
├── coa_template.go                     — SAK EMKM default template + seed logic
├── item_service.go                     — Item CRUD
├── rules.go                            — Rule engine (category → account mapping)
├── categorizer.go                      — AI categorization (OpenAI, sandboxed, strict)
├── ledger_worker.go                    — NATS consumer (posting + balance update)
├── report_service.go                   — Report generation (neraca, laba rugi, etc.)
├── dto.go                              — All request/response DTOs
├── constants.go                        — Category enums, account types, status constants
└── errors.go                           — Sentinel errors
internal/api/handler/accounting.handler.go  — HTTP handlers
internal/api/router.go                      — Route registration (updated)
internal/events/types.go                    — Event types (already defined)
cmd/consumer/main.go                        — Register ledger worker (updated)
```

### Implementation Order

| Step | Task | Depends On |
|------|------|------------|
| 1 | Migration `014_accounting.sql` | — |
| 2 | SQLC queries `queries/accounting.sql` | Step 1 |
| 3 | Run `make sqlc` | Step 2 |
| 4 | `internal/accounting/constants.go` (enums, categories) | — |
| 5 | `internal/accounting/errors.go` (sentinel errors) | — |
| 6 | `internal/accounting/coa_template.go` (SAK EMKM seed data) | Step 4 |
| 7 | `internal/accounting/coa_service.go` (CRUD + seed) | Step 3, 6 |
| 8 | `internal/accounting/rules.go` (category → account mapping) | Step 4 |
| 9 | `internal/accounting/categorizer.go` (AI + security) | Step 4 |
| 10 | `internal/accounting/item_service.go` (CRUD) | Step 3 |
| 11 | `internal/accounting/dto.go` (all DTOs) | Step 4 |
| 12 | `internal/accounting/service.go` (transaction create/list/get/void) | Step 7, 8, 9, 10 |
| 13 | `internal/accounting/ledger_worker.go` (NATS consumer) | Step 3, 12 |
| 14 | `internal/accounting/report_service.go` (reports) | Step 3, 13 |
| 15 | `internal/api/handler/accounting.handler.go` | Step 12, 14 |
| 16 | Update `internal/api/router.go` (register routes) | Step 15 |
| 17 | Update `cmd/consumer/main.go` (register ledger worker) | Step 13 |
| 18 | Hook COA seed into workspace.created consumer | Step 7 |
| 19 | Swagger docs | All |
| 20 | Tests (unit + integration) | All |

### 7A. Chart of Accounts

- [x] Migration: `accounts` table (per workspace, parent-child hierarchy)
- [x] COA template: SAK EMKM standard (seeded via workspace.created event)
- [x] Account types: ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE
- [x] Normal balance: DEBIT or CREDIT per account type
- [x] Account hierarchy (parent_id, level)
- [x] Account codes (format: "X-XXXX", unique per workspace)
- [x] System accounts (is_system=true, cannot be deleted)
- [x] SQLC queries + coa_service + handler
- [x] Hook: auto-seed on workspace.created event (NATS consumer)

### 7B. Items & Products

- [x] Migration: `items` table (per workspace, privacy boundary)
- [x] Item types: BARANG, JASA, PROYEK, AHSP_RAKITAN
- [x] Item CRUD with workspace isolation
- [x] Unit types: Pcs, Kg, Liter, Meter, M2, M3, Jam, Hari, Paket, Unit, Box, Lusin
- [x] Default account linking (account_id FK for auto-categorization)
- [x] Soft delete (is_active flag)
- [x] SQLC queries + item_service + handler

### 7C. Transactions

- [x] Migration: `transactions`, `transaction_line_items`, `journal_entries`
- [x] Transaction types: CASH_IN, CASH_OUT, SALES, PURCHASE, JOURNAL, REVERSAL
- [x] Input modes: SIMPLE, ADVANCED, OCR
- [x] Multi-line items: `transaction_line_items` table (qty, unit_price, item_id, etc.)
- [x] Dual-mode input:
  - [x] Simple mode: user picks direction + category → auto journal via rule engine
  - [x] Advanced mode: user manually specifies debit/credit accounts
- [x] AI categorization: strict whitelist, sandboxed prompt, double-check validation
- [x] Double-entry enforcement: sum(debit) MUST = sum(credit) per transaction
- [x] Transaction status lifecycle: DRAFT → POSTED (async) → VOID (jurnal pembalik)
- [x] Counterparty linking (counterparty_entity_id FK)
- [ ] Shadow entity auto-creation for unknown counterparties (reuse Phase 3)
- [x] Void = Jurnal Pembalik: create REVERSAL transaction, swap debit↔credit
- [x] SQLC queries + service + handler

### 7D. Ledger Worker (Async)

- [x] NATS consumer: accounting.transaction.created
- [x] Validate posting rules (sum debit = sum credit)
- [x] Generate ledger_entries from journal_entries
- [x] Calculate running_balance per account (ordered by posted_at)
- [x] Upsert account_balances (period: YYYY-MM)
- [x] Validate accounting equation: total ASSET = total LIABILITY + total EQUITY
- [x] Mark transaction status: POSTED + set posted_at
- [x] Emit: accounting.ledger.posted event
- [x] Error handling: if validation fails → mark FAILED + emit error event

### 7E. Reporting (Basic)

- [x] Trial Balance (Neraca Saldo): sum debit/credit per account for period
- [x] Balance Sheet (Neraca): Assets, Liabilities, Equity at point-in-time
- [x] Income Statement (Laba Rugi): Revenue - Expenses for period
- [x] Cash Flow (Arus Kas): Cash account movements for period
- [x] General Ledger (Buku Besar): all entries for one account with running balance
- [x] All reports use account_balances table (fast, pre-aggregated)
- [x] Synchronous for now (data from account_balances is already aggregated)
- [ ] Future: async generation via Asynq for large datasets + PDF export

### 7F. AI Categorization Service

- [x] Sandboxed OpenAI integration (reuse internal/ai client)
- [x] Hardcoded system prompt (not user-configurable)
- [x] Input sanitization: strip control chars, limit length, wrap in delimiters
- [x] Prompt injection defense: delimiter-wrapped user input, instruction hierarchy
- [x] Output validation: MUST match category enum whitelist exactly
- [x] Double-check: backend validates AI response against constants.go enums
- [x] Fallback: invalid/low-confidence → "beban_lain" or "pendapatan_lain"
- [x] Token efficiency: ~200 input tokens, ~50 output tokens per request
- [x] NO sensitive data sent: only description + amount + direction
- [x] Confidence score: 0.0-1.0, frontend can show "suggested" vs "confident"

---

## Phase 8 — Company Identity & Claim Workflow

> Verification system for entities. Proves ownership of a company.
>
> **Why after Accounting?** Because the claim workflow references
> transactions. "PT Maju Jaya" becomes a shadow entity through
> transactions first, then gets claimed and verified later.

### 8A. Company Identity

- [x] Verification status on entities (unverified, pending, verified, rejected)
- [x] Legal identifiers (NPWP, NIB, SIUP)
- [x] Normalized company names (for matching)
- [x] Company aliases
- [x] Duplicate detection (fuzzy matching)

### 8B. Company Claim Workflow

- [x] Migration: company_claims, claim_documents
- [x] Claim request (user claims a shadow entity)
- [x] Document submission (upload legal docs to R2)
- [x] Admin review queue (REVIEWER role)
- [x] Approve/reject/dispute flow
- [x] Link shadow entity to verified entity on approval
- [x] Audit trail for all claim actions
- [x] NATS event: company.claim_requested, company.claim_approved
- [x] Claim verification worker (NATS consumer)

### 8C. Counterparty Management

- [x] Privacy-safe counterparty lookup
- [x] Counterparty matching (suggest existing entities)
- [x] Alias mapping (nama_alias_kustom)
- [x] Cross-tenant reference (without data leakage)

---

## Phase 9 — Document & OCR ✅

> Document upload, storage, and AI-powered extraction.
>
> **Stakeholder Vision:** User scans receipt -> system extracts vendor,
> amount, date, NPWP -> auto-creates transaction.

### 9A. Document Storage & API (Backend)

- [x] Migration: `017_documents.sql` (documents table)
- [x] Document upload flow (presigned URL → R2 → confirm)
- [x] Document metadata (type, upload/extraction/verification status, transaction link)
- [x] Workspace-scoped handlers: `POST/GET /documents`, confirm, get by ID
- [x] Access control: workspace middleware + `transaction:create/read` permissions
- [x] Plan gate: `ocr_enabled` feature check on upload
- [x] R2 key helper: `storage.WorkspaceDocumentKey()`
- [x] Handler Swagger annotations (`document_handler.go`)

### 9B. OCR Worker & Transaction Automation

- [x] NATS consumer: `document-worker` on `document.uploaded`
- [x] OpenAI Vision OCR (`ChatVisionJSON`) — structured JSON extraction
- [x] Entity extraction from documents (vendor name, NPWP, amount, date, category)
- [x] Counterparty resolution (fuzzy match + shadow entity via `AddCounterparty`)
- [x] Auto-create transaction from extracted data (`input_mode=OCR`)
- [x] Link document → transaction_id on success
- [x] Extraction status tracking (PENDING → PROCESSING → COMPLETED/FAILED)
- [x] Emit `document.uploaded` event on confirm (outbox → NATS)

### 9C. User Panel Frontend (`azzetuserpanel`)

- [x] `/accounting/documents` — upload, list, status badges
- [x] Presigned upload flow (request → PUT R2 → confirm)
- [x] Auto-polling while extraction in progress
- [x] Link to auto-created transaction draft
- [x] Sidebar: Akuntansi → Dokumen & OCR
- [x] `document.service.ts`, `use-documents.ts`, types

### 9D. Deferred / Future

- [ ] PDF document support (images only: JPEG/PNG/WebP)
- [ ] `image_detail: high` on vision requests (better small-text OCR)
- [ ] Split `OPENAI_OCR_MODEL` vs categorizer model env
- [ ] Admin document verification UI (→ Phase 12)
- [ ] Document detail page (per-doc extraction breakdown)
- [x] Regenerate Swagger bundle (`make swag`) if not yet run post-handler

---

## Phase 10 — Tax ✅

> Tax calculation, profiles, and reporting.
> Indonesian tax system: PPN (VAT), PPh (Income Tax).

- [x] Migration: tax_profiles, tax_calculations
- [x] Tax profile per entity (NPWP, tax status)
- [x] PPN calculation hooks (on transactions)
- [x] PPh calculation hooks (on income)
- [x] Tax document references
- [x] Tax report generation (async)
- [x] Future: e-Faktur / e-Bupot integration ready
- [x] Handler + Swagger docs

### 10A. Backend (`azzetbe`)

- [x] Migration `018_tax.sql` — tax_profiles, tax_calculations, tax_document_refs, tax_report_jobs
- [x] `internal/tax/` — service, calculator, worker (ledger.posted hook + report worker)
- [x] API: `GET/PUT /tax/profile`, calculations, PPN/PPh summary, document refs, async reports
- [x] Consumer: tax calc on `accounting.ledger.posted`, report on `report.generation_requested`

### 10B. User Panel (`azzetuserpanel`)

- [x] `/accounting/tax` — profil PKP/NPWP, ringkasan PPN, daftar perhitungan, laporan async
- [x] Sidebar: Akuntansi → Pajak
- [x] `tax.service.ts`, `use-tax.ts`

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
- [x] Company claim review (REVIEWER+): approve, reject
- [ ] Document verification (REVIEWER+)
- [ ] System health dashboard (ENGINEER+)
- [ ] Audit log viewer (all admins)
- [ ] Metrics endpoint (Prometheus-compatible)

### 12A. Company Claim Review — Admin Panel (`azzetadminpanel`)

- [x] `/claims` — list with status filters + pending count badge
- [x] `/claims/$claimId` — detail, documents (presigned view), audit trail
- [x] Assign → Under Review, Approve, Reject (with reason)
- [x] Sidebar nav + `claimsService`, `use-claims` hooks

---

## Phase Pre-13 — Tweaks & Enhancements

> Small improvements and missing features discovered during integration testing.
> Should be done before hardening (Phase 13) to ensure all business logic is solid.

### Accounting Enhancements

- [ ] Custom category rules per workspace (user-defined category → account mapping for SIMPLE mode)
  - Migration: `workspace_category_rules` table
  - Rule engine: check workspace custom rules first → fallback to hardcoded rules
  - API: CRUD endpoints for custom rules
  - Allows user's custom accounts (e.g., "5-1013 Beban Parkir") to be used in SIMPLE mode
- [ ] Dynamic units per workspace (like COA, seeded with defaults, user can add custom)
  - Migration: `workspace_units` table (workspace_id, name, symbol, is_system)
  - Seed default units on workspace.created (Pcs, Kg, Liter, Meter, M2, M3, Jam, Hari, Paket, Unit, Box, Lusin, Set, Rim)
  - Validation: check against workspace's units table instead of hardcoded list
  - Case-insensitive matching
  - API: `GET /api/v1/units` (list), `POST /api/v1/units` (create custom)
  - Items reference unit from this table
- [ ] Shadow entity auto-creation for unknown counterparties (Phase 7C leftover)
  - When `counterparty_name` is provided but `counterparty_entity_id` is empty
  - Auto-create shadow entity + relation via existing Phase 3 counterparty logic
- [ ] Async report generation via Asynq for large datasets + PDF export (Phase 7E leftover)
- [ ] Manual COA seed endpoint: `POST /api/v1/accounts/seed` for workspaces created before Phase 7 deploy
- [ ] Transaction pagination params: support `status`, `type`, `date_from`, `date_to` filters in list endpoint

### API & Integration Fixes

- [ ] `amount` field: support both JSON string and number in request body (or document that number is required)
- [ ] Validate `journal_entries` array: reject empty objects `[{}, {}]`, only accept when `input_mode=ADVANCED`
- [ ] Add `counterparty_name` search/filter to `GET /transactions`

### Frontend Alignment

- [ ] Document: `X-Workspace-ID` must use `entity_id` from workspace list, NOT relation `id`
- [ ] Document: all numeric fields (`amount`, `unit_price`, `debit`, `credit`, `quantity`) must be JSON numbers, not strings
- [ ] Document: report endpoints require period/date query params (400 without them)

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
Phase Pre-13: Tweaks & Enhancements  <-- Polish before hardening
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
Phase 7:  ████████████████████ 100%
Phase 8:  ████████████████████ 100%
Phase 9:  ████████████████████ 100%
Phase 10: ████████████████████ 100%
Phase 11: ░░░░░░░░░░░░░░░░░░░░   0%
Phase 12: ███░░░░░░░░░░░░░░░░░  17%
Pre-13:   ░░░░░░░░░░░░░░░░░░░░   0%
Phase 13: ░░░░░░░░░░░░░░░░░░░░   0%
```

**Next up:** Phase 11 — Notification & Webhooks

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
| 10 | 010_fix_otp_code_column.sql | OTP code column fix |
| 11 | 011_abac_permissions.sql | workspace_roles, workspace_role_assignments, drop master_roles + role_id |
| 12 | 012_workspace_invites.sql | workspace_invites table |
| 13 | 013_subscription_pending_payment.sql | Add pending_payment to check_sub_status constraint |
| 14 | 014_accounting.sql | accounts, items, transactions, transaction_line_items, journal_entries, ledger_entries, account_balances |
| 15 | 015_company_identity.sql | entity_verification, entity_legal_ids, entity_aliases, company_claims, claim_documents, claim_audit_log, counterparty_aliases |
| 16 | 016_backfill_nama_normalized.sql | Backfill nama_normalized for existing entities |
| 17 | 017_documents.sql | Workspace documents for OCR (upload, extraction, transaction link) |
| 18 | 018_tax.sql | Tax profiles, calculations, document refs, report jobs |

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

**Last Updated:** 2026-06-02
