package model

// AsaasSubscriptionDiscount models Asaas discount object for subscriptions.
type AsaasSubscriptionDiscount struct {
	Value            *float64           `json:"value,omitempty"`
	DueDateLimitDays *int32             `json:"dueDateLimitDays,omitempty"`
	Type             *AsaasDiscountType `json:"type,omitempty"` // FIXED | PERCENTAGE
}

// AsaasSubscriptionInterest models Asaas interest object for subscriptions.
type AsaasSubscriptionInterest struct {
	Value *float64 `json:"value,omitempty"`
}

// AsaasSubscriptionFine models Asaas fine object for subscriptions.
type AsaasSubscriptionFine struct {
	Value *float64       `json:"value,omitempty"`
	Type  *AsaasFineType `json:"type,omitempty"` // FIXED | PERCENTAGE
}

// AsaasCreateSubscriptionRequest is the payload we send to Asaas /v3/subscriptions.
// Our API resolves "customer" internally by company_id (mapping table), same pattern as charges.
type AsaasCreateSubscriptionRequest struct {
	BillingType AsaasBillingType `json:"billingType"` // BOLETO | CREDIT_CARD | PIX | UNDEFINED
	Value       float64          `json:"value"`
	NextDueDate string           `json:"nextDueDate"` // YYYY-MM-DD
	Cycle       string           `json:"cycle"`       // WEEKLY | BIWEEKLY | MONTHLY | BIMONTHLY | QUARTERLY | SEMIANNUALLY | YEARLY
	Description *string          `json:"description,omitempty"`
	EndDate     *string          `json:"endDate,omitempty"` // YYYY-MM-DD
	MaxPayments *int32           `json:"maxPayments,omitempty"`

	ExternalReference *string `json:"externalReference,omitempty"`

	Discount *AsaasSubscriptionDiscount `json:"discount,omitempty"`
	Interest *AsaasSubscriptionInterest `json:"interest,omitempty"`
	Fine     *AsaasSubscriptionFine     `json:"fine,omitempty"`
}

// AsaasUpdateSubscriptionRequest is the payload for updating an existing subscription in Asaas.
// https://docs.asaas.com/reference/atualizar-assinatura-existente
// All fields are optional; only provided fields will be updated.
// Note: nextDueDate sets the due date for the NEXT installment to be generated (not the current pending one).
// Set updatePendingPayments=true to also update existing pending payments with new billingType/value.
type AsaasUpdateSubscriptionRequest struct {
	BillingType *AsaasBillingType `json:"billingType,omitempty"` // BOLETO | CREDIT_CARD | PIX | UNDEFINED
	Status      *string           `json:"status,omitempty"`      // ACTIVE | INACTIVE
	Value       *float64          `json:"value,omitempty"`
	NextDueDate *string           `json:"nextDueDate,omitempty"` // YYYY-MM-DD — next installment due date
	Cycle       *string           `json:"cycle,omitempty"`       // WEEKLY | BIWEEKLY | MONTHLY | BIMONTHLY | QUARTERLY | SEMIANNUALLY | YEARLY
	Description *string           `json:"description,omitempty"`
	EndDate     *string           `json:"endDate,omitempty"` // YYYY-MM-DD

	// When true, updates pending (unpaid) payments with the new billingType and/or value.
	UpdatePendingPayments *bool `json:"updatePendingPayments,omitempty"`

	ExternalReference *string `json:"externalReference,omitempty"`

	Discount *AsaasSubscriptionDiscount `json:"discount,omitempty"`
	Interest *AsaasSubscriptionInterest `json:"interest,omitempty"`
	Fine     *AsaasSubscriptionFine     `json:"fine,omitempty"`
}

// AsaasSubscriptionResponse is a partial representation of the subscription object returned by Asaas.
// We use it mainly to extract id and show on logs/response; handlers pass-through raw JSON.
type AsaasSubscriptionResponse struct {
	Object      string  `json:"object"`
	ID          string  `json:"id"`
	DateCreated string  `json:"dateCreated"`
	Customer    string  `json:"customer"`

	BillingType string  `json:"billingType"`
	Cycle       string  `json:"cycle"`
	Value       float64 `json:"value"`
	NextDueDate string  `json:"nextDueDate"`
	EndDate     string  `json:"endDate"`
	Description string  `json:"description"`
	Status      string  `json:"status"`

	Discount any `json:"discount,omitempty"`
	Fine     any `json:"fine,omitempty"`
	Interest any `json:"interest,omitempty"`

	MaxPayments      *int32  `json:"maxPayments,omitempty"`
	ExternalReference *string `json:"externalReference,omitempty"`
	Deleted          bool    `json:"deleted"`
}

