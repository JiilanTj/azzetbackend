package accounting

import "errors"

// --- Service Errors ---

var (
	// Account errors
	ErrAccountNotFound     = errors.New("account not found")
	ErrAccountCodeExists   = errors.New("account code already exists in this workspace")
	ErrAccountIsSystem     = errors.New("cannot modify or delete system account")
	ErrAccountHasChildren  = errors.New("cannot deactivate account with active children")
	ErrInvalidAccountType  = errors.New("invalid account type")
	ErrInvalidParentAccount = errors.New("invalid parent account")

	// Item errors
	ErrItemNotFound    = errors.New("item not found")
	ErrInvalidItemType = errors.New("invalid item type")
	ErrInvalidUnit     = errors.New("invalid unit")

	// Transaction errors
	ErrTransactionNotFound    = errors.New("transaction not found")
	ErrTransactionNotDraft    = errors.New("can only modify transactions in DRAFT status")
	ErrTransactionNotPosted   = errors.New("can only void transactions in POSTED status")
	ErrTransactionAlreadyVoid = errors.New("transaction is already voided")
	ErrInvalidTransactionType = errors.New("invalid transaction type")
	ErrInvalidInputMode       = errors.New("invalid input mode")
	ErrInvalidPaymentMethod   = errors.New("invalid payment method")
	ErrInvalidCategory        = errors.New("invalid category for this transaction type")
	ErrAmountMustBePositive   = errors.New("amount must be greater than zero")
	ErrDebitCreditMismatch    = errors.New("total debit must equal total credit")
	ErrNoJournalEntries       = errors.New("transaction must have at least one journal entry")
	ErrNoLineItems            = errors.New("transaction must have at least one line item")

	// Ledger errors
	ErrLedgerPostingFailed    = errors.New("ledger posting failed")
	ErrAccountingEquationFail = errors.New("accounting equation violated: Assets != Liabilities + Equity")

	// Categorization errors
	ErrCategorizationFailed = errors.New("AI categorization failed")
	ErrLowConfidence        = errors.New("AI confidence too low, using fallback category")

	// COA errors
	ErrCOAAlreadySeeded = errors.New("chart of accounts already seeded for this workspace")
)
