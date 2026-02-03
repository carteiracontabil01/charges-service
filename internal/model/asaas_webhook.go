package model

// AsaasWebhookEvent represents the event object sent by Asaas webhooks.
// Reference: https://docs.asaas.com/docs/receba-eventos-do-asaas-no-seu-endpoint-de-webhook
type AsaasWebhookEvent struct {
	ID          string                `json:"id"`
	Event       string                `json:"event"` // e.g., PAYMENT_CREATED, PAYMENT_RECEIVED, PAYMENT_UPDATED, etc.
	DateCreated string                `json:"dateCreated"` // Format: "2026-01-24 16:13:02"
	Account     *AsaasWebhookAccount  `json:"account,omitempty"`
	Payment     *AsaasPaymentResponse `json:"payment,omitempty"` // The payment object (when event is payment-related)
}

// AsaasWebhookAccount represents the account object in webhook events
type AsaasWebhookAccount struct {
	ID      string  `json:"id"`
	OwnerID *string `json:"ownerId"`
}

// AsaasWebhookEventType defines the webhook events we handle.
// Full list: https://docs.asaas.com/docs/eventos-de-webhooks#eventos-para-cobran%C3%A7as
const (
	EventPaymentCreated           = "PAYMENT_CREATED"
	EventPaymentUpdated           = "PAYMENT_UPDATED"
	EventPaymentConfirmed         = "PAYMENT_CONFIRMED"
	EventPaymentReceived          = "PAYMENT_RECEIVED"
	EventPaymentOverdue           = "PAYMENT_OVERDUE"
	EventPaymentDeleted           = "PAYMENT_DELETED"
	EventPaymentRestored          = "PAYMENT_RESTORED"
	EventPaymentRefunded          = "PAYMENT_REFUNDED"
	EventPaymentReceivedInCash    = "PAYMENT_RECEIVED_IN_CASH"
	EventPaymentChargebackRequested = "PAYMENT_CHARGEBACK_REQUESTED"
	EventPaymentChargebackDispute   = "PAYMENT_CHARGEBACK_DISPUTE"
	EventPaymentAwaitingChargeback  = "PAYMENT_AWAITING_CHARGEBACK_REVERSAL"
	EventPaymentDunningReceived     = "PAYMENT_DUNNING_RECEIVED"
	EventPaymentDunningRequested    = "PAYMENT_DUNNING_REQUESTED"
	EventPaymentBankSlipViewed      = "PAYMENT_BANK_SLIP_VIEWED"
	EventPaymentCheckoutViewed      = "PAYMENT_CHECKOUT_VIEWED"
)
