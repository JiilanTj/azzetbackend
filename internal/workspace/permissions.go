package workspace

// ============================================================
// ABAC Permission Keys
// Used in workspace_roles.permissions[] array.
// Owner (PEMILIK) always has wildcard "*".
// ============================================================

const (
	// Wildcard — full access (owner only)
	PermAll = "*"

	// Transactions
	PermTransactionCreate = "transaction:create"
	PermTransactionRead   = "transaction:read"
	PermTransactionUpdate = "transaction:update"
	PermTransactionDelete = "transaction:delete"

	// Reports
	PermReportRead   = "report:read"
	PermReportExport = "report:export"

	// Members
	PermMemberInvite = "member:invite"
	PermMemberManage = "member:manage"
	PermMemberRemove = "member:remove"

	// Roles
	PermRoleCreate = "role:create"
	PermRoleUpdate = "role:update"
	PermRoleDelete = "role:delete"
	PermRoleAssign = "role:assign"

	// Workspace Settings
	PermWorkspaceSettings = "workspace:settings"

	// Billing
	PermBillingRead   = "billing:read"
	PermBillingManage = "billing:manage"

	// Items / Products
	PermItemCreate = "item:create"
	PermItemRead   = "item:read"
	PermItemUpdate = "item:update"
	PermItemDelete = "item:delete"

	// Accounts (Chart of Accounts)
	PermAccountCreate = "account:create"
	PermAccountRead   = "account:read"
	PermAccountUpdate = "account:update"
	PermAccountDelete = "account:delete"
)

// AllPermissions is the full list of valid permission keys (excluding wildcard)
var AllPermissions = []string{
	PermTransactionCreate, PermTransactionRead, PermTransactionUpdate, PermTransactionDelete,
	PermReportRead, PermReportExport,
	PermMemberInvite, PermMemberManage, PermMemberRemove,
	PermRoleCreate, PermRoleUpdate, PermRoleDelete, PermRoleAssign,
	PermWorkspaceSettings,
	PermBillingRead, PermBillingManage,
	PermItemCreate, PermItemRead, PermItemUpdate, PermItemDelete,
	PermAccountCreate, PermAccountRead, PermAccountUpdate, PermAccountDelete,
}

// IsValidPermission checks if a permission key is valid
func IsValidPermission(perm string) bool {
	if perm == PermAll {
		return true
	}
	for _, p := range AllPermissions {
		if p == perm {
			return true
		}
	}
	return false
}

// HasPermission checks if a permission set contains the required permission
// Supports wildcard "*" which grants all permissions.
func HasPermission(permissions []string, required string) bool {
	for _, p := range permissions {
		if p == PermAll || p == required {
			return true
		}
	}
	return false
}
