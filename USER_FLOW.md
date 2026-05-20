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

> After registration + verification + first login, the system automatically creates:
> 1. Personal entity (ORANG_PRIBADI)
> 2. Personal workspace
>
> This happens via event system (async), usually within 1-2 seconds.

**Frontend should:**

1. After first login, call `GET /api/v1/workspaces`
2. If empty (event not processed yet), show loading/onboarding screen
3. Poll every 2 seconds until workspace appears
4. Once workspace exists, proceed to plan selection

**Request:**
```http
GET /api/v1/workspaces
Authorization: Bearer <access_token>
```

**Response (workspace ready):**
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
      "created_at": "2026-05-20T10:00:05Z"
    }
  ]
}
```

**Frontend Action:**
- Store `entity_id` as default workspace
- Set `X-Workspace-ID` header for subsequent requests
- Redirect to plan selection page (`/plans`)

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

**Frontend Action after subscription:**
- Redirect to dashboard
- If paid plan → redirect to payment page (Flow 10)
- If free/trial → ready to use

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

**Frontend:** Workspace switcher dropdown in navbar

All workspace-scoped requests need `X-Workspace-ID` header:

```http
GET /api/v1/subscription
Authorization: Bearer <access_token>
X-Workspace-ID: <selected-workspace-entity-id>
```

### Invite Team Member

**Page:** `/workspace/settings/members`

```http
POST /api/v1/workspaces/members
Authorization: Bearer <access_token>
X-Workspace-ID: <workspace-entity-id>
Content-Type: application/json

{
  "entity_id": "<member-entity-id>",
  "role": "KASIR",
  "custom_alias": "Andi Kasir"
}
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
    │   ├── NO → Redirect to /login (except public pages)
    │   └── YES ↓
    │
    ├── Token expired?
    │   ├── YES → Try refresh (POST /auth/refresh)
    │   │         ├── Success → continue
    │   │         └── Fail → clear state, redirect to /login
    │   └── NO ↓
    │
    ├── GET /api/v1/workspaces → has workspaces?
    │   ├── NO (empty array) → entity not created yet
    │   │   └── Show loading screen, poll every 2s
    │   │       (event system creating entity + workspace)
    │   │       └── Once workspace appears → continue ↓
    │   └── YES ↓
    │
    ├── Has X-Workspace-ID selected?
    │   ├── NO → Redirect to /workspaces (workspace picker)
    │   └── YES ↓
    │
    ├── GET /api/v1/subscription → has active subscription?
    │   ├── NO (404 or status=expired/cancelled)
    │   │   └── Redirect to /plans (plan selection page)
    │   └── YES (status=active or status=trial) ↓
    │
    ├── If trial: is trial expiring soon? (< 3 days)
    │   └── Show banner: "Trial expires in X days. Upgrade now."
    │
    └── ALLOW ACCESS to workspace features
```

---

### Page-by-Page Routing Rules

#### Public Pages (No Auth Required)

| Page | Path | Condition |
|------|------|-----------|
| Landing | `/` | Always accessible |
| Login | `/login` | If already logged in → redirect to `/dashboard` |
| Register | `/register` | If already logged in → redirect to `/dashboard` |
| Verify Email | `/verify-email` | Accessible after registration |
| Verify WhatsApp | `/verify-whatsapp` | Accessible after registration |
| Forgot Password | `/forgot-password` | Always accessible |
| Reset Password | `/reset-password` | Always accessible |
| Pricing | `/plans` | Always accessible (public plan list) |

#### Authenticated Pages (Requires Access Token)

| Page | Path | Guard Logic |
|------|------|-------------|
| Dashboard | `/dashboard` | Requires workspace + subscription |
| Workspace Picker | `/workspaces` | Shows all workspaces, user picks one |
| Create Workspace | `/workspaces/new` | User creates business entity + workspace |
| Plan Selection | `/plans/select` | Shown when workspace has no subscription |
| Payment | `/billing/pay` | Shown when paid plan selected |

#### Workspace-Scoped Pages (Requires X-Workspace-ID + Active Subscription)

| Page | Path | Guard Logic |
|------|------|-------------|
| Dashboard | `/dashboard` | Main workspace view |
| Members | `/workspace/members` | Requires PEMILIK role |
| Counterparties | `/workspace/counterparties` | Any workspace member |
| Billing | `/workspace/billing` | Requires PEMILIK role |
| Settings | `/workspace/settings` | Requires PEMILIK role |
| Subscription | `/workspace/subscription` | Requires PEMILIK role |

---

### Detailed Guard Logic Per Scenario

#### Scenario 1: Brand New User (Just Registered)

```
1. User registers → status = UNVERIFIED
2. Redirect to /verify-email or /verify-whatsapp
3. User verifies OTP → status = ACTIVE
4. Redirect to /login
5. User logs in → access_token received
6. GET /workspaces → empty (event system processing)
7. Show: "Setting up your workspace..." (loading)
8. Poll GET /workspaces every 2 seconds
9. Workspace appears → store entity_id as X-Workspace-ID
10. GET /subscription → 404 (no subscription yet)
11. Redirect to /plans/select
12. User picks Free plan → POST /subscription
13. Subscription active → redirect to /dashboard
14. User can now use features!
```

#### Scenario 2: Returning User (Has Everything Set Up)

```
1. User opens app → has access_token in memory? 
   - If page refresh: token gone → try refresh cookie
   - POST /auth/refresh → new access_token
