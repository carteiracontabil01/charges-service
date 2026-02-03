package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/seuuser/charges-service/internal/integrations/asaas"
	"github.com/seuuser/charges-service/internal/model"
	"github.com/seuuser/charges-service/internal/supabase"
)

func isDebugEnabled() bool {
	return strings.TrimSpace(strings.ToLower(os.Getenv("CHARGES_DEBUG"))) == "true"
}

func maskToken(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return "(empty)"
	}
	// show only last 4 chars, keep length
	last := t
	if len(t) > 4 {
		last = t[len(t)-4:]
	}
	return "len=" + itoa(len(t)) + " ****" + last
}

func itoa(n int) string {
	// tiny helper to avoid importing strconv everywhere in handlers
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [32]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func normalizeProvider(p string) string {
	p = strings.ToUpper(strings.TrimSpace(p))
	if strings.HasPrefix(p, "ASSAS") || strings.HasPrefix(p, "ASAAS") {
		return "ASAAS"
	}
	if i := strings.IndexByte(p, '_'); i > 0 {
		return p[:i]
	}
	return p
}

// CreateAsaasCustomer godoc
// @Summary      Criar novo cliente no Asaas
// @Description  Cria um cliente no Asaas usando a integração ativa (is_active=true) do escritório (accounting_office_id) cadastrada no Supabase (iam.billing_integrations). Quando existir mais de uma integração ativa, o serviço prioriza is_default=true e depois a mais recente.
// @Tags         asaas
// @Accept       json
// @Produce      json
// @Param        accounting_office_id  query     string  true  "ID do accounting_office (UUID)"
// @Param        company_id            query     string  true  "ID da empresa (company_id, UUID)"
// @Param        body                  body      model.AsaasCreateCustomerRequest  true  "Payload (campos mínimos)"
// @Success      200  {object}  model.AsaasCustomerResponse
// @Failure      400  {object}  map[string]any
// @Failure      404  {object}  map[string]any
// @Failure      502  {object}  map[string]any
// @Router       /v1/asaas/customers [post]
func CreateAsaasCustomer(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("[asaas] request received: accounting_office_id=%s company_id=%s", accountingOfficeID, companyID)
	}

	var req model.AsaasCreateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json body"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.CpfCnpj = strings.TrimSpace(req.CpfCnpj)
	req.Email = strings.TrimSpace(req.Email)
	req.MobilePhone = strings.TrimSpace(req.MobilePhone)

	if req.Name == "" || req.CpfCnpj == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name and cpfCnpj are required"})
		return
	}

	// Avoid duplicate customers in Asaas if integration already exists for this company (scoped by current tenant inside RPC)
	existingCustomerID, err := supabase.GetCompanyAsaasCustomerID(companyID)
	if err != nil {
		if isDebugEnabled() {
			log.Printf("[asaas] warning: error checking existing company.asaas_integration (will continue): %v", err)
		}
	} else if strings.TrimSpace(existingCustomerID) != "" {
		if isDebugEnabled() {
			log.Printf("[asaas] existing asaas_integration found for company_id=%s asaas_customer_id=%s", companyID, existingCustomerID)
		}
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":             "asaas integration already exists for this company (current tenant)",
			"asaas_customer_id": existingCustomerID,
		})
		return
	}

	if isDebugEnabled() {
		log.Printf("[asaas] no existing asaas_integration found for company_id=%s", companyID)
		log.Printf("[asaas] loading billing integration config for office_id=%s provider=%s", accountingOfficeID, normalizeProvider("ASAAS"))
	}

	cfg, err := supabase.GetBillingIntegrationForOffice(accountingOfficeID, normalizeProvider("ASAAS"))
	if err != nil || cfg == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "billing integration not found for office/provider"})
		return
	}

	if isDebugEnabled() {
		log.Printf(
			"[asaas] using integration cfg: id=%s provider=%s env=%s is_default=%v base_api=%q token=%s",
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
	status, body, callErr := client.CreateCustomer(asaas.CreateCustomerRequest{
		Name:                 req.Name,
		CpfCnpj:              req.CpfCnpj,
		Email:                req.Email,
		MobilePhone:          req.MobilePhone,
		NotificationDisabled: req.NotificationDisabled,
		Company:              req.Company,
	})
	if callErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": callErr.Error()})
		return
	}

	if isDebugEnabled() {
		// log response status + truncated body (avoid huge logs)
		raw := string(body)
		if len(raw) > 800 {
			raw = raw[:800] + "…(truncated)"
		}
		log.Printf("[asaas] create customer response: status=%d body=%s", status, raw)
	}

	// Parse Asaas response to extract customer id
	if status >= 200 && status < 300 {
		var asaasResp model.AsaasCustomerResponse
		if err := json.Unmarshal(body, &asaasResp); err != nil || strings.TrimSpace(asaasResp.ID) == "" {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "invalid asaas response (missing id)"})
			return
		}

		// Persist mapping in Supabase (company.asaas_integration) via RPC (public schema)
		if err := supabase.UpsertCompanyAsaasIntegration(companyID, asaasResp.ID); err != nil {
			log.Printf("[asaas] ERROR persisting company.asaas_integration: company_id=%s asaas_customer_id=%s err=%v", companyID, asaasResp.ID, err)
			// We must persist this id; otherwise future charges can't be generated reliably.
			if isDebugEnabled() {
				writeJSON(w, http.StatusBadGateway, map[string]any{
					"error":   "failed to persist asaas integration",
					"details": err.Error(),
				})
				return
			}
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to persist asaas integration"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
		return
	}

	// Non-2xx: pass-through Asaas error payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
