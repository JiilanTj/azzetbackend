package document

import "errors"

var (
	ErrDocumentNotFound    = errors.New("document not found")
	ErrStorageNotConfigured = errors.New("storage not configured")
	ErrUploadNotConfirmed  = errors.New("document not found in storage")
	ErrInvalidDocumentType = errors.New("invalid document_type")
	ErrOCRNotEnabled       = errors.New("OCR feature not enabled on current plan")
	ErrInvalidStatus       = errors.New("invalid document status for this operation")
	ErrExtractionFailed    = errors.New("document extraction failed")
)
