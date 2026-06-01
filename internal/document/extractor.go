package document

import (
	"context"
	"fmt"
	"strings"

	"codeberg.org/azzet/azzetbe/internal/accounting"
	"codeberg.org/azzet/azzetbe/internal/ai"
)

const extractionSystemPrompt = `You are an OCR assistant for Indonesian business receipts and invoices.

Extract structured data from the document image and return JSON with these fields:
- vendor_name: string (business or person name)
- vendor_npwp: string (optional, NPWP if visible)
- amount: number (total amount in IDR, no currency symbol)
- date: string (YYYY-MM-DD format)
- transaction_type: one of CASH_IN, CASH_OUT, PURCHASE, SALES
- category: one of the valid categories for the transaction type (Indonesian accounting)
- payment_method: one of TUNAI, TRANSFER, KREDIT (optional)
- description: string (brief summary of items/services)
- confidence: number 0.0-1.0 (your confidence in the extraction)

Rules:
- For expense receipts (buying goods/services): use CASH_OUT or PURCHASE
- For sales invoices (selling): use SALES or CASH_IN
- amount must be positive number
- If unsure about category, pick the closest match from valid categories
- date must be parseable as YYYY-MM-DD`

// Extractor performs AI-powered document extraction.
type Extractor struct {
	AI *ai.Client
}

func NewExtractor(aiClient *ai.Client) *Extractor {
	return &Extractor{AI: aiClient}
}

func (e *Extractor) ExtractFromImageURL(ctx context.Context, imageURL, docType string) (*ExtractionResult, error) {
	if e.AI == nil {
		return nil, fmt.Errorf("AI client not configured")
	}

	userPrompt := fmt.Sprintf("Document type: %s\nExtract all financial data from this document.", docType)

	var result ExtractionResult
	if err := e.AI.ChatVisionJSON(ctx, extractionSystemPrompt, userPrompt, imageURL, &result); err != nil {
		return nil, fmt.Errorf("OCR extraction failed: %w", err)
	}

	if err := validateExtraction(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func validateExtraction(r *ExtractionResult) error {
	r.VendorName = strings.TrimSpace(r.VendorName)
	r.Category = strings.TrimSpace(r.Category)
	r.TransactionType = strings.ToUpper(strings.TrimSpace(r.TransactionType))
	r.PaymentMethod = strings.ToUpper(strings.TrimSpace(r.PaymentMethod))

	if r.VendorName == "" {
		return fmt.Errorf("vendor_name is required")
	}
	if r.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if r.Date == "" {
		return fmt.Errorf("date is required")
	}
	if !accounting.IsValidTxType(r.TransactionType) {
		return fmt.Errorf("invalid transaction_type: %s", r.TransactionType)
	}
	if r.Category == "" {
		r.Category = accounting.GetFallbackCategory(r.TransactionType)
	}
	if !accounting.IsValidCategoryForType(r.Category, r.TransactionType) {
		r.Category = accounting.GetFallbackCategory(r.TransactionType)
	}
	if r.PaymentMethod != "" && !accounting.IsValidPaymentMethod(r.PaymentMethod) {
		r.PaymentMethod = accounting.PaymentMethodTunai
	}
	if r.Confidence <= 0 {
		r.Confidence = 0.5
	}
	if r.Confidence > 1 {
		r.Confidence = 1
	}
	return nil
}

func IsImageMimeType(mime string) bool {
	return strings.HasPrefix(mime, "image/")
}
