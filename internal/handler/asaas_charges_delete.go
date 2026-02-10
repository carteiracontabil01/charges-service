package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/seuuser/charges-service/internal/integrations/asaas"
	"github.com/seuuser/charges-service/internal/supabase"
)

// DeleteAsaasCharge godoc
// @Summary Excluir cobrança do Asaas
// @Description Exclui uma cobrança (payment) do Asaas e remove do banco de dados iam.charges
// @Tags asaas
// @Accept json
// @Produce json
// @Param id path string true "ID da cobrança no Asaas (pay_xxx)"
// @Param accounting_office_id query string true "ID do escritório contábil"
// @Success 200 {object} map[string]interface{} "Cobrança excluída com sucesso"
// @Failure 400 {object} map[string]interface{} "Requisição inválida"
// @Failure 404 {object} map[string]interface{} "Cobrança não encontrada"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /v1/asaas/charges/{id} [delete]
func DeleteAsaasCharge(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "id")
	if paymentID == "" {
		http.Error(w, `{"error":"payment ID is required"}`, http.StatusBadRequest)
		return
	}

	accountingOfficeID := strings.TrimSpace(r.URL.Query().Get("accounting_office_id"))
	if accountingOfficeID == "" {
		http.Error(w, `{"error":"accounting_office_id is required"}`, http.StatusBadRequest)
		return
	}

	log.Printf("[asaas] DELETE /v1/asaas/charges/%s | office=%s", paymentID, accountingOfficeID)

	// Get billing integration for current tenant
	integration, err := supabase.GetBillingIntegrationForOffice(accountingOfficeID, normalizeProvider("ASAAS"))
	if err != nil {
		log.Printf("[asaas] ERROR get billing integration: %v", err)
		http.Error(w, `{"error":"billing integration not found"}`, http.StatusInternalServerError)
		return
	}

	// Delete payment from Asaas
	asaasClient := asaas.NewClient(integration.BaseAPI, integration.Token)
	statusCode, body, err := asaasClient.DeletePayment(paymentID)
	if err != nil {
		log.Printf("[asaas] ERROR delete payment: %v", err)
		http.Error(w, fmt.Sprintf(`{"error":"failed to delete payment: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	if statusCode != http.StatusOK && statusCode != http.StatusNoContent {
		log.Printf("[asaas] ERROR delete payment returned status %d: %s", statusCode, string(body))
		http.Error(w, string(body), statusCode)
		return
	}

	// Delete charge from iam.charges
	if err := supabase.DeleteChargeByProviderID("ASAAS", paymentID); err != nil {
		log.Printf("[asaas] WARNING failed to delete charge from database: %v", err)
		// Don't fail the request if we can't delete from database
		// The charge was already deleted from Asaas
	} else {
		log.Printf("[asaas] ✅ Charge deleted from database: %s", paymentID)
	}

	log.Printf("[asaas] ✅ Payment deleted successfully from Asaas: %s", paymentID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Cobrança excluída com sucesso",
		"id":      paymentID,
	})
}
