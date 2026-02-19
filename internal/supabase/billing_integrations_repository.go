package supabase

import (
	"fmt"
	"strings"

	"github.com/seuuser/charges-service/internal/model"
	"github.com/supabase-community/postgrest-go"
)

// GetBillingIntegrationForOffice loads the billing integration configuration for a given office/provider.
// It reads from schema `iam` (configured in InitClient via SUPABASE_SCHEMA).
//
// Rule:
// - Always returns ONE active config (is_active = true)
// - If multiple exist (e.g. HML/PRD), prefer is_default=true (you can enforce this in UI).
func GetBillingIntegrationForOffice(accountingOfficeID string, provider string) (*model.BillingIntegrationRow, error) {
	c := GetClient()
	if c == nil {
		return nil, fmt.Errorf("supabase client não inicializado")
	}

	accountingOfficeID = strings.TrimSpace(accountingOfficeID)
	provider = strings.ToUpper(strings.TrimSpace(provider))

	if accountingOfficeID == "" {
		return nil, fmt.Errorf("accounting_office_id vazio")
	}
	if provider == "" {
		return nil, fmt.Errorf("provider vazio")
	}

	var rows []model.BillingIntegrationRow
	_, err := c.
		From("billing_integrations").
		Select("id, accounting_office_id, provider, environment, base_api, token, is_active, is_default, updated_at, created_at", "exact", false).
		Eq("accounting_office_id", accountingOfficeID).
		Eq("provider", provider).
		Eq("is_active", "true").
		// Prefer default config first, then most recently updated
		Order("is_default", &postgrest.OrderOpts{Ascending: false, NullsFirst: false}).
		Order("updated_at", &postgrest.OrderOpts{Ascending: false, NullsFirst: false}).
		Order("created_at", &postgrest.OrderOpts{Ascending: false, NullsFirst: false}).
		Limit(1, "").
		ExecuteTo(&rows)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("billing integration not found")
	}

	return &rows[0], nil
}

// GetBillingIntegrationForOfficeAndEnvironment loads the active billing integration configuration
// for a given office/provider/environment (iam.billing_integrations).
//
// It follows the same ordering rule as GetBillingIntegrationForOffice: prefer is_default=true, then most recent.
func GetBillingIntegrationForOfficeAndEnvironment(accountingOfficeID string, provider string, environment string) (*model.BillingIntegrationRow, error) {
	c := GetClient()
	if c == nil {
		return nil, fmt.Errorf("supabase client não inicializado")
	}

	accountingOfficeID = strings.TrimSpace(accountingOfficeID)
	provider = strings.ToUpper(strings.TrimSpace(provider))
	environment = strings.TrimSpace(environment)

	if accountingOfficeID == "" {
		return nil, fmt.Errorf("accounting_office_id vazio")
	}
	if provider == "" {
		return nil, fmt.Errorf("provider vazio")
	}
	if environment == "" {
		return nil, fmt.Errorf("environment vazio")
	}

	var rows []model.BillingIntegrationRow
	_, err := c.
		From("billing_integrations").
		Select("id, accounting_office_id, provider, environment, base_api, token, is_active, is_default, updated_at, created_at", "exact", false).
		Eq("accounting_office_id", accountingOfficeID).
		Eq("provider", provider).
		Eq("environment", environment).
		Eq("is_active", "true").
		// Prefer default config first, then most recently updated
		Order("is_default", &postgrest.OrderOpts{Ascending: false, NullsFirst: false}).
		Order("updated_at", &postgrest.OrderOpts{Ascending: false, NullsFirst: false}).
		Order("created_at", &postgrest.OrderOpts{Ascending: false, NullsFirst: false}).
		Limit(1, "").
		ExecuteTo(&rows)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("billing integration not found")
	}

	return &rows[0], nil
}

// GetBillingIntegrationByID loads a billing integration by id (iam.billing_integrations).
// Useful when the contract explicitly selects a billing_integration_id.
func GetBillingIntegrationByID(id string) (*model.BillingIntegrationRow, error) {
	c := GetClient()
	if c == nil {
		return nil, fmt.Errorf("supabase client não inicializado")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("billing_integration_id vazio")
	}

	var rows []model.BillingIntegrationRow
	_, err := c.
		From("billing_integrations").
		Select("id, accounting_office_id, provider, environment, base_api, token, is_active, is_default, updated_at, created_at", "exact", false).
		Eq("id", id).
		Eq("is_active", "true").
		Limit(1, "").
		ExecuteTo(&rows)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("billing integration not found")
	}
	return &rows[0], nil
}
