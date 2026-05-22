# Azzet Backend

Enterprise-grade Go backend for a multi-tenant SaaS platform focused on **Accounting**, **Tax**, **Finance**, and **Company Identity Verification**.

This backend is designed for medium-to-heavy workload from launch, with enterprise clients, sensitive financial/legal data, strict tenant isolation, strong auditability, and event-driven processing.

---

## Table of Contents

- [Core Concept](#core-concept)
- [Tech Stack](#tech-stack)
- [Architecture Style](#architecture-style)
- [High-Level Flow](#high-level-flow)
- [Event-Driven Design](#event-driven-design)
- [Event Envelope](#event-envelope)
- [Transactional Outbox Pattern](#transactional-outbox-pattern)
- [Idempotent Consumers](#idempotent-consumers)
- [Dead Letter Queue](#dead-letter-queue)
- [Required Modules](#required-modules)
- [Project Structure](#project-structure)
- [Database Design](#database-design)
- [API Routes](#api-routes)
- [Synchronous vs Asynchronous Operations](#synchronous-vs-asynchronous-operations)
- [Workers](#workers)
- [Security Requirements](#security-requirements)
- [Tenant Isolation](#tenant-isolation)
- [Observability](#observability)
- [Testing Strategy](#testing-strategy)
- [Deployment](#deployment)
- [Environment Variables](#environment-variables)
- [Local Development](#local-development)
- [Technology Choices](#technology-choices)
- [Common Mistakes to Avoid](#common-mistakes-to-avoid)
- [Implementation Roadmap](#implementation-roadmap)
- [Production Readiness Checklist](#production-readiness-checklist)
- [Stakeholder Vision & Context](#stakeholder-vision--context)

---

## Stakeholder Vision & Context

> **Note:** This section contains the original stakeholder conversation transcript that provides crucial business context and vision. This represents the client's perspective on how the system should work from a user's point of view. While this is essential for understanding business requirements, it needs to be translated into proper technical specifications and architecture.

### Client Meeting Transcript - Core Business Flow Discussion

**Topic: End-to-End Transaction Flow (Hulu ke Hilir)**

**Key Points from Stakeholder:**

#### 1. User Journey - From Individual to Company

The stakeholder describes a natural progression:
- Individual starts with an idea
- Validates the idea
- Finds market
- Seeks capital
- Before opening a bank account, needs identity (KTP, NPWP)
- Starts with personal transactions
- Eventually forms a PT (company) when cash flow is sufficient

**System Requirement:** The platform must support users starting as individuals (ORANG_PRIBADI) and later transitioning to business entities (BADAN_USAHA) without losing transaction history.

#### 2. Flexible Transaction Recording Methods

The stakeholder emphasizes multiple entry methods for different user sophistication levels:

**Method A - Simple Cash Recording:**
- User just wants to record: "Today I received 100,000"
- System should automatically handle double-entry behind the scenes
- User doesn't need to understand accounting

**Method B - Sales Recording:**
- User records a sale transaction
- System automatically updates both cash and revenue accounts

**Method C - Advanced/Automatic:**
- User scans receipt with OCR
- System extracts: amount, vendor name, date, items
- Automatically creates proper journal entries

**System Requirement:** Multiple input interfaces (simple cash entry, sales form, OCR scan) that all feed into the same double-entry accounting engine.

#### 3. Entity Relationship & Multi-Tenant Isolation

**Critical Business Logic:**

When User A (individual) receives salary from PT A:
```
Transaction affects 4 accounts:
1. User A's Cash (Debit) - Balance Sheet
2. User A's Income (Credit) - Income Statement
3. PT A's Salary Expense (Debit) - Income Statement
4. PT A's Cash (Credit) - Balance Sheet
```

**Key Insight:** Even if PT A is not yet using the application, the system should:
- Create a "shadow entity" for PT A
- Record User A's side of the transaction
- When PT A eventually joins the platform, they can see all entities that have reported transactions with them
- PT A can then "claim" their entity and see historical references

**System Requirement:** 
- Shadow entity creation for unregistered counterparties
- Entity claiming workflow
- Strict privacy: User A cannot see PT A's internal transactions, only their own transactions with PT A

#### 4. Custom Naming & Aliasing

**Scenario:**
- User A transacts with "Toko Maju"
- User B transacts with "Pak Budi" (same physical entity)
- Each user sees their own custom name
- System maintains entity matching in the background

**System Requirement:** `nama_alias_kustom` in `entity_relations` table allows each tenant to name counterparties as they wish.

#### 5. Automatic Double-Entry Accounting

**Stakeholder's Vision:**
- User doesn't need to understand debit/credit
- User just records: "I bought lunch at KFC for 50,000"
- System automatically:
  - Debits: Food Expense (Income Statement)
  - Credits: Cash (Balance Sheet)
  - Creates entity reference for KFC
  - Extracts NPWP if available from receipt (OCR)

**System Requirement:**
- Intelligent transaction categorization
- Automatic chart of accounts mapping
- OCR integration for receipt scanning
- Entity extraction from documents

#### 6. OCR & Document Intelligence

**Stakeholder mentions:**
- "If user scans receipt, system should automatically extract vendor name, NPWP, amount"
- "OCR development was challenging but necessary"

**System Requirement:**
- Document upload and OCR processing
- Entity matching based on extracted data
- NPWP/NIK extraction and validation
- Automatic transaction creation from scanned documents

#### 7. Reporting & Visibility

**Implicit Requirements:**
- Balance Sheet (Neraca) - shows assets, liabilities, equity
- Income Statement (Laba Rugi) - shows revenue, expenses, net income
- Cash Flow tracking
- Entity-wise transaction history
- Tax reporting (linked to NPWP)

### Technical Translation of Stakeholder Vision

Based on the conversation, the system must provide:

1. **Dual-Mode Accounting:**
   - Simple mode: Cash in/out recording
   - Advanced mode: Full double-entry with journal entries
   - Both modes write to the same underlying ledger

2. **Entity Graph:**
   - Every transaction involves at least 2 entities
   - Entities can be: verified users, verified companies, or shadow entities
   - Shadow entities can be "claimed" later

3. **Smart Categorization:**
   - AI/rule-based expense categorization
   - Automatic account code assignment
   - User can override if needed

4. **Document-Driven Transactions:**
   - Upload receipt → OCR → Extract data → Create transaction
   - Link documents to transactions
   - Extract counterparty information

5. **Privacy-First Multi-Tenant:**
   - User A sees: their transactions + their view of counterparties
   - User A cannot see: PT A's internal transactions or PT A's transactions with others
   - PT A sees: their transactions + entities that reported transactions with them

6. **Progressive Disclosure:**
   - Start simple (cash tracking)
   - Grow to sales/purchase recording
   - Eventually full accounting with reports
   - All historical data remains consistent

---

## Core Concept

This platform supports financial operations such as:

- Cash in / cash out
- Journal entries
- Ledger posting
- Chart of accounts
- Tax calculation hooks
- Financial reports
- Legal document management
- Company identity verification
- Counterparty management
- Admin review workflow
- Notification and webhook delivery

### Unverified Counterparty Log & Company Claim Workflow

A key feature is the **unverified counterparty log** and **company claim workflow**.

**Example Flow:**

1. Company A creates a cash-out transaction to Company B.
2. Company B is not registered yet.
3. Company A creates an unverified counterparty reference using only limited data, such as the company name.
4. Later, Company C also records a transaction to the same or similar Company B.
5. The system may match it to an existing company candidate.
6. Company B later registers and submits official legal documents.
7. The system detects possible existing candidates.
8. Company B can request a claim.
9. Admin verifies the legal documents.
10. Only after approval can the candidate be linked to a verified company.

**Important principle:**

> Company name similarity is **not proof of ownership**.

No tenant should ever see another tenant's private transaction data.

---

## Tech Stack

### Backend Core

- **Go 1.26.2** — Programming language
- **Chi v5** (`github.com/go-chi/chi/v5`) — HTTP router/framework
- **pgx/v5** (`github.com/jackc/pgx/v5`) — PostgreSQL driver with connection pooling
- **SQLC** — Type-safe SQL code generator
- **PostgreSQL** — Source of truth database
- **Custom Migration Tool** — Built-in database migration runner
- **slog** (`log/slog`) — Structured logging (Go standard library)
- **Swaggo** (`github.com/swaggo/swag`) — OpenAPI/Swagger documentation generator

### Authentication & Security

- **JWT** (`github.com/golang-jwt/jwt/v5`) — Token-based authentication
- **bcrypt** (`golang.org/x/crypto`) — Password hashing
- **Custom Validator** — Manual validation functions for request validation
- **CORS** (`github.com/go-chi/cors`) — Cross-origin resource sharing

### Event-Driven Architecture

- **NATS JetStream** — Primary message broker for event streaming and pub/sub
- **Asynq** (`github.com/hibiken/asynq`) — Redis-based distributed task queue for:
  - Email sending
  - Image processing
  - Webhook retry
  - Scheduled jobs
  - Invoice generation
  - Background processing

### Caching & Session Management

- **Redis** (`github.com/redis/go-redis/v9`) — Caching, session storage, and task queue backend

### External Integrations

- **OpenAI API** — AI-powered features (document extraction, categorization)
- **SMTP** — Email notifications
- **WhatsApp API** — Messaging and OTP delivery
- **Xendit** — Payment gateway integration
- **Cloudflare R2** — S3-compatible object storage for documents

### Configuration & Utilities

- **godotenv** (`github.com/joho/godotenv`) — Environment variable management
- **UUID** (`github.com/google/uuid`) — UUID generation

### Infrastructure

- **Docker** — Containerization
- **PostgreSQL** — Primary database
- **Redis** — Cache and queue
- **NATS JetStream** — Event streaming
- **Cloudflare R2** — Object storage
- **Prometheus / Grafana** (optional) — Metrics and monitoring
- **Sentry** (optional) — Error tracking

---

## Architecture Style

This project follows **Domain-Driven Design (DDD)** principles within a **modular monolith with event-driven architecture**.

### Domain-Driven Design (DDD)

The system is organized around business domains, not technical layers:

- **Bounded Contexts:** Each module (auth, tenant, accounting, etc.) represents a bounded context with clear boundaries
- **Domain Models:** Rich domain models that encapsulate business logic and rules
- **Ubiquitous Language:** Code reflects business terminology used by stakeholders
- **Aggregates:** Transaction boundaries align with business invariants
- **Domain Events:** Business events drive asynchronous workflows

### Event-Driven Architecture

The system uses **NATS JetStream** for event streaming and **Asynq** for task queuing:

**NATS JetStream** handles:
- Domain events (e.g., `company.claim_requested`, `ledger.posted`)
- Event sourcing patterns
- Cross-context communication
- Event replay and audit trails
- Pub/sub messaging

**Asynq** handles:
- Email sending
- Image processing
- Webhook retry with exponential backoff
- Scheduled jobs (e.g., daily reports, subscription renewals)
- Invoice generation
- Background processing tasks

### Project Structure

```
cmd/
  api/          # HTTP API server
  worker/       # Asynq background worker
  migrate/      # Database migration tool

internal/
  auth/         # Authentication & authorization domain
  tenant/       # Multi-tenancy domain
  company/      # Company identity domain
  counterparty/ # Counterparty management domain
  claim/        # Company claim workflow domain
  accounting/   # Accounting & ledger domain
  tax/          # Tax calculation domain
  document/     # Document management domain
  admin_review/ # Admin review workflow domain
  audit/        # Audit logging domain
  events/       # Event definitions & handlers
  messaging/    # NATS JetStream client
  config/       # Configuration management
  database/     # Database connection & pooling
  security/     # Security utilities
  shared/       # Shared utilities & helpers
  redis/        # Redis client
  smtp/         # Email client
  ai/           # AI/OpenAI integration

queries/        # SQLC SQL queries
migrations/     # Database migrations
docs/           # Swagger documentation
```

### Design Principles

The backend is **not** a simple CRUD system. It follows these principles:

1. **Domain-Centric:** Business logic lives in domain modules, not in handlers
2. **Event-Driven:** Heavy or slow tasks run asynchronously through events
3. **Transactional Consistency:** Use outbox pattern for reliable event publishing
4. **Idempotency:** All event consumers and API endpoints are idempotent
5. **Tenant Isolation:** Strict data isolation enforced at database and application layers

**Asynchronous Operations:**

- OCR and document extraction
- PDF generation
- Financial report generation
- Ledger posting
- Tax recalculation
- Document verification
- Webhook delivery
- Email and WhatsApp notifications
- Reconciliation
- Invoice generation

---

## High-Level Flow

### Event-Driven Architecture with Dual Queue System

```
Client / Frontend
      |
      v
HTTP API (Chi Router)
      |
      v
Domain Service (Business Logic)
      |
      v
PostgreSQL Transaction
      |
      |-- write business data
      |-- write outbox event
      |
      v
Commit Transaction
      |
      v
Outbox Publisher
      |
      |-- publishes to NATS JetStream (domain events)
      |-- enqueues to Asynq (background tasks)
      |
      v
┌─────────────────────────────────────────────────────┐
│                                                     │
│  NATS JetStream              Asynq (Redis)         │
│  (Domain Events)             (Task Queue)          │
│                                                     │
│  • company.claim_requested   • email.send          │
│  • ledger.posted             • image.process       │
│  • document.extracted        • webhook.retry       │
│  • transaction.created       • invoice.generate    │
│  • claim.approved            • report.schedule     │
│                                                     │
└─────────────────────────────────────────────────────┘
      |                              |
      v                              v
Event Consumers              Asynq Workers
      |                              |
      |-- ledger worker              |-- email worker
      |-- document worker            |-- image worker
      |-- claim worker               |-- webhook worker
      |-- notification worker        |-- invoice worker
      |-- report worker              |-- scheduled job worker
```

### Why Two Queue Systems?

**NATS JetStream** for domain events:
- Event sourcing and replay
- Cross-domain communication
- Audit trail and compliance
- Stream persistence
- Multiple consumers per event
- Event ordering guarantees

**Asynq** for background tasks:
- Simple task execution
- Retry with exponential backoff
- Scheduled/delayed tasks
- Task prioritization
- Redis-based (simpler ops)
- Task deduplication

---

## Event-Driven Design

The system uses **domain events** published to **NATS JetStream** for asynchronous processing and cross-domain communication.

### Domain Events

**Example events:**

```
# Authentication & Tenant
user.created
user.verified
tenant.created
tenant.subscription_changed

# Company & Counterparty
company.registered
company.candidate_created
company.claim_requested
company.claim_approved
company.claim_rejected
counterparty.reference_created
counterparty.matched

# Documents
document.uploaded
document.extraction_requested
document.extracted
document.verified

# Accounting
cash_transaction.created
journal_entry.created
ledger.posting_requested
ledger.posted
account.balance_updated

# Tax
tax.calculation_requested
tax.calculated
tax.report_generated

# Reports
report.generation_requested
report.generated
report.exported

# Notifications
notification.requested
notification.sent
notification.failed

# Webhooks
webhook.delivery_requested
webhook.delivered
webhook.failed
```

### Background Tasks (Asynq)

**Example tasks:**

```
# Email
email:send
email:verification
email:password_reset
email:invoice

# Processing
image:resize
image:ocr
document:extract
pdf:generate

# Webhooks
webhook:deliver
webhook:retry

# Scheduled
invoice:generate_monthly
report:daily_summary
subscription:check_expiry
trial:expiry_reminder

# Cleanup
session:cleanup
token:cleanup
cache:invalidate
```

---

## Event Envelope

All events should use a consistent envelope.

```json
{
  "event_id": "uuid",
  "event_type": "company.claim_requested",
  "event_version": 1,
  "tenant_id": "uuid",
  "actor_id": "uuid",
  "correlation_id": "uuid",
  "causation_id": "uuid",
  "idempotency_key": "string",
  "occurred_at": "2026-05-18T10:00:00Z",
  "payload": {},
  "metadata": {
    "source": "api",
    "request_id": "uuid"
  }
}
```

**Required fields:**

- `event_id`
- `event_type`
- `event_version`
- `tenant_id`
- `correlation_id`
- `occurred_at`
- `payload`

**Optional but recommended:**

- `actor_id`
- `causation_id`
- `idempotency_key`
- `metadata`

---

## Transactional Outbox Pattern

The system must use the **Transactional Outbox** pattern.

When the API handles a command, it should:

1. Start a PostgreSQL transaction.
2. Write business data.
3. Write an event to `outbox_events`.
4. Commit the transaction.
5. Let the outbox publisher publish the event to the message broker.

This prevents losing events when the database transaction succeeds but broker publishing fails.

```
API request
   |
   v
DB transaction
   |
   |-- insert cash_transaction
   |-- insert outbox event: cash_transaction.created
   |
   v
commit
   |
   v
outbox publisher publishes event
```

---

## Idempotent Consumers

Every worker must be idempotent.

Workers should store consumed event IDs in `inbox_consumed_events`.

```
Worker receives event
      |
      v
Check inbox_consumed_events
      |
      |-- already consumed -> skip
      |-- not consumed -> process
      |
      v
write result + mark consumed in one transaction
```

This prevents duplicate processing during retries, broker redelivery, or deployment restarts.

---

## Dead Letter Queue

Failed messages should not retry forever.

**Recommended behavior:**

```
event received
   |
   v
process
   |
   |-- success -> ack
   |-- transient failure -> retry with backoff
   |-- repeated failure -> send to DLQ
   |-- invalid event -> send to DLQ
```

DLQ messages must be observable and reviewable by admin or engineering.

---

## Required Modules

### Auth

Responsible for:

- Login
- Logout
- Password hashing
- Refresh token rotation
- JWT/PASETO issuing
- MFA-ready structure
- API keys
- Service-to-service auth

**Recommended:**

- bcrypt or Argon2id for password hashing
- Refresh token rotation
- Session tracking
- Device/IP tracking

### Tenant

Responsible for:

- Tenant workspace
- Company workspace
- Tenant membership
- Role assignment
- Permission checks
- Tenant isolation

Every tenant-owned table must include `tenant_id`.

### Company Identity

Responsible for:

- Verified companies
- Company candidates
- Company aliases
- Legal identifiers
- Normalized company names
- Duplicate detection hooks

**Important:**

```
A company candidate is not a verified company.
```

### Counterparty

Responsible for:

- Unverified counterparty references
- Tenant-specific counterparty display name
- Candidate matching
- Alias mapping
- Privacy-safe lookup

**Important:**

```
Company A must not see Company C's transaction details.
```

### Company Claim

Responsible for:

- Claim request
- Claim evidence
- Document submission
- Admin review
- Approve/reject/dispute
- Linking candidate to verified company
- Audit trail

**Important:**

```
No automatic company claim approval.
```

### Accounting

Responsible for:

- Cash in
- Cash out
- Chart of accounts
- Journal entries
- Ledger entries
- Transaction status
- Async ledger posting

Ledger posting should be event-driven when appropriate.

### Tax

Responsible for:

- Tax profiles
- Tax calculation hooks
- PPN-ready structure
- PPh-ready structure
- Tax document references
- Future e-Faktur/e-Bupot integrations

### Document

Responsible for:

- Document metadata
- Upload references
- Object storage paths
- Document type
- Verification status
- OCR jobs
- Access control

Files should be stored in S3-compatible storage, not directly in PostgreSQL.

### Admin Review

Responsible for:

- Admin cases
- Review queue
- Admin decisions
- Review notes
- Escalation
- Claim approval/rejection
- Document verification

All admin actions must be audited.

### Audit

Responsible for:

- Immutable audit logs
- Actor tracking
- Tenant tracking
- Request correlation
- Admin action logs
- Security-sensitive events

Audit logs should be append-only.

### Notification

Responsible for:

- Email notification
- WhatsApp notification
- In-app notification
- Event-driven delivery
- Retry
- DLQ

### Reporting

Responsible for:

- Financial reports
- Tax reports
- Async report generation
- Export jobs
- Materialized views
- Cached reports

Reports must not block API request-response paths.

### Webhooks

Responsible for:

- Outbound webhooks
- Inbound webhook verification
- Idempotency keys
- Delivery attempts
- Retry
- DLQ
- Enterprise API integrations

---

## Project Structure

```
.
├── cmd
│   ├── api/                # HTTP API server
│   ├── worker/             # Asynq background task worker
│   ├── consumer/           # NATS JetStream event consumer
│   ├── publisher/          # Outbox event publisher
│   ├── migrate/            # Database migration tool
│   ├── admin-seed/         # Platform admin seeder
│   └── plan-seed/          # Plan data seeder
│
├── internal
│   ├── api/
│   │   ├── handler/        # HTTP handlers (auth, workspace, entity, plan, subscription, billing, invite)
│   │   └── middleware/     # Auth, workspace, admin middleware
│   ├── auth/               # User authentication (register, login, OTP, sessions)
│   ├── admin/              # Platform admin (login, MFA, CRUD admins)
│   ├── workspace/          # Workspace management, ABAC roles, invites, members, counterparties
│   ├── entity/             # Entity CRUD (ORANG_PRIBADI, BADAN_USAHA)
│   ├── plan/               # Plan management (CRUD, features)
│   ├── subscription/       # Subscription lifecycle (subscribe, trial, cancel, usage)
│   ├── billing/            # Invoices, payments, Xendit integration
│   ├── events/             # Event system (outbox, publisher, consumer, NATS)
│   ├── ai/                 # OpenAI integration (OCR, categorization)
│   ├── config/             # Environment configuration
│   ├── database/           # PostgreSQL connection pooling
│   ├── db/                 # SQLC generated code (queries, models)
│   ├── redis/              # Redis client wrapper
│   ├── smtp/               # SMTP email client
│   └── shared/             # Shared utilities (JWT, OTP, Zenziva, validator, response helpers)
│
├── queries/                # SQLC query definitions (.sql)
├── migrations/             # Database migrations (numbered .sql files)
├── docs/                   # Swagger/OpenAPI generated docs
│
├── docker-compose.yml
├── Makefile
├── sqlc.yaml
├── go.mod
├── go.sum
├── .env.example
├── PROJECT_TRACKER.md
├── USER_FLOW.md
└── README.md
```

---

## Database Design

### Core Tables

```
users
tenants
tenant_memberships
roles
permissions
sessions
refresh_tokens
api_keys

verified_companies
company_candidates
company_aliases
counterparty_references

company_claim_requests
verification_documents
admin_review_cases

chart_of_accounts
cash_transactions
journal_entries
ledger_entries

tax_profiles
tax_calculations

documents

audit_logs
outbox_events
inbox_consumed_events

notification_jobs
webhook_deliveries
report_jobs
```

### Required Columns for Tenant-Owned Tables

Most business tables should include:

```
id
tenant_id
created_at
updated_at
created_by
updated_by
```

### Large Tables

Consider partitioning:

```
audit_logs
ledger_entries
cash_transactions
outbox_events
```

**Partitioning strategy:**

```
by month
by tenant hash
or hybrid depending on query pattern
```

### Important Indexes

**Examples:**

```sql
CREATE INDEX idx_cash_transactions_tenant_created_at
ON cash_transactions (tenant_id, created_at DESC);

CREATE INDEX idx_ledger_entries_tenant_account_period
ON ledger_entries (tenant_id, account_id, posted_at DESC);

CREATE INDEX idx_counterparty_references_tenant_name
ON counterparty_references (tenant_id, normalized_name);

CREATE INDEX idx_company_candidates_normalized_name
ON company_candidates (normalized_name);

CREATE INDEX idx_outbox_events_status_created_at
ON outbox_events (status, created_at);

CREATE INDEX idx_audit_logs_tenant_created_at
ON audit_logs (tenant_id, created_at DESC);
```

### Stakeholder's Initial Database Schema Proposal

> **Note:** This is an initial concept from the stakeholder that requires further development and refinement. While this represents the stakeholder's perspective and must be carefully considered, it should not be directly implemented without proper analysis, normalization, and alignment with the overall architecture described in this document.

#### 1. Table: users (Authentication & Credentials Management)

- **id_user** (PK, UUID): System-generated identity key
- **whatsapp** (VARCHAR, UNIQUE, NULLABLE): Phone number for OTP WhatsApp login
- **email** (VARCHAR, UNIQUE, NULLABLE): Email for login or password recovery
- **password_hash** (VARCHAR, NULLABLE): Encrypted password. Can be null if user only uses OTP authentication
- **status_akun** (VARCHAR): Account status flag (e.g., ACTIVE, SUSPENDED, UNVERIFIED)
- **created_at** (TIMESTAMP): Account registration timestamp

#### 2. Table: entities (ENTITAS)

- **id_entitas** (UUID, PK): Unique key for physical entity (System-generated)
- **id_user** (UUID, FK -> users.id_user, NULLABLE): Link to authentication table (Null if entity is a Shadow Entity or pure company)
- **jenis_entitas** (ENUM, NOT NULL): Standard classification (ORANG_PRIBADI or BADAN_USAHA)
- **nama_utama** (VARCHAR, NOT NULL): Original name from cashier, store name, or legal entity name
- **nik_npwp** (VARCHAR, NULLABLE): Official identity number for legal purposes, tax invoices, or receipts
- **nomor_wa** (VARCHAR, NULLABLE): Number for operational communication and billing delivery
- **alamat_lengkap** (TEXT, NULLABLE): Domicile address or billing address

#### 3. Table: entity_meta (Administrative Vault & Compliance)

This table stores long and heavy attribute data, keeping the entities table lean for very fast query searches (High-Speed Read).

- **id_meta** (PK, UUID)
- **id_entitas** (FK -> entities.id_entitas, UNIQUE): Ownership of administrative data
- **bidang_usaha** (VARCHAR, NULLABLE): Specific to BADAN_USAHA entity type
- **logo_profil_url** (TEXT, NULLABLE): Cloud storage link for images

#### 4. Table: entity_relations (Multi-Tenant Isolation & Business Core)

- **id_relasi** (PK, UUID)
- **id_objek** (FK -> entities.id_entitas): Acts as "Room Owner" (e.g., PT A)
- **id_subjek** (FK -> entities.id_entitas): Acts as "Guest/Member" (e.g., WhatsApp Customer)
- **jenis_relasi** (VARCHAR): Business relationship category (PELANGGAN, VENDOR, KARYAWAN, PEMILIK)
- **nama_alias_kustom** (VARCHAR, NULLABLE): **Key to Naming Flexibility.** Allows PT A to store subject as "Toko Maju", while PT B stores the same physical subject as "Pak Budi", without corrupting each other
- **status_relasi** (VARCHAR): Relationship activity flag (ACTIVE, INACTIVE)

#### 5. Table: workspace_roles (ABAC Permission System)

> Replaces the old `master_roles` table. Roles are now per-workspace, custom, and managed by workspace owners.

- **id** (UUID, PK)
- **workspace_id** (UUID, FK -> entities.id): The workspace this role belongs to
- **name** (VARCHAR): Role name (e.g., "Akuntan", "Kasir", "Manager")
- **description** (TEXT, NULLABLE): Human-readable description
- **permissions** (TEXT[]): Array of permission keys (e.g., `["transaction:create", "report:read"]`)
- **is_system** (BOOLEAN): System roles (e.g., "Owner") cannot be modified/deleted
- **created_by** (UUID, FK -> users.id): Who created this role
- **created_at**, **updated_at** (TIMESTAMPTZ)

#### 6. Table: workspace_role_assignments

> Links members to roles within a workspace. A member can have multiple roles.

- **id** (UUID, PK)
- **workspace_id** (UUID, FK -> entities.id)
- **member_entity_id** (UUID, FK -> entities.id): The member's personal entity
- **role_id** (UUID, FK -> workspace_roles.id, ON DELETE CASCADE)
- **assigned_by** (UUID, FK -> users.id)
- **created_at** (TIMESTAMPTZ)
- **UNIQUE** (workspace_id, member_entity_id, role_id)

#### 6b. Table: workspace_invites (Email-based Invitations)

- **id** (UUID, PK)
- **workspace_id** (UUID, FK -> entities.id)
- **invited_email** (VARCHAR): Must be a registered user's email
- **role_id** (UUID, FK -> workspace_roles.id): Role to assign on accept
- **token** (VARCHAR(64), UNIQUE): Secure invite token
- **invited_by** (UUID, FK -> users.id)
- **expires_at** (TIMESTAMPTZ): 24h from creation
- **accepted_at** (TIMESTAMPTZ, NULLABLE): Set when invite is accepted
- **created_at** (TIMESTAMPTZ)

#### 7. Table: items

- **id_entitas_pemilik** (UUID, Foreign Key): Absolute privacy boundary. Ensures PT A can never see price lists, materials, or AHSP recipes owned by PT B
- **nama_item** (VARCHAR, Not Null): Universal name of product, material, service, or project
- **tipe_item** (ENUM, Not Null): (BARANG_FISIK, JASA, PROYEK, AHSP_RAKITAN)
- **satuan_dasar** (VARCHAR, Not Null): Standard measurement unit reference (Kg, Jam, Paket, M2, M3)
- **harga_satuan** (NUMERIC, Not Null): Base cost price or reference Cost of Goods Sold
- **status_aktif** (BOOLEAN): Marker to archive products/projects no longer in use, without deleting data (Soft Delete)

#### 8. Table: transaksi (Transactions)

- **id_transaksi**
- **meta_transaksi**
- **id_entitas_sumber_transaksi**
- **id_entitas_lawan_transaksi**
- **tanggal_transaksi**
- **keterangan_sumber_transaksi**
- **keterangan_lawan_transaksi**
- **jenis_akun_sumber** (D/K - Debit/Kredit)
- **kode_akun_sumber**
- **kode_akun_pembantu_sumber**
- **jenis_akun_lawan** (D/K - Debit/Kredit)
- **kode_akun_lawan**
- **kode_akun_pembantu_lawan**
- **debit** (Rp.)
- **kredit** (Rp.)
- **accounting_balance** (1, 0)
- **fiskal** (1, 0)
- **auditor_status** (1, 0)
- **dilaporkan_lawan** (1, 0)

#### 9. Table: akun (Accounts)

- **id_entity**
- **kode_akun_pembantu**
- **nama_akun_pembantu**

---

## API Routes

### Route Groups

```
/api/v1/auth              — User authentication (register, login, OTP, sessions)
/api/v1/plans             — Public plan listing
/api/v1/entities          — Entity CRUD (authenticated)
/api/v1/workspaces        — Workspace management, members, roles, invites, counterparties
/api/v1/subscription      — Subscription management (workspace-scoped)
/api/v1/billing           — Invoices & payments (workspace-scoped)
/api/v1/webhooks/xendit   — Xendit payment webhook (public, token-verified)

/api/v1/admin/auth        — Admin authentication (login, MFA, sessions)
/api/v1/admin/admins      — Admin management (SUPER_ADMIN only)
/api/v1/admin/plans       — Plan management (SUPER_ADMIN + ENGINEER)
/api/v1/admin/subscriptions — Subscription overview (SUPER_ADMIN + ENGINEER)
/api/v1/admin/billing     — Invoice overview (SUPER_ADMIN + ENGINEER)
```

### User API Routes

```
# Auth (public)
POST   /api/v1/auth/register          — Register new account
POST   /api/v1/auth/login/email       — Login with email + password
POST   /api/v1/auth/login/otp         — Login with WhatsApp OTP
POST   /api/v1/auth/otp/request       — Request OTP code
POST   /api/v1/auth/verify            — Verify email/WhatsApp OTP
POST   /api/v1/auth/password/reset    — Reset password via OTP
POST   /api/v1/auth/refresh           — Refresh access token (HttpOnly cookie)

# Auth (authenticated)
GET    /api/v1/auth/me                — Get current user profile
POST   /api/v1/auth/logout            — Logout current session
POST   /api/v1/auth/logout-all        — Revoke all sessions
POST   /api/v1/auth/password/change   — Change password
GET    /api/v1/auth/sessions          — List active sessions
DELETE /api/v1/auth/sessions/{id}     — Revoke specific session

# Plans (public)
GET    /api/v1/plans                  — List all active plans
GET    /api/v1/plans/{slug}           — Get plan details with features

# Entities (authenticated)
POST   /api/v1/entities               — Create entity
GET    /api/v1/entities               — List my entities
GET    /api/v1/entities/search        — Search entities
GET    /api/v1/entities/{id}          — Get entity by ID
PATCH  /api/v1/entities/{id}          — Update entity
PATCH  /api/v1/entities/{id}/meta     — Update entity metadata

# Workspaces (authenticated)
POST   /api/v1/workspaces             — Create workspace
GET    /api/v1/workspaces             — List my workspaces (includes subscription_status)
POST   /api/v1/workspaces/invites/accept — Accept invite (token-based)

# Workspaces (workspace-scoped, requires X-Workspace-ID header)
GET    /api/v1/workspaces/members             — List members
PATCH  /api/v1/workspaces/members/{id}        — Update member [member:manage]
DELETE /api/v1/workspaces/members/{id}        — Remove member [member:remove]

GET    /api/v1/workspaces/roles               — List workspace roles
POST   /api/v1/workspaces/roles               — Create custom role [role:create]
PATCH  /api/v1/workspaces/roles/{id}          — Update role [role:update]
DELETE /api/v1/workspaces/roles/{id}          — Delete role [role:delete]
POST   /api/v1/workspaces/roles/assign        — Assign role to member [role:assign]
POST   /api/v1/workspaces/roles/unassign      — Unassign role [role:assign]

POST   /api/v1/workspaces/invites             — Send invite email [member:invite]
GET    /api/v1/workspaces/invites             — List pending invites
DELETE /api/v1/workspaces/invites/{id}        — Revoke invite [member:invite]

POST   /api/v1/workspaces/counterparties      — Add counterparty
GET    /api/v1/workspaces/counterparties      — List counterparties

# Subscription (workspace-scoped)
POST   /api/v1/subscription            — Subscribe to plan (returns payment_url for paid plans)
GET    /api/v1/subscription            — Get active subscription
GET    /api/v1/subscription/history    — List subscription history
POST   /api/v1/subscription/cancel     — Cancel subscription
POST   /api/v1/subscription/change     — Change plan
GET    /api/v1/subscription/usage      — Get quota usage

# Billing (workspace-scoped)
GET    /api/v1/billing/invoices        — List invoices
GET    /api/v1/billing/invoices/{id}   — Get invoice details
POST   /api/v1/billing/pay             — Initiate payment (returns Xendit URL)
GET    /api/v1/billing/payments        — List payments

# Webhook (public, token-verified)
POST   /api/v1/webhooks/xendit         — Xendit payment callback
```

### Admin API Routes

```
# Admin Auth
POST   /api/v1/admin/auth/login        — Admin login (step 1)
POST   /api/v1/admin/auth/mfa/verify   — MFA verification (step 2)
POST   /api/v1/admin/auth/mfa/setup    — Setup MFA (authenticated)
POST   /api/v1/admin/auth/mfa/confirm  — Confirm MFA setup
POST   /api/v1/admin/auth/refresh      — Refresh admin token
POST   /api/v1/admin/auth/logout       — Admin logout
GET    /api/v1/admin/auth/me           — Admin profile

# Admin Management (SUPER_ADMIN only)
POST   /api/v1/admin/admins            — Invite admin
GET    /api/v1/admin/admins            — List admins
PATCH  /api/v1/admin/admins/{id}       — Update admin
DELETE /api/v1/admin/admins/{id}       — Delete admin

# Plan Management (SUPER_ADMIN + ENGINEER)
GET    /api/v1/admin/plans             — List all plans (including inactive)
POST   /api/v1/admin/plans             — Create plan
GET    /api/v1/admin/plans/{id}        — Get plan
PATCH  /api/v1/admin/plans/{id}        — Update plan
DELETE /api/v1/admin/plans/{id}        — Soft-delete plan
POST   /api/v1/admin/plans/{id}/features          — Set feature
DELETE /api/v1/admin/plans/{id}/features/{key}     — Remove feature

# Subscription & Billing Overview (SUPER_ADMIN + ENGINEER)
GET    /api/v1/admin/subscriptions     — List all subscriptions
GET    /api/v1/admin/billing/invoices  — List all invoices
```

---

## Synchronous vs Asynchronous Operations

### Synchronous

Good for:

```
authentication
simple reads
simple writes
validation
creating transaction draft
creating claim request
requesting upload URL
```

### Asynchronous

Must be async:

```
ledger posting
OCR
document extraction
report generation
PDF generation
tax recalculation
notification sending
webhook delivery
large imports
reconciliation
```

---

## Workers

The system has two types of workers:

1. **Event Consumers** (NATS JetStream) - Domain event handlers
2. **Task Workers** (Asynq) - Background task processors

### Event Consumers (NATS JetStream)

#### Outbox Publisher

Reads `outbox_events` table and publishes to NATS JetStream.

**Responsibilities:**

- Fetch pending events from outbox table
- Publish to NATS JetStream streams
- Mark as published in database
- Retry failed publish with exponential backoff
- Preserve event ordering when required
- Handle dead letter queue for poison messages

#### Ledger Worker

**Consumes (NATS):**

```
ledger.posting_requested
cash_transaction.created
journal_entry.created
```

**Responsibilities:**

- Validate double-entry posting rules
- Generate ledger entries
- Update account balances
- Mark transaction as posted
- Emit `ledger.posted` event
- Ensure accounting equation balance (Assets = Liabilities + Equity)

#### Document Worker

**Consumes (NATS):**

```
document.uploaded
document.extraction_requested
```

**Responsibilities:**

- Trigger OCR/extraction via OpenAI
- Extract structured data (vendor, amount, date, NPWP)
- Update document extraction state
- Create entity references from extracted data
- Emit `document.extracted` event

#### Claim Verification Worker

**Consumes (NATS):**

```
company.claim_requested
document.extracted
```

**Responsibilities:**

- Prepare admin review case
- Run duplicate/similarity matching
- Calculate risk score based on documents
- Attach document findings to review case
- **Never auto-approve ownership claims**
- Emit events for admin notification

#### Notification Worker

**Consumes (NATS):**

```
notification.requested
company.claim_approved
company.claim_rejected
report.generated
user.verified
subscription.expiring
```

**Responsibilities:**

- Route notifications to appropriate channels (email/WhatsApp/in-app)
- Enqueue email tasks to Asynq
- Enqueue WhatsApp tasks to Asynq
- Track notification delivery status
- Handle notification preferences

#### Report Worker

**Consumes (NATS):**

```
report.generation_requested
```

**Responsibilities:**

- Generate financial reports (Balance Sheet, Income Statement, Cash Flow)
- Generate tax reports (PPN, PPh)
- Use materialized views for performance
- Enqueue PDF generation to Asynq
- Emit `report.generated` event
- Store report metadata

#### Webhook Worker

**Consumes (NATS):**

```
webhook.delivery_requested
```

**Responsibilities:**

- Deliver webhook payloads to external URLs
- Sign payloads with HMAC
- Enqueue retry tasks to Asynq on failure
- Record delivery attempts
- Send failed deliveries to DLQ after max retries

### Task Workers (Asynq)

#### Email Worker

**Processes tasks:**

```
email:send
email:verification
email:password_reset
email:invoice
email:notification
```

**Responsibilities:**

- Connect to SMTP server
- Send emails with templates
- Retry on transient failures (connection issues)
- Track delivery status
- Handle bounce notifications

#### Image Worker

**Processes tasks:**

```
image:resize
image:compress
image:ocr
image:thumbnail
```

**Responsibilities:**

- Resize/compress uploaded images
- Generate thumbnails
- Perform OCR on image documents
- Store processed images to R2/S3
- Update document metadata

#### Webhook Retry Worker

**Processes tasks:**

```
webhook:retry
webhook:deliver
```

**Responsibilities:**

- Retry failed webhook deliveries
- Exponential backoff (1min, 5min, 15min, 1hr, 6hr, 24hr)
- Track retry attempts
- Move to DLQ after max retries
- Update webhook delivery status

#### Invoice Worker

**Processes tasks:**

```
invoice:generate
invoice:send
invoice:reminder
```

**Responsibilities:**

- Generate invoice PDFs
- Calculate tax amounts
- Send invoice via email
- Schedule payment reminders
- Update invoice status

#### Scheduled Job Worker

**Processes tasks:**

```
job:daily_report
job:subscription_check
job:trial_expiry_reminder
job:session_cleanup
job:token_cleanup
job:cache_invalidate
```

**Responsibilities:**

- Run scheduled/cron jobs
- Generate daily/weekly/monthly reports
- Check subscription expiry
- Send trial expiry reminders
- Cleanup expired sessions and tokens
- Invalidate stale cache entries

---

## Security Requirements

This backend handles sensitive financial, tax, and legal data.

Security must be designed from the start.

**Required:**

```
strict tenant isolation
RBAC/ABAC
least privilege
bcrypt/Argon2id password hashing
refresh token rotation
JWT or PASETO
MFA-ready design
API key support
service-to-service auth
secure document access
signed object storage URLs
encryption at rest
TLS in transit
secrets management
audit logs
admin action tracking
rate limiting
request body limits
input validation
idempotency keys
session/device tracking
```

**Never trust:**

```
company name similarity
email alone
user-submitted documents without verification
client-side tenant_id
```

---

## Workspace Isolation (Multi-Tenant)

Every workspace-scoped request must resolve workspace context and verify permissions.

**Flow (implemented in middleware):**

```
Authenticate user (JWT access token)
      |
      v
Extract X-Workspace-ID header
      |
      v
VerifyWorkspaceAccess: check entity_relations
(user's personal entity must have PEMILIK or KARYAWAN relation to workspace)
      |
      v
Resolve permissions:
  - PEMILIK → wildcard ["*"] (full access)
  - KARYAWAN → aggregate permissions from workspace_role_assignments
      |
      v
RequirePermission("action:resource") middleware check
      |
      v
Execute query scoped to workspace entity ID
```

**Key principles:**

- Never accept workspace ID from request body — always from `X-Workspace-ID` header
- Workspace access derived from `entity_relations` table (not a separate membership table)
- Permissions are per-workspace custom roles (ABAC), not global roles
- Owner (PEMILIK) always bypasses permission checks via wildcard `["*"]`
- Permission keys follow `resource:action` pattern (e.g., `transaction:create`, `member:invite`)
- `RequirePermission` middleware supports exact match, wildcard `*`, and resource wildcard `transaction:*`

---

## Observability

Every request and event should include:

```
request_id
correlation_id
causation_id
tenant_id
actor_id
event_id
event_type
```

**Use:**

```
structured logging (zap/zerolog)
OpenTelemetry
metrics
distributed traces
error tracking (Sentry)
```

**Recommended metrics:**

```
http_request_duration
http_request_count
db_query_duration
outbox_pending_count
event_publish_failure_count
worker_processing_duration
worker_retry_count
dlq_message_count
document_processing_duration
ledger_posting_duration
report_generation_duration
```

---

## Testing Strategy

**Use:**

```
go test
testcontainers-go
integration tests
contract tests for events
table-driven tests
```

**Test categories:**

```
unit tests
integration tests
database tests
event consumer tests
tenant isolation tests
auth tests
idempotency tests
audit log tests
claim workflow tests
ledger posting tests
```

**Critical tests:**

```
tenant A cannot access tenant B data
company claim cannot auto-approve
duplicate event is processed only once
failed event goes to DLQ
outbox event is written in same transaction as business data
ledger entries balance correctly
admin approval is audited
```

---

## Deployment

**Recommended infrastructure:**

```
Docker
PostgreSQL (managed database)
NATS JetStream (event streaming)
Redis (cache & task queue)
Cloudflare R2 or S3-compatible storage
Prometheus/Grafana (monitoring)
Sentry (error tracking)
```

**Services:**

```
api                 # HTTP API server
worker              # Asynq task worker
event-consumer      # NATS JetStream event consumer
outbox-publisher    # Outbox pattern publisher
postgres            # Primary database
nats                # NATS JetStream server
redis               # Cache & Asynq backend
```

**Example deployment separation:**

```
api service:              horizontally scalable (multiple instances)
worker service:           horizontally scalable (multiple instances)
event-consumer service:   horizontally scalable (consumer groups)
outbox-publisher:         one or more instances with distributed locking
document worker:          separately scalable (high CPU/memory)
report worker:            separately scalable (high CPU/memory)
```

**Scaling considerations:**

- **API:** Scale based on HTTP request load
- **Asynq Workers:** Scale based on queue depth and task processing time
- **Event Consumers:** Scale based on event throughput and processing latency
- **Outbox Publisher:** Use distributed locking (Redis) to prevent duplicate publishing
- **Database:** Use connection pooling, read replicas for reporting queries
- **NATS JetStream:** Use clustering for high availability

---

## Environment Variables

**Example `.env`:**

```env
# Application
APP_ENV=production
APP_PORT=8080
APP_SECRET=your-secret-key-min-32-chars
REFRESH_TOKEN_SECRET=your-different-secret-key-min-32-chars

# Auth Token Expiry
ACCESS_TOKEN_EXPIRY_MINUTES=15
REFRESH_TOKEN_EXPIRY_DAYS=7

# CORS (comma-separated origins)
CORS_ALLOWED_ORIGINS=https://app.azzet.id
ADMIN_CORS_ALLOWED_ORIGINS=https://admin.azzet.id

# Database
DATABASE_URL=postgres://user:pass@localhost:5432/azzet?sslmode=require

# Redis
REDIS_URL=redis://localhost:6379

# NATS JetStream
NATS_URL=nats://localhost:4222

# Object Storage (Cloudflare R2)
R2_ACCOUNT_ID=your-account-id
R2_ACCESS_KEY_ID=your-access-key
R2_SECRET_ACCESS_KEY=your-secret-key
R2_BUCKET_NAME=azzet-documents
R2_ENDPOINT=https://your-account.r2.cloudflarestorage.com

# OpenAI
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4-turbo

# Zenziva WhatsApp OTP
ZENZIVA_URL=https://console.zenziva.net/waofficial/api/sendWAOfficial/
ZENZIVA_USERKEY=your-zenziva-userkey
ZENZIVA_PASSKEY=your-zenziva-passkey
ZENZIVA_BRAND=Azzet

# Xendit Payment Gateway
XENDIT_API_KEY=your-xendit-api-key
XENDIT_WEBHOOK_SECRET=your-webhook-secret
XENDIT_CALLBACK_URL=https://api.azzet.id/api/v1/webhooks/xendit
XENDIT_SUCCESS_URL=https://app.azzet.id/payment/success
XENDIT_FAILURE_URL=https://app.azzet.id/payment/failed

# SMTP (Email)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
SMTP_FROM=noreply@azzet.com

# Frontend URL (used for invite email links)
FRONTEND_URL=https://app.azzet.id

# Worker
WORKER_CONCURRENCY=50
```

---

## Local Development

### Prerequisites

- **Go 1.26.2+**
- **Docker & Docker Compose**
- **PostgreSQL client tools**
- **SQLC** (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)
- **Swag** (`go install github.com/swaggo/swag/cmd/swag@latest`)

### Start dependencies

```bash
docker compose up -d
```

This starts:
- PostgreSQL
- Redis
- NATS JetStream
- Cloudflare R2 (S3-compatible storage)

### Run migrations

```bash
go run cmd/migrate/main.go
```

The custom migration tool will:
- Create `schema_migrations` table if not exists
- Apply all pending migrations from `migrations/` directory
- Track applied migrations

### Generate SQLC code

```bash
sqlc generate
```

This generates type-safe Go code from SQL queries in `queries/` directory.

### Generate Swagger docs

```bash
swag init -g cmd/api/main.go -o docs
```

### Run API

```bash
go run cmd/api/main.go
```

API will be available at:
- `http://localhost:8080/api/v1`
- Swagger UI: `http://localhost:8080/swagger/index.html`

### Run worker

```bash
go run cmd/worker/main.go
```

Worker will:
- Connect to Redis for Asynq tasks
- Process background tasks (email, image processing, webhooks, etc.)

### Run event consumer

```bash
# TODO: Implement event consumer
go run cmd/consumer/main.go
```

Event consumer will:
- Connect to NATS JetStream
- Subscribe to domain event streams
- Process events idempotently

### Run tests

```bash
go test ./...
```

### Run tests with coverage

```bash
go test -cover ./...
# or
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Development workflow

```bash
# Install dependencies
go mod download

# Run linter (if golangci-lint installed)
golangci-lint run

# Format code
go fmt ./...

# Vet code
go vet ./...

# Build
go build -o bin/api cmd/api/main.go
go build -o bin/worker cmd/worker/main.go
go build -o bin/migrate cmd/migrate/main.go
```
make test
# or
go test ./...
```

### Run tests with coverage

```bash
make test-coverage
# or
go test -cover ./...
```

---

## Technology Choices

### Why NATS JetStream?

**We chose NATS JetStream** for domain event streaming because:

```
✓ Lightweight and fast (written in Go)
✓ Stream persistence with replay capability
✓ Pub/sub with multiple consumers per event
✓ Event ordering guarantees
✓ Simpler operations than Kafka
✓ Built-in clustering and high availability
✓ Perfect for event sourcing patterns
✓ Low latency (<1ms)
✓ Horizontal scalability
✓ Native Go client
```

**Use cases in Azzet:**
- Domain event streaming (company.claim_requested, ledger.posted, etc.)
- Cross-domain communication
- Event sourcing and audit trails
- Real-time notifications
- Event replay for debugging and recovery

### Why Asynq?

**We chose Asynq** for background task processing because:

```
✓ Redis-based (simple infrastructure)
✓ Built-in retry with exponential backoff
✓ Task scheduling and delayed execution
✓ Task prioritization
✓ Task deduplication
✓ Web UI for monitoring
✓ Simpler than full message broker for simple tasks
✓ Perfect for fire-and-forget tasks
✓ Native Go library
```

**Use cases in Azzet:**
- Email sending
- Image processing and OCR
- Webhook retry
- Scheduled jobs (daily reports, subscription checks)
- Invoice generation
- Background cleanup tasks

### Why SQLC + pgx/v5?

**We chose SQLC with pgx/v5** over ORMs because:

**SQLC:**
```
✓ Type-safe SQL at compile time
✓ Write raw SQL, get Go code
✓ No runtime reflection
✓ Better performance than ORMs
✓ Full control over queries
✓ Perfect for complex accounting queries
✓ No N+1 query problems
✓ Explicit and predictable
```

**pgx/v5:**
```
✓ Best PostgreSQL driver for Go
✓ Connection pooling built-in
✓ Prepared statement caching
✓ Binary protocol support
✓ Better performance than database/sql
✓ Rich PostgreSQL feature support
✓ Context-aware
```

**Why not GORM?**
- Accounting systems need explicit SQL control
- Complex joins and aggregations are clearer in SQL
- Performance is critical for financial queries
- Type safety at compile time vs runtime

### Why Chi Router?

**We chose Chi** over Gin/Echo/Fiber because:

```
✓ Idiomatic Go (uses standard http.Handler)
✓ Lightweight and fast
✓ Composable middleware
✓ Context-based routing
✓ Sub-router support
✓ No external dependencies
✓ Stable and mature
✓ Great for building RESTful APIs
```

### Why JWT (not PASETO)?

**We chose JWT** because:

```
✓ Widely supported across platforms
✓ Many client libraries available
✓ Enterprise integration compatibility
✓ Familiar to most developers
✓ Good enough with proper implementation
```

**Security measures:**
- Use HS256 or RS256 only (no algorithm confusion)
- Short-lived access tokens (15 minutes)
- Refresh token rotation
- Token blacklisting via Redis
- Secure secret management

### Why Custom Migration Tool?

**We built a custom migration tool** instead of using golang-migrate because:

```
✓ Simple and lightweight
✓ No external CLI dependency
✓ Embedded in application
✓ Transaction-based migrations
✓ Version tracking in database
✓ Easy to customize
✓ No up/down file complexity
```

### Why Custom Validator?

**We use custom validation functions** instead of validator libraries because:

```
✓ No struct tags needed
✓ Explicit validation logic
✓ Better error messages
✓ Domain-specific validation rules
✓ No reflection overhead
✓ Easier to test
✓ Full control over validation flow
```

---

## Common Mistakes to Avoid

**Do not:**

```
perform OCR inside HTTP handlers
generate large reports inside request-response path
trust company name similarity as ownership proof
leak counterparty data across tenants
skip transactional outbox pattern
skip idempotent consumers
ignore dead letter queue
mix NATS events with Asynq tasks (use each for its purpose)
store large files directly in PostgreSQL
accept tenant_id blindly from client
skip audit logs for admin actions
use ORM for complex accounting queries
forget to index tenant_id columns
skip connection pooling
ignore event ordering requirements
auto-approve company ownership claims
```

---

## Implementation Roadmap

### Phase 1 — Foundation (Week 1-2)

```
✓ Go project setup with proper structure
✓ Chi router setup
✓ SQLC configuration
✓ pgx/v5 connection pooling
✓ Custom migration tool
✓ Health endpoint
✓ Swagger/OpenAPI setup
✓ Docker Compose (PostgreSQL, Redis, NATS)
✓ Environment configuration
✓ Logging with slog
✓ Basic middleware (CORS, RequestID, Logger)
```

### Phase 2 — Auth & Tenant (Week 3-4)

```
□ User registration and login
□ JWT token generation and validation
□ Refresh token rotation
□ Password hashing with bcrypt
□ Session management with Redis
□ Tenant creation and management
□ Tenant membership
□ RBAC (roles and permissions)
□ Permission middleware
□ Tenant context resolver
□ Multi-tenant isolation enforcement
```

### Phase 3 — Event System (Week 5-6)

```
□ Event envelope definition
□ outbox_events table
□ inbox_consumed_events table
□ Outbox publisher (NATS JetStream)
□ NATS JetStream client setup
□ Event consumer base with idempotency
□ Asynq task queue setup
□ Dead letter queue handling
□ Event replay mechanism
□ Monitoring and observability
```

### Phase 4 — Company & Counterparty (Week 7-8)

```
□ Entity model (ORANG_PRIBADI, BADAN_USAHA)
□ Verified companies
□ Company candidates (shadow entities)
□ Company aliases
□ Counterparty references
□ Custom naming (nama_alias_kustom)
□ Candidate matching algorithm
□ Privacy-safe lookup
□ Entity relationship management
```

### Phase 5 — Claim Workflow (Week 9-10)

```
□ Company claim request
□ Document submission
□ Admin review case creation
□ Approve/reject/dispute workflow
□ Claim verification worker (NATS)
□ Audit logs for all claim actions
□ Claim events (claim_requested, claim_approved, etc.)
□ Notification on claim status change
```

### Phase 6 — Accounting Core (Week 11-13)

```
□ Chart of accounts (COA)
□ Account hierarchy
□ Cash transactions
□ Journal entries
□ Ledger entries
□ Double-entry posting rules
□ Ledger worker (NATS consumer)
□ Account balance calculation
□ Transaction status tracking
□ Accounting equation validation
```

### Phase 7 — Documents & OCR (Week 14-15)

```
□ Document metadata model
□ Upload flow with presigned URLs
□ Cloudflare R2 integration
□ Document worker (NATS consumer)
□ OpenAI OCR integration
□ Entity extraction from documents
□ NPWP/NIK extraction
□ Document verification status
□ Access control for documents
```

### Phase 8 — Tax & Reporting (Week 16-17)

```
□ Tax profiles
□ Tax calculation hooks (PPN, PPh)
□ Tax document references
□ Report jobs table
□ Report worker (NATS consumer)
□ Financial reports (Balance Sheet, Income Statement, Cash Flow)
□ Tax reports
□ Materialized views for performance
□ PDF generation (Asynq task)
□ Export jobs (Excel, PDF)
```

### Phase 9 — Notification & Webhooks (Week 18-19)

```
□ Notification worker (NATS consumer)
□ Email worker (Asynq task)
□ SMTP integration
□ WhatsApp API integration
□ In-app notification
□ Webhook delivery worker (NATS consumer)
□ Webhook retry worker (Asynq task)
□ HMAC signature for webhooks
□ Delivery attempt tracking
□ Webhook DLQ
```

### Phase 10 — Background Jobs (Week 20)

```
□ Invoice generation worker (Asynq)
□ Scheduled job worker (Asynq)
□ Daily/weekly/monthly reports
□ Subscription expiry checker
□ Trial expiry reminder
□ Session cleanup job
□ Token cleanup job
□ Cache invalidation job
```

### Phase 11 — Hardening (Week 21-22)

```
□ Load testing
□ Security testing
□ Tenant isolation tests
□ Idempotency tests
□ Event replay tests
□ Rate limiting
□ Request body size limits
□ Database query optimization
□ Index review and optimization
□ Connection pool tuning
□ Monitoring dashboards
□ Error tracking (Sentry)
□ Backup/restore procedures
□ Production readiness review
```

---

## Production Readiness Checklist

### Infrastructure

```
[ ] PostgreSQL configured with connection pooling
[ ] PostgreSQL read replicas for reporting queries
[ ] NATS JetStream cluster setup (3+ nodes)
[ ] Redis cluster or sentinel for high availability
[ ] Cloudflare R2 or S3 bucket configured
[ ] Database backups automated (daily + WAL archiving)
[ ] Backup restore procedure tested
[ ] Disaster recovery plan documented
```

### Security

```
[ ] Tenant isolation tested (cannot access other tenant data)
[ ] RBAC tested (role-based access control)
[ ] JWT secret rotation strategy
[ ] Secrets stored in secure vault (not in code)
[ ] TLS/HTTPS enabled for all endpoints
[ ] Rate limiting enabled per tenant/user
[ ] Request body size limits configured
[ ] SQL injection prevention verified
[ ] XSS prevention verified
[ ] CSRF protection for state-changing operations
[ ] Security headers configured (CSP, HSTS, etc.)
[ ] Dependency vulnerability scanning
[ ] Security review completed
[ ] Penetration testing completed
```

### Observability

```
[ ] Structured logging with slog
[ ] Request correlation IDs implemented
[ ] Event correlation IDs implemented
[ ] Log aggregation configured (ELK, Loki, etc.)
[ ] Metrics collection (Prometheus)
[ ] Dashboards created (Grafana)
[ ] Alerting rules configured
[ ] Error tracking enabled (Sentry)
[ ] Performance monitoring enabled
[ ] Database query performance monitoring
[ ] NATS JetStream monitoring
[ ] Asynq queue monitoring
```

### Event-Driven Architecture

```
[ ] Outbox pattern implemented
[ ] Outbox publisher running
[ ] Idempotent consumers implemented
[ ] Event replay tested
[ ] Dead letter queue implemented
[ ] DLQ monitoring and alerting
[ ] Event ordering verified where required
[ ] NATS JetStream streams configured
[ ] Asynq queues configured
[ ] Worker scaling tested
```

### Database

```
[ ] All tenant-owned tables have tenant_id column
[ ] All tenant queries filter by tenant_id
[ ] Database indexes reviewed and optimized
[ ] Slow query log analyzed
[ ] Large tables partitioning plan ready
[ ] Migration rollback strategy tested
[ ] Database connection pool tuned
[ ] Query timeout configured
[ ] Transaction isolation level verified
```

### Audit & Compliance

```
[ ] Audit logs implemented for all sensitive actions
[ ] Admin actions audited
[ ] User actions audited
[ ] Financial transactions audited
[ ] Company claim workflow audited
[ ] Audit logs are append-only
[ ] Audit log retention policy defined
[ ] GDPR compliance reviewed (if applicable)
```

### Testing

```
[ ] Unit tests coverage > 70%
[ ] Integration tests for critical flows
[ ] Tenant isolation tests
[ ] Idempotency tests
[ ] Event consumer tests
[ ] Load testing completed
[ ] Stress testing completed
[ ] Chaos engineering tests (optional)
```

### Performance

```
[ ] API response time < 200ms (p95)
[ ] Database query time < 100ms (p95)
[ ] Event processing latency < 1s (p95)
[ ] Background task processing time monitored
[ ] Connection pooling optimized
[ ] Cache hit rate > 80% (where applicable)
[ ] CDN configured for static assets
```

### Documentation

```
[ ] API documentation (Swagger) up to date
[ ] Architecture documentation complete
[ ] Deployment runbook created
[ ] Incident response playbook created
[ ] Onboarding documentation for new developers
[ ] Database schema documented
[ ] Event catalog documented
```

---

## Final Principles

This backend is designed around **five core principles**:

```
1. Domain-Driven Design (DDD)
   - Business logic in domain modules
   - Ubiquitous language
   - Bounded contexts
   - Rich domain models

2. Event-Driven Architecture
   - NATS JetStream for domain events
   - Asynq for background tasks
   - Transactional outbox pattern
   - Idempotent consumers

3. No Cross-Tenant Data Leakage
   - Strict tenant isolation at all layers
   - Never trust client-provided tenant_id
   - All queries filtered by tenant_id

4. No Financial/Legal Action Without Auditability
   - All sensitive actions logged
   - Immutable audit trail
   - Admin actions tracked
   - Event sourcing for compliance

5. No Heavy Workload Inside Request-Response Handlers
   - Async processing via events and tasks
   - Quick API responses
   - Background workers for heavy lifting
```

---

## Architecture Summary

```
┌─────────────────────────────────────────────────────────────┐
│                     Azzet Backend                           │
│                                                             │
│  Domain-Driven Design + Event-Driven Architecture          │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   Chi API    │  │ NATS Events  │  │ Asynq Tasks  │    │
│  │   (HTTP)     │  │ (Streaming)  │  │   (Queue)    │    │
│  └──────────────┘  └──────────────┘  └──────────────┘    │
│         │                  │                  │            │
│         v                  v                  v            │
│  ┌──────────────────────────────────────────────────┐    │
│  │           Domain Modules (DDD)                   │    │
│  │  auth | tenant | company | accounting | tax      │    │
│  │  counterparty | claim | document | audit         │    │
│  └──────────────────────────────────────────────────┘    │
│         │                                                  │
│         v                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │  PostgreSQL  │  │    Redis     │  │ Cloudflare R2│   │
│  │   (SQLC)     │  │   (Cache)    │  │   (Storage)  │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## License

Proprietary - All rights reserved

---

## Contact

For questions or support, contact the development team.

---

**Last Updated:** 2026-05-18
