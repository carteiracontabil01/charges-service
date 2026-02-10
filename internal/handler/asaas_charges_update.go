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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json body"})
		return
	}

	if isDebugEnabled() {
		log.Printf("[asaas] update charge request received: rid=%s payment_id=%s accounting_office_id=%s",
			rid, paymentID, accountingOfficeID,
		)
	}

	// Get billing integration for the accounting office
	cfg, err := supabase.GetBillingIntegrationForOffice(accountingOfficeID, normalizeProvider("ASAAS"))
	if err != nil || cfg == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "billing integration not found for office/provider"})
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

	if req.BillingType != nil {
		bt := string(*req.BillingType)
		out.BillingType = &bt
	}
	out.Value = req.Value
	out.DueDate = req.DueDate
	out.Description = req.Description
	out.DaysAfterDueDateToRegistrationCancellation = req.DaysAfterDueDateToRegistrationCancellation
	out.ExternalReference = req.ExternalReference
	out.PostalService = req.PostalService
	out.Callback = req.Callback

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

	if len(req.Split) > 0 {
		splits := make([]asaas.PaymentSplit, 0, len(req.Split))
		for _, s := range req.Split {
			splits = append(splits, asaas.PaymentSplit{
				WalletID:         s.WalletID,
				FixedValue:       s.FixedValue,
				PercentualValue:  s.PercentualValue,
				ExternalReference: s.ExternalReference,
				Description:      s.Description,
			})
		}
		out.Split = splits
	}

	return out
}
