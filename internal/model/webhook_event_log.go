package model

import "encoding/json"

// AsaasWebhookEventLog represents a row in logs.asaas_webhook_events.
// It is written whenever an Asaas webhook event cannot be processed successfully,
// providing full context for debugging and manual resolution.
type AsaasWebhookEventLog struct {
	// EventType is the event name sent by Asaas (e.g. "PAYMENT_CREATED").
	EventType string `json:"event_type,omitempty"`

	// PaymentID is the Asaas charge/payment identifier (e.g. "pay_xxx").
	PaymentID string `json:"payment_id,omitempty"`

	// SubscriptionID is the Asaas subscription identifier (e.g. "sub_xxx").
	SubscriptionID string `json:"subscription_id,omitempty"`

	// ExternalReference is the external_reference field from the payment object.
	ExternalReference string `json:"external_reference,omitempty"`

	// ErrorStage indicates where in the processing pipeline the failure occurred.
	// Expected values: "decode_payload", "resolve_contract_context", "upsert_charge".
	ErrorStage string `json:"error_stage,omitempty"`

	// ErrorMessage is the human-readable description of what went wrong.
	ErrorMessage string `json:"error_message,omitempty"`

	// RawPayload is the full JSON body received from the Asaas webhook.
	// Stored as JSONB for reprocessing and forensic analysis.
	RawPayload json.RawMessage `json:"raw_payload,omitempty"`
}
