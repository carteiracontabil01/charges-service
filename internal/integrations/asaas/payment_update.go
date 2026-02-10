package asaas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// PaymentSplit represents a split configuration for updating a charge.
type PaymentSplit struct {
	WalletID         string   `json:"walletId"`
	FixedValue       *float64 `json:"fixedValue,omitempty"`
	PercentualValue  *float64 `json:"percentualValue,omitempty"`
	ExternalReference *string  `json:"externalReference,omitempty"`
	Description      *string  `json:"description,omitempty"`
}

// UpdatePaymentRequest is the payload for updating an existing charge (payment) in Asaas.
// https://docs.asaas.com/reference/atualizar-cobranca-existente
type UpdatePaymentRequest struct {
	BillingType                                *string  `json:"billingType,omitempty"` // BOLETO | CREDIT_CARD | PIX | UNDEFINED
	Value                                      *float64 `json:"value,omitempty"`
	DueDate                                    *string  `json:"dueDate,omitempty"` // YYYY-MM-DD
	Description                                *string  `json:"description,omitempty"`
	DaysAfterDueDateToRegistrationCancellation *int32   `json:"daysAfterDueDateToRegistrationCancellation,omitempty"`
	ExternalReference                          *string  `json:"externalReference,omitempty"`

	Discount      *PaymentDiscount `json:"discount,omitempty"`
	Interest      *PaymentInterest `json:"interest,omitempty"`
	Fine          *PaymentFine     `json:"fine,omitempty"`
	PostalService *bool            `json:"postalService,omitempty"`

	// Callback controls automatic redirect behavior after payment (commonly used for payment links).
	// We keep it flexible because the exact schema may evolve in Asaas without notice.
	Callback map[string]any `json:"callback,omitempty"`

	Split []PaymentSplit `json:"split,omitempty"`
}

// UpdatePayment calls the Asaas API to update an existing charge (payment).
// paymentID is the Asaas payment ID (e.g. "pay_080225913252").
// Note: Only charges awaiting payment or overdue can be updated.
// The customer cannot be changed once the charge is created.
func (c *Client) UpdatePayment(paymentID string, req UpdatePaymentRequest) (int, []byte, error) {
	if c.BaseURL == "" {
		return 0, nil, fmt.Errorf("asaas baseURL is empty")
	}
	if c.Token == "" {
		return 0, nil, fmt.Errorf("asaas token is empty")
	}
	if strings.TrimSpace(paymentID) == "" {
		return 0, nil, fmt.Errorf("paymentID is required")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v3/payments/%s", c.BaseURL, paymentID)
	httpReq, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, fmt.Errorf("new request: %w", err)
	}

	httpReq.Header.Set("access_token", c.Token)
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body, nil
}