2. GET /workspaces → has workspaces ✓
3. Last used workspace stored in localStorage (just the ID)
4. Set X-Workspace-ID header
5. GET /subscription → active ✓
6. Show dashboard directly
```

#### Scenario 3: User with Expired Trial

```
1. User logs in → access_token received
2. GET /workspaces → has workspace ✓
3. Set X-Workspace-ID
4. GET /subscription → status = "expired" (trial ended)
5. Redirect to /plans/select
6. Show message: "Your trial has ended. Choose a plan to continue."
7. User picks paid plan → POST /subscription { plan_id, billing_cycle: "monthly" }
8. Invoice created → redirect to /billing/pay
9. POST /billing/pay → get payment_url
10. Redirect to Xendit checkout
11. User pays → webhook activates subscription
12. User returns → subscription active → dashboard
```

#### Scenario 4: User with Multiple Workspaces

```
1. User logs in
2. GET /workspaces → returns multiple:
   [
     { entity_name: "Jiilan (Personal)", entity_type: "ORANG_PRIBADI" },
     { entity_name: "PT Azzet", entity_type: "BADAN_USAHA" },
     { entity_name: "CV Jiilan", entity_type: "BADAN_USAHA" }
   ]
3. Show workspace picker (/workspaces)
4. User selects "PT Azzet" → store as X-Workspace-ID
5. GET /subscription for PT Azzet → check status
6. If active → dashboard
7. If no subscription → /plans/select
```

#### Scenario 5: User Creates New Business Workspace

```
1. User is on /dashboard (personal workspace active)
2. Clicks "Create Business Workspace"
3. Redirect to /workspaces/new
4. Fill form: nama_utama, entity_type=BADAN_USAHA, nik_npwp, etc.
5. POST /entities → entity created
6. POST /workspaces { entity_id } → workspace created
7. Switch X-Workspace-ID to new workspace
8. GET /subscription → 404 (new workspace, no plan)
9. Redirect to /plans/select
10. User subscribes → ready to use
```

#### Scenario 6: User Tries to Access Feature Without Subscription

```
1. User on workspace that has no subscription
2. Tries to access /workspace/counterparties
3. Frontend guard: GET /subscription → 404
4. Redirect to /plans/select
5. Show: "Subscribe to a plan to access this feature"
```

#### Scenario 7: User's Payment Failed

```
1. User subscribed to paid plan
2. Invoice created but payment failed/expired
3. GET /subscription → status = "active" (still within grace period)
   OR status = "expired" (payment overdue)
