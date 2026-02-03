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

// UpdateAsaasCustomerByID godoc
// @Summary      Atualizar cliente existente no Asaas
// @Description  Atualiza um cliente existente no Asaas pelo ID (Asaas customer id), usando a integração ativa (is_active=true) do escritório (accounting_office_id) cadastrada no Supabase (iam.billing_integrations). Referência Asaas: https://docs.asaas.com/reference/atualizar-cliente-existente
// @Tags         asaas
// @Accept       json
// @Produce      json
// @Param        accounting_office_id  query     string  true  "ID do accounting_office (UUID)"
// @Param        id                   path      string  true  "ID do cliente no Asaas"
// @Param        body                 body      model.AsaasUpdateCustomerRequest  true  "Campos para atualização (parcial)"
// @Success      200  {object}  model.AsaasCustomerResponse
// @Failure      400  {object}  map[string]any
// @Failure      404  {object}  map[string]any
// @Failure      502  {object}  map[string]any
// @Router       /v1/asaas/customers/{id} [put]
func UpdateAsaasCustomerByID(w http.ResponseWriter, r *http.Request) {
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

	var req model.AsaasUpdateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json body"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.CpfCnpj = strings.TrimSpace(req.CpfCnpj)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)
	req.MobilePhone = strings.TrimSpace(req.MobilePhone)
	req.Address = strings.TrimSpace(req.Address)
	req.AddressNumber = strings.TrimSpace(req.AddressNumber)
	req.Complement = strings.TrimSpace(req.Complement)
	req.Province = strings.TrimSpace(req.Province)
	req.State = strings.TrimSpace(req.State)
	req.Country = strings.TrimSpace(req.Country)
	req.PostalCode = strings.TrimSpace(req.PostalCode)
	req.AdditionalEmails = strings.TrimSpace(req.AdditionalEmails)
	req.ExternalReference = strings.TrimSpace(req.ExternalReference)
	req.Observations = strings.TrimSpace(req.Observations)

	// Avoid sending an empty update to Asaas (usually returns 400).
	if req.Name == "" &&
		req.CpfCnpj == "" &&
		req.Email == "" &&
		req.Phone == "" &&
		req.MobilePhone == "" &&
		req.Address == "" &&
		req.AddressNumber == "" &&
		req.Complement == "" &&
		req.Province == "" &&
		req.City == nil &&
		req.State == "" &&
		req.Country == "" &&
		req.PostalCode == "" &&
		req.AdditionalEmails == "" &&
		req.ExternalReference == "" &&
		req.Observations == "" &&
		req.PersonType == "" &&
		req.NotificationDisabled == nil &&
		req.Company == nil &&
		req.ForeignCustomer == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "at least one field is required to update"})
		return
	}

	if isDebugEnabled() {
		log.Printf("[asaas] update customer by id: accounting_office_id=%s customer_id=%s", accountingOfficeID, customerID)
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
	status, body, callErr := client.UpdateCustomer(customerID, asaas.UpdateCustomerRequest{
		Name:                 req.Name,
		CpfCnpj:              req.CpfCnpj,
		Email:                req.Email,
		Phone:                req.Phone,
		MobilePhone:          req.MobilePhone,
		Address:              req.Address,
		AddressNumber:        req.AddressNumber,
		Complement:           req.Complement,
		Province:             req.Province,
		City:                 req.City,
		State:                req.State,
		Country:              req.Country,
		PostalCode:           req.PostalCode,
		AdditionalEmails:     req.AdditionalEmails,
		ExternalReference:    req.ExternalReference,
		Observations:         req.Observations,
		PersonType:           string(req.PersonType),
		NotificationDisabled: req.NotificationDisabled,
		Company:              req.Company,
		ForeignCustomer:      req.ForeignCustomer,
	})
	if callErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": callErr.Error()})
		return
	}

	if isDebugEnabled() {
		raw := string(body)
		if len(raw) > 800 {
			raw = raw[:800] + "…(truncated)"
		}
		log.Printf("[asaas] update customer response: status=%d body=%s", status, raw)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// UpdateAsaasCustomerByCompanyID godoc
// @Summary      Atualizar cliente existente no Asaas por company_id (mapeamento interno)
// @Description  Resolve o customer id do Asaas via company.asaas_integration (RPC em public) e atualiza o cliente no Asaas. Referência Asaas: https://docs.asaas.com/reference/atualizar-cliente-existente
// @Tags         asaas
// @Accept       json
// @Produce      json
// @Param        accounting_office_id  query     string  true  "ID do accounting_office (UUID)"
// @Param        company_id            query     string  true  "ID da empresa (company_id, UUID)"
// @Param        body                  body      model.AsaasUpdateCustomerRequest  true  "Campos para atualização (parcial)"
// @Success      200  {object}  model.AsaasCustomerResponse
// @Failure      400  {object}  map[string]any
// @Failure      404  {object}  map[string]any
// @Failure      502  {object}  map[string]any
// @Router       /v1/asaas/customers/by-company [put]
func UpdateAsaasCustomerByCompanyID(w http.ResponseWriter, r *http.Request) {
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

	var req model.AsaasUpdateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json body"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.CpfCnpj = strings.TrimSpace(req.CpfCnpj)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)
	req.MobilePhone = strings.TrimSpace(req.MobilePhone)
	req.Address = strings.TrimSpace(req.Address)
	req.AddressNumber = strings.TrimSpace(req.AddressNumber)
	req.Complement = strings.TrimSpace(req.Complement)
	req.Province = strings.TrimSpace(req.Province)
	req.State = strings.TrimSpace(req.State)
	req.Country = strings.TrimSpace(req.Country)
	req.PostalCode = strings.TrimSpace(req.PostalCode)
	req.AdditionalEmails = strings.TrimSpace(req.AdditionalEmails)
	req.ExternalReference = strings.TrimSpace(req.ExternalReference)
	req.Observations = strings.TrimSpace(req.Observations)

	if req.Name == "" &&
		req.CpfCnpj == "" &&
		req.Email == "" &&
		req.Phone == "" &&
		req.MobilePhone == "" &&
		req.Address == "" &&
		req.AddressNumber == "" &&
		req.Complement == "" &&
		req.Province == "" &&
		req.City == nil &&
		req.State == "" &&
		req.Country == "" &&
		req.PostalCode == "" &&
		req.AdditionalEmails == "" &&
		req.ExternalReference == "" &&
		req.Observations == "" &&
		req.PersonType == "" &&
		req.NotificationDisabled == nil &&
		req.Company == nil &&
		req.ForeignCustomer == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "at least one field is required to update"})
		return
	}

	if isDebugEnabled() {
		log.Printf("[asaas] update customer by company: accounting_office_id=%s company_id=%s", accountingOfficeID, companyID)
	}

	asaasCustomerID, err := supabase.GetCompanyAsaasCustomerID(companyID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to resolve asaas customer id"})
		return
	}
	if strings.TrimSpace(asaasCustomerID) == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "asaas integration not found for this company (current tenant)"})
		return
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
	status, body, callErr := client.UpdateCustomer(asaasCustomerID, asaas.UpdateCustomerRequest{
		Name:                 req.Name,
		CpfCnpj:              req.CpfCnpj,
		Email:                req.Email,
		Phone:                req.Phone,
		MobilePhone:          req.MobilePhone,
		Address:              req.Address,
		AddressNumber:        req.AddressNumber,
		Complement:           req.Complement,
		Province:             req.Province,
		City:                 req.City,
		State:                req.State,
		Country:              req.Country,
		PostalCode:           req.PostalCode,
		AdditionalEmails:     req.AdditionalEmails,
		ExternalReference:    req.ExternalReference,
		Observations:         req.Observations,
		PersonType:           string(req.PersonType),
		NotificationDisabled: req.NotificationDisabled,
		Company:              req.Company,
		ForeignCustomer:      req.ForeignCustomer,
	})
	if callErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": callErr.Error()})
		return
	}

	if isDebugEnabled() {
		raw := string(body)
		if len(raw) > 800 {
			raw = raw[:800] + "…(truncated)"
		}
		log.Printf("[asaas] update customer (by-company) response: status=%d body=%s", status, raw)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
