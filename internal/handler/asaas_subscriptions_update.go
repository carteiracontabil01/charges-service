package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/seuuser/charges-service/internal/integrations/asaas"
	"github.com/seuuser/charges-service/internal/model"
	"github.com/seuuser/charges-service/internal/supabase"
)

// UpdateAsaasSubscription godoc
// @Summary      Atualizar assinatura existente no Asaas
// @Description  Atualiza uma assinatura (subscription) no Asaas. O parâmetro nextDueDate indica o vencimento da PRÓXIMA mensalidade a ser gerada (não altera a já existente). Para atualizar mensalidades pendentes existentes com o novo valor/forma de pagamento, passe updatePendingPayments=true.
// @Tags         asaas
// @Accept       json
// @Produce      json
// @Param        id           path      string  true  "ID da assinatura no Asaas (sub_...)"
// @Param        contract_id  query     string  true  "ID do contrato (UUID) — usado para resolver a integração"
// @Param        body         body      model.AsaasUpdateSubscriptionRequest  true  "Payload da atualização (todos os campos são opcionais)"
// @Success      200  {object}  model.AsaasSubscriptionResponse
// @Failure      400  {object}  map[string]any
// @Failure      404  {object}  map[string]any
// @Failure      502  {object}  map[string]any
// @Router       /v1/asaas/subscriptions/{id} [put]
func UpdateAsaasSubscription(w http.ResponseWriter, r *http.Request) {
	rid := newRequestID()

	subscriptionID := strings.TrimSpace(chi.URLParam(r, "id"))
	if subscriptionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "subscription id is required"})
		return
	}

	contractID := strings.TrimSpace(r.URL.Query().Get("contract_id"))
	if contractID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "contract_id is required"})
		return
	}

	var req model.AsaasUpdateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json body"})
		return
	}

	// At least one updatable field must be present
	if req.BillingType == nil &&
		req.Status == nil &&
		req.Value == nil &&
		req.NextDueDate == nil &&
		req.Cycle == nil &&
		req.Description == nil &&
		req.EndDate == nil &&
		req.UpdatePendingPayments == nil &&
		req.ExternalReference == nil &&
		req.Discount == nil &&
		req.Interest == nil &&
		req.Fine == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "no updatable fields provided"})
		return
	}

	if isDebugEnabled() {
		log.Printf("[asaas] update subscription request: rid=%s subscription_id=%s contract_id=%s",
			rid, subscriptionID, contractID,
		)
	}

	// Resolve contract → billing integration (same strategy as CreateAsaasSubscription)
	contract, err := supabase.GetFeeContractByID(contractID)
	if err != nil {
		if isDebugEnabled() {
			log.Printf("[supabase] ERROR loading fee_contract: rid=%s contract_id=%s err=%v", rid, contractID, err)
		}
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to load contract", "request_id": rid})
		return
	}
	if contract == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "contract not found"})
		return
	}

	provider := "ASAAS"
	if contract.Provider != nil && strings.TrimSpace(*contract.Provider) != "" {
		provider = normalizeProvider(*contract.Provider)
	}

	var cfg *model.BillingIntegrationRow
	if contract.BillingIntegrationID != nil && strings.TrimSpace(*contract.BillingIntegrationID) != "" {
		cfg, err = supabase.GetBillingIntegrationByID(strings.TrimSpace(*contract.BillingIntegrationID))
	} else if contract.ProviderEnvironment != nil && strings.TrimSpace(*contract.ProviderEnvironment) != "" {
		cfg, err = supabase.GetBillingIntegrationForOfficeAndEnvironment(contract.AccountingOfficeID, provider, strings.TrimSpace(*contract.ProviderEnvironment))
		if err != nil {
			cfg, err = supabase.GetBillingIntegrationForOffice(contract.AccountingOfficeID, provider)
		}
	} else {
		cfg, err = supabase.GetBillingIntegrationForOffice(contract.AccountingOfficeID, provider)
	}

	if err != nil || cfg == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":      "billing integration not found for contract/office/provider",
			"contract_id": contractID,
			"provider":   provider,
		})
		return
	}
	if strings.TrimSpace(cfg.BaseAPI) == "" || strings.TrimSpace(cfg.Token) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "integration config missing base_api or token"})
		return
	}

	if isDebugEnabled() {
		log.Printf(
			"[asaas] update subscription: rid=%s sub_id=%s cfg_id=%s provider=%s env=%s base_api=%q token=%s",
			rid, subscriptionID, cfg.ID, cfg.Provider, cfg.Environment, cfg.BaseAPI, maskToken(cfg.Token),
		)
	}

	client := asaas.NewClient(cfg.BaseAPI, cfg.Token)

	// Map model → integration request
	updateReq := mapUpdateSubscriptionRequest(req)

	status, body, callErr := client.UpdateSubscription(subscriptionID, updateReq)
	if callErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": callErr.Error()})
		return
	}

	if isDebugEnabled() {
		raw := string(body)
		if len(raw) > 800 {
			raw = raw[:800] + "…(truncated)"
		}
		log.Printf("[asaas] update subscription response: rid=%s status=%d body=%s", rid, status, raw)
	}

	// Pass-through Asaas payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func mapUpdateSubscriptionRequest(req model.AsaasUpdateSubscriptionRequest) asaas.UpdateSubscriptionRequest {
	out := asaas.UpdateSubscriptionRequest{}

	if req.BillingType != nil {
		s := string(*req.BillingType)
		out.BillingType = &s
	}
	out.Status = req.Status
	out.Value = req.Value
	out.NextDueDate = req.NextDueDate
	out.Cycle = req.Cycle
	out.Description = req.Description
	out.EndDate = req.EndDate
	out.UpdatePendingPayments = req.UpdatePendingPayments
	out.ExternalReference = req.ExternalReference

	if req.Discount != nil {
		var t *string
		if req.Discount.Type != nil {
			s := string(*req.Discount.Type)
			t = &s
		}
		out.Discount = &asaas.PaymentDiscount{
			Value:            req.Discount.Value,
			DueDateLimitDays: req.Discount.DueDateLimitDays,
			Type:             t,
		}
	}
	if req.Interest != nil {
		out.Interest = &asaas.PaymentInterest{Value: req.Interest.Value}
	}
	if req.Fine != nil {
		var t *string
		if req.Fine.Type != nil {
			s := string(*req.Fine.Type)
			t = &s
		}
		out.Fine = &asaas.PaymentFine{Value: req.Fine.Value, Type: t}
	}

	return out
}
