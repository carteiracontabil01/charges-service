package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/seuuser/charges-service/internal/integrations/asaas"
	"github.com/seuuser/charges-service/internal/model"
	"github.com/seuuser/charges-service/internal/supabase"
)

// CreateAsaasSubscription godoc
// @Summary      Criar assinatura (subscription) no Asaas
// @Description  Cria uma assinatura (subscription) no Asaas para um contrato. O serviço resolve a integração ativa a partir do contrato (billing_integration_id / provider_environment) e resolve o customer_id via mapeamento em company.asaas_integration (RPC em public). Se não existir, auto-cria o customer com dados da empresa e persiste o mapping.
// @Tags         asaas
// @Accept       json
// @Produce      json
// @Param        contract_id           query     string  true  "ID do contrato (UUID)"
// @Param        body                  body      model.AsaasCreateSubscriptionRequest  true  "Payload da assinatura"
// @Success      200  {object}  model.AsaasSubscriptionResponse
// @Failure      400  {object}  map[string]any
// @Failure      404  {object}  map[string]any
// @Failure      502  {object}  map[string]any
// @Router       /v1/asaas/subscriptions [post]
func CreateAsaasSubscription(w http.ResponseWriter, r *http.Request) {
	rid := newRequestID()

	contractID := strings.TrimSpace(r.URL.Query().Get("contract_id"))
	if contractID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "contract_id is required"})
		return
	}

	var req model.AsaasCreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json body"})
		return
	}

	req.NextDueDate = strings.TrimSpace(req.NextDueDate)
	req.Cycle = strings.TrimSpace(req.Cycle)
	if strings.TrimSpace(string(req.BillingType)) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "billingType is required"})
		return
	}
	if req.Value <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "value must be > 0"})
		return
	}
	if req.NextDueDate == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "nextDueDate is required"})
		return
	}
	if req.Cycle == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "cycle is required"})
		return
	}

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

	// Resolve integration config:
	// 1) If contract explicitly selected billing_integration_id, use it.
	// 2) Else, prefer provider_environment if present, fallback to default integration for office/provider.
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
			// fallback to any active/default integration (previous behavior)
			cfg, err = supabase.GetBillingIntegrationForOffice(contract.AccountingOfficeID, provider)
		}
	} else {
		cfg, err = supabase.GetBillingIntegrationForOffice(contract.AccountingOfficeID, provider)
	}

	if err != nil || cfg == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "billing integration not found for contract/office/provider"})
		return
	}
	if strings.TrimSpace(cfg.BaseAPI) == "" || strings.TrimSpace(cfg.Token) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "integration config missing base_api or token"})
		return
	}

	if isDebugEnabled() {
		log.Printf(
			"[asaas] create subscription: rid=%s contract_id=%s cfg_id=%s provider=%s env=%s base_api=%q token=%s",
			rid,
			contractID,
			cfg.ID,
			cfg.Provider,
			cfg.Environment,
			cfg.BaseAPI,
			maskToken(cfg.Token),
		)
	}

	client := asaas.NewClient(cfg.BaseAPI, cfg.Token)

	// Resolve customer (auto-create if missing)
	asaasCustomerID, err := supabase.GetCompanyAsaasCustomerID(contract.CompanyID)
	if err != nil {
		if isDebugEnabled() {
			log.Printf("[asaas] error resolving asaas_customer_id: rid=%s company_id=%s err=%v", rid, contract.CompanyID, err)
		}
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to resolve asaas customer id", "request_id": rid})
		return
	}
	if strings.TrimSpace(asaasCustomerID) == "" {
		payload, perr := supabase.GetCompanyAsaasCustomerPayload(contract.CompanyID)
		if perr != nil {
			log.Printf("[asaas] ERROR loading company payload for customer create: rid=%s company_id=%s err=%v", rid, contract.CompanyID, perr)
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to load company data to create asaas customer", "request_id": rid})
			return
		}
		if payload == nil {
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

		if err := supabase.UpsertCompanyAsaasIntegration(contract.CompanyID, createdCustomer.ID); err != nil {
			log.Printf("[asaas] ERROR persisting company.asaas_integration: rid=%s company_id=%s asaas_customer_id=%s err=%v",
				rid, contract.CompanyID, createdCustomer.ID, err,
			)
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to persist asaas integration", "request_id": rid})
			return
		}

		asaasCustomerID = createdCustomer.ID
	}

	// Auto-fill financial settings from contract when not provided by client
	if req.Discount == nil && contract.DiscountType != nil && strings.TrimSpace(*contract.DiscountType) != "" {
		var v *float64
		t := strings.TrimSpace(strings.ToUpper(*contract.DiscountType))
		if t == "PERCENTAGE" && contract.DiscountPercentage != nil {
			v = contract.DiscountPercentage
		}
		if t == "FIXED" && contract.DiscountValue != nil {
			v = contract.DiscountValue
		}
		if v != nil {
			dt := model.AsaasDiscountType(t)
			req.Discount = &model.AsaasSubscriptionDiscount{
				Value:            v,
				DueDateLimitDays: contract.DiscountDueLimitDays,
				Type:             &dt,
			}
		}
	}
	if req.Interest == nil && contract.InterestPercentage != nil {
		req.Interest = &model.AsaasSubscriptionInterest{Value: contract.InterestPercentage}
	}
	if req.Fine == nil && contract.FineType != nil && strings.TrimSpace(*contract.FineType) != "" {
		t := strings.TrimSpace(strings.ToUpper(*contract.FineType))
		var v *float64
		if t == "PERCENTAGE" && contract.FinePercentage != nil {
			v = contract.FinePercentage
		}
		if t == "FIXED" && contract.FineValue != nil {
			v = contract.FineValue
		}
		if v != nil {
			ft := model.AsaasFineType(t)
			req.Fine = &model.AsaasSubscriptionFine{
				Value: v,
				Type:  &ft,
			}
		}
	}

	// Create subscription in Asaas
	status, body, callErr := client.CreateSubscription(mapCreateSubscriptionRequest(asaasCustomerID, req))
	if callErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": callErr.Error()})
		return
	}

	if isDebugEnabled() {
		raw := string(body)
		if len(raw) > 800 {
			raw = raw[:800] + "…(truncated)"
		}
		log.Printf("[asaas] create subscription response: rid=%s status=%d body=%s", rid, status, raw)
	}

	// On success, persist a "subscription header" row and also list+persist generated payments (charges)
	if status >= 200 && status < 300 {
		var created model.AsaasSubscriptionResponse
		_ = json.Unmarshal(body, &created)

		subID := strings.TrimSpace(created.ID)
		if subID != "" {
			subIDPtr := subID

			// List payments for this subscription and upsert them with provider_subscription_id set.
			// IMPORTANT:
			// - We do NOT persist a "header row" for the subscription itself in iam.charges to avoid duplicates in the Charges UI.
			// - We persist only actual payments (pay_...) and link them to the subscription via provider_subscription_id.
			// so the webhook can update them later (it requires existing rows to infer tenant/company context).
			params := url.Values{}
			params.Set("customer", asaasCustomerID)
			params.Set("limit", "100")
			params.Set("offset", "0")
			params.Set("subscription", subID)

			listStatus, listBody, listErr := client.ListPayments(params)
			if listErr != nil || listStatus < 200 || listStatus >= 300 {
				// fallback: try by externalReference (when Asaas doesn't support subscription filter)
				eref := strings.TrimSpace(func() string {
					if req.ExternalReference == nil {
						return ""
					}
					return *req.ExternalReference
				}())
				if eref != "" {
					params2 := url.Values{}
					params2.Set("customer", asaasCustomerID)
					params2.Set("limit", "100")
					params2.Set("offset", "0")
					params2.Set("externalReference", eref)
					listStatus, listBody, listErr = client.ListPayments(params2)
				}
			}

			if listErr == nil && listStatus >= 200 && listStatus < 300 {
				var list model.AsaasPaymentsListResponse
				_ = json.Unmarshal(listBody, &list)

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
					erefP := strings.TrimSpace(p.ExternalReference)
					var erefPtr *string
					if erefP != "" {
						erefPtr = &erefP
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
						TenantID:              contract.TenantID,
						AccountingOfficeID:    contract.AccountingOfficeID,
						CompanyID:             contract.CompanyID,
						ContractID:            contract.ID,
						Provider:              "ASAAS",
						ProviderChargeID:      p.ID,
						ProviderInstallmentID: instIDPtr,
						ProviderSubscriptionID: &subIDPtr,
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

				if len(rows) > 0 {
					_ = supabase.UpsertCharges(rows)
				}
			}
		}
	}

	// Pass-through Asaas payload (it matches model.AsaasSubscriptionResponse on success)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func mapCreateSubscriptionRequest(asaasCustomerID string, req model.AsaasCreateSubscriptionRequest) asaas.CreateSubscriptionRequest {
	out := asaas.CreateSubscriptionRequest{
		Customer:    asaasCustomerID,
		BillingType: string(req.BillingType),
		Value:       req.Value,
		NextDueDate: req.NextDueDate,
		Cycle:       req.Cycle,
	}

	out.Description = req.Description
	out.EndDate = req.EndDate
	out.MaxPayments = req.MaxPayments
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

