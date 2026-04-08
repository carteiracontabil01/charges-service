package supabase

import (
	"fmt"
	"log"

	"github.com/seuuser/charges-service/internal/model"
)

// InsertAsaasWebhookEventLog persists a failed or unprocessable Asaas webhook
// event into logs.asaas_webhook_events for diagnostics and manual resolution.
//
// This function is intentionally non-fatal: if the insert itself fails, it logs
// the error to stdout but does NOT propagate it — the caller should never fail
// a webhook response just because the log write failed.
func InsertAsaasWebhookEventLog(entry model.AsaasWebhookEventLog) {
	c := GetLogsClient()
	if c == nil {
		log.Printf("⚠️  [webhook_log] logs client não inicializado — impossível persistir log. stage=%s err=%s",
			entry.ErrorStage, entry.ErrorMessage)
		return
	}

	_, _, err := c.
		From("asaas_webhook_events").
		Insert(entry, false, "", "minimal", "").
		Execute()
	if err != nil {
		// Last resort: log to stdout — do NOT propagate.
		log.Printf("❌ [webhook_log] ERRO ao persistir em logs.asaas_webhook_events: %v | event=%s payment=%s stage=%s",
			err, entry.EventType, entry.PaymentID, entry.ErrorStage)
		return
	}

	log.Printf("📝 [webhook_log] Log registrado: event=%s payment=%s stage=%s",
		entry.EventType, entry.PaymentID, entry.ErrorStage)
}

// MustInsertAsaasWebhookEventLog is like InsertAsaasWebhookEventLog but returns
// an error. Use this when the caller needs to be aware of log write failures
// (e.g. in integration tests).
func MustInsertAsaasWebhookEventLog(entry model.AsaasWebhookEventLog) error {
	c := GetLogsClient()
	if c == nil {
		return fmt.Errorf("logs client não inicializado")
	}

	_, _, err := c.
		From("asaas_webhook_events").
		Insert(entry, false, "", "minimal", "").
		Execute()
	if err != nil {
		return fmt.Errorf("failed to insert asaas webhook event log: %w", err)
	}
	return nil
}
