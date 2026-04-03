package supabase

import (
	"fmt"
	"strings"
)

// SyncOneOffChargeFromProvider calls the iam.sync_one_off_charge_from_provider RPC.
//
// The RPC runs in two phases:
//  1. If a fee_contract_one_off_charges row already has provider_charge_id set,
//     it only updates provider_status + updated_at (fast path, e.g. after an edit).
//  2. Otherwise it parses externalReference (format: "fee_contract:{uuid}:one_off:{description}")
//     to locate the correct row and links it by setting provider_charge_id for the first time
//     (e.g. right after the charge is first created in Asaas).
//
// Call sites:
//   - asaas_charges.go  (CreateAsaasCharge)  → initial linking, pass externalReference
//   - asaas_charges_update.go (UpdateAsaasCharge) → status sync, externalReference is a fallback
//
// Non-fatal design: if the row is not found the RPC returns silently; the caller logs but
// does not fail — iam.charges is always kept up-to-date regardless.
func SyncOneOffChargeFromProvider(providerChargeID, providerStatus, externalReference string) error {
	providerChargeID = strings.TrimSpace(providerChargeID)
	if providerChargeID == "" {
		return fmt.Errorf("provider_charge_id is required")
	}

	body := map[string]any{
		"p_provider_charge_id": providerChargeID,
		"p_provider_status":    strings.TrimSpace(providerStatus),
	}
	if ref := strings.TrimSpace(externalReference); ref != "" {
		body["p_external_reference"] = ref
	}

	if _, err := RpcIAM("sync_one_off_charge_from_provider", body); err != nil {
		return fmt.Errorf("sync_one_off_charge_from_provider failed: %w", err)
	}
	return nil
}
