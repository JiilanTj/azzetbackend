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
│       │                       ├── Record Transactions (Phase 7) │
│       │                       └── Generate Reports (Phase 7)    │
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

**Last Updated:** 2026-05-22
