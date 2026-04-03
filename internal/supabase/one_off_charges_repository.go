package supabase

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// extRefRe parses the external_reference format used for one-off charges:
// "fee_contract:{contract_uuid}:one_off:{description}"
var extRefRe = regexp.MustCompile(
	`^fee_contract:([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}):one_off:(.+)$`,
)

// SyncOneOffChargeFromProvider updates iam.fee_contract_one_off_charges using the
// Asaas provider data.  It uses direct SDK calls (no SQL RPC required).
//
// Two-phase logic
//
//  1. Status refresh (fast path):
//     If a row already has provider_charge_id = providerChargeID, update provider_status
//     and updated_at only.  This is the normal path for charge UPDATES.
//
//  2. Initial link (slow path – CREATE case):
//     When the row still has provider_charge_id = NULL, parse externalReference to
//     extract contract_id + description and link the row by setting provider_charge_id,
//     provider_status and provider.
//
// Non-fatal: if no row matches either phase the function returns nil; the caller is
// responsible for logging a warning.  iam.charges is always kept up-to-date regardless.
func SyncOneOffChargeFromProvider(providerChargeID, providerStatus, externalReference string) error {
	c := GetIAMClient()
	if c == nil {
		return fmt.Errorf("supabase iam client not initialised")
	}

	providerChargeID = strings.TrimSpace(providerChargeID)
	if providerChargeID == "" {
		return fmt.Errorf("provider_charge_id is required")
	}
	providerStatus = strings.TrimSpace(providerStatus)
	now := time.Now().UTC().Format(time.RFC3339)

	// ── Phase 1: status refresh ────────────────────────────────────────────────
	// Rows that are already linked just need their status refreshed.
	_, _, err := c.
		From("fee_contract_one_off_charges").
		Update(map[string]any{
			"provider_status": providerStatus,
			"updated_at":      now,
		}, "", "").
		Eq("provider_charge_id", providerChargeID).
		Execute()
	if err != nil {
		return fmt.Errorf("one_off phase-1 status refresh failed: %w", err)
	}

	// ── Phase 2: initial link ──────────────────────────────────────────────────
	// Only needed for the CREATE case where provider_charge_id is still NULL.
	// We use the external_reference to locate the correct row.
	externalReference = strings.TrimSpace(externalReference)
	if externalReference == "" {
		return nil // no reference to link with — non-fatal
	}

	m := extRefRe.FindStringSubmatch(externalReference)
	if m == nil {
		return nil // format not recognised — non-fatal
	}
	contractID := m[1]
	description := m[2]

	_, _, linkErr := c.
		From("fee_contract_one_off_charges").
		Update(map[string]any{
			"provider":           "ASAAS",
			"provider_charge_id": providerChargeID,
			"provider_status":    providerStatus,
			"updated_at":         now,
		}, "", "").
		Eq("contract_id", contractID).
		Eq("description", description).
		Is("provider_charge_id", "null").
		Execute()
	if linkErr != nil {
		return fmt.Errorf("one_off phase-2 initial link failed: %w", linkErr)
	}

	return nil
}
