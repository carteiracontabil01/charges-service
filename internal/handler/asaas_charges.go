package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/seuuser/charges-service/internal/integrations/asaas"
	"github.com/seuuser/charges-service/internal/model"
	"github.com/seuuser/charges-service/internal/supabase"
)

func newRequestID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// CreateAsaasCharge godoc
// @Summary      Criar cobrança no Asaas
// @Description  Cria uma cobrança (payment) no Asaas para uma empresa (company_id). O serviço resolve o customer_id no Asaas via mapeamento em company.asaas_integration (RPC em public) e usa a integração ativa (is_active=true) do escritório (accounting_office_id) em iam.billing_integrations.
// @Tags         asaas
// @Accept       json
// @Produce      json
// @Param        accounting_office_id  query     string  true  "ID do accounting_office (UUID)"
// @Param        company_id            query     string  true  "ID da empresa (company_id, UUID)"
// @Param        contract_id           query     string  true  "ID do contrato (UUID)"
// @Param        body                  body      model.AsaasCreateChargeRequest  true  "Payload da cobrança"
// @Success      200  {object}  model.AsaasPaymentResponse
// @Failure      400  {object}  map[string]any
// @Failure      404  {object}  map[string]any
// @Failure      502  {object}  map[string]any
// @Router       /v1/asaas/charges [post]
func CreateAsaasCharge(w http.ResponseWriter, r *http.Request) {
	rid := newRequestID()
	accountingOfficeID := strings.TrimSpace(r.URL.Query().Get("accounting_office_id"))
	if accountingOfficeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "accounting_office_id is required"})
		return
	}

	companyID := strings.TrimSpace(r.URL.Query().Get("company_id"))
	if companyID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "company_id is required"})
		return
	}

	contractID := strings.TrimSpace(r.URL.Query().Get("contract_id"))
	if contractID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "contract_id is required"})
		return
	}

	var req model.AsaasCreateChargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json body"})
		return
	}

	req.DueDate = strings.TrimSpace(req.DueDate)
	if strings.TrimSpace(string(req.BillingType)) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "billingType is required"})
		return
	}
	if req.Value <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "value must be > 0"})
		return
	}
	if req.DueDate == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "dueDate is required"})
		return
	}

	if isDebugEnabled() {
		log.Printf("[asaas] create charge request received: accounting_office_id=%s company_id=%s billingType=%s value=%v dueDate=%s",
			accountingOfficeID, companyID, req.BillingType, req.Value, req.DueDate,
		)
	}

	cfg, err := supabase.GetBillingIntegrationForOffice(accountingOfficeID, normalizeProvider("ASAAS"))
	if err != nil || cfg == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "billing integration not found for office/provider"})
		return
	}
	if isDebugEnabled() {
		log.Printf(
			"[asaas] using integration cfg for charge: id=%s provider=%s env=%s is_default=%v base_api=%q token=%s",
			cfg.ID,
			cfg.Provider,
			cfg.Environment,
			cfg.IsDefault,
			cfg.BaseAPI,
			maskToken(cfg.Token),
		)
	}
	if strings.TrimSpace(cfg.BaseAPI) == "" || strings.TrimSpace(cfg.Token) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "integration config missing base_api or token"})
		return
	}

	client := asaas.NewClient(cfg.BaseAPI, cfg.Token)

	// Resolve Asaas customer id. If missing, auto-create customer from company data via RPC and persist mapping.
	asaasCustomerID, err := supabase.GetCompanyAsaasCustomerID(companyID)
	if err != nil {
		if isDebugEnabled() {
			log.Printf("[asaas] error resolving asaas_customer_id: company_id=%s err=%v", companyID, err)
		}
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to resolve asaas customer id"})
		return
	}
	if strings.TrimSpace(asaasCustomerID) == "" {
		if isDebugEnabled() {
			log.Printf("[asaas] asaas_integration missing; auto-creating customer: rid=%s company_id=%s", rid, companyID)
		}

		payload, perr := supabase.GetCompanyAsaasCustomerPayload(companyID)
		if perr != nil {
			if isDebugEnabled() {
				log.Printf("[asaas] error loading company payload for customer create: rid=%s company_id=%s err=%v", rid, companyID, perr)
			}
			log.Printf("[asaas] ERROR loading company payload for customer create: rid=%s company_id=%s err=%v", rid, companyID, perr)
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to load company data to create asaas customer"})
			return
		}
		if payload == nil {
			log.Printf("[asaas] ERROR company not found for customer auto-create: rid=%s company_id=%s", rid, companyID)
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "company not found to create asaas customer"})
			return
		}

		cStatus, cBody, cErr := client.CreateCustomer(asaas.CreateCustomerRequest{
			Name:                 payload.Name,
			CpfCnpj:              payload.CpfCnpj,
			Email:                payload.Email,
			MobilePhone:          payload.MobilePhone,
			NotificationDisabled: payload.NotificationDisabled,
			Company:              payload.Company,
		})
		if cErr != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": cErr.Error()})
			return
		}
		if cStatus < 200 || cStatus >= 300 {
			// pass-through Asaas error payload
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(cStatus)
			_, _ = w.Write(cBody)
			return
		}

		var createdCustomer model.AsaasCustomerResponse
		if err := json.Unmarshal(cBody, &createdCustomer); err != nil || strings.TrimSpace(createdCustomer.ID) == "" {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "invalid asaas response (missing customer id)"})
			return
		}

		if err := supabase.UpsertCompanyAsaasIntegration(companyID, createdCustomer.ID); err != nil {
			// Always log the underlying error server-side (no secrets). Use request_id for correlation.
			log.Printf("[asaas] ERROR persisting company.asaas_integration: rid=%s company_id=%s asaas_customer_id=%s err=%v",
				rid, companyID, createdCustomer.ID, err,
			)
			if isDebugEnabled() {
				writeJSON(w, http.StatusBadGateway, map[string]any{
					"error":      "failed to persist asaas integration",
					"details":    err.Error(),
					"request_id": rid,
				})
				return
			}
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to persist asaas integration", "request_id": rid})
			return
		}

		asaasCustomerID = createdCustomer.ID
	}

	status, body, callErr := client.CreatePayment(mapCreatePaymentRequest(asaasCustomerID, req))
	if callErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": callErr.Error()})
		return
	}

	if isDebugEnabled() {
		raw := string(body)
		if len(raw) > 800 {
			raw = raw[:800] + "…(truncated)"
		}
		log.Printf("[asaas] create charge response: status=%d body=%s", status, raw)
	}

	// If 2xx, persist charges in our provider-agnostic table (iam.charges).
	if status >= 200 && status < 300 {
		var created model.AsaasPaymentResponse
		_ = json.Unmarshal(body, &created)

		// List charges created. If installment id exists, list by installment; else fallback to externalReference (if any).
		params := url.Values{}
		params.Set("customer", asaasCustomerID)
		params.Set("limit", "100")
		params.Set("offset", "0")

		if strings.TrimSpace(created.Installment) != "" {
			params.Set("installment", created.Installment)
		} else if req.ExternalReference != nil && strings.TrimSpace(*req.ExternalReference) != "" {
			params.Set("externalReference", strings.TrimSpace(*req.ExternalReference))
		} else if strings.TrimSpace(created.ID) != "" {
			// last-resort: persist at least the created payment (list may not be filterable)
			// no extra filter
		}

		listStatus, listBody, listErr := client.ListPayments(params)
		if listErr != nil {
			if isDebugEnabled() {
				log.Printf("[asaas] ERROR listing charges after create: status=%d err=%v", listStatus, listErr)
			}
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to list charges after create"})
			return
		}

		var list model.AsaasPaymentsListResponse
		_ = json.Unmarshal(listBody, &list)

		tenantID, terr := supabase.GetCompanyTenantID(companyID)
		if terr != nil {
			if isDebugEnabled() {
				log.Printf("[supabase] ERROR resolving tenant_id for company_id=%s err=%v", companyID, terr)
			}
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to resolve tenant for company"})
			return
		}

		rows := make([]model.IamChargeRow, 0, len(list.Data))
		for _, p := range list.Data {
			if strings.TrimSpace(p.ID) == "" {
				continue
			}

			desc := strings.TrimSpace(p.Description)
			var descPtr *string
			if desc != "" {
				descPtr = &desc
			}
			bt := strings.TrimSpace(p.BillingType)
			var btPtr *string
			if bt != "" {
				btPtr = &bt
			}
			st := strings.TrimSpace(p.Status)
			var stPtr *string
			if st != "" {
				stPtr = &st
			}
			due := strings.TrimSpace(p.DueDate)
			var duePtr *string
			if due != "" {
				duePtr = &due
			}
			odue := strings.TrimSpace(p.OriginalDueDate)
			var oduePtr *string
			if odue != "" {
				oduePtr = &odue
			}
			iurl := strings.TrimSpace(p.InvoiceURL)
			var iurlPtr *string
			if iurl != "" {
				iurlPtr = &iurl
			}
			inum := strings.TrimSpace(p.InvoiceNumber)
			var inumPtr *string
			if inum != "" {
				inumPtr = &inum
			}
			eref := strings.TrimSpace(p.ExternalReference)
			var erefPtr *string
			if eref != "" {
				erefPtr = &eref
			}
			var netPtr *float64
			if p.NetValue != 0 {
				v := p.NetValue
				netPtr = &v
			}
			var instNumPtr *int32
			if p.InstallmentNumber != 0 {
				v := p.InstallmentNumber
				instNumPtr = &v
			}
			var instIDPtr *string
			if strings.TrimSpace(p.Installment) != "" {
				v := strings.TrimSpace(p.Installment)
				instIDPtr = &v
			}

			payload, _ := json.Marshal(p)

			rows = append(rows, model.IamChargeRow{
				TenantID:              tenantID,
				AccountingOfficeID:    accountingOfficeID,
				CompanyID:             companyID,
				ContractID:            contractID,
				Provider:              "ASAAS",
				ProviderChargeID:      p.ID,
				ProviderInstallmentID: instIDPtr,
				InstallmentNumber:     instNumPtr,
				Value:                 p.Value,
				NetValue:              netPtr,
				Description:           descPtr,
				BillingType:           btPtr,
				Status:                stPtr,
				DueDate:               duePtr,
				OriginalDueDate:       oduePtr,
				InvoiceURL:            iurlPtr,
				InvoiceNumber:         inumPtr,
				ExternalReference:     erefPtr,
				ProviderPayload:       payload,
			})
		}

		// If Asaas list didn't return anything (rare), persist at least the created payment.
		if len(rows) == 0 && strings.TrimSpace(created.ID) != "" {
			payload, _ := json.Marshal(created)
			rows = append(rows, model.IamChargeRow{
				TenantID:           tenantID,
				AccountingOfficeID: accountingOfficeID,
				CompanyID:          companyID,
				ContractID:         contractID,
				Provider:           "ASAAS",
				ProviderChargeID:   created.ID,
				Value:              created.Value,
				ProviderPayload:    payload,
			})
		}

		if err := supabase.UpsertCharges(rows); err != nil {
			if isDebugEnabled() {
				log.Printf("[supabase] ERROR upserting iam.charges: err=%v", err)
				writeJSON(w, http.StatusBadGateway, map[string]any{
					"error":   "failed to persist charges",
					"details": err.Error(),
				})
				return
			}
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to persist charges"})
			return
		}
	}

	// Pass-through Asaas payload (it matches model.AsaasPaymentResponse on success)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func mapCreatePaymentRequest(asaasCustomerID string, req model.AsaasCreateChargeRequest) asaas.CreatePaymentRequest {
	out := asaas.CreatePaymentRequest{
		Customer:    asaasCustomerID,
		BillingType: string(req.BillingType),
		Value:       req.Value,
		DueDate:     req.DueDate,
	}

	out.Description = req.Description
	out.DaysAfterDueDateToRegistrationCancellation = req.DaysAfterDueDateToRegistrationCancellation
	out.ExternalReference = req.ExternalReference
	out.InstallmentCount = req.InstallmentCount
	out.TotalValue = req.TotalValue
	out.InstallmentValue = req.InstallmentValue
	out.PostalService = req.PostalService

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
