package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

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

	// Validate access token (if configured in Asaas webhook settings)
	expectedToken := os.Getenv("ASAAS_WEBHOOK_SECRET")
	if expectedToken != "" {
		providedToken := r.Header.Get("asaas-access-token")
		log.Printf("🔐 [webhook] Validando token de acesso (esperado=%t, fornecido=%t)", expectedToken != "", providedToken != "")
		if providedToken != expectedToken {
			log.Printf("❌ [webhook] ERRO: Token inválido ou não fornecido")
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		log.Printf("✅ [webhook] Token validado com sucesso")
	} else {
		log.Printf("⚠️  [webhook] ASAAS_WEBHOOK_SECRET não configurado - webhook sem autenticação")
	}

	var event model.AsaasWebhookEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("❌ [webhook] ERRO ao decodificar JSON: %v", err)
		log.Printf("❌ [webhook] Body recebido: %s", r.Body)
		http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	log.Printf("📦 [webhook] Evento recebido: %s | ID=%s | DateCreated=%s", event.Event, event.ID, event.DateCreated)
	
	// Log do payload completo para debug
	eventJSON, _ := json.MarshalIndent(event, "", "  ")
	log.Printf("📄 [webhook] Payload completo:\n%s", string(eventJSON))

	// Only process payment-related events
	if event.Payment == nil {
		log.Printf("⚠️  [webhook] Evento %s não possui objeto payment, pulando processamento", event.Event)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"received": true, "processed": false, "reason": "no_payment_object"})
		log.Printf("✅ [webhook] ========== WEBHOOK FINALIZADO (SEM PROCESSAMENTO) ==========\n")
		return
	}

	log.Printf("💳 [webhook] Processando payment: ID=%s | Status=%s | Value=%.2f", event.Payment.ID, event.Payment.Status, event.Payment.Value)

	// Update charge status in iam.charges
	log.Printf("💾 [webhook] Iniciando atualização da cobrança no banco de dados...")
	if err := updateChargeFromWebhook(event); err != nil {
		log.Printf("❌ [webhook] ERRO CRÍTICO ao atualizar cobrança: %v", err)
		log.Printf("❌ [webhook] Event: %s | Payment ID: %s", event.Event, event.Payment.ID)
		log.Printf("❌ [webhook] Stack trace disponível para análise")
		http.Error(w, `{"error":"Failed to update charge"}`, http.StatusInternalServerError)
		log.Printf("❌ [webhook] ========== WEBHOOK FINALIZADO COM ERRO ==========\n")
		return
	}

	log.Printf("✅ [webhook] Cobrança atualizada com sucesso!")
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

// updateChargeFromWebhook updates the charge in iam.charges based on the webhook event.
func updateChargeFromWebhook(event model.AsaasWebhookEvent) error {
	log.Printf("🔄 [updateCharge] Iniciando atualização da cobrança...")
	
	if event.Payment == nil {
		log.Printf("⚠️  [updateCharge] Payment é nil, retornando sem atualização")
		return nil
	}

	p := event.Payment
	log.Printf("🔍 [updateCharge] Mapeando payment: ID=%s | Status=%s | Value=%.2f", p.ID, p.Status, p.Value)

	// Map Asaas payment to IamChargeRow
	charge := model.IamChargeRow{
		Provider:          "ASAAS",
		ProviderChargeID:  p.ID,
		Value:             p.Value,
		NetValue:          &p.NetValue,
		Description:       &p.Description,
		BillingType:       &p.BillingType,
		Status:            &p.Status,
		DueDate:           &p.DueDate,
		OriginalDueDate:   &p.OriginalDueDate,
		InvoiceURL:        &p.InvoiceURL,
		InvoiceNumber:     &p.InvoiceNumber,
		ExternalReference: &p.ExternalReference,
	}

	// Store the full payment object as JSONB for reference
	payloadBytes, _ := json.Marshal(p)
	charge.ProviderPayload = payloadBytes
	log.Printf("📦 [updateCharge] Payload serializado com sucesso (%d bytes)", len(payloadBytes))

	// Fetch tenant_id, accounting_office_id, company_id, contract_id from existing charge
	// (these were set when the charge was created via CreateAsaasCharge)
	log.Printf("🔎 [updateCharge] Buscando cobrança existente no banco (provider_charge_id=%s)...", p.ID)
	existingCharge, err := supabase.GetChargeByProviderID("ASAAS", p.ID)
	if err != nil {
		log.Printf("⚠️  [updateCharge] Cobrança não encontrada no banco (provider_charge_id=%s)", p.ID)
		log.Printf("⚠️  [updateCharge] Erro: %v", err)
		log.Printf("⚠️  [updateCharge] Pulando atualização pois não há contexto de tenant/company")
		// If the charge doesn't exist yet, we can't upsert without tenant/company context.
		// In production, you may want to fetch these from external_reference or skip the update.
		return nil
	}

	log.Printf("✅ [updateCharge] Cobrança encontrada! TenantID=%s | CompanyID=%s | ContractID=%s", 
		existingCharge.TenantID, 
		func() string { if existingCharge.CompanyID != "" { return existingCharge.CompanyID } else { return "null" } }(),
		func() string { if existingCharge.ContractID != "" { return existingCharge.ContractID } else { return "null" } }())

	charge.TenantID = existingCharge.TenantID
	charge.AccountingOfficeID = existingCharge.AccountingOfficeID
	charge.CompanyID = existingCharge.CompanyID
	charge.ContractID = existingCharge.ContractID
	charge.InstallmentNumber = existingCharge.InstallmentNumber
	charge.ProviderInstallmentID = existingCharge.ProviderInstallmentID

	// Upsert the charge
	log.Printf("💾 [updateCharge] Executando upsert no banco de dados...")
	if err := supabase.UpsertCharges([]model.IamChargeRow{charge}); err != nil {
		log.Printf("❌ [updateCharge] ERRO ao fazer upsert: %v", err)
		return err
	}
	
	log.Printf("✅ [updateCharge] Upsert executado com sucesso!")
	return nil
}
