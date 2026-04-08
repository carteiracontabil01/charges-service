package supabase

import (
	"encoding/json"
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

// feeContractSubscriptionRow is used only internally to unmarshal the contract_id
// from iam.fee_contract_subscriptions.
type feeContractSubscriptionRow struct {
	ContractID string `json:"contract_id"`
}

// GetFeeContractBySubscriptionProviderID finds the fee contract linked to a given
// Asaas subscription ID (e.g. "sub_VXJBYgP2u0eO") by querying
// iam.fee_contract_subscriptions.provider_subscription_id and then fetching
// the parent iam.fee_contracts row.
//
// Returns (nil, nil) when no matching subscription/contract is found.
func GetFeeContractBySubscriptionProviderID(providerSubID string) (*model.FeeContractRow, error) {
	c := GetIAMClient()
	if c == nil {
		return nil, fmt.Errorf("supabase iam client não inicializado")
	}
	providerSubID = strings.TrimSpace(providerSubID)
	if providerSubID == "" {
		return nil, fmt.Errorf("provider_subscription_id is required")
	}

	// Step 1: resolve contract_id from fee_contract_subscriptions
	data, _, err := c.
		From("fee_contract_subscriptions").
		Select("contract_id", "", false).
		Eq("provider_subscription_id", providerSubID).
		Limit(1, "").
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("fee_contract_subscriptions lookup failed (sub=%s): %w", providerSubID, err)
	}

	var sub feeContractSubscriptionRow
	if err := json.Unmarshal(data, &sub); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fee_contract_subscriptions row: %w", err)
	}
	if strings.TrimSpace(sub.ContractID) == "" {
		return nil, fmt.Errorf("contract_id is empty for provider_subscription_id=%s", providerSubID)
	}

	// Step 2: fetch the full contract row
	return GetFeeContractByID(sub.ContractID)
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

