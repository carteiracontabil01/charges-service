package model

// AsaasChargeDiscount models Asaas discount object for charges (payments).
type AsaasChargeDiscount struct {
	Value            *float64           `json:"value,omitempty"`
	DueDateLimitDays *int32             `json:"dueDateLimitDays,omitempty"`
	Type             *AsaasDiscountType `json:"type,omitempty"` // FIXED | PERCENTAGE
}

// AsaasChargeInterest models Asaas interest object for charges (payments).
type AsaasChargeInterest struct {
	Value *float64 `json:"value,omitempty"`
}

// AsaasChargeFine models Asaas fine object for charges (payments).
type AsaasChargeFine struct {
	Value *float64       `json:"value,omitempty"`
	Type  *AsaasFineType `json:"type,omitempty"` // FIXED | PERCENTAGE
}

// AsaasCreateChargeRequest is the payload we accept from our API to create a charge (payment) in Asaas.
// Note: the Asaas customer id is resolved internally from company_id (mapping table).
type AsaasCreateChargeRequest struct {
	BillingType                                AsaasBillingType `json:"billingType"` // BOLETO | CREDIT_CARD | PIX | UNDEFINED
	Value                                      float64          `json:"value"`
	DueDate                                    string           `json:"dueDate"` // YYYY-MM-DD
	Description                                *string          `json:"description,omitempty"`
	DaysAfterDueDateToRegistrationCancellation *int32           `json:"daysAfterDueDateToRegistrationCancellation,omitempty"` // boleto only
	ExternalReference                          *string          `json:"externalReference,omitempty"`

	InstallmentCount *int32   `json:"installmentCount,omitempty"`
	TotalValue       *float64 `json:"totalValue,omitempty"`
	InstallmentValue *float64 `json:"installmentValue,omitempty"`

	Discount      *AsaasChargeDiscount `json:"discount,omitempty"`
	Interest      *AsaasChargeInterest `json:"interest,omitempty"`
	Fine          *AsaasChargeFine     `json:"fine,omitempty"`
	PostalService *bool                `json:"postalService,omitempty"`
}

// AsaasPaymentResponse is a partial representation of the payment object returned by Asaas.
// We keep it for Swagger typing; handlers currently pass-through the raw Asaas JSON payload.
type AsaasPaymentResponse struct {
	Object            string  `json:"object"`
	ID                string  `json:"id"`
	DateCreated       string  `json:"dateCreated"`
	Customer          string  `json:"customer"`
	Installment       string  `json:"installment"`
	CheckoutSession   any     `json:"checkoutSession"`
	PaymentLink       any     `json:"paymentLink"`
	Value             float64 `json:"value"`
	NetValue          float64 `json:"netValue"`
	OriginalValue     any     `json:"originalValue"`
	InterestValue     any     `json:"interestValue"`
	Description       string  `json:"description"`
	BillingType       string  `json:"billingType"`
	CanBePaidAfterDueDate bool `json:"canBePaidAfterDueDate"`
	PixTransaction    any     `json:"pixTransaction"`
	Status            string  `json:"status"`
	DueDate           string  `json:"dueDate"`
	OriginalDueDate   string  `json:"originalDueDate"`
	PaymentDate       string  `json:"paymentDate,omitempty"`
	ClientPaymentDate string  `json:"clientPaymentDate,omitempty"`
	ConfirmedDate     string  `json:"confirmedDate,omitempty"`
	InstallmentNumber int32   `json:"installmentNumber"`
	ExternalReference string  `json:"externalReference"`

	InvoiceURL    string `json:"invoiceUrl"`
	BankSlipURL   string `json:"bankSlipUrl"`
	PaymentLink2  string `json:"paymentLink"` // duplicated field name workaround
	InvoiceNumber string `json:"invoiceNumber"`

	Deleted       bool `json:"deleted"`
	PostalService bool `json:"postalService"`
	Anticipated   bool `json:"anticipated"`
	Anticipable   bool `json:"anticipable"`

	CreditDate          string `json:"creditDate,omitempty"`
	EstimatedCreditDate string `json:"estimatedCreditDate,omitempty"`
	TransactionReceiptURL string `json:"transactionReceiptUrl,omitempty"`
	NossoNumero         string `json:"nossoNumero,omitempty"`

	LastInvoiceViewedDate   any `json:"lastInvoiceViewedDate"`
	LastBankSlipViewedDate  any `json:"lastBankSlipViewedDate"`

	Discount      any `json:"discount,omitempty"`
	Fine          any `json:"fine,omitempty"`
	Interest      any `json:"interest,omitempty"`
	Escrow        any `json:"escrow"`
	Refunds       any `json:"refunds"`
}
