package tax

import "errors"

var (
	ErrProfileNotFound      = errors.New("tax profile not found")
	ErrCalculationNotFound  = errors.New("tax calculation not found")
	ErrReportJobNotFound    = errors.New("tax report job not found")
	ErrInvalidTaxStatus     = errors.New("invalid tax status")
	ErrInvalidReportType    = errors.New("invalid report type")
	ErrInvalidDocRefType    = errors.New("invalid document reference type")
	ErrDocumentNotFound     = errors.New("document not found")
	ErrCalculationExists    = errors.New("tax calculation already exists for this transaction")
)
