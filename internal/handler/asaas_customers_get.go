package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/seuuser/charges-service/internal/integrations/asaas"
	"github.com/seuuser/charges-service/internal/supabase"
)

// GetAsaasCustomerByID godoc
// @Summary      Recuperar um único cliente no Asaas
// @Description  Recupera um cliente no Asaas pelo ID (Asaas customer id), usando a integração ativa (is_active=true) do escritório (accounting_office_id) cadastrada no Supabase (iam.billing_integrations). Referência Asaas: https://docs.asaas.com/reference/recuperar-um-unico-cliente
// @Tags         asaas
// @Accept       json
// @Produce      json
// @Param        accounting_office_id  query     string  true  "ID do accounting_office (UUID)"
// @Param        id                   path      string  true  "ID do cliente no Asaas"
// @Success      200  {object}  model.AsaasCustomerResponse
// @Failure      400  {object}  map[string]any
// @Failure      404  {object}  map[string]any
// @Failure      502  {object}  map[string]any
// @Router       /v1/asaas/customers/{id} [get]
func GetAsaasCustomerByID(w http.ResponseWriter, r *http.Request) {
	accountingOfficeID := strings.TrimSpace(r.URL.Query().Get("accounting_office_id"))
	if accountingOfficeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "accounting_office_id is required"})
		return
	}

	customerID := strings.TrimSpace(chi.URLParam(r, "id"))
	if customerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
		return
	}

	if isDebugEnabled() {
		log.Printf("[asaas] get customer by id: accounting_office_id=%s customer_id=%s", accountingOfficeID, customerID)
	}

	cfg, err := supabase.GetBillingIntegrationForOffice(accountingOfficeID, normalizeProvider("ASAAS"))
	if err != nil || cfg == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "billing integration not found for office/provider"})
		return
	}
	if strings.TrimSpace(cfg.BaseAPI) == "" || strings.TrimSpace(cfg.Token) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "integration config missing base_api or token"})
		return
	}

	client := asaas.NewClient(cfg.BaseAPI, cfg.Token)
	status, body, callErr := client.GetCustomer(customerID)
	if callErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": callErr.Error()})
		return
	}

	if isDebugEnabled() {
		raw := string(body)
		if len(raw) > 800 {
			raw = raw[:800] + "…(truncated)"
		}
		log.Printf("[asaas] get customer response: status=%d body=%s", status, raw)
	}

	// Pass-through Asaas payload (it matches model.AsaasCustomerResponse on success)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// GetAsaasCustomerByCompanyID godoc
// @Summary      Recuperar cliente no Asaas por company_id (mapeamento interno)
// @Description  Recupera o customer do Asaas usando o mapeamento salvo em company.asaas_integration (via RPC em public), e então consulta o Asaas. Útil porque o front normalmente tem company_id. Referência Asaas: https://docs.asaas.com/reference/recuperar-um-unico-cliente
// @Tags         asaas
// @Accept       json
// @Produce      json
// @Param        accounting_office_id  query     string  true  "ID do accounting_office (UUID)"
// @Param        company_id            query     string  true  "ID da empresa (company_id, UUID)"
// @Success      200  {object}  model.AsaasCustomerResponse
// @Failure      400  {object}  map[string]any
// @Failure      404  {object}  map[string]any
// @Failure      502  {object}  map[string]any
// @Router       /v1/asaas/customers/by-company [get]
func GetAsaasCustomerByCompanyID(w http.ResponseWriter, r *http.Request) {
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

	if isDebugEnabled() {
		log.Printf("[asaas] get customer by company: accounting_office_id=%s company_id=%s", accountingOfficeID, companyID)
	}

	asaasCustomerID, err := supabase.GetCompanyAsaasCustomerID(companyID)
	if err != nil {
		if isDebugEnabled() {
			log.Printf("[asaas] error loading asaas_customer_id from mapping: company_id=%s err=%v", companyID, err)
		}
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to resolve asaas customer id"})
		return
	}
	if strings.TrimSpace(asaasCustomerID) == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "asaas integration not found for this company (current tenant)"})
		return
	}

	// Reuse same flow as get-by-id
	rctx := r.Clone(r.Context())
	q := rctx.URL.Query()
	q.Set("accounting_office_id", accountingOfficeID)
	rctx.URL.RawQuery = q.Encode()

	// We can't easily "call" the other handler with a path param, so just repeat the call.
	cfg, err := supabase.GetBillingIntegrationForOffice(accountingOfficeID, normalizeProvider("ASAAS"))
	if err != nil || cfg == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "billing integration not found for office/provider"})
		return
	}
	if strings.TrimSpace(cfg.BaseAPI) == "" || strings.TrimSpace(cfg.Token) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "integration config missing base_api or token"})
		return
	}

	client := asaas.NewClient(cfg.BaseAPI, cfg.Token)
	status, body, callErr := client.GetCustomer(asaasCustomerID)
	if callErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": callErr.Error()})
		return
	}

	if isDebugEnabled() {
		raw := string(body)
		if len(raw) > 800 {
			raw = raw[:800] + "…(truncated)"
		}
		log.Printf("[asaas] get customer (by-company) response: status=%d body=%s", status, raw)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
