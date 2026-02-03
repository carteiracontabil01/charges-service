package model

import "encoding/json"

// IamChargeRow is the row we persist in iam.charges (provider-agnostic).
type IamChargeRow struct {
	TenantID           string `json:"tenant_id"`
	AccountingOfficeID string `json:"accounting_office_id"`
	CompanyID          string `json:"company_id"`
	ContractID         string `json:"contract_id"`

	Provider              string  `json:"provider"`
	ProviderChargeID      string  `json:"provider_charge_id"`
	ProviderInstallmentID *string `json:"provider_installment_id,omitempty"`

	InstallmentNumber *int32   `json:"installment_number,omitempty"`
	Value             float64  `json:"value"`
	NetValue          *float64 `json:"net_value,omitempty"`
	Description       *string  `json:"description,omitempty"`
	BillingType       *string  `json:"billing_type,omitempty"`
	Status            *string  `json:"status,omitempty"`
	DueDate           *string  `json:"due_date,omitempty"`          // YYYY-MM-DD
	OriginalDueDate   *string  `json:"original_due_date,omitempty"` // YYYY-MM-DD
	InvoiceURL        *string  `json:"invoice_url,omitempty"`
	InvoiceNumber     *string  `json:"invoice_number,omitempty"`
	ExternalReference *string  `json:"external_reference,omitempty"`

	ProviderPayload json.RawMessage `json:"provider_payload,omitempty"`
}
