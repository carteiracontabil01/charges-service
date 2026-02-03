package supabase

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// GetCompanyTenantID resolves the tenant_id for a given company_id (company.companies).
func GetCompanyTenantID(companyID string) (string, error) {
	c := GetCompanyClient()
	if c == nil {
		return "", fmt.Errorf("supabase company client não inicializado")
	}

	body, _, err := c.
		From("companies").
		Select("id,tenant_id", "", false).
		Eq("id", companyID).
		Single().
		Execute()
	if err != nil {
		return "", err
	}

	var row struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&row); err != nil {
		return "", fmt.Errorf("erro ao decodificar companies.tenant_id: %w", err)
	}
	if row.TenantID == "" {
		return "", fmt.Errorf("tenant_id não encontrado para company_id=%s", companyID)
	}
	return row.TenantID, nil
}
