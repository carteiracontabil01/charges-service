package supabase

import (
	"encoding/json"
	"fmt"
	"strings"
)

type CompanyAsaasCustomerPayload struct {
	Name                 string `json:"name"`
	CpfCnpj              string `json:"cpfCnpj"`
	Email                string `json:"email"`
	MobilePhone          string `json:"mobilePhone"`
	Company              bool   `json:"company"`
	NotificationDisabled bool   `json:"notificationDisabled"`
}

// GetCompanyAsaasCustomerPayload loads minimal company data needed to create an Asaas customer.
// It uses RPC in public schema because company schema might not be exposed in PostgREST.
func GetCompanyAsaasCustomerPayload(companyID string) (*CompanyAsaasCustomerPayload, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("company_id is required")
	}

	raw, err := RpcPublic("rpc_get_company_asaas_customer_payload", map[string]any{
		"p_company_id": companyID,
	})
	if err != nil {
		return nil, err
	}

	s := strings.TrimSpace(raw)
	if s == "" || s == "null" {
		return nil, nil
	}

	var out CompanyAsaasCustomerPayload
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("invalid rpc_get_company_asaas_customer_payload response: %w", err)
	}

	out.Name = strings.TrimSpace(out.Name)
	out.CpfCnpj = strings.TrimSpace(out.CpfCnpj)
	out.Email = strings.TrimSpace(out.Email)
	out.MobilePhone = strings.TrimSpace(out.MobilePhone)

	if out.Name == "" || out.CpfCnpj == "" {
		return nil, fmt.Errorf("company payload missing name or cpfCnpj")
	}

	return &out, nil
}
