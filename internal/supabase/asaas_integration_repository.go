package supabase

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// GetCompanyAsaasCustomerID returns the current Asaas customer id for a company,
// scoped by the company's CURRENT tenant (resolved inside the RPC).
func GetCompanyAsaasCustomerID(companyID string) (string, error) {
	raw, err := RpcPublic("rpc_get_company_asaas_customer_id", map[string]any{
		"p_company_id": companyID,
	})
	if err != nil {
		return "", err
	}

	s := strings.TrimSpace(raw)
	if s == "" || s == "null" {
		return "", nil
	}

	// PostgREST may return a JSON string (e.g. "cus_123")
	var out string
	if json.Unmarshal([]byte(s), &out) == nil && strings.TrimSpace(out) != "" {
		return out, nil
	}

	// Fallback: raw string
	return strings.Trim(s, `"`), nil
}

// UpsertCompanyAsaasIntegration stores the Asaas customer id for the company (tenant is derived inside the RPC).
func UpsertCompanyAsaasIntegration(companyID string, asaasCustomerID string) error {
	_, err := RpcPublic("rpc_upsert_company_asaas_integration", map[string]any{
		"p_company_id":        companyID,
		"p_asaas_customer_id": asaasCustomerID,
	})
	if err != nil {
		// Always log server-side (this is actionable and contains no secrets).
		log.Printf("[supabase] ERROR rpc_upsert_company_asaas_integration: company_id=%s asaas_customer_id=%s err=%v", companyID, asaasCustomerID, err)
		return fmt.Errorf("rpc_upsert_company_asaas_integration failed: %w", err)
	}
	return nil
}
