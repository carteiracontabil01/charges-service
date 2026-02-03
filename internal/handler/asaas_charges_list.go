package handler

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/seuuser/charges-service/internal/integrations/asaas"
	"github.com/seuuser/charges-service/internal/supabase"
)

// ListAsaasCharges godoc
// @Summary      Listar cobranças no Asaas (paginado)
// @Description  Lista cobranças (payments) do Asaas com filtros. Se company_id for informado e customer não, o serviço resolve o customer_id do Asaas via mapeamento (RPC em public) e aplica o filtro customer automaticamente.
// @Tags         asaas
// @Accept       json
// @Produce      json
// @Param        accounting_office_id  query     string  true   "ID do accounting_office (UUID)"
// @Param        company_id            query     string  false  "ID da empresa (UUID) - resolve customer automaticamente"
// @Param        offset                query     int     false  "Elemento inicial da lista"
// @Param        limit                 query     int     false  "Número de elementos da lista (max: 100)"
// @Param        customer              query     string  false  "Filtrar pelo Identificador único do cliente no Asaas"
// @Param        customerGroupName     query     string  false  "Filtrar pelo nome do grupo de cliente"
// @Param        billingType           query     string  false  "Filtrar por forma de pagamento" Enums(UNDEFINED,BOLETO,CREDIT_CARD,PIX)
// @Param        status                query     string  false  "Filtrar por status" Enums(PENDING,RECEIVED,CONFIRMED,OVERDUE,REFUNDED,RECEIVED_IN_CASH,REFUND_REQUESTED,REFUND_IN_PROGRESS,CHARGEBACK_REQUESTED,CHARGEBACK_DISPUTE,AWAITING_CHARGEBACK_REVERSAL,DUNNING_REQUESTED,DUNNING_RECEIVED,AWAITING_RISK_ANALYSIS)
// @Param        subscription          query     string  false  "Filtrar pelo Identificador único da assinatura"
// @Param        installment           query     string  false  "Filtrar pelo Identificador único do parcelamento"
// @Param        externalReference     query     string  false  "Filtrar pelo identificador do seu sistema"
// @Param        paymentDate           query     string  false  "Filtrar pela data de pagamento"
// @Param        invoiceStatus         query     string  false  "Filtro para cobranças que possuam ou não nota fiscal" Enums(SCHEDULED,AUTHORIZED,PROCESSING_CANCELLATION,CANCELED,CANCELLATION_DENIED,ERROR)
// @Param        estimatedCreditDate   query     string  false  "Filtrar pela data estimada de crédito"
// @Param        pixQrCodeId           query     string  false  "Filtrar recebimentos originados de um QrCode estático utilizando o id"
// @Param        anticipated           query     bool    false  "Filtrar registros antecipados ou não"
// @Param        anticipable           query     bool    false  "Filtrar registros antecipáveis ou não"
// @Param        dateCreated[ge]       query     string  false  "Filtrar a partir da data de criação inicial"
// @Param        dateCreated[le]       query     string  false  "Filtrar até a data de criação final"
// @Param        paymentDate[ge]       query     string  false  "Filtrar a partir da data de recebimento inicial"
// @Param        paymentDate[le]       query     string  false  "Filtrar até a data de recebimento final"
// @Param        estimatedCreditDate[ge] query   string  false  "Filtrar a partir da data estimada de crédito inicial"
// @Param        estimatedCreditDate[le] query   string  false  "Filtrar até a data estimada de crédito final"
// @Param        dueDate[ge]           query     string  false  "Filtrar a partir da data de vencimento inicial"
// @Param        dueDate[le]           query     string  false  "Filtrar até a data de vencimento final"
// @Param        user                  query     string  false  "Filtrar pelo endereço de e-mail do usuário que criou a cobrança"
// @Param        checkoutSession       query     string  false  "Filtrar pelo identificador único da checkout"
// @Success      200  {object}  model.AsaasPaymentsListResponse
// @Failure      400  {object}  map[string]any
// @Failure      404  {object}  map[string]any
// @Failure      502  {object}  map[string]any
// @Router       /v1/asaas/charges [get]
func ListAsaasCharges(w http.ResponseWriter, r *http.Request) {
	accountingOfficeID := strings.TrimSpace(r.URL.Query().Get("accounting_office_id"))
	if accountingOfficeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "accounting_office_id is required"})
		return
	}

	q := r.URL.Query()
	params := url.Values{}

	// pagination (optional)
	if s := strings.TrimSpace(q.Get("offset")); s != "" {
		if _, err := strconv.Atoi(s); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "offset must be an integer"})
			return
		}
		params.Set("offset", s)
	}
	if s := strings.TrimSpace(q.Get("limit")); s != "" {
		if n, err := strconv.Atoi(s); err != nil || n < 0 || n > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be an integer between 0 and 100"})
			return
		}
		params.Set("limit", s)
	}

	// customer filter: prefer explicit customer; else resolve from company_id
	customer := strings.TrimSpace(q.Get("customer"))
	if customer == "" {
		companyID := strings.TrimSpace(q.Get("company_id"))
		if companyID != "" {
			asaasCustomerID, err := supabase.GetCompanyAsaasCustomerID(companyID)
			if err != nil {
				if isDebugEnabled() {
					log.Printf("[asaas] error resolving asaas_customer_id for list: company_id=%s err=%v", companyID, err)
				}
				writeJSON(w, http.StatusBadGateway, map[string]any{"error": "failed to resolve asaas customer id"})
				return
			}
			if strings.TrimSpace(asaasCustomerID) == "" {
				writeJSON(w, http.StatusNotFound, map[string]any{"error": "asaas integration not found for this company (current tenant)"})
				return
			}
			customer = asaasCustomerID
		}
	}
	if customer != "" {
		params.Set("customer", customer)
	}

	// direct pass-through filters (if present)
	for _, key := range []string{
		"customerGroupName",
		"billingType",
		"status",
		"subscription",
		"installment",
		"externalReference",
		"paymentDate",
		"invoiceStatus",
		"estimatedCreditDate",
		"pixQrCodeId",
		"user",
		"checkoutSession",
	} {
		if v := strings.TrimSpace(q.Get(key)); v != "" {
			params.Set(key, v)
		}
	}

	// booleans (optional)
	for _, key := range []string{"anticipated", "anticipable"} {
		if s := strings.TrimSpace(q.Get(key)); s != "" {
			if s != "true" && s != "false" {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": key + " must be true or false"})
				return
			}
			params.Set(key, s)
		}
	}

	// range filters with brackets
	for _, key := range []string{
		"dateCreated[ge]",
		"dateCreated[le]",
		"paymentDate[ge]",
		"paymentDate[le]",
		"estimatedCreditDate[ge]",
		"estimatedCreditDate[le]",
		"dueDate[ge]",
		"dueDate[le]",
	} {
		if v := strings.TrimSpace(q.Get(key)); v != "" {
			params.Set(key, v)
		}
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

	if isDebugEnabled() {
		log.Printf("[asaas] list charges: accounting_office_id=%s params=%s cfg_base_api=%q token=%s",
			accountingOfficeID, params.Encode(), cfg.BaseAPI, maskToken(cfg.Token),
		)
	}

	client := asaas.NewClient(cfg.BaseAPI, cfg.Token)
	status, body, callErr := client.ListPayments(params)
	if callErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": callErr.Error()})
		return
	}

	if isDebugEnabled() {
		raw := string(body)
		if len(raw) > 800 {
			raw = raw[:800] + "…(truncated)"
		}
		log.Printf("[asaas] list charges response: status=%d body=%s", status, raw)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
