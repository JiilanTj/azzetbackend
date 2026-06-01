# Azzet - User Flow Documentation

> Dokumen ini menjelaskan flow user dari pertama kali buka aplikasi sampai bisa menggunakan fitur.
> Ditujukan untuk tim frontend sebagai referensi implementasi UI/UX.

---

## Table of Contents

- [Overview](#overview)
- [Flow 1: Registration](#flow-1-registration)
- [Flow 2: Email Verification](#flow-2-email-verification)
- [Flow 3: WhatsApp Verification](#flow-3-whatsapp-verification)
- [Flow 4: Login](#flow-4-login)
- [Flow 5: Token Refresh](#flow-5-token-refresh)
- [Flow 6: First Time Setup (Post-Registration)](#flow-6-first-time-setup-post-registration)
- [Flow 7: Create Business Workspace](#flow-7-create-business-workspace)
- [Flow 8: Subscribe to Plan](#flow-8-subscribe-to-plan)
- [Flow 9: Billing & Payment](#flow-9-billing--payment)
- [Flow 10: Workspace Management](#flow-10-workspace-management)
- [Flow 11: Session Management](#flow-11-session-management)
- [Flow 12: Password Management](#flow-12-password-management)
- [Flow 13: Accounting](#flow-13-accounting-phase-7)
- [Flow 14: Entity Identity & Verification](#flow-14-entity-identity--verification)
- [Flow 15: Company Claim Workflow](#flow-15-company-claim-workflow)
- [Flow 16: Counterparty Alias Management](#flow-16-counterparty-alias-management)
- [State Diagram](#state-diagram)
- [Headers Reference](#headers-reference)
- [Error Handling](#error-handling)

---

## Overview

### User States

```
UNVERIFIED → ACTIVE → (using app)
                ↓
           SUSPENDED (by admin)
```

### High-Level Journey

```
1. User opens app (no account)
2. Register (email or whatsapp + password)
3. Verify account (OTP)
4. Login → get access token
5. Personal entity + workspace auto-created (via event system)
6. Choose plan (free/trial/paid)
7. Start using features
```

---

## Flow 1: Registration

### 1A. Register with Email + Password

**Page:** `/register`

**Request:**
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "name": "Jiilan Nashrulloh",
  "email": "jiilan@example.com",
  "password": "SecurePass123"
}
```

**Success Response (201):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Jiilan Nashrulloh",
      "email": "jiilan@example.com",
      "email_verified": false,
      "whatsapp_verified": false,
      "status": "UNVERIFIED",
      "created_at": "2026-05-20T10:00:00Z"
    },
    "message": "Registration successful. Please verify your account."
  }
}
```

**Frontend Action:**
- Show success message
- Redirect to email verification page (`/verify-email`)
- Tell user to check inbox for OTP

---

### 1B. Register with WhatsApp + Password

**Page:** `/register`

**Request:**
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "name": "Jiilan Nashrulloh",
  "whatsapp": "+628123456789",
  "password": "SecurePass123"
}
```

**Success Response (201):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Jiilan Nashrulloh",
      "whatsapp": "+628123456789",
      "email_verified": false,
      "whatsapp_verified": false,
      "status": "UNVERIFIED",
      "created_at": "2026-05-20T10:00:00Z"
    },
    "message": "Registration successful. Please verify your account."
  }
}
```

**Frontend Action:**
- Show success message
- Redirect to WhatsApp verification page (`/verify-whatsapp`)
- Tell user to check WhatsApp for OTP

---

### 1C. Register with Both Email + WhatsApp

**Request:**
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "name": "Jiilan Nashrulloh",
  "email": "jiilan@example.com",
  "whatsapp": "+628123456789",
  "password": "SecurePass123"
}
```

**Frontend Action:**
- Both OTPs sent (email + WhatsApp)
- User can verify either one to activate account
- Redirect to verification page

---

### Registration Validation Errors (400)

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "validation failed",
    "domain": "auth",
    "details": [
      {"field": "email", "message": "invalid email format"},
      {"field": "password", "message": "password must be at least 8 characters"}
    ]
  }
}
```

---

## Flow 2: Email Verification

**Page:** `/verify-email`

**Step 1:** User receives OTP via email (6-digit code)

**Step 2:** User enters OTP

**Request:**
```http
POST /api/v1/auth/verify
Content-Type: application/json

{
  "identifier": "jiilan@example.com",
  "otp": "123456",
  "purpose": "verify_email"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "message": "Verification successful"
  }
}
```

**Frontend Action:**
- Show success message "Account verified!"
- Redirect to login page (`/login`)

**Error Cases:**
- Invalid OTP → `"invalid OTP"` (user can retry, max 3 attempts)
- Expired OTP → `"invalid or expired OTP"` (user needs to re-register or request new OTP)
- Too many attempts → `"too many failed attempts"`

---

## Flow 3: WhatsApp Verification

**Page:** `/verify-whatsapp`

Same as email verification but with WhatsApp number:

**Request:**
```http
POST /api/v1/auth/verify
Content-Type: application/json

{
  "identifier": "+628123456789",
  "otp": "123456",
  "purpose": "verify_whatsapp"
}
```

**Note:** OTP is sent via Zenziva WhatsApp Official. Message format:
```
Kode OTP Azzet Anda 123456. Jaga kerahasiaan OTP Anda.
```

---

## Flow 4: Login

### 4A. Login with Email + Password

**Page:** `/login`

**Request:**
```http
POST /api/v1/auth/login/email
Content-Type: application/json

{
  "email": "jiilan@example.com",
  "password": "SecurePass123"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Jiilan Nashrulloh",
      "email": "jiilan@example.com",
      "email_verified": true,
      "whatsapp_verified": false,
      "status": "ACTIVE",
      "created_at": "2026-05-20T10:00:00Z"
    }
  }
}
```

**Response Headers:**
```
Set-Cookie: refresh_token=eyJ...; Path=/api/v1/auth; HttpOnly; Secure; SameSite=Strict; Max-Age=604800
```

**Frontend Action:**
- Store `access_token` in memory (NOT localStorage)
- Cookie is auto-managed by browser (HttpOnly, not accessible via JS)
- Redirect to dashboard (`/dashboard`)
- Set timer for token refresh (before 900s expires)

---

### 4B. Login with WhatsApp OTP

**Page:** `/login`

**Step 1:** Request OTP

```http
POST /api/v1/auth/otp/request
Content-Type: application/json

{
  "whatsapp": "+628123456789",
  "purpose": "login"
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "message": "OTP sent successfully"
  }
}
```

**Step 2:** Enter OTP

```http
POST /api/v1/auth/login/otp
Content-Type: application/json

{
  "whatsapp": "+628123456789",
  "otp": "123456"
}
```

**Response:** Same as email login (access_token + refresh cookie)

**Frontend Action:**
- Show OTP input field after requesting
- 5 minute countdown timer (OTP expiry)
- Max 3 attempts before OTP is invalidated

---

### 4C. Login Fallback

If WhatsApp OTP fails (Zenziva down), user can always login with password:

```
User enters WhatsApp number → "Request OTP" fails
    ↓
Show fallback: "Login with password instead"
    ↓
User enters password → POST /api/v1/auth/login/email (using whatsapp as identifier won't work)
```

**Important:** Password login only works with email. If user registered with WhatsApp only, they MUST use OTP. This is why password is required during registration — as fallback.

**Recommendation for frontend:** If user has both email + whatsapp, show both login options.

---

## Flow 5: Token Refresh

**When:** Access token is about to expire (frontend should refresh at ~80% of expires_in, e.g., at 720s for 900s token)

**Request:**
```http
POST /api/v1/auth/refresh
Cookie: refresh_token=eyJ...
```

**Note:** No body needed. Refresh token comes from HttpOnly cookie (auto-sent by browser).

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "access_token": "new-eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900,
    "user": { ... }
  }
}
```

**Response Headers:**
```
Set-Cookie: refresh_token=new-eyJ...; (rotated)
```

**Frontend Action:**
- Replace old access_token with new one
- Cookie auto-rotated by browser
- Reset refresh timer

**Error (401):**
- Refresh token expired or invalid
- Clear local state
- Redirect to login page

---

## Flow 6: First Time Setup (Post-Registration)

> After registration + verification + first login, the system has already created:
> 1. Personal entity (ORANG_PRIBADI)
> 2. Personal workspace
> 3. Free plan subscription (if a free plan exists in DB)
>
> This happens synchronously during registration — no delay, no polling needed.

**Frontend flow:**

1. After login, frontend always redirects to `/workspaces` (workspace selection page)
2. User sees all their workspaces with subscription status
3. User picks a workspace:
   - Personal workspace with free plan → go to `/dashboard`
   - Business workspace with active plan → go to `/dashboard`
   - Workspace without plan → go to `/plans`

**Request:**
```http
GET /api/v1/workspaces
Authorization: Bearer <access_token>
```

**Response (workspace ready with subscription info):**
```json
{
  "success": true,
  "data": [
    {
      "id": "rel-uuid",
      "entity_id": "entity-uuid",
      "entity_name": "Jiilan Nashrulloh",
      "entity_type": "ORANG_PRIBADI",
      "role": "PEMILIK",
      "subscription_status": "active",
      "plan_name": "Free",
      "created_at": "2026-05-20T10:00:05Z"
    }
  ]
}
```

**Frontend Action:**
- Always show workspace selection page (every login)
- User must manually click a workspace to proceed
- Store selected `entity_id` as `X-Workspace-ID` for subsequent requests
- If `subscription_status` is `active` or `trial` → navigate to `/dashboard`
- If `subscription_status` is null/expired/cancelled → navigate to `/plans`

---

## Flow 7: Create Business Workspace

**Page:** `/workspaces/new`

**Use Case:** User wants to manage a company (PT, CV, UD)

**Step 1:** Create business entity

```http
POST /api/v1/entities
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "entity_type": "BADAN_USAHA",
  "nama_utama": "PT Maju Jaya",
  "nik_npwp": "01.234.567.8-901.000",
  "nomor_wa": "+628111222333",
  "alamat_lengkap": "Jl. Sudirman No. 1, Jakarta"
}
```

**Step 2:** Create workspace from entity

```http
POST /api/v1/workspaces
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "entity_id": "<entity-id-from-step-1>"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "rel-uuid",
    "entity_id": "entity-uuid",
    "entity_name": "PT Maju Jaya",
    "entity_type": "BADAN_USAHA",
    "role": "PEMILIK",
    "created_at": "2026-05-20T10:05:00Z"
  }
}
```

**Frontend Action:**
- Add new workspace to workspace switcher
- Prompt user to subscribe to a plan for this workspace

---

## Flow 8: Subscribe to Plan

**Page:** `/plans` or `/workspace/settings/plan`

**Step 1:** Show available plans

```http
GET /api/v1/plans
```

**Response (200):** (No auth required - public pricing page)
```json
{
  "success": true,
  "data": [
    {
      "id": "plan-uuid-1",
      "name": "Free",
      "slug": "free",
      "type": "free",
      "price_monthly": 0,
      "price_yearly": 0,
      "is_trial": false,
      "tier": 0,
      "features": [
        {"feature_key": "max_entities", "feature_type": "quota", "value_int": 5},
        {"feature_key": "ocr_enabled", "feature_type": "boolean", "value_bool": false}
      ]
    },
    {
      "id": "plan-uuid-2",
      "name": "Starter",
      "slug": "starter",
      "type": "paid",
      "price_monthly": 99000,
      "price_yearly": 990000,
      "is_trial": true,
      "trial_days": 14,
      "tier": 1,
      "features": [...]
    }
  ]
}
```

**Step 2:** User selects plan

### Subscribe to Free Plan (instant):

```http
POST /api/v1/subscription
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "plan_id": "<free-plan-uuid>"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "sub-uuid",
    "workspace_id": "...",
    "plan_id": "...",
    "status": "active",
    "started_at": "2026-05-20T10:10:00Z"
  }
}
```

### Start Trial (14 days free):

```http
POST /api/v1/subscription
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "plan_id": "<starter-plan-uuid>"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "sub-uuid",
    "status": "trial",
    "trial_ends_at": "2026-06-03T10:10:00Z",
    "expires_at": "2026-06-03T10:10:00Z"
  }
}
```

### Subscribe to Paid Plan:

```http
POST /api/v1/subscription
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "plan_id": "<professional-plan-uuid>",
  "billing_cycle": "monthly"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "sub-uuid",
    "workspace_id": "...",
    "plan_id": "...",
    "status": "pending_payment",
    "billing_cycle": "monthly",
    "started_at": "2026-05-20T10:10:00Z",
    "expires_at": "2026-06-20T10:10:00Z",
    "payment_url": "https://checkout.xendit.co/web/abc123"
  }
}
```

**Frontend Action after subscription:**
- If `payment_url` is present → redirect user to Xendit checkout page
- If status is `active` (free plan) or `trial` → redirect to `/dashboard`
- After payment at Xendit:
  - Success → Xendit redirects to `/payment/success` → auto-redirect to dashboard
  - Failed → Xendit redirects to `/payment/failed` → user can retry

---

## Flow 9: Billing & Payment

**Page:** `/workspace/settings/billing`

**Use Case:** User subscribed to paid plan or trial expired

**Step 1:** View invoices

```http
GET /api/v1/billing/invoices
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

**Step 2:** Pay invoice

```http
POST /api/v1/billing/pay
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "invoice_id": "<invoice-uuid>"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "payment-uuid",
    "invoice_id": "...",
    "amount": 299000,
    "currency": "IDR",
    "status": "pending",
    "payment_url": "https://checkout.xendit.co/web/abc123",
    "expires_at": "2026-05-21T10:00:00Z"
  }
}
```

**Frontend Action:**
- Redirect user to `payment_url` (Xendit checkout page)
- Or open in new tab/iframe
- After payment, Xendit redirects to success/failure URL
- Backend receives webhook → activates subscription automatically

**After payment success:**
- User returns to app
- Subscription status = "active"
- All paid features unlocked

---

## Flow 10: Workspace Management

### Switch Workspace

**Frontend:** Workspace selection page (`/workspaces`) shown every login.
User can also switch workspace from within the app.

All workspace-scoped requests need `X-Workspace-ID` header:

```http
GET /api/v1/subscription
Authorization: Bearer <access_token>
X-Workspace-ID: <selected-workspace-entity-id>
```

### Invite Team Member (Email-based)

**Page:** `/users` (workspace members page)

**Requirements:**
- Inviter must have `member:invite` permission (Owner has this by default via wildcard)
- Invited email must be registered on the platform
- No duplicate pending invites for same email + workspace
- User must not already be a member of the workspace

**Step 1:** Send invite

```http
POST /api/v1/workspaces/invites
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "email": "andi@example.com",
  "role_id": "<workspace-role-uuid>"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "invite-uuid",
    "workspace_id": "...",
    "invited_email": "andi@example.com",
    "role_name": "Akuntan",
    "token": "a1b2c3d4...",
    "invited_by": "...",
    "expires_at": "2026-05-21T10:00:00Z",
    "created_at": "2026-05-20T10:00:00Z"
  }
}
```

**What happens:**
- Backend sends styled HTML email to `andi@example.com`
- Email contains link: `{FRONTEND_URL}/invite/{token}`
- Link expires in 24 hours

**Step 2:** Invitee accepts (must be logged in)

```http
POST /api/v1/workspaces/invites/accept
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "token": "a1b2c3d4..."
}
```

**Validation:**
- Token must be valid and not expired (24h)
- Logged-in user's email must match `invited_email`
- User must not already be a member

**Result:** Creates `KARYAWAN` relation + assigns the specified role.

**Note:** Invite only works via email. WhatsApp is reserved for OTP only.

### Manage Workspace Roles (ABAC)

**Page:** Workspace settings / roles management

**List roles:**
```http
GET /api/v1/workspaces/roles
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

**Create custom role:**
```http
POST /api/v1/workspaces/roles
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "name": "Akuntan",
  "description": "Akses laporan dan transaksi",
  "permissions": ["transaction:read", "transaction:create", "report:read", "report:export"]
}
```

**Assign role to member:**
```http
POST /api/v1/workspaces/roles/assign
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "member_entity_id": "<member-personal-entity-uuid>",
  "role_id": "<workspace-role-uuid>"
}
```

**Available permission keys:**
```
transaction:create, transaction:read, transaction:update, transaction:delete
report:read, report:export
member:invite, member:manage, member:remove
role:create, role:update, role:delete, role:assign
workspace:settings
billing:read, billing:manage
item:create, item:read, item:update, item:delete
account:create, account:read, account:update, account:delete
* (wildcard — owner only, auto-assigned)
```

### Add Counterparty (Customer/Vendor)

**Page:** `/workspace/counterparties`

```http
POST /api/v1/workspaces/counterparties
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "relation_type": "PELANGGAN",
  "nama_utama": "Toko Maju",
  "entity_type": "BADAN_USAHA",
  "custom_alias": "Toko Maju Cabang Utara"
}
```

**Note:** This creates a shadow entity automatically if `entity_id` is not provided.

---

## Flow 11: Session Management

**Page:** `/settings/sessions`

### View Active Sessions

```http
GET /api/v1/auth/sessions
Authorization: Bearer <access_token>
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "session-uuid",
      "device_name": "Chrome on MacOS",
      "ip_address": "192.168.1.1",
      "last_used_at": "2026-05-20T10:00:00Z",
      "created_at": "2026-05-19T09:00:00Z"
    }
  ]
}
```

### Revoke Session

```http
DELETE /api/v1/auth/sessions/<session-id>
Authorization: Bearer <access_token>
```

### Logout Current Session

```http
POST /api/v1/auth/logout
Authorization: Bearer <access_token>
Cookie: refresh_token=...
```

### Logout All Sessions

```http
POST /api/v1/auth/logout-all
Authorization: Bearer <access_token>
```

---

## Flow 12: Password Management

### Change Password (Authenticated)

**Page:** `/settings/security`

```http
POST /api/v1/auth/password/change
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "old_password": "OldPass123",
  "new_password": "NewSecurePass456"
}
```

### Reset Password (Forgot Password)

**Page:** `/forgot-password`

**Step 1:** Request OTP

```http
POST /api/v1/auth/otp/request
Content-Type: application/json

