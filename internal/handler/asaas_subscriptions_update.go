package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
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

	// When updatePendingPayments=true and the update succeeded, re-sync the affected
	// pending payments to iam.charges so the local mirror reflects the new value/billing type.
	if status >= 200 && status < 300 &&
		req.UpdatePendingPayments != nil && *req.UpdatePendingPayments {
		syncSubscriptionChargesToIAM(rid, client, contract, subscriptionID)
	}

	// Pass-through Asaas payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// syncSubscriptionChargesToIAM lists all PENDING payments for a subscription in Asaas
// and upserts them into iam.charges to keep the local mirror in sync.
// Non-fatal: errors are logged but do not affect the HTTP response already sent.
func syncSubscriptionChargesToIAM(
	rid string,
	client *asaas.Client,
	contract *model.FeeContractRow,
	subscriptionID string,
) {
	listParams := url.Values{}
	listParams.Set("subscription", subscriptionID)
	listParams.Set("status", "PENDING")
	listParams.Set("limit", "100")
	listParams.Set("offset", "0")

	listStatus, listBody, listErr := client.ListPayments(listParams)
	if listErr != nil {
		log.Printf("[asaas] syncSubscriptionChargesToIAM: ERROR listing payments: rid=%s sub=%s err=%v",
			rid, subscriptionID, listErr)
		return
	}
	if listStatus < 200 || listStatus >= 300 {
		log.Printf("[asaas] syncSubscriptionChargesToIAM: non-2xx listing payments: rid=%s sub=%s status=%d",
			rid, subscriptionID, listStatus)
		return
	}

	var list model.AsaasPaymentsListResponse
	if err := json.Unmarshal(listBody, &list); err != nil {
		log.Printf("[asaas] syncSubscriptionChargesToIAM: ERROR unmarshalling payment list: rid=%s err=%v", rid, err)
		return
	}

	subIDPtr := subscriptionID
	rows := make([]model.IamChargeRow, 0, len(list.Data))

	for _, p := range list.Data {
		if strings.TrimSpace(p.ID) == "" {
			continue
		}

		var netPtr *float64
		if p.NetValue != 0 {
			v := p.NetValue
			netPtr = &v
		}
		toStrPtr := func(s string) *string {
			if s == "" {
				return nil
			}
			return &s
		}
		payload, _ := json.Marshal(p)

		rows = append(rows, model.IamChargeRow{
			TenantID:               contract.TenantID,
			AccountingOfficeID:     contract.AccountingOfficeID,
			CompanyID:              contract.CompanyID,
			ContractID:             contract.ID,
			Provider:               "ASAAS",
			ProviderChargeID:       p.ID,
			ProviderSubscriptionID: &subIDPtr,
			Value:                  p.Value,
			NetValue:               netPtr,
			Description:            toStrPtr(p.Description),
			BillingType:            toStrPtr(p.BillingType),
			Status:                 toStrPtr(p.Status),
			DueDate:                toStrPtr(p.DueDate),
			OriginalDueDate:        toStrPtr(p.OriginalDueDate),
			InvoiceURL:             toStrPtr(p.InvoiceURL),
			InvoiceNumber:          toStrPtr(p.InvoiceNumber),
			ExternalReference:      toStrPtr(p.ExternalReference),
			ProviderPayload:        payload,
		})
	}

	if len(rows) == 0 {
		if isDebugEnabled() {
			log.Printf("[asaas] syncSubscriptionChargesToIAM: no PENDING payments found for sub=%s", subscriptionID)
		}
		return
	}

	if err := supabase.UpsertCharges(rows); err != nil {
		log.Printf("[asaas] syncSubscriptionChargesToIAM: ERROR upserting iam.charges: rid=%s sub=%s err=%v",
			rid, subscriptionID, err)
		return
	}

	log.Printf("[asaas] syncSubscriptionChargesToIAM: upserted %d charges for sub=%s", len(rows), subscriptionID)
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
