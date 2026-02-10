package model

// AsaasUpdateChargeRequest is the payload to update an existing charge (payment) in Asaas.
// According to Asaas docs: only charges awaiting payment or overdue can be updated.
// Once created, the customer cannot be changed.
// https://docs.asaas.com/reference/atualizar-cobranca-existente
type AsaasUpdateChargeRequest struct {
	BillingType                                *AsaasBillingType `json:"billingType,omitempty"` // BOLETO | CREDIT_CARD | PIX | UNDEFINED
	Value                                      *float64          `json:"value,omitempty"`
	DueDate                                    *string           `json:"dueDate,omitempty"` // YYYY-MM-DD
	Description                                *string           `json:"description,omitempty"`
	DaysAfterDueDateToRegistrationCancellation *int32            `json:"daysAfterDueDateToRegistrationCancellation,omitempty"`
	ExternalReference                          *string           `json:"externalReference,omitempty"`

	Discount      *AsaasChargeDiscount `json:"discount,omitempty"`
	Interest      *AsaasChargeInterest `json:"interest,omitempty"`
	Fine          *AsaasChargeFine     `json:"fine,omitempty"`
	PostalService *bool                `json:"postalService,omitempty"`

	// Callback controls automatic redirect behavior after payment (commonly used for payment links).
	// Asaas docs show this object as a free-form payload; we keep it flexible to avoid breaking changes.
	Callback map[string]any `json:"callback,omitempty"`

	// Split is allowed for credit/debit card charges up to 1 business day before expected payment date.
	// Can only update CONFIRMED status charges without anticipation.
	// Exception: split divergence block allows updating even RECEIVED/anticipated charges (split field only).
	Split []AsaasChargeSplit `json:"split,omitempty"`
}

// AsaasChargeSplit represents a split configuration for a charge.
type AsaasChargeSplit struct {
	WalletID         string   `json:"walletId"`
	FixedValue       *float64 `json:"fixedValue,omitempty"`
	PercentualValue  *float64 `json:"percentualValue,omitempty"`
	ExternalReference *string  `json:"externalReference,omitempty"`
	Description      *string  `json:"description,omitempty"`
}
