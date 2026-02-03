package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/seuuser/charges-service/internal/integrations/asaas"
	"github.com/seuuser/charges-service/internal/supabase"
)

// GetAsaasChargeDigitableLine godoc
// @Summary      Linha digitável do boleto (Asaas)
// @Description  Retorna a linha digitável (identificationField) para uma cobrança no Asaas.
// @Tags         asaas
// @Produce      json
// @Param        accounting_office_id  query     string  true  "ID do accounting_office (UUID)"
// @Param        id                   path      string  true  "ID da cobrança no Asaas (payment id)"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  map[string]any
// @Failure      404  {object}  map[string]any
// @Failure      502  {object}  map[string]any
// @Router       /v1/asaas/charges/{id}/digitable-line [get]
func GetAsaasChargeDigitableLine(w http.ResponseWriter, r *http.Request) {
	accountingOfficeID := strings.TrimSpace(r.URL.Query().Get("accounting_office_id"))
	if accountingOfficeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "accounting_office_id is required"})
		return
	}
	paymentID := strings.TrimSpace(chi.URLParam(r, "id"))
	if paymentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
		return
	}

	cfg, err := supabase.GetBillingIntegrationForOffice(accountingOfficeID, normalizeProvider("ASAAS"))
	if err != nil || cfg == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "billing integration not found for office/provider"})
		return
	}
	if strings.TrimSpace(cfg.BaseAPI) == "" || strings.TrimSpace(cfg.Token) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "integration config missing base_api or token"})
		return
	}

	client := asaas.NewClient(cfg.BaseAPI, cfg.Token)
	status, body, callErr := client.GetPaymentIdentificationField(paymentID)
	if callErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": callErr.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// GetAsaasChargePixQrCode godoc
// @Summary      QRCode Pix (Asaas)
// @Description  Retorna o QRCode Pix (encodedImage/payload) para uma cobrança no Asaas.
// @Tags         asaas
// @Produce      json
// @Param        accounting_office_id  query     string  true  "ID do accounting_office (UUID)"
// @Param        id                   path      string  true  "ID da cobrança no Asaas (payment id)"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  map[string]any
// @Failure      404  {object}  map[string]any
// @Failure      502  {object}  map[string]any
// @Router       /v1/asaas/charges/{id}/pix-qrcode [get]
func GetAsaasChargePixQrCode(w http.ResponseWriter, r *http.Request) {
	accountingOfficeID := strings.TrimSpace(r.URL.Query().Get("accounting_office_id"))
	if accountingOfficeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "accounting_office_id is required"})
		return
	}
	paymentID := strings.TrimSpace(chi.URLParam(r, "id"))
	if paymentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
		return
	}

	cfg, err := supabase.GetBillingIntegrationForOffice(accountingOfficeID, normalizeProvider("ASAAS"))
	if err != nil || cfg == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "billing integration not found for office/provider"})
		return
	}
	if strings.TrimSpace(cfg.BaseAPI) == "" || strings.TrimSpace(cfg.Token) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "integration config missing base_api or token"})
		return
	}

	client := asaas.NewClient(cfg.BaseAPI, cfg.Token)
	status, body, callErr := client.GetPaymentPixQrCode(paymentID)
	if callErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": callErr.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
