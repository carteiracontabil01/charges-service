package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/seuuser/charges-service/internal/integrations/asaas"
	"github.com/seuuser/charges-service/internal/model"
	"github.com/seuuser/charges-service/internal/supabase"
)

// UpdateAsaasCharge godoc
// @Summary      Atualizar cobrança existente no Asaas
// @Description  Atualiza uma cobrança (payment) no Asaas. Somente é possível atualizar cobranças aguardando pagamento ou vencidas. Uma vez criada, não é possível alterar o cliente ao qual a cobrança pertence. Para atualizar split após confirmação, existem regras específicas (ver documentação Asaas).
// @Tags         asaas
// @Accept       json
// @Produce      json
// @Param        id                    path      string  true   "ID da cobrança no Asaas (payment_id)"
// @Param        accounting_office_id  query     string  true   "ID do accounting_office (UUID)"
// @Param        body                  body      model.AsaasUpdateChargeRequest  true  "Payload da atualização"
// @Success      200  {object}  model.AsaasPaymentResponse
// @Failure      400  {object}  map[string]any
// @Failure      404  {object}  map[string]any
// @Failure      502  {object}  map[string]any
// @Router       /v1/asaas/charges/{id} [put]
func UpdateAsaasCharge(w http.ResponseWriter, r *http.Request) {
	rid := newRequestID()
	paymentID := strings.TrimSpace(chi.URLParam(r, "id"))
	if paymentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "payment id is required"})
		return
	}

	accountingOfficeID := strings.TrimSpace(r.URL.Query().Get("accounting_office_id"))
	if accountingOfficeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "accounting_office_id is required"})
		return
	}

	var req model.AsaasUpdateChargeRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json body"})
		return
	}

	// Validate: at least one field must be provided
	if req.Value == nil && req.DueDate == nil && req.Discount == nil && req.Interest == nil && req.Fine == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "no updatable fields provided"})
		return
	}

	if isDebugEnabled() {
		log.Printf("[asaas] update charge request received: rid=%s payment_id=%s accounting_office_id=%s",
			rid, paymentID, accountingOfficeID,
		)
	}

	// Resolve integration config:
	// We should NOT guess by office/provider only, because an office may have multiple integrations
	// and the contract explicitly selects one (billing_integration_id).
	//
	// Strategy:
	// 1) Load charge (iam.charges) by payment_id + accounting_office_id
	// 2) Load contract (iam.fee_contracts) by charge.contract_id
	// 3) If contract has billing_integration_id, use it; else fallback to env/default like subscriptions handler
	chargeRow, chErr := supabase.GetChargeByProviderIDAndOffice("ASAAS", paymentID, accountingOfficeID)
	if chErr != nil || chargeRow == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":                "charge not found for office/provider",
			"accounting_office_id": accountingOfficeID,
			"provider":             "ASAAS",
			"provider_charge_id":   paymentID,
		})
		return
	}

	contract, cErr := supabase.GetFeeContractByID(chargeRow.ContractID)
	if cErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to load contract", "request_id": rid})
		return
	}
	if contract == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "contract not found", "contract_id": chargeRow.ContractID})
		return
	}

	provider := "ASAAS"
	if contract.Provider != nil && strings.TrimSpace(*contract.Provider) != "" {
		provider = normalizeProvider(*contract.Provider)
	}

	var cfg *model.BillingIntegrationRow
	var err error
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
			"error":                "billing integration not found for contract/office/provider",
			"accounting_office_id": contract.AccountingOfficeID,
			"provider":             provider,
			"contract_id":          contract.ID,
			"billing_integration_id": func() any {
				if contract.BillingIntegrationID != nil {
					return strings.TrimSpace(*contract.BillingIntegrationID)
				}
				return nil
			}(),
			"hint": "Garanta que o contrato tenha billing_integration_id válido e que exista registro ativo em iam.billing_integrations (is_active=true).",
		})
		return
	}
	if isDebugEnabled() {
		log.Printf(
			"[asaas] using integration cfg for charge update: id=%s provider=%s env=%s base_api=%q token=%s",
			cfg.ID,
			cfg.Provider,
			cfg.Environment,
			cfg.BaseAPI,
			maskToken(cfg.Token),
		)
	}
	if strings.TrimSpace(cfg.BaseAPI) == "" || strings.TrimSpace(cfg.Token) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "integration config missing base_api or token"})
		return
	}

	client := asaas.NewClient(cfg.BaseAPI, cfg.Token)

	// Build update request
	updateReq := mapUpdatePaymentRequest(req)

	status, body, callErr := client.UpdatePayment(paymentID, updateReq)
	if callErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": callErr.Error()})
		return
	}

	if isDebugEnabled() {
		raw := string(body)
		if len(raw) > 800 {
			raw = raw[:800] + "…(truncated)"
		}
		log.Printf("[asaas] update charge response: rid=%s status=%d body=%s", rid, status, raw)
	}

	// If 2xx, sync both iam.charges and (if one-off) iam.fee_contract_one_off_charges
	if status >= 200 && status < 300 {
		var updated model.AsaasPaymentResponse
		if err := json.Unmarshal(body, &updated); err == nil && strings.TrimSpace(updated.ID) != "" {
			// Find the existing charge in our database to get required IDs
			charge, fetchErr := supabase.GetChargeByProviderID("ASAAS", updated.ID)
			if fetchErr != nil {
				if isDebugEnabled() {
					log.Printf("[supabase] ERROR fetching charge for update: payment_id=%s err=%v", updated.ID, fetchErr)
				}
				// Non-fatal: we updated in Asaas but couldn't sync to our DB
			} else if charge != nil {
				now := time.Now().UTC().Format(time.RFC3339)

				// ── helpers: pointer from non-empty string ──────────────────
				strPtr := func(s string) *string {
					s = strings.TrimSpace(s)
					if s == "" {
						return nil
					}
					return &s
				}

				var netPtr *float64
				if updated.NetValue != 0 {
					v := updated.NetValue
					netPtr = &v
				}

				rawPayload, _ := json.Marshal(updated)

				// ── 1. Upsert iam.charges ────────────────────────────────────
				row := model.IamChargeRow{
					TenantID:               charge.TenantID,
					AccountingOfficeID:     charge.AccountingOfficeID,
					CompanyID:              charge.CompanyID,
					ContractID:             charge.ContractID,
					Provider:               "ASAAS",
					ProviderChargeID:       updated.ID,
					ProviderInstallmentID:  charge.ProviderInstallmentID,
					ProviderSubscriptionID: charge.ProviderSubscriptionID,
					InstallmentNumber:      charge.InstallmentNumber,
					Value:                  updated.Value,
					NetValue:               netPtr,
					Description:            strPtr(updated.Description),
					BillingType:            strPtr(updated.BillingType),
					Status:                 strPtr(updated.Status),
					DueDate:                strPtr(updated.DueDate),
					OriginalDueDate:        strPtr(updated.OriginalDueDate),
					InvoiceURL:             strPtr(updated.InvoiceURL),
					InvoiceNumber:          strPtr(updated.InvoiceNumber),
					ExternalReference:      strPtr(updated.ExternalReference),
					UpdatedAt:              &now,
					ProviderPayload:        rawPayload,
				}

				if upsertErr := supabase.UpsertCharges([]model.IamChargeRow{row}); upsertErr != nil {
					if isDebugEnabled() {
						log.Printf("[supabase] ERROR upserting charge after update: payment_id=%s err=%v", updated.ID, upsertErr)
					}
					// Non-fatal
				} else if isDebugEnabled() {
					log.Printf("[supabase] iam.charges updated: payment_id=%s status=%s due=%s value=%.2f",
						updated.ID, updated.Status, updated.DueDate, updated.Value)
				}

				// ── 2. Sync iam.fee_contract_one_off_charges (only for one-off charges) ──
				// If provider_subscription_id is set, this is a subscription instalment —
				// in that case we only update iam.charges (above), NOT the one-off table.
				isOneOff := charge.ProviderSubscriptionID == nil || strings.TrimSpace(*charge.ProviderSubscriptionID) == ""
				if isOneOff {
					oneOffPayload := supabase.OneOffChargeUpdatePayload{
						ProviderStatus: strPtr(updated.Status),
						UpdatedAt:      now,
					}

					if oneOffErr := supabase.UpdateOneOffChargeByProviderChargeID(updated.ID, oneOffPayload); oneOffErr != nil {
						if isDebugEnabled() {
							log.Printf("[supabase] ERROR updating one_off_charge after update: payment_id=%s err=%v", updated.ID, oneOffErr)
						}
						// Non-fatal: iam.charges is already updated; the one-off sync is best-effort
					} else if isDebugEnabled() {
						log.Printf("[supabase] fee_contract_one_off_charges updated: provider_charge_id=%s new_status=%s",
							updated.ID, updated.Status)
					}
				}
			}
		}
	}

	// Pass-through Asaas payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func mapUpdatePaymentRequest(req model.AsaasUpdateChargeRequest) asaas.UpdatePaymentRequest {
	out := asaas.UpdatePaymentRequest{}

	out.Value = req.Value
	out.DueDate = req.DueDate

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
