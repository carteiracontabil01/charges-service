package supabase

import (
	"encoding/json"
	"fmt"

	"github.com/seuuser/charges-service/internal/model"
)

// UpsertCharges stores provider-agnostic charges into iam.charges.
// Requires a unique constraint matching (tenant_id, provider, provider_charge_id).
func UpsertCharges(rows []model.IamChargeRow) error {
	c := GetIAMClient()
	if c == nil {
		return fmt.Errorf("supabase iam client não inicializado")
	}
	if len(rows) == 0 {
		return nil
	}

	// postgrest-go: Upsert(value, onConflict, returning, count)
	_, _, err := c.
		From("charges").
		Upsert(rows, "tenant_id,provider,provider_charge_id", "minimal", "").
		Execute()
	if err != nil {
		return err
	}
	return nil
}

// GetChargeByProviderID retrieves a charge from iam.charges by provider and provider_charge_id.
func GetChargeByProviderID(provider, providerChargeID string) (*model.IamChargeRow, error) {
	c := GetIAMClient()
	if c == nil {
		return nil, fmt.Errorf("supabase iam client não inicializado")
	}

	data, _, err := c.
		From("charges").
		Select("*", "", false).
		Eq("provider", provider).
		Eq("provider_charge_id", providerChargeID).
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch charge: %w", err)
	}

	var charge model.IamChargeRow
	if err := json.Unmarshal(data, &charge); err != nil {
		return nil, fmt.Errorf("failed to unmarshal charge: %w", err)
	}

	return &charge, nil
}

// DeleteChargeByProviderID deletes a charge from iam.charges by provider and provider_charge_id.
func DeleteChargeByProviderID(provider, providerChargeID string) error {
	c := GetIAMClient()
	if c == nil {
		return fmt.Errorf("supabase iam client não inicializado")
	}

	_, _, err := c.
		From("charges").
		Delete("", "").
		Eq("provider", provider).
		Eq("provider_charge_id", providerChargeID).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to delete charge: %w", err)
	}

	return nil
}
