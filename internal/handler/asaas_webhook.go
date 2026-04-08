package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/seuuser/charges-service/internal/model"
	"github.com/seuuser/charges-service/internal/supabase"
)

// ReceiveAsaasWebhook godoc
// @Summary Recebe eventos do Asaas via webhook
// @Description Endpoint para receber notificações de atualização de status de cobranças do Asaas
// @Tags Webhook
// @Accept json
// @Produce json
// @Param asaas-access-token header string false "Token de acesso configurado no webhook do Asaas"
// @Param event body model.AsaasWebhookEvent true "Evento do webhook"
// @Success 200 {object} map[string]interface{} "Evento recebido com sucesso"
// @Failure 401 {object} map[string]interface{} "Não autorizado"
// @Failure 400 {object} map[string]interface{} "JSON inválido"
// @Failure 500 {object} map[string]interface{} "Erro interno"
// @Router /asaas/feecharges [post]
func ReceiveAsaasWebhook(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔔 [webhook] ========== WEBHOOK ASAAS INICIADO ==========")
	log.Printf("🔔 [webhook] Method: %s | Path: %s | RemoteAddr: %s", r.Method, r.URL.Path, r.RemoteAddr)
	log.Printf("🔔 [webhook] Headers: User-Agent=%s | Content-Type=%s", r.Header.Get("User-Agent"), r.Header.Get("Content-Type"))

	// ── Auth ──────────────────────────────────────────────────────────────────
	expectedToken := os.Getenv("ASAAS_WEBHOOK_SECRET")
	if expectedToken != "" {
		providedToken := r.Header.Get("asaas-access-token")
		if providedToken != expectedToken {
			log.Printf("❌ [webhook] Token inválido ou não fornecido")
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		log.Printf("✅ [webhook] Token validado")
	} else {
		log.Printf("⚠️  [webhook] ASAAS_WEBHOOK_SECRET não configurado — webhook sem autenticação")
	}

	// ── Read raw body (needed for logs) ───────────────────────────────────────
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ [webhook] ERRO ao ler body: %v", err)
		http.Error(w, `{"error":"Failed to read request body"}`, http.StatusBadRequest)
		return
	}
	rawPayload := json.RawMessage(rawBody)

	// ── Decode JSON ───────────────────────────────────────────────────────────
	var event model.AsaasWebhookEvent
	if err := json.Unmarshal(rawBody, &event); err != nil {
		log.Printf("❌ [webhook] ERRO ao decodificar JSON: %v", err)
		supabase.InsertAsaasWebhookEventLog(model.AsaasWebhookEventLog{
			ErrorStage:   "decode_payload",
			ErrorMessage: fmt.Sprintf("falha ao decodificar JSON do webhook: %v", err),
			RawPayload:   rawPayload,
		})
		http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	log.Printf("📦 [webhook] Evento recebido: %s | ID=%s | DateCreated=%s", event.Event, event.ID, event.DateCreated)

	// Log do payload completo para debug
	eventJSON, _ := json.MarshalIndent(event, "", "  ")
	log.Printf("📄 [webhook] Payload completo:\n%s", string(eventJSON))

	// ── Skip non-payment events ───────────────────────────────────────────────
	if event.Payment == nil {
		log.Printf("⚠️  [webhook] Evento %s sem objeto payment, pulando", event.Event)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"received": true, "processed": false, "reason": "no_payment_object"})
		log.Printf("✅ [webhook] ========== WEBHOOK FINALIZADO (SEM PROCESSAMENTO) ==========\n")
		return
	}

	log.Printf("💳 [webhook] Processando payment: ID=%s | Status=%s | Value=%.2f",
		event.Payment.ID, event.Payment.Status, event.Payment.Value)

	// ── Process ───────────────────────────────────────────────────────────────
	if err := updateChargeFromWebhook(event, rawPayload); err != nil {
		log.Printf("❌ [webhook] ERRO CRÍTICO ao processar cobrança: %v", err)
		http.Error(w, `{"error":"Failed to update charge"}`, http.StatusInternalServerError)
		log.Printf("❌ [webhook] ========== WEBHOOK FINALIZADO COM ERRO ==========\n")
		return
	}

	log.Printf("✅ [webhook] Payment ID: %s | Novo Status: %s", event.Payment.ID, event.Payment.Status)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"received":   true,
		"processed":  true,
		"payment_id": event.Payment.ID,
		"status":     event.Payment.Status,
	})
	log.Printf("✅ [webhook] ========== WEBHOOK FINALIZADO COM SUCESSO ==========\n")
}

