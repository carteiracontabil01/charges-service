package model

// BillingIntegrationRow represents the minimal fields we need to read from iam.billing_integrations.
// NOTE: token is a secret; only backend services should access it.
type BillingIntegrationRow struct {
	ID                 string `json:"id"`
	AccountingOfficeID string `json:"accounting_office_id"`
	Provider           string `json:"provider"`
	Environment        string `json:"environment"`
	BaseAPI            string `json:"base_api"`
	Token              string `json:"token"`
	IsActive           bool   `json:"is_active"`
	IsDefault          bool   `json:"is_default"`
	UpdatedAt          string `json:"updated_at"`
	CreatedAt          string `json:"created_at"`
}
