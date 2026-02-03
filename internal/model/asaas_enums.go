package model

// AsaasPersonType represents the customer personType field returned/accepted by Asaas.
type AsaasPersonType string

const (
	AsaasPersonTypeJuridica AsaasPersonType = "JURIDICA"
	AsaasPersonTypeFisica   AsaasPersonType = "FISICA"
)

// AsaasBillingType represents the billing type used when creating charges (payments) in Asaas.
type AsaasBillingType string

const (
	AsaasBillingTypeBoleto     AsaasBillingType = "BOLETO"
	AsaasBillingTypeCreditCard AsaasBillingType = "CREDIT_CARD"
	AsaasBillingTypePix        AsaasBillingType = "PIX"
	AsaasBillingTypeUndefined  AsaasBillingType = "UNDEFINED"
)

// AsaasDiscountType represents the discount type in Asaas charges.
type AsaasDiscountType string

const (
	AsaasDiscountTypeFixed      AsaasDiscountType = "FIXED"
	AsaasDiscountTypePercentage AsaasDiscountType = "PERCENTAGE"
)

// AsaasFineType represents the fine type in Asaas charges.
type AsaasFineType string

const (
	AsaasFineTypeFixed      AsaasFineType = "FIXED"
	AsaasFineTypePercentage AsaasFineType = "PERCENTAGE"
)

// AsaasPaymentStatus represents payment status in Asaas list filters.
type AsaasPaymentStatus string

const (
	AsaasPaymentStatusPending                    AsaasPaymentStatus = "PENDING"
	AsaasPaymentStatusReceived                   AsaasPaymentStatus = "RECEIVED"
	AsaasPaymentStatusConfirmed                  AsaasPaymentStatus = "CONFIRMED"
	AsaasPaymentStatusOverdue                    AsaasPaymentStatus = "OVERDUE"
	AsaasPaymentStatusRefunded                   AsaasPaymentStatus = "REFUNDED"
	AsaasPaymentStatusReceivedInCash             AsaasPaymentStatus = "RECEIVED_IN_CASH"
	AsaasPaymentStatusRefundRequested            AsaasPaymentStatus = "REFUND_REQUESTED"
	AsaasPaymentStatusRefundInProgress           AsaasPaymentStatus = "REFUND_IN_PROGRESS"
	AsaasPaymentStatusChargebackRequested        AsaasPaymentStatus = "CHARGEBACK_REQUESTED"
	AsaasPaymentStatusChargebackDispute          AsaasPaymentStatus = "CHARGEBACK_DISPUTE"
	AsaasPaymentStatusAwaitingChargebackReversal AsaasPaymentStatus = "AWAITING_CHARGEBACK_REVERSAL"
	AsaasPaymentStatusDunningRequested           AsaasPaymentStatus = "DUNNING_REQUESTED"
	AsaasPaymentStatusDunningReceived            AsaasPaymentStatus = "DUNNING_RECEIVED"
	AsaasPaymentStatusAwaitingRiskAnalysis       AsaasPaymentStatus = "AWAITING_RISK_ANALYSIS"
)

// AsaasInvoiceStatus represents invoice status filter in list payments.
type AsaasInvoiceStatus string

const (
	AsaasInvoiceStatusScheduled              AsaasInvoiceStatus = "SCHEDULED"
	AsaasInvoiceStatusAuthorized             AsaasInvoiceStatus = "AUTHORIZED"
	AsaasInvoiceStatusProcessingCancellation AsaasInvoiceStatus = "PROCESSING_CANCELLATION"
	AsaasInvoiceStatusCanceled               AsaasInvoiceStatus = "CANCELED"
	AsaasInvoiceStatusCancellationDenied     AsaasInvoiceStatus = "CANCELLATION_DENIED"
	AsaasInvoiceStatusError                  AsaasInvoiceStatus = "ERROR"
)
