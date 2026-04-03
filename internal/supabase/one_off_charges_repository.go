package supabase

import (
	"fmt"
	"strings"
)

// SyncOneOffChargeFromProvider calls iam.sync_one_off_charge_from_provider via PostgREST RPC.
//
// The RPC runs in SECURITY DEFINER context (bypasses RLS) and handles two cases:
//  1. Fast path  – row already has provider_charge_id → update provider_status + updated_at.
//  2. Initial link – provider_charge_id is NULL → parse external_reference to find the row
//     and set provider_charge_id for the first time.
//
// Non-fatal by design: if no matching row is found the RPC returns without error; the
// caller logs a warning but does not fail.
// external_reference format expected: "fee_contract:{contract_uuid}:one_off:{description}"
func SyncOneOffChargeFromProvider(providerChargeID, providerStatus, externalReference string) error {
	providerChargeID = strings.TrimSpace(providerChargeID)
	if providerChargeID == "" {
		return fmt.Errorf("provider_charge_id is required for one-off sync")
	}
	providerStatus = strings.TrimSpace(providerStatus)

	body := map[string]any{
		"p_provider_charge_id": providerChargeID,
		"p_provider_status":    providerStatus,
	}
	if ref := strings.TrimSpace(externalReference); ref != "" {
		body["p_external_reference"] = ref
	}

	// NOTE: The function lives in the iam schema but is wrapped by a public alias
	// (public.sync_one_off_charge_from_provider) so PostgREST can call it without
	// needing extra schema exposure config.
	rawResp, err := RpcPublic("sync_one_off_charge_from_provider", body)
	if err != nil {
		return fmt.Errorf("sync_one_off_charge_from_provider RPC failed (charge_id=%s): %w", providerChargeID, err)
	}

	_ = rawResp // RETURNS void; response body is empty — ignored intentionally
	return nil
}
