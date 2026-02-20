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

	// If 2xx, update the charge in our database (iam.charges)
	if status >= 200 && status < 300 {
		var updated model.AsaasPaymentResponse
		if err := json.Unmarshal(body, &updated); err == nil && strings.TrimSpace(updated.ID) != "" {
			// Find the existing charge in our database to get required IDs
			charge, err := supabase.GetChargeByProviderID("ASAAS", updated.ID)
			if err != nil {
				if isDebugEnabled() {
					log.Printf("[supabase] ERROR fetching charge for update: payment_id=%s err=%v", updated.ID, err)
				}
				// Non-fatal: we updated in Asaas but couldn't sync to our DB
			} else if charge != nil {
				// Update the charge record
				desc := strings.TrimSpace(updated.Description)
				var descPtr *string
				if desc != "" {
					descPtr = &desc
				}
				bt := strings.TrimSpace(updated.BillingType)
				var btPtr *string
				if bt != "" {
					btPtr = &bt
				}
				st := strings.TrimSpace(updated.Status)
				var stPtr *string
				if st != "" {
					stPtr = &st
				}
				due := strings.TrimSpace(updated.DueDate)
				var duePtr *string
				if due != "" {
					duePtr = &due
				}
				odue := strings.TrimSpace(updated.OriginalDueDate)
				var oduePtr *string
				if odue != "" {
					oduePtr = &odue
				}
				iurl := strings.TrimSpace(updated.InvoiceURL)
				var iurlPtr *string
				if iurl != "" {
					iurlPtr = &iurl
				}
				inum := strings.TrimSpace(updated.InvoiceNumber)
				var inumPtr *string
				if inum != "" {
					inumPtr = &inum
				}
				eref := strings.TrimSpace(updated.ExternalReference)
				var erefPtr *string
				if eref != "" {
					erefPtr = &eref
				}
				var netPtr *float64
				if updated.NetValue != 0 {
					v := updated.NetValue
					netPtr = &v
				}

				payload, _ := json.Marshal(updated)

				row := model.IamChargeRow{
					TenantID:           charge.TenantID,
					AccountingOfficeID: charge.AccountingOfficeID,
					CompanyID:          charge.CompanyID,
					ContractID:         charge.ContractID,
					Provider:           "ASAAS",
					ProviderChargeID:   updated.ID,
					ProviderInstallmentID:  charge.ProviderInstallmentID,
					ProviderSubscriptionID: charge.ProviderSubscriptionID,
					InstallmentNumber:      charge.InstallmentNumber,
					Value:              updated.Value,
					NetValue:           netPtr,
					Description:        descPtr,
					BillingType:        btPtr,
					Status:             stPtr,
					DueDate:            duePtr,
					OriginalDueDate:    oduePtr,
					InvoiceURL:         iurlPtr,
					InvoiceNumber:      inumPtr,
					ExternalReference:  erefPtr,
					ProviderPayload:    payload,
				}

				if err := supabase.UpsertCharges([]model.IamChargeRow{row}); err != nil {
					if isDebugEnabled() {
						log.Printf("[supabase] ERROR upserting charge after update: payment_id=%s err=%v", updated.ID, err)
					}
					// Non-fatal: we updated in Asaas but couldn't sync to our DB
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