4. If expired:
   - Show banner: "Payment overdue. Please pay to continue."
   - Redirect to /workspace/billing
   - Show unpaid invoice
   - User clicks "Pay Now" → POST /billing/pay → Xendit checkout
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
│  └── /plans (public pricing)                                 │
│                                                              │
│  ─── Login Success ───────────────────────────────────────── │
│                                                              │
│  AUTHENTICATED (has access_token)                            │
│  │                                                           │
│  ├── NO WORKSPACE YET                                        │
│  │   └── Show loading (poll GET /workspaces)                 │
│  │                                                           │
│  ├── HAS WORKSPACE, NO SUBSCRIPTION                          │
│  │   ├── /workspaces (picker)                                │
│  │   ├── /workspaces/new (create business)                   │
│  │   └── /plans/select (choose plan) ← FORCED               │
│  │                                                           │
│  ├── HAS WORKSPACE + ACTIVE SUBSCRIPTION                     │
│  │   ├── /dashboard                                          │
│  │   ├── /workspace/members                                  │
│  │   ├── /workspace/counterparties                           │
│  │   ├── /workspace/billing                                  │
│  │   ├── /workspace/settings                                 │
│  │   └── (all business features)                             │
│  │                                                           │
│  └── SUBSCRIPTION EXPIRED                                    │
│      └── /plans/select ← FORCED (must re-subscribe)          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

---

### API Calls for Guard Logic (Frontend Init)

```javascript
// On app load / page navigation:
async function initGuard() {
  // 1. Check auth
  if (!accessToken) {
    const refreshed = await tryRefresh();
    if (!refreshed) return redirect('/login');
  }

  // 2. Check workspaces
  const { data: workspaces } = await api.get('/workspaces');
  
  if (workspaces.length === 0) {
    // Entity not created yet (event processing)
    return showLoading('Setting up your workspace...');
    // Poll every 2s until workspace appears
  }

  // 3. Check workspace selection
  const wsId = localStorage.getItem('workspace_id') || workspaces[0].entity_id;
  setWorkspaceHeader(wsId);

  // 4. Check subscription
  try {
    const { data: sub } = await api.get('/subscription');
    
    if (sub.status === 'trial' && isExpiringSoon(sub.trial_ends_at)) {
      showBanner('Trial expires soon. Upgrade now.');
    }
    
    // All good - allow access
    return ALLOW;
    
  } catch (err) {
    if (err.status === 404) {
      // No subscription
      return redirect('/plans/select');
    }
    throw err;
  }
}
```

---

### localStorage Keys (Frontend Reference)

| Key | Value | Purpose |
|-----|-------|---------|
| `workspace_id` | UUID string | Last selected workspace (for X-Workspace-ID header) |
| `workspace_name` | string | Display name in navbar |
| `theme` | "light" / "dark" | UI preference |

**Note:** Access token is NEVER stored in localStorage. Only in memory (JS variable / React state).

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
/api/v1/workspaces/counterparties/*
/api/v1/subscription/*
/api/v1/billing/*
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

```javascript
// Store access token in memory (NOT localStorage)
let accessToken = null;

// Set refresh timer
function scheduleRefresh(expiresIn) {
  // Refresh at 80% of expiry time
  const refreshAt = expiresIn * 0.8 * 1000; // ms
  setTimeout(refreshToken, refreshAt);
}

// Axios interceptor for auto-refresh
axios.interceptors.response.use(
  response => response,
  async error => {
    if (error.response?.status === 401 && !error.config._retry) {
      error.config._retry = true;
      const newToken = await refreshToken();
      if (newToken) {
        error.config.headers.Authorization = `Bearer ${newToken}`;
        return axios(error.config);
      }
    }
    return Promise.reject(error);
  }
);
```

---

## Important Notes for Frontend

1. **Refresh token is HttpOnly cookie** — Frontend cannot read it. Browser sends it automatically with requests to `/api/v1/auth/*` path.

2. **Access token in memory only** — Never store in localStorage (XSS risk). Store in JS variable or React state.

3. **X-Workspace-ID is mandatory** for business endpoints. Frontend should have a workspace switcher and always send this header.

4. **Personal entity is auto-created** after registration (via event system). May take 1-2 seconds. Poll `GET /workspaces` until it appears.

5. **Password is always required** during registration (even for WhatsApp users) as fallback when OTP service is down.

6. **OTP expires in 5 minutes**, max 3 attempts. After that, user needs to request a new one.

7. **OTP expires in 5 minutes**, max 3 attempts. After that, user needs to request a new one.

---

**Last Updated:** 2026-05-20