// updateChargeFromWebhook upserts the charge in iam.charges based on the webhook event.
//
// For existing charges (already in iam.charges) it refreshes all mutable fields.
// For new charges (e.g. auto-generated installments from a subscription) it resolves
// the contract context via the subscription ID and inserts them for the first time.
//
// rawPayload is the original JSON body and is used exclusively for error logging.
func updateChargeFromWebhook(event model.AsaasWebhookEvent, rawPayload json.RawMessage) error {
	log.Printf("🔄 [updateCharge] Iniciando processamento da cobrança...")

	if event.Payment == nil {
		return nil
	}

	p := event.Payment
	log.Printf("🔍 [updateCharge] payment: ID=%s | Status=%s | Value=%.2f | Sub=%s | ExtRef=%s",
		p.ID, p.Status, p.Value, p.Subscription, p.ExternalReference)

	// ── Build base charge row ─────────────────────────────────────────────────
	toPtr := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}

	charge := model.IamChargeRow{
		Provider:          "ASAAS",
		ProviderChargeID:  p.ID,
		Value:             p.Value,
		NetValue:          &p.NetValue,
		Description:       toPtr(p.Description),
		BillingType:       toPtr(p.BillingType),
		Status:            toPtr(p.Status),
		DueDate:           toPtr(p.DueDate),
		OriginalDueDate:   toPtr(p.OriginalDueDate),
		InvoiceURL:        toPtr(p.InvoiceURL),
		InvoiceNumber:     toPtr(p.InvoiceNumber),
		ExternalReference: toPtr(p.ExternalReference),
	}
	if p.Subscription != "" {
		charge.ProviderSubscriptionID = &p.Subscription
	}

	payloadBytes, _ := json.Marshal(p)
	charge.ProviderPayload = payloadBytes

	// ── Try to find existing charge ───────────────────────────────────────────
	log.Printf("🔎 [updateCharge] Buscando cobrança existente (provider_charge_id=%s)...", p.ID)
	existingCharge, err := supabase.GetChargeByProviderID("ASAAS", p.ID)
	if err != nil {
		// Charge not in iam.charges yet — likely PAYMENT_CREATED for a subscription installment.
		// Resolve the contract context so we can insert it.
		log.Printf("⚠️  [updateCharge] Cobrança não encontrada no banco (id=%s) — resolvendo contexto...", p.ID)

		contract, resolveErr := resolveContractContextFromPayment(p)
		if resolveErr != nil || contract == nil {
			msg := fmt.Sprintf("contrato não encontrado para payment=%s sub=%q extRef=%q", p.ID, p.Subscription, p.ExternalReference)
			log.Printf("⚠️  [updateCharge] %s — persistindo log de erro", msg)

			supabase.InsertAsaasWebhookEventLog(model.AsaasWebhookEventLog{
				EventType:         event.Event,
				PaymentID:         p.ID,
				SubscriptionID:    p.Subscription,
				ExternalReference: p.ExternalReference,
				ErrorStage:        "resolve_contract_context",
				ErrorMessage:      msg,
				RawPayload:        rawPayload,
			})
			// Non-fatal: return nil so Asaas receives 200 and doesn't retry indefinitely.
			return nil
		}

		charge.TenantID           = contract.TenantID
		charge.AccountingOfficeID = contract.AccountingOfficeID
		charge.CompanyID          = contract.CompanyID
		charge.ContractID         = contract.ID

		log.Printf("✅ [updateCharge] Contexto resolvido: contract=%s tenant=%s company=%s",
			contract.ID, contract.TenantID, contract.CompanyID)
	} else {
		// Existing charge — preserve immutable context fields
		log.Printf("✅ [updateCharge] Cobrança existente encontrada: tenant=%s contract=%s",
			existingCharge.TenantID, existingCharge.ContractID)

		charge.TenantID               = existingCharge.TenantID
		charge.AccountingOfficeID     = existingCharge.AccountingOfficeID
		charge.CompanyID              = existingCharge.CompanyID
		charge.ContractID             = existingCharge.ContractID
		charge.InstallmentNumber      = existingCharge.InstallmentNumber
		charge.ProviderInstallmentID  = existingCharge.ProviderInstallmentID
		if charge.ProviderSubscriptionID == nil {
			charge.ProviderSubscriptionID = existingCharge.ProviderSubscriptionID
		}
	}

	// ── Upsert ───────────────────────────────────────────────────────────────
	log.Printf("💾 [updateCharge] Executando upsert (tenant=%s, charge=%s)...", charge.TenantID, charge.ProviderChargeID)
	if upsertErr := supabase.UpsertCharges([]model.IamChargeRow{charge}); upsertErr != nil {
		msg := fmt.Sprintf("erro ao fazer upsert em iam.charges: %v", upsertErr)
		log.Printf("❌ [updateCharge] %s", msg)

		supabase.InsertAsaasWebhookEventLog(model.AsaasWebhookEventLog{
			EventType:         event.Event,
			PaymentID:         p.ID,
			SubscriptionID:    p.Subscription,
			ExternalReference: p.ExternalReference,
			ErrorStage:        "upsert_charge",
			ErrorMessage:      msg,
			RawPayload:        rawPayload,
		})
		return upsertErr
	}

	log.Printf("✅ [updateCharge] Upsert concluído! payment=%s status=%s", p.ID, p.Status)
	return nil
}

