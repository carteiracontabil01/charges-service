package supabase

import (
	"fmt"
	"strings"
	"time"
)

// OneOffChargeUpdatePayload contains the fields that may be updated in iam.fee_contract_one_off_charges
// after a successful charge update in the provider (e.g. Asaas).
type OneOffChargeUpdatePayload struct {
	ProviderStatus   *string `json:"provider_status,omitempty"`
	ProviderChargeID *string `json:"provider_charge_id,omitempty"`
	UpdatedAt        string  `json:"updated_at"`
}

// UpdateOneOffChargeByProviderChargeID updates iam.fee_contract_one_off_charges
// matching the given provider_charge_id (e.g. Asaas payment ID).
// This is used after a successful update of a one-off charge in the provider to keep our DB in sync.
// Non-fatal: if the charge is not found in fee_contract_one_off_charges (e.g. it is a subscription
// instalment), the function returns nil without error.
func UpdateOneOffChargeByProviderChargeID(providerChargeID string, payload OneOffChargeUpdatePayload) error {
	providerChargeID = strings.TrimSpace(providerChargeID)
	if providerChargeID == "" {
		return fmt.Errorf("provider_charge_id is required")
	}

	c := GetIAMClient()
	if c == nil {
		return fmt.Errorf("supabase iam client não inicializado")
	}

	if payload.UpdatedAt == "" {
		payload.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	// Build the update map with only the fields we want to set
	update := map[string]any{
		"updated_at": payload.UpdatedAt,
	}
	if payload.ProviderStatus != nil {
		update["provider_status"] = *payload.ProviderStatus
	}
	if payload.ProviderChargeID != nil {
		update["provider_charge_id"] = *payload.ProviderChargeID
	}

	_, _, err := c.
		From("fee_contract_one_off_charges").
		Update(update, "", "").
		Eq("provider_charge_id", providerChargeID).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to update one_off_charge (provider_charge_id=%s): %w", providerChargeID, err)
	}

	return nil
}
