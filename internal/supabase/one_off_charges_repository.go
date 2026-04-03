package supabase

import (
	"fmt"
	"strings"
)

// OneOffSyncFields carries optional provider-side fields to be persisted back into
// iam.fee_contract_one_off_charges alongside the mandatory status fields.
type OneOffSyncFields struct {
	Value       *float64 // charge value → base_amount
	DueDate     *string  // YYYY-MM-DD  → start_date
	BillingType *string  // BOLETO|PIX|CREDIT_CARD|UNDEFINED → payment_method
}

// SyncOneOffChargeFromProvider calls public.sync_one_off_charge_from_provider via PostgREST.
//
// The function (SECURITY DEFINER) handles two cases:
//  1. Fast path  – row already linked (provider_charge_id set) → refreshes status + value fields.
//  2. Initial link – provider_charge_id is NULL → parses externalReference to locate the row
//     and sets provider_charge_id for the first time.
//
// extra is optional; pass nil to skip updating value/dueDate/billingType.
// external_reference expected format: "fee_contract:{uuid}:one_off:{description}"
func SyncOneOffChargeFromProvider(
	providerChargeID, providerStatus, externalReference string,
	extra *OneOffSyncFields,
) error {
	providerChargeID = strings.TrimSpace(providerChargeID)
	if providerChargeID == "" {
		return fmt.Errorf("provider_charge_id is required for one-off sync")
	}

	body := map[string]any{
		"p_provider_charge_id": providerChargeID,
		"p_provider_status":    strings.TrimSpace(providerStatus),
	}
	if ref := strings.TrimSpace(externalReference); ref != "" {
		body["p_external_reference"] = ref
	}
	if extra != nil {
		if extra.Value != nil {
			body["p_value"] = *extra.Value
		}
		if extra.DueDate != nil && strings.TrimSpace(*extra.DueDate) != "" {
			body["p_due_date"] = strings.TrimSpace(*extra.DueDate)
		}
		if extra.BillingType != nil && strings.TrimSpace(*extra.BillingType) != "" {
			body["p_billing_type"] = strings.TrimSpace(*extra.BillingType)
		}
	}

	rawResp, err := RpcPublic("sync_one_off_charge_from_provider", body)
	if err != nil {
		return fmt.Errorf("sync_one_off_charge_from_provider RPC failed (charge_id=%s): %w", providerChargeID, err)
	}

	_ = rawResp // RETURNS void — ignored intentionally
	return nil
}