// resolveContractContextFromPayment attempts to find the fee contract associated with
// a payment by looking up the subscription ID in iam.fee_contract_subscriptions.
// Falls back to parsing the external_reference field as "fee_contract:{uuid}:...".
//
// Returns (nil, nil) when the context cannot be determined (non-fatal).
func resolveContractContextFromPayment(p *model.AsaasPaymentResponse) (*model.FeeContractRow, error) {
	// ── Strategy 1: via subscription ID (most reliable for recurring charges) ──
	if subID := strings.TrimSpace(p.Subscription); subID != "" {
		log.Printf("🔗 [resolveContext] Tentando resolver via subscription id=%s", subID)
		contract, err := supabase.GetFeeContractBySubscriptionProviderID(subID)
		if err == nil && contract != nil {
			return contract, nil
		}
		log.Printf("⚠️  [resolveContext] Lookup por subscription falhou (sub=%s): %v", subID, err)
	}

	// ── Strategy 2: via external_reference "fee_contract:{uuid}:..." ─────────
	if contractID := contractIDFromExtRef(p.ExternalReference); contractID != "" {
		log.Printf("🔗 [resolveContext] Tentando resolver via external_reference contract_id=%s", contractID)
		contract, err := supabase.GetFeeContractByID(contractID)
		if err == nil && contract != nil {
			return contract, nil
		}
		log.Printf("⚠️  [resolveContext] Lookup por external_reference falhou (contract_id=%s): %v", contractID, err)
	}

	return nil, fmt.Errorf("não foi possível resolver o contrato para payment=%s sub=%q extRef=%q",
		p.ID, p.Subscription, p.ExternalReference)
}

// contractIDFromExtRef extracts the contract UUID from the external_reference format
// "fee_contract:{uuid}:..." used when creating charges/subscriptions.
func contractIDFromExtRef(extRef string) string {
	parts := strings.SplitN(strings.TrimSpace(extRef), ":", 3)
	if len(parts) >= 2 && parts[0] == "fee_contract" {
		return strings.TrimSpace(parts[1])
	}
	return ""
}
