package model

// FeeContractRow represents the minimal fields we need from iam.fee_contracts to generate charges.
type FeeContractRow struct {
	ID                string `json:"id"`
	TenantID          string `json:"tenant_id"`
	AccountingOfficeID string `json:"accounting_office_id"`
	CompanyID         string `json:"company_id"`
	ContractNumber    string `json:"contract_number"`

	Provider            *string `json:"provider"`
	ProviderEnvironment *string `json:"provider_environment"`
	BillingIntegrationID *string `json:"billing_integration_id"`

	StartDate *string `json:"start_date"` // date
	EndDate   *string `json:"end_date"`   // date

	// Financial settings (contract-level)
	InterestPercentage     *float64 `json:"interest_percentage"`
	FineType              *string  `json:"fine_type"` // FIXED | PERCENTAGE
	FinePercentage         *float64 `json:"fine_percentage"`
	FineValue              *float64 `json:"fine_value"`
	DiscountType           *string  `json:"discount_type"` // FIXED | PERCENTAGE
	DiscountPercentage     *float64 `json:"discount_percentage"`
	DiscountValue          *float64 `json:"discount_value"`
	DiscountDueLimitDays   *int32   `json:"discount_due_limit_days"`
}

// FeeContractServiceItemRow represents recurring service items in iam.fee_contract_service_items.
type FeeContractServiceItemRow struct {
	ID            string   `json:"id"`
	ContractID    string   `json:"contract_id"`
	LineNo        int32    `json:"line_no"`
	Name          string   `json:"name"`
	BillingType   string   `json:"billing_type"` // RECURRING | ONE_TIME
	Periodicity   *string  `json:"periodicity"`  // MONTHLY | BIMONTHLY | QUARTERLY | SEMIANNUAL | ANNUAL | SPORADIC
	FinalAmount   float64  `json:"final_amount"`
	DueDay        *int32   `json:"due_day"`
	StartDate     *string  `json:"start_date"` // date
	PaymentMethod *string  `json:"payment_method"`
}