{
  "whatsapp": "+628123456789",
  "purpose": "reset_password"
}
```

**Step 2:** Reset with OTP

```http
POST /api/v1/auth/password/reset
Content-Type: application/json

{
  "identifier": "+628123456789",
  "otp": "123456",
  "new_password": "NewSecurePass456"
}
```

---

## State Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  [Landing Page]                                                 │
│       │                                                         │
│       ├── "Register" ──→ [Register Page]                        │
│       │                       │                                 │
│       │                       ├── Email + Password              │
│       │                       └── WhatsApp + Password           │
│       │                              │                          │
│       │                              ▼                          │
│       │                    [Verify OTP Page]                     │
│       │                              │                          │
│       │                              ▼                          │
│       │                    Account ACTIVE                       │
│       │                              │                          │
│       ├── "Login" ────→ [Login Page]  │                         │
│       │                       │       │                         │
│       │                       ▼       ▼                         │
│       │              [Dashboard / Workspace List]                │
│       │                       │                                 │
│       │                       ├── First time? → [Select Plan]   │
│       │                       │                      │          │
│       │                       │                      ▼          │
│       │                       │              [Subscribe]         │
│       │                       │                      │          │
│       │                       │         ┌────────────┼────────┐ │
│       │                       │         │ Free       │ Paid   │ │
│       │                       │         │ (instant)  │ (pay)  │ │
│       │                       │         └────────────┼────────┘ │
│       │                       │                      │          │
│       │                       ▼                      ▼          │
│       │              [Workspace Active - Use Features]          │
│       │                       │                                 │
│       │                       ├── Create Business Workspace     │
│       │                       ├── Invite Members                │
│       │                       ├── Add Counterparties            │
│       │                       ├── Manage Counterparty Aliases   │
│       │                       ├── Record Transactions (Phase 7) │
│       │                       ├── Generate Reports (Phase 7)    │
│       │                       ├── Manage Entity Identity        │
│       │                       ├── Claim Shadow Entities         │
│       │                       └── Fuzzy Search Entities         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Page Routing Logic (Frontend Guard)

> This section defines the routing/redirect logic for the frontend.
> Use this as the basis for route guards, middleware, or layout wrappers.

### Decision Tree (On Every Page Load)

```
User opens any page
    │
    ├── Has access_token in memory?
    │   ├── NO → Try silent refresh (POST /auth/refresh via HttpOnly cookie)
    │   │         ├── Success → continue with new token
    │   │         └── Fail → redirect to /login
    │   └── YES ↓
    │
    ├── Is this a public page? (/login, /register, /verify-*, /forgot-password, /payment/*)
    │   └── YES → If authenticated, redirect to /workspaces (except /payment/*)
    │
    ├── Always redirect to /workspaces (workspace selection page)
    │   User MUST manually select a workspace every login
    │
    ├── User selects workspace → check subscription_status from response
    │   ├── status = "active" or "trial" → navigate to /dashboard
    │   ├── status = null/expired/cancelled/pending_payment → navigate to /plans
    │   └── (subscription_status is included in GET /workspaces response)
    │
    ├── Inside /dashboard (_authed layout):
    │   ├── No activeWorkspace in store? → redirect to /workspaces
    │   ├── GET /subscription → check status
    │   │   ├── active/trial → allow access
    │   │   ├── expired/cancelled/pending_payment → redirect to /plans
    │   │   └── 404 (no subscription) → redirect to /plans
    │   └── If trial expiring soon (< 3 days) → show warning banner
    │
    └── ALLOW ACCESS to workspace features
```

---

### Route Categories

#### Public Pages (No Auth Required)

| Page | Path | Behavior if Authenticated |
|------|------|---------------------------|
| Index | `/` | Redirect to `/workspaces` |
| Login | `/login` | Redirect to `/workspaces` |
| Register | `/register` | Redirect to `/workspaces` |
| Verify Email | `/verify-email` | Accessible (no redirect) |
| Verify WhatsApp | `/verify-whatsapp` | Accessible (no redirect) |
| Forgot Password | `/forgot-password` | Accessible (no redirect) |
| Payment Success | `/payment/success` | Show success + auto-redirect to dashboard (5s) |
| Payment Failed | `/payment/failed` | Show failure + retry button |

#### Onboarding Pages (Auth Required, No Sidebar)

| Page | Path | Guard Logic |
|------|------|-------------|
| Setup | `/setup` | Always redirects to `/workspaces` |
| Workspace Selection | `/workspaces` | Shows all workspaces with plan status. Always shown on login. |
| Create Workspace | `/workspaces/new` | Form to create business entity + workspace |
| Plan Selection | `/plans` | Standalone page. Shown when workspace has no active subscription. |
| Accept Invite | `/invite/{token}` | Validates token, accepts invite, redirects to `/workspaces` |

#### Protected Pages (Auth + Active Workspace + Subscription, With Sidebar)

| Page | Path | Permission Required |
|------|------|---------------------|
| Dashboard | `/dashboard` | Any workspace member |
| Members | `/users` | Any (view), `member:manage` (edit), `member:remove` (delete) |
| Billing | `/billing` | `billing:read` |
| Settings | `/settings` | `workspace:settings` |
| UI Overview | `/ui-overview` | Any workspace member |

---

### Detailed Guard Logic Per Scenario

#### Scenario 1: Brand New User (Just Registered)

```
1. User registers → status = UNVERIFIED
2. Redirect to /verify-email or /verify-whatsapp
3. User verifies OTP → status = ACTIVE
4. Redirect to /login
5. User logs in → access_token received
6. Redirect to /workspaces (always)
7. Workspace list shows personal workspace with "Free" plan badge (auto-assigned)
8. User clicks personal workspace → subscription_status = "active"
9. Navigate to /dashboard → ready to use!
```

#### Scenario 2: Returning User (Has Everything Set Up)

```
1. User opens app → access_token gone (page refresh)
2. Silent refresh via HttpOnly cookie → new access_token
3. Redirect to /workspaces (always shown on login)
4. User clicks their workspace
5. subscription_status = "active" → navigate to /dashboard
```

#### Scenario 3: User with Expired Trial

```
1. User logs in → redirect to /workspaces
2. Workspace list shows workspace with subscription_status = null (expired)
3. User clicks workspace → navigate to /plans
4. Show message: "Masa uji coba telah berakhir. Pilih plan untuk melanjutkan."
5. User picks paid plan → POST /subscription → response has payment_url
6. Frontend redirects to Xendit checkout
7. User pays → Xendit redirects to /payment/success
8. Webhook activates subscription
9. User clicks "Masuk ke Dashboard" → /dashboard
```

#### Scenario 4: User with Multiple Workspaces

```
1. User logs in → redirect to /workspaces
2. Workspace list shows:
   [
     { entity_name: "Jiilan (Personal)", subscription_status: "active", plan_name: "Free" },
     { entity_name: "PT Azzet", subscription_status: "active", plan_name: "Professional" },
     { entity_name: "CV Jiilan", subscription_status: null }
   ]
3. User clicks "PT Azzet" → active → /dashboard
4. Or user clicks "CV Jiilan" → no plan → /plans
```

#### Scenario 5: User Creates New Business Workspace

```
1. User on /workspaces page
2. Clicks "Buat Workspace Baru"
3. Navigate to /workspaces/new
4. Fill form: nama_utama, nik_npwp, nomor_wa, alamat_lengkap
5. POST /entities → entity created
6. POST /workspaces { entity_id } → workspace created (Owner role bootstrapped)
7. Navigate to /plans (new workspace has no subscription)
8. User subscribes → payment flow or instant activation
9. Navigate to /dashboard
```

#### Scenario 6: User Accepts Workspace Invite

```
1. User receives email with invite link: https://app.azzet.id/invite/{token}
2. User clicks link → frontend /invite/{token} page
3. If not logged in → redirect to /login?redirect=/invite/{token}
4. After login → back to /invite/{token}
5. Frontend calls POST /workspaces/invites/accept { token }
6. Backend validates: token valid, not expired (24h), email matches, not already member
7. Creates KARYAWAN relation + assigns role
8. Show success → navigate to /workspaces
9. New workspace appears in list
```

#### Scenario 7: User's Payment Failed

```
1. User subscribed to paid plan → redirected to Xendit
2. Payment fails → Xendit redirects to /payment/failed
3. User sees "Pembayaran Gagal" page
4. Clicks "Coba Lagi" → navigate to /plans
5. Or clicks "Kembali ke Workspace" → /workspaces
```

---

### Frontend State Machine

```
┌──────────────────────────────────────────────────────────────┐
│                                                              │
│  UNAUTHENTICATED                                             │
│  ├── /login                                                  │
│  ├── /register                                               │
│  ├── /verify-email                                           │
│  ├── /verify-whatsapp                                        │
│  ├── /forgot-password                                        │
│  ├── /payment/success (public, no auth needed)               │
│  └── /payment/failed (public, no auth needed)                │
│                                                              │
│  ─── Login Success ───────────────────────────────────────── │
│                                                              │
│  AUTHENTICATED (has access_token)                            │
│  │                                                           │
│  ├── WORKSPACE SELECTION (always shown on login)             │
│  │   ├── /workspaces (picker — shows plan status per ws)     │
│  │   ├── /workspaces/new (create business workspace)         │
│  │   └── /invite/{token} (accept invite)                     │
│  │                                                           │
│  ├── PLAN SELECTION (workspace selected, no subscription)    │
│  │   └── /plans (choose plan → free/trial/paid)              │
│  │       └── Paid → redirect to Xendit → /payment/success    │
│  │                                                           │
│  ├── HAS WORKSPACE + ACTIVE SUBSCRIPTION (_authed layout)    │
│  │   ├── /dashboard                                          │
│  │   ├── /users (members)                                    │
│  │   ├── /billing                                            │
│  │   ├── /settings                                           │
│  │   └── (all business features)                             │
│  │                                                           │
│  └── SUBSCRIPTION EXPIRED/CANCELLED/PENDING                  │
│      └── /plans ← FORCED (must subscribe/pay)                │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

---

### API Calls for Guard Logic (Frontend Implementation)

```typescript
// Auth middleware (src/middleware/auth.middleware.ts)

async function tryRefresh(): Promise<boolean> {
  const { isAuthenticated, setAuth } = useAuthStore.getState()
  if (isAuthenticated) return true

  try {
    const data = await authService.refresh()
    setAuth(data.access_token, data.user)
    return true
  } catch {
    return false
  }
}

// requireAuth — for protected routes (beforeLoad)
async function requireAuth({ location }) {
  const authed = await tryRefresh()
  if (!authed) throw redirect({ to: '/login', search: { redirect: location.href } })
}

// requireGuest — for login/register (beforeLoad)
async function requireGuest() {
  const authed = await tryRefresh()
  if (authed) throw redirect({ to: '/workspaces' })
}

// _authed layout (loader) — checks workspace + subscription
async function authedLayoutLoader() {
  // 1. If no active workspace → redirect to /workspaces
  // 2. GET /subscription → check status
  //    - active/trial → allow, show trial banner if < 3 days
  //    - expired/cancelled/pending_payment/404 → redirect to /plans
}
```

---

### Storage Strategy (Frontend Reference)

| Storage | Key | Value | Purpose |
|---------|-----|-------|---------|
| In-memory (Zustand) | `accessToken` | JWT string | Auth header — never persisted |
| sessionStorage (Zustand persist) | `user` | User object | Display name, email — survives refresh within tab |
| Zustand (workspace store) | `activeWorkspace` | WorkspaceResponse | Selected workspace for X-Workspace-ID |
| HttpOnly cookie (set by backend) | `refresh_token` | JWT string | Silent refresh — frontend cannot read |

**Note:** Access token is NEVER stored in localStorage/sessionStorage. Only in JS memory via Zustand (non-persisted).

---

## Headers Reference

### Required for ALL authenticated requests:

```
Authorization: Bearer <access_token>
```

### Required for workspace-scoped requests:

```
X-Workspace-ID: <workspace-entity-id>
```

### Optional (for device tracking):

```
X-Device-Name: Chrome on MacOS
```

### Endpoints that DON'T need auth:

```
GET  /api/v1/plans
GET  /api/v1/plans/{slug}
GET  /api/v1/health
POST /api/v1/auth/register
POST /api/v1/auth/login/email
POST /api/v1/auth/login/otp
POST /api/v1/auth/otp/request
POST /api/v1/auth/refresh
POST /api/v1/auth/verify
POST /api/v1/auth/password/reset
POST /api/v1/webhooks/xendit
```

### Endpoints that need X-Workspace-ID:

```
/api/v1/workspaces/members/*
/api/v1/workspaces/roles/*
/api/v1/workspaces/invites (POST, GET, DELETE — not /accept)
/api/v1/workspaces/counterparties/*
/api/v1/workspaces/counterparties/aliases
/api/v1/workspaces/counterparties/search
/api/v1/subscription/*
/api/v1/billing/*
```

### Endpoints that need auth but NOT X-Workspace-ID:

```
GET  /api/v1/workspaces
POST /api/v1/workspaces
POST /api/v1/workspaces/invites/accept
GET  /api/v1/entities
POST /api/v1/entities
GET  /api/v1/entities/search?q=...
GET  /api/v1/entities/{id}/verification
POST /api/v1/entities/{id}/legal-ids
GET  /api/v1/entities/{id}/legal-ids
PATCH /api/v1/entities/{id}/legal-ids/{type}
DELETE /api/v1/entities/{id}/legal-ids/{type}
POST /api/v1/entities/{id}/aliases
GET  /api/v1/entities/{id}/aliases
DELETE /api/v1/entities/{id}/aliases/{alias_id}
GET  /api/v1/entities/{id}/duplicates
POST /api/v1/claims
GET  /api/v1/claims
GET  /api/v1/claims/{id}
POST /api/v1/claims/{id}/submit
POST /api/v1/claims/{id}/documents
POST /api/v1/claims/{id}/documents/{doc_id}/confirm
GET  /api/v1/claims/{id}/documents
POST /api/v1/claims/{id}/dispute
GET  /api/v1/auth/me
POST /api/v1/auth/logout
GET  /api/v1/auth/sessions
```

---

## Error Handling

### Standard Error Format

```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "invalid credentials",
    "domain": "auth",
    "request_id": "archlinux/abc123",
    "timestamp": "2026-05-20T10:00:00Z"
  }
}
```

### Common Error Codes

| Code | HTTP Status | Meaning |
|------|-------------|---------|
| `BAD_REQUEST` | 400 | Invalid input |
| `VALIDATION_ERROR` | 400 | Field validation failed (has `details` array) |
| `UNAUTHORIZED` | 401 | Not authenticated or token invalid |
| `FORBIDDEN` | 403 | Authenticated but no permission |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource already exists |
| `INTERNAL_ERROR` | 500 | Server error |

### Token Expiry Handling

```
Access token expires (900s / 15 min)
    ↓
Frontend detects 401 response
    ↓
Call POST /api/v1/auth/refresh (cookie auto-sent)
    ↓
    ├── Success → retry original request with new token
    └── Failure (401) → redirect to login page
```

### Recommended Frontend Token Strategy

```typescript
// Access token stored in Zustand (in-memory only, not persisted)
// On 401 response → ky afterResponse hook auto-refreshes:

// src/lib/api/client.ts
afterResponse: [
  async ({ request, response, retryCount }) => {
    if (response.status === 401 && retryCount === 0) {
      try {
        const newToken = await doRefresh() // POST /auth/refresh (cookie auto-sent)
        const headers = new Headers(request.headers)
        headers.set('Authorization', `Bearer ${newToken}`)
        return ky(new Request(request, { headers })) // retry original request
      } catch {
        return response // refresh failed, let error propagate
      }
    }
    return response
  }
]
```

---

## Important Notes for Frontend

1. **Refresh token is HttpOnly cookie** (`SameSite=Lax`, `Secure=true`) — Frontend cannot read it. Browser sends it automatically with requests to `/api/v1/auth/*` path.

2. **Access token in memory only** — Never stored in localStorage/sessionStorage (XSS risk). Stored in Zustand state (non-persisted). Lost on page refresh → silent refresh via cookie.

3. **X-Workspace-ID is mandatory** for business endpoints. Frontend auto-sets this from `activeWorkspace.entity_id` in the ky `beforeRequest` hook.

4. **Personal entity + workspace + free plan are created instantly** during registration (synchronous). Workspace is always ready when user first logs in.

5. **Workspace selection page is always shown** on every login. User must manually pick a workspace.

6. **Password is always required** during registration (even for WhatsApp users) as fallback when OTP service is down.

7. **OTP expires in 5 minutes**, max 3 attempts. After that, user needs to request a new one.

8. **Invite flow is email-only.** WhatsApp is reserved for OTP. Invited user must already have an Azzet account. Invite expires in 24 hours.

9. **ABAC permissions** — Roles are per-workspace custom roles. Owner always has wildcard `["*"]`. Permission check happens via `RequirePermission` middleware on backend.

10. **Payment pages (`/payment/success`, `/payment/failed`) are public** — no auth required. This is intentional because cross-site redirect from Xendit may not carry cookies.

---

## Flow 13: Accounting (Phase 7)

> **Prerequisite:** User sudah punya workspace aktif dengan subscription aktif.
> COA (Chart of Accounts) otomatis ter-seed saat workspace dibuat.

### Overview: Dari Setup Sampai Laporan

```
Owner creates workspace
       │
       ▼ (auto via workspace.created event)
COA ter-seed (55+ akun SAK EMKM/ETAP)
       │
       ▼
Owner bootstrap roles + assign permissions
       │
       ▼
Owner invite team members (Kasir, Akuntan, dll)
       │
       ▼
Team mulai kerja:
  ├── Kasir: buat transaksi (SIMPLE mode)
  ├── Akuntan: buat transaksi (ADVANCED mode) + lihat laporan
  └── Owner: semua akses + void transaksi
       │
       ▼
Ledger Worker posting otomatis (async)
       │
       ▼
Laporan keuangan tersedia (Neraca, Laba Rugi, dll)
```

---

### 13A. Owner Bootstrap: Setup Roles & Permissions

**Page:** `/workspace/settings/roles`

**Context:** Setelah workspace dibuat, Owner perlu setup roles untuk tim.

**Step 1:** Lihat roles yang ada (Owner role sudah auto-created)

```http
GET /api/v1/workspaces/roles
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "role-uuid-owner",
      "name": "Owner",
      "description": "Full access",
      "permissions": ["*"],
      "is_system": true,
      "created_at": "2026-05-20T10:00:00Z"
    }
  ]
}
```

**Step 2:** Owner buat role "Kasir"

```http
POST /api/v1/workspaces/roles
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "name": "Kasir",
  "description": "Bisa buat dan lihat transaksi, tidak bisa void atau lihat laporan",
  "permissions": [
    "transaction:create",
    "transaction:read"
  ]
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "role-uuid-kasir",
    "name": "Kasir",
    "permissions": ["transaction:create", "transaction:read"],
    "is_system": false,
    "created_at": "2026-05-20T10:05:00Z"
  }
}
```

**Step 3:** Owner buat role "Akuntan"

```http
POST /api/v1/workspaces/roles
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "name": "Akuntan",
  "description": "Full akses transaksi + laporan, bisa void",
  "permissions": [
    "transaction:create",
    "transaction:read",
    "transaction:void",
    "report:read",
    "report:export"
  ]
}
```

**Step 4:** Owner invite Kasir via email

```http
POST /api/v1/workspaces/invites
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "email": "kasir@example.com",
  "role_id": "role-uuid-kasir"
}
```

**Result:** Email terkirim ke kasir@example.com dengan link invite (expire 24 jam).

**Step 5:** Kasir accept invite (harus sudah login)

```http
POST /api/v1/workspaces/invites/accept
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "token": "invite-token-from-email"
}
```

**Result:** Kasir sekarang punya akses ke workspace dengan role "Kasir".

---

### 13B. Lihat Chart of Accounts (COA)

**Page:** `/workspace/accounts`

**Permission required:** `transaction:read`

```http
GET /api/v1/accounts
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "acc-uuid-1",
      "code": "1-0000",
      "name": "Aset",
      "account_type": "ASSET",
      "normal_balance": "DEBIT",
      "level": 1,
      "is_system": true,
      "is_active": true
    },
    {
      "id": "acc-uuid-2",
      "code": "1-1001",
      "name": "Kas",
      "account_type": "ASSET",
      "normal_balance": "DEBIT",
      "level": 3,
      "parent_id": "acc-uuid-parent",
      "is_system": true,
      "is_active": true
    }
  ]
}
```

**Filter by type:**
```http
GET /api/v1/accounts?type=EXPENSE
```

**Owner bisa tambah akun custom:**
```http
POST /api/v1/accounts
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "code": "5-1013",
  "name": "Beban Parkir",
  "account_type": "EXPENSE",
  "parent_id": "acc-uuid-beban-operasional"
}
```

**Note:** System accounts (is_system=true) tidak bisa diedit/dihapus.

---

### 13C. Manage Items (Produk/Jasa)

**Page:** `/workspace/items`

**Permission required:** `transaction:create` (untuk buat/edit), `transaction:read` (untuk lihat)

**Buat item baru:**
```http
POST /api/v1/items
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "name": "Nasi Goreng Spesial",
  "item_type": "BARANG",
  "unit": "Pcs",
  "unit_price": 25000,
  "description": "Nasi goreng dengan telur dan ayam"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "item-uuid",
    "name": "Nasi Goreng Spesial",
    "item_type": "BARANG",
    "unit": "Pcs",
    "unit_price": "25000.00",
    "description": "Nasi goreng dengan telur dan ayam",
    "is_active": true,
    "created_at": "2026-05-20T10:10:00Z"
  }
}
```

**Item types:** `BARANG`, `JASA`, `PROYEK`, `AHSP_RAKITAN`

**Units:** `Pcs`, `Kg`, `Liter`, `Meter`, `M2`, `M3`, `Jam`, `Hari`, `Paket`, `Unit`, `Box`, `Lusin`, `Set`, `Rim`

**List items:**
```http
GET /api/v1/items?limit=50&offset=0
GET /api/v1/items?type=JASA
```

**Soft-delete item:**
```http
DELETE /api/v1/items/<item-id>
```

---

### 13D. Buat Transaksi — SIMPLE Mode (Kasir-friendly)

**Page:** `/workspace/transactions/new`

**Permission required:** `transaction:create`

**Scenario:** Kasir terima uang Rp 100.000 dari Pak Budi untuk jual nasi goreng.

**Step 1 (Optional):** Minta AI suggestion dulu

```http
POST /api/v1/transactions/categorize
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "transaction_type": "CASH_IN",
  "description": "Terima uang dari Pak Budi untuk nasi goreng",
  "amount": 100000
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "category": "pendapatan_usaha",
    "confidence": 0.94,
    "used_fallback": false
  }
}
```

**Frontend:** Tampilkan "Suggested: Pendapatan Usaha (94%)" — user bisa confirm atau override.

**Step 2:** Buat transaksi

```http
POST /api/v1/transactions
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "transaction_type": "CASH_IN",
  "input_mode": "SIMPLE",
  "description": "Terima uang dari Pak Budi untuk nasi goreng",
  "transaction_date": "2026-05-20",
  "amount": 100000,
  "category": "pendapatan_usaha",
  "counterparty_name": "Pak Budi"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "tx-uuid",
    "workspace_id": "ws-uuid",
    "transaction_number": "TXN-000001",
    "transaction_type": "CASH_IN",
    "input_mode": "SIMPLE",
    "status": "DRAFT",
    "description": "Terima uang dari Pak Budi untuk nasi goreng",
    "transaction_date": "2026-05-20",
    "amount": "100000.00",
    "currency": "IDR",
    "category": "pendapatan_usaha",
    "ai_confidence": 0.94,
    "counterparty_name": "Pak Budi",
    "created_by": "user-uuid",
    "created_at": "2026-05-20T10:15:00Z"
  }
}
```

**Apa yang terjadi di balik layar:**
1. Rule engine lookup: `CASH_IN` + `pendapatan_usaha` → Debit 1-1001 (Kas), Credit 4-1001 (Pendapatan Usaha)
2. Journal entries auto-created:
   - Debit: Kas Rp 100.000
   - Credit: Pendapatan Usaha Rp 100.000
3. Status = DRAFT
4. Event `accounting.transaction.created` emitted ke NATS
5. Ledger Worker (async) akan posting → status jadi POSTED

**Setelah beberapa detik (async posting):**
```
Status: DRAFT → POSTED
posted_at: "2026-05-20T10:15:02Z"
```

---

### 13E. Buat Transaksi — CASH_OUT (Bayar Beban)

**Scenario:** Bayar listrik Rp 500.000

```http
POST /api/v1/transactions
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "transaction_type": "CASH_OUT",
  "input_mode": "SIMPLE",
  "description": "Bayar listrik bulan Mei",
  "transaction_date": "2026-05-20",
  "amount": 500000,
  "category": "beban_listrik",
  "counterparty_name": "PLN"
}
```

**Di balik layar:**
- Debit: 5-1003 (Beban Listrik & Air) Rp 500.000
- Credit: 1-1001 (Kas) Rp 500.000

---

### 13F. Buat Transaksi — SALES dengan Multi-Item + PPN

**Scenario:** Jual 5 Nasi Goreng @25.000 + 3 Es Teh @8.000 ke PT Maju, kredit (piutang), include PPN 11%.

```http
POST /api/v1/transactions
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "transaction_type": "SALES",
  "input_mode": "SIMPLE",
  "description": "Penjualan catering ke PT Maju",
  "transaction_date": "2026-05-20",
  "amount": 149000,
  "category": "penjualan_barang_kredit",
  "payment_method": "KREDIT",
  "includes_tax": true,
  "counterparty_name": "PT Maju",
  "line_items": [
    {
      "item_id": "item-uuid-nasi-goreng",
      "description": "Nasi Goreng Spesial",
      "quantity": 5,
      "unit": "Pcs",
      "unit_price": 25000
    },
    {
      "item_id": "item-uuid-es-teh",
      "description": "Es Teh Manis",
      "quantity": 3,
      "unit": "Pcs",
      "unit_price": 8000
    }
  ]
}
```

**Di balik layar (auto-generated journal):**
```
Debit:  1-1003 Piutang Usaha      Rp 134.234 (base amount = 149000/1.11)
Credit: 4-1001 Pendapatan Usaha   Rp 134.234
Debit:  5-3001 HPP                Rp 134.234
Credit: 1-1004 Persediaan         Rp 134.234
Debit:  1-1003 Piutang Usaha      Rp  14.766 (PPN 11%)
Credit: 2-1005 PPN Keluaran       Rp  14.766
```

---

### 13G. Buat Transaksi — ADVANCED Mode (Akuntan)

**Scenario:** Akuntan mau buat jurnal manual (penyesuaian akhir bulan).

**Permission required:** `transaction:create`

```http
POST /api/v1/transactions
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "transaction_type": "JOURNAL",
  "input_mode": "ADVANCED",
  "description": "Penyesuaian beban penyusutan peralatan bulan Mei",
  "transaction_date": "2026-05-31",
  "amount": 500000,
  "journal_entries": [
    {
      "account_code": "5-1008",
      "description": "Beban Penyusutan",
      "debit": 500000,
      "credit": 0
    },
    {
      "account_code": "1-2099",
      "description": "Akumulasi Penyusutan",
      "debit": 0,
      "credit": 500000
    }
  ]
}
```

**Validasi backend:**
- sum(debit) HARUS = sum(credit) → 500.000 = 500.000 PASS
- Semua account_code harus ada di COA workspace ini
- Kalau tidak balance → 400 "total debit must equal total credit"

---

### 13H. Lihat Detail Transaksi

**Page:** `/workspace/transactions/<id>`

**Permission required:** `transaction:read`

```http
GET /api/v1/transactions/<tx-uuid>
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "tx-uuid",
    "transaction_number": "TXN-000001",
    "transaction_type": "CASH_IN",
    "input_mode": "SIMPLE",
    "status": "POSTED",
    "description": "Terima uang dari Pak Budi untuk nasi goreng",
    "transaction_date": "2026-05-20",
    "amount": "100000.00",
    "currency": "IDR",
    "category": "pendapatan_usaha",
    "ai_confidence": 0.94,
    "counterparty_name": "Pak Budi",
    "posted_at": "2026-05-20T10:15:02Z",
    "journal_entries": [
      {
        "id": "je-uuid-1",
        "account_id": "acc-uuid-kas",
        "account_code": "1-1001",
        "account_name": "Kas",
        "debit": "100000.00",
        "credit": "0.00",
        "sort_order": 0
      },
      {
        "id": "je-uuid-2",
        "account_id": "acc-uuid-pendapatan",
        "account_code": "4-1001",
        "account_name": "Pendapatan Usaha",
        "debit": "0.00",
        "credit": "100000.00",
        "sort_order": 1
      }
    ],
    "line_items": []
  }
}
```

---

### 13I. List Transaksi

**Page:** `/workspace/transactions`

```http
GET /api/v1/transactions?limit=50&offset=0
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

---

### 13J. Void Transaksi (Jurnal Pembalik)

**Page:** `/workspace/transactions/<id>` → tombol "Void"

**Permission required:** `transaction:void` (Owner dan Akuntan punya ini, Kasir tidak)

**Scenario:** Transaksi TXN-000001 salah, perlu di-void.

```http
POST /api/v1/transactions/<tx-uuid>/void
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

**Validasi:**
- Transaksi HARUS berstatus `POSTED` (tidak bisa void DRAFT atau yang sudah VOID)
- User HARUS punya permission `transaction:void`

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "reversal-tx-uuid",
    "transaction_number": "TXN-000002",
    "transaction_type": "REVERSAL",
    "status": "DRAFT",
    "description": "Jurnal Pembalik: TXN-000001",
    "amount": "100000.00",
    "reversed_transaction_id": "tx-uuid-original"
  }
}
```

**Apa yang terjadi:**
1. Transaksi asli (TXN-000001) di-mark `VOID`
2. Transaksi baru (TXN-000002) dibuat dengan type `REVERSAL`
3. Journal entries di-swap (debit↔credit):
   - Debit: 4-1001 Pendapatan Usaha Rp 100.000
   - Credit: 1-1001 Kas Rp 100.000
4. Reversal di-posting async → net effect = 0
5. Kedua transaksi tetap ada di audit trail selamanya

**Kasir coba void → DITOLAK:**
```json
{
  "success": false,
  "error": {
    "code": "FORBIDDEN",
    "message": "insufficient permissions",
    "domain": "workspace"
  }
}
```

---

### 13K. Laporan Keuangan

**Page:** `/workspace/reports`

**Permission required:** `report:read`

#### Trial Balance (Neraca Saldo)

```http
GET /api/v1/reports/trial-balance?period_from=2026-01&period_to=2026-05
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "account_id": "acc-uuid",
      "code": "1-1001",
      "name": "Kas",
      "account_type": "ASSET",
      "normal_balance": "DEBIT",
      "total_debit": "5000000.00",
      "total_credit": "2000000.00",
      "balance": "3000000.00"
    },
    {
      "code": "4-1001",
      "name": "Pendapatan Usaha",
      "account_type": "REVENUE",
      "normal_balance": "CREDIT",
      "total_debit": "0.00",
      "total_credit": "5000000.00",
      "balance": "-5000000.00"
    }
  ]
}
```

#### Balance Sheet (Neraca)

```http
GET /api/v1/reports/balance-sheet?period=2026-05
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "assets": [
      {"code": "1-1001", "name": "Kas", "balance": "3000000.00"},
      {"code": "1-1003", "name": "Piutang Usaha", "balance": "500000.00"}
    ],
    "liabilities": [
      {"code": "2-1001", "name": "Hutang Usaha", "balance": "-200000.00"}
    ],
    "equity": [
      {"code": "3-1001", "name": "Modal Pemilik", "balance": "-1000000.00"}
    ],
    "total_assets": "3500000.00",
    "total_liabilities": "200000.00",
    "total_equity": "3300000.00",
    "is_balanced": true
  }
}
```

#### Income Statement (Laba Rugi)

```http
GET /api/v1/reports/income-statement?period_from=2026-05&period_to=2026-05
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "revenue": [
      {"code": "4-1001", "name": "Pendapatan Usaha", "balance": "5000000.00"},
      {"code": "4-1002", "name": "Pendapatan Jasa", "balance": "1000000.00"}
    ],
    "expenses": [
      {"code": "5-1001", "name": "Beban Gaji & Tunjangan", "balance": "2000000.00"},
      {"code": "5-1003", "name": "Beban Listrik & Air", "balance": "500000.00"}
    ],
    "total_revenue": "6000000.00",
    "total_expenses": "2500000.00",
    "net_income": "3500000.00"
  }
}
```

#### Cash Flow (Arus Kas)

```http
GET /api/v1/reports/cash-flow?date_from=2026-05-01&date_to=2026-05-31
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {"date": "2026-05-01", "total_debit": "1000000.00", "total_credit": "200000.00", "net_flow": "800000.00"},
    {"date": "2026-05-02", "total_debit": "500000.00", "total_credit": "1500000.00", "net_flow": "-1000000.00"}
  ]
}
```

#### General Ledger / Buku Besar (per akun)

```http
GET /api/v1/reports/ledger/<account-uuid>?limit=50&offset=0
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "ledger-uuid",
      "transaction_number": "TXN-000001",
      "tx_description": "Terima uang dari Pak Budi",
      "transaction_date": "2026-05-20",
      "debit": "100000.00",
      "credit": "0.00",
      "running_balance": "100000.00",
      "posted_at": "2026-05-20T10:15:02Z"
    },
    {
      "id": "ledger-uuid-2",
      "transaction_number": "TXN-000003",
      "tx_description": "Bayar listrik",
      "transaction_date": "2026-05-20",
      "debit": "0.00",
      "credit": "500000.00",
      "running_balance": "-400000.00",
      "posted_at": "2026-05-20T11:00:00Z"
    }
  ]
}
```

---

### 13L. Permission Matrix (Siapa Bisa Apa)

| Action | Owner | Akuntan | Kasir |
|--------|-------|---------|-------|
| Lihat COA | Yes | Yes | Yes |
| Tambah akun custom | Yes | Yes | No |
| Buat item | Yes | Yes | Yes |
| Buat transaksi (SIMPLE) | Yes | Yes | Yes |
| Buat transaksi (ADVANCED/JOURNAL) | Yes | Yes | Yes |
| Lihat transaksi | Yes | Yes | Yes |
| Void transaksi | Yes | Yes | **No** |
| Lihat laporan | Yes | Yes | **No** |
| Export laporan | Yes | Yes | **No** |
| Invite member | Yes | No | No |
| Manage roles | Yes | No | No |

**Permission keys yang dipakai:**
```
transaction:create  → buat/edit transaksi + items + akun custom
transaction:read    → lihat transaksi + items + COA
transaction:void    → void transaksi (jurnal pembalik)
report:read         → lihat semua laporan keuangan
report:export       → export laporan (future)
member:invite       → invite member baru
role:create/update/delete/assign → manage roles
```

---

### 13M. Transaction Status Lifecycle

```
                    ┌─────────────────────────────────────┐
                    │                                     │
User creates  ──►  DRAFT  ──► (Ledger Worker) ──►  POSTED
                    │                                     │
                    │ (validation fail)                   │ (void request)
                    ▼                                     ▼
                  FAILED                                VOID
                                                         │
                                                         ▼
                                              New REVERSAL transaction
                                              (DRAFT → POSTED)
```

- **DRAFT:** Baru dibuat, belum di-posting ke ledger
- **POSTED:** Sudah di-posting, mempengaruhi saldo akun
- **VOID:** Di-void via jurnal pembalik, efek net = 0
- **FAILED:** Gagal posting (debit != credit, akun tidak ditemukan, dll)

---

### 13N. AI Categorization — Kapan Dipanggil

| Kondisi | AI Dipanggil? | Alasan |
|---------|---------------|--------|
| SIMPLE mode, category kosong | **Ya** | AI suggest category |
| SIMPLE mode, category diisi user | Tidak | User sudah pilih sendiri |
| ADVANCED mode | Tidak | User manual input journal |
| JOURNAL type | Tidak | Tidak butuh category |
| Endpoint `/categorize` | **Ya** | User minta suggestion eksplisit |

**Security:** AI hanya terima description + amount + type. Tidak ada data tenant, financials, atau info sensitif yang dikirim ke OpenAI.

---

---

## Flow 14: Entity Identity & Verification

> **Prerequisite:** User sudah punya entity (personal atau business).
> Fitur ini memungkinkan user mengelola identitas legal entity: verifikasi status, legal IDs, aliases, dan pencarian duplikat.

### Overview

```
Entity punya:
  ├── Verification status (UNVERIFIED → PENDING → VERIFIED/REJECTED)
  ├── Legal IDs (NPWP, NIB, SIUP, KTP, AKTA)
  ├── Aliases (nama alternatif, dari berbagai source)
  └── Normalized name (auto-generated, untuk fuzzy search)
```

---

### 14A. Lihat Status Verifikasi

**Page:** `/entities/{id}/verification`

```http
GET /api/v1/entities/{entity-id}/verification
Authorization: Bearer <access_token>
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "entity_id": "entity-uuid",
    "status": "UNVERIFIED"
  }
}
```

**Status values:**
| Status | Arti |
|--------|------|
| `UNVERIFIED` | Belum diverifikasi (default) |
| `PENDING` | Sedang menunggu review admin |
| `VERIFIED` | Sudah terverifikasi |
| `REJECTED` | Ditolak (ada rejection_reason) |

**Response jika sudah verified:**
```json
{
  "success": true,
  "data": {
    "entity_id": "entity-uuid",
    "status": "VERIFIED",
    "verified_by": "admin-uuid",
    "verified_at": "2026-05-27T10:00:00Z"
  }
}
```

**Response jika rejected:**
```json
{
  "success": true,
  "data": {
    "entity_id": "entity-uuid",
    "status": "REJECTED",
    "rejection_reason": "Dokumen NPWP tidak valid",
    "notes": "Silakan upload ulang dengan foto yang lebih jelas"
  }
}
```

---

### 14B. Manage Legal IDs

Legal ID adalah identitas resmi entity (NPWP, NIB, dll). Bisa punya lebih dari satu tipe.

**Page:** `/entities/{id}/identity`

#### Tambah Legal ID

```http
POST /api/v1/entities/{entity-id}/legal-ids
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "id_type": "NPWP",
  "id_value": "01.234.567.8-901.000"
}
```

**Valid tipe:** `NPWP`, `NIB`, `SIUP`, `KTP`, `AKTA`

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "legal-id-uuid",
    "entity_id": "entity-uuid",
    "id_type": "NPWP",
    "id_value": "01.234.567.8-901.000",
    "is_verified": false,
    "created_at": "2026-05-27T10:00:00Z"
  }
}
```

#### List Legal IDs

```http
GET /api/v1/entities/{entity-id}/legal-ids
Authorization: Bearer <access_token>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "legal-id-uuid",
      "entity_id": "entity-uuid",
      "id_type": "NPWP",
      "id_value": "01.234.567.8-901.000",
      "is_verified": false,
      "created_at": "2026-05-27T10:00:00Z"
    }
  ]
}
```

#### Update Legal ID

```http
PATCH /api/v1/entities/{entity-id}/legal-ids/NPWP
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "id_value": "02.345.678.9-012.000"
}
```

#### Hapus Legal ID

```http
DELETE /api/v1/entities/{entity-id}/legal-ids/NPWP
Authorization: Bearer <access_token>
```

---

### 14C. Manage Aliases

Aliases adalah nama alternatif untuk entity. Berguna ketika entity dikenal dengan nama berbeda di konteks berbeda.

**Source values:** `MANUAL` (user input), `CLAIM` (dari claim workflow), `COUNTERPARTY` (dari counterparty mapping), `SYSTEM` (auto-generated)

#### Tambah Alias

```http
POST /api/v1/entities/{entity-id}/aliases
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "alias": "PT Maju Jaya Sejahtera",
  "source": "MANUAL"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "alias-uuid",
    "entity_id": "entity-uuid",
    "alias": "PT Maju Jaya Sejahtera",
    "source": "MANUAL",
    "created_at": "2026-05-27T10:00:00Z"
  }
}
```

#### List Aliases

```http
GET /api/v1/entities/{entity-id}/aliases
Authorization: Bearer <access_token>
```

#### Hapus Alias

```http
DELETE /api/v1/entities/{entity-id}/aliases/{alias-id}
Authorization: Bearer <access_token>
```

---

### 14D. Fuzzy Search Entities

Pencarian entity berdasarkan kesamaan nama (trigram similarity). Berguna untuk menemukan duplikat atau mencari entity yang mirip.

**Page:** `/entities/search` atau komponen search di mana saja

```http
GET /api/v1/entities/search?q=PT%20Maju&limit=20&offset=0
Authorization: Bearer <access_token>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "entity-uuid-1",
      "nama_utama": "PT Maju Jaya",
      "entity_type": "BADAN_USAHA",
      "is_shadow": false,
      "match_score": 0.85
    },
    {
      "id": "entity-uuid-2",
      "nama_utama": "PT Maju Makmur",
      "entity_type": "BADAN_USAHA",
      "is_shadow": true,
      "match_score": 0.72
    }
  ]
}
```

**Match score:** 0.0 – 1.0 (semakin tinggi semakin mirip)

**Frontend notes:**
- Gunakan debounce 300ms sebelum panggil API
- Tampilkan `match_score` sebagai badge (e.g., "85% match")
- Shadow entities (`is_shadow: true`) ditandai berbeda di UI

---

### 14E. Cari Duplikat Entity

Khusus untuk satu entity — cari entity lain yang mirip namanya.

```http
GET /api/v1/entities/{entity-id}/duplicates?limit=10
Authorization: Bearer <access_token>
```

**Response:** Sama seperti fuzzy search, tapi exclude entity itu sendiri.

**Use case:** Sebelum merge entity, user bisa cek apakah ada duplikat.

---

## Flow 15: Company Claim Workflow

> **Prerequisite:** User sudah login. Entity yang di-claim harus shadow entity (is_shadow=true).
> Claim workflow memungkinkan user meng-claim shadow entity sebagai miliknya, dengan upload dokumen bukti dan review admin.

### Claim State Machine

```
                ┌─────────────────────────────────────────────────┐
                │                                                 │
User creates ──►  DRAFT  ──► (user submits) ──►  SUBMITTED        │
                │                                                 │
                │                                    │            │
                │                     ┌──────────────┼───────┐    │
                │                     ▼              ▼       │    │
                │            UNDER_REVIEW        DISPUTED    │    │
                │                     │                       │    │
                │              ┌──────┴──────┐               │    │
                │              ▼             ▼               │    │
                │          APPROVED      REJECTED            │    │
                │              │                             │    │
                │              ▼                             │    │
                │    Shadow entity linked                    │    │
                │    (user_id set, is_shadow=false)          │    │
                └─────────────────────────────────────────────────┘
```

| Status | Arti |
|--------|------|
| `DRAFT` | Baru dibuat, belum submit. User bisa edit/submit |
| `SUBMITTED` | Sudah di-submit, menunggu review admin |
| `UNDER_REVIEW` | Admin sedang review (otomatis jika duplikat terdeteksi) |
| `APPROVED` | Disetujui, shadow entity linked ke user |
| `REJECTED` | Ditolak oleh admin |
| `DISPUTED` | User menolak keputusan reject, minta review ulang |

---

### 15A. Buat Claim

**Page:** `/entities/{id}/claim` (hanya muncul di shadow entities)

**Step 1:** User melihat shadow entity dan ingin meng-claimnya

```http
POST /api/v1/claims
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "entity_id": "shadow-entity-uuid",
  "claim_type": "OWNER",
  "notes": "Ini adalah perusahaan saya"
}
```

**claim_type values:**
- `OWNER` — User adalah pemilik entity
- `AUTHORIZED_REP` — User adalah perwakilan resmi

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "claim-uuid",
    "entity_id": "shadow-entity-uuid",
    "claim_type": "OWNER",
    "status": "DRAFT",
    "notes": "Ini adalah perusahaan saya",
    "created_at": "2026-05-27T10:00:00Z"
  }
}
```

---

### 15B. Upload Dokumen Bukti

User upload dokumen pendukung (NPWP, NIB, AKTA, dll) via presigned URL ke Cloudflare R2.

**Step 1:** Request presigned upload URL

```http
POST /api/v1/claims/{claim-id}/documents
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "doc_type": "NPWP",
  "filename": "npwp-scan.pdf",
  "mime_type": "application/pdf"
}
```

**doc_type values:** `NPWP`, `NIB`, `SIUP`, `AKTA`, `KTP`, `OTHER`

**Response (201):**
```json
{
  "success": true,
  "data": {
    "document_id": "doc-uuid",
    "upload_url": "https://azzet-storage.r2.cloudflarestorage.com/claims/claim-uuid/doc-uuid/npwp-scan.pdf?X-Amz-...",
    "expires_at": "2026-05-27T10:15:00Z"
  }
}
```

**Step 2:** Frontend upload file langsung ke R2 via presigned URL

```typescript
// Frontend upload ke R2
await fetch(uploadUrl, {
  method: 'PUT',
  body: file,
  headers: { 'Content-Type': 'application/pdf' }
})
```

**Step 3:** Konfirmasi upload berhasil

```http
POST /api/v1/claims/{claim-id}/documents/{doc-id}/confirm
Authorization: Bearer <access_token>
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "document_id": "doc-uuid",
    "status": "UPLOADED"
  }
}
```

---

### 15C. Submit Claim

Setelah semua dokumen di-upload, user submit claim untuk review admin.

```http
POST /api/v1/claims/{claim-id}/submit
Authorization: Bearer <access_token>
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "claim-uuid",
    "status": "SUBMITTED",
    "updated_at": "2026-05-27T10:05:00Z"
  }
}
```

**What happens behind the scenes:**
1. Status berubah DRAFT → SUBMITTED
2. Event `CompanyClaimRequested` dikirim ke NATS
3. Claim Worker otomatis menjalankan duplicate detection
4. Jika ada entity mirip (match_score >= 0.8), status berubah SUBMITTED → UNDER_REVIEW (perlu perhatian admin ekstra)

---

### 15D. Lihat Status Claim

**Page:** `/claims` (list semua claim user) atau `/claims/{id}` (detail)

#### List My Claims

```http
GET /api/v1/claims
Authorization: Bearer <access_token>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "claim-uuid",
      "entity_id": "shadow-entity-uuid",
      "entity_name": "PT Maju Jaya",
      "claim_type": "OWNER",
      "status": "SUBMITTED",
      "created_at": "2026-05-27T10:00:00Z"
    }
  ]
}
```

#### Get Claim Detail

```http
GET /api/v1/claims/{claim-id}
Authorization: Bearer <access_token>
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "claim-uuid",
    "entity_id": "shadow-entity-uuid",
    "entity_name": "PT Maju Jaya",
    "claim_type": "OWNER",
    "status": "UNDER_REVIEW",
    "notes": "Ini adalah perusahaan saya",
    "reviewed_by": "admin-uuid",
    "rejection_reason": null,
    "created_at": "2026-05-27T10:00:00Z",
    "updated_at": "2026-05-27T10:10:00Z"
  }
}
```

#### List Claim Documents

```http
GET /api/v1/claims/{claim-id}/documents
Authorization: Bearer <access_token>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "doc-uuid",
      "doc_type": "NPWP",
      "filename": "npwp-scan.pdf",
      "mime_type": "application/pdf",
      "status": "UPLOADED",
      "uploaded_at": "2026-05-27T10:02:00Z"
    }
  ]
}
```

---

### 15E. Dispute Claim (User Tolak Keputusan)

Jika claim di-reject, user bisa dispute untuk minta review ulang.

```http
POST /api/v1/claims/{claim-id}/dispute
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "reason": "Dokumen sudah benar, mohon review ulang"
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "claim-uuid",
    "status": "DISPUTED"
  }
}
```

---

### 15F. Admin Review Claims

> **Endpoint admin:** `/admin/claims/*`
> **Permission:** REVIEWER role atau higher

#### List Claims untuk Review

```http
GET /admin/claims?status=SUBMITTED&limit=20&offset=0
Authorization: Bearer <admin-access-token>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "claim-uuid",
      "entity_id": "shadow-entity-uuid",
      "entity_name": "PT Maju Jaya",
      "claim_type": "OWNER",
      "status": "SUBMITTED",
      "user_name": "Jiilan Nashrulloh",
      "created_at": "2026-05-27T10:00:00Z"
    }
  ]
}
```

#### Get Claim Detail (Admin)

```http
GET /admin/claims/{claim-id}
Authorization: Bearer <admin-access-token>
```

#### Assign Claim ke Admin

```http
POST /admin/claims/{claim-id}/assign
Authorization: Bearer <admin-access-token>
```

#### Lihat Dokumen (Presigned GET URL)

```http
GET /admin/claims/{claim-id}/documents/{doc-id}/view
Authorization: Bearer <admin-access-token>
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "url": "https://azzet-storage.r2.cloudflarestorage.com/claims/...?X-Amz-...",
    "expires_at": "2026-05-27T10:30:00Z"
  }
}
```

#### Approve Claim

```http
POST /admin/claims/{claim-id}/approve
Authorization: Bearer <admin-access-token>
Content-Type: application/json

{
  "notes": "Dokumen lengkap dan valid"
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "claim-uuid",
    "status": "APPROVED"
  }
}
```

**What happens:**
1. Status → APPROVED
2. Shadow entity linked ke user: `user_id` di-set, `is_shadow` = false, `status` = `CLAIMED`
3. Event `CompanyClaimApproved` dikirim ke NATS
4. Claim Worker menjalankan `NormalizeName` pada entity

#### Reject Claim

```http
POST /admin/claims/{claim-id}/reject
Authorization: Bearer <admin-access-token>
Content-Type: application/json

{
  "reason": "Dokumen NPWP tidak sesuai dengan nama entity"
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "claim-uuid",
    "status": "REJECTED"
  }
}
```

#### Lihat Audit Log

```http
GET /admin/claims/{claim-id}/audit
Authorization: Bearer <admin-access-token>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "audit-uuid",
      "action": "CLAIM_CREATED",
      "performed_by": "user-uuid",
      "performed_by_name": "Jiilan Nashrulloh",
      "notes": "",
      "created_at": "2026-05-27T10:00:00Z"
    },
    {
      "id": "audit-uuid-2",
      "action": "CLAIM_SUBMITTED",
      "performed_by": "user-uuid",
      "performed_by_name": "Jiilan Nashrulloh",
      "notes": "",
      "created_at": "2026-05-27T10:05:00Z"
    },
    {
      "id": "audit-uuid-3",
      "action": "CLAIM_APPROVED",
      "performed_by": "admin-uuid",
      "performed_by_name": "Admin Azzet",
      "notes": "Dokumen lengkap dan valid",
      "created_at": "2026-05-27T10:15:00Z"
    }
  ]
}
```

#### Count Pending Claims

```http
GET /admin/claims/stats
Authorization: Bearer <admin-access-token>
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "pending_count": 5
  }
}
```

---

## Flow 16: Counterparty Alias Management

> **Prerequisite:** User sudah punya workspace aktif.
> Counterparty aliases memungkinkan workspace memberikan nama custom ke counterparty (entity lain) tanpa mengubah nama asli entity.

### Kenapa Butuh Alias?

```
Entity: "PT Maju Jaya Sejahtera" (nama resmi)
  ├── Workspace A punya alias: "Maju Jaya" (karena klien biasa sebut begitu)
  ├── Workspace B punya alias: "Supplier Utama" (internal reference)
  └── Nama asli entity TIDAK berubah
```

**Privacy:** Workspace hanya bisa search counterparty berdasarkan alias yang mereka punya hubungan bisnis, bukan semua entity di sistem.

---

### 16A. Set Counterparty Alias

**Page:** `/workspace/counterparties/{id}` → edit alias

```http
POST /api/v1/workspaces/counterparties/aliases
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "counterparty_entity_id": "counterparty-entity-uuid",
  "alias": "Maju Jaya"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "alias-uuid",
    "counterparty_entity_id": "counterparty-entity-uuid",
    "alias": "Maju Jaya",
    "created_at": "2026-05-27T10:00:00Z"
  }
}
```

---

### 16B. List Counterparty Aliases

```http
GET /api/v1/workspaces/counterparties/aliases
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "alias-uuid",
      "counterparty_entity_id": "counterparty-entity-uuid",
      "alias": "Maju Jaya",
      "created_at": "2026-05-27T10:00:00Z"
    }
  ]
}
```

---

### 16C. Hapus Counterparty Alias

```http
DELETE /api/v1/workspaces/counterparties/aliases/{counterparty-entity-id}
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

---

### 16D. Search Counterparties

Pencarian counterparties berdasarkan nama asli ATAU alias yang dimiliki workspace ini.

```http
GET /api/v1/workspaces/counterparties/search?q=Maju&limit=20
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
```

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "entity_id": "counterparty-entity-uuid",
      "nama_utama": "PT Maju Jaya Sejahtera",
      "alias": "Maju Jaya",
      "relation_type": "PELANGGAN",
      "match_score": 0.92
    }
  ]
}
```

**Privacy:** Hanya counterparty yang punya hubungan bisnis dengan workspace ini yang muncul di hasil search.

---

### 16E. Alur Lengkap: Shadow Entity → Claim → Verified

```
1. User A buat transaksi dengan "PT Maju Jaya"
   → Shadow entity auto-created (is_shadow=true, user_id=null)

2. User B (pemilik PT Maju Jaya) daftar di Azzet
   → Personal entity + workspace created

3. User B lihat shadow entity "PT Maju Jaya" di search
   → Muncul dengan badge "Shadow Entity"

4. User B buat claim (Flow 15A-15C)
   → Upload NPWP + NIB, submit claim

5. Admin review (Flow 15F)
   → Approve claim

6. Shadow entity linked ke User B
   → user_id = User B, is_shadow = false, status = CLAIMED
   → User B sekarang punya workspace untuk PT Maju Jaya

7. Semua workspace yang punya "PT Maju Jaya" sebagai counterparty
   → Otomatis ter-update (entity bukan shadow lagi)
```

---

**Last Updated:** 2026-05-27
