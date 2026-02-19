package supabase

import (
	"fmt"
	"strings"

	"github.com/seuuser/charges-service/internal/model"
)

func GetFeeContractByID(contractID string) (*model.FeeContractRow, error) {
	c := GetIAMClient()
	if c == nil {
		return nil, fmt.Errorf("supabase iam client não inicializado")
	}
	contractID = strings.TrimSpace(contractID)
	if contractID == "" {
		return nil, fmt.Errorf("contract_id is required")
	}

	var rows []model.FeeContractRow
	_, err := c.
		From("fee_contracts").
		Select(
			"id, tenant_id, accounting_office_id, company_id, contract_number, provider, provider_environment, billing_integration_id, start_date, end_date, interest_percentage, fine_type, fine_percentage, fine_value, discount_type, discount_percentage, discount_value, discount_due_limit_days",
			"exact",
			false,
		).
		Eq("id", contractID).
		Limit(1, "").
		ExecuteTo(&rows)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func ListFeeContractServiceItems(contractID string) ([]model.FeeContractServiceItemRow, error) {
	c := GetIAMClient()
	if c == nil {
		return nil, fmt.Errorf("supabase iam client não inicializado")
	}
	contractID = strings.TrimSpace(contractID)
	if contractID == "" {
		return nil, fmt.Errorf("contract_id is required")
	}

	var rows []model.FeeContractServiceItemRow
	_, err := c.
		From("fee_contract_service_items").
		Select("id, contract_id, line_no, name, billing_type, periodicity, final_amount, due_day, start_date, payment_method", "exact", false).
		Eq("contract_id", contractID).
		Order("line_no", nil).
		ExecuteTo(&rows)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

