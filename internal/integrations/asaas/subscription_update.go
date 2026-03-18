package asaas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// UpdateSubscriptionRequest is the payload for updating an existing subscription in Asaas.
// https://docs.asaas.com/reference/atualizar-assinatura-existente
// All fields are optional.
type UpdateSubscriptionRequest struct {
	BillingType *string `json:"billingType,omitempty"` // BOLETO | CREDIT_CARD | PIX | UNDEFINED
	Status      *string `json:"status,omitempty"`      // ACTIVE | INACTIVE
	Value       *float64 `json:"value,omitempty"`
	NextDueDate *string `json:"nextDueDate,omitempty"` // YYYY-MM-DD
	Cycle       *string `json:"cycle,omitempty"`
	Description *string `json:"description,omitempty"`
	EndDate     *string `json:"endDate,omitempty"`

	// When true, updates existing pending payments with the new billingType and/or value.
	UpdatePendingPayments *bool `json:"updatePendingPayments,omitempty"`

	ExternalReference *string `json:"externalReference,omitempty"`

	Discount *PaymentDiscount `json:"discount,omitempty"`
	Interest *PaymentInterest `json:"interest,omitempty"`
	Fine     *PaymentFine     `json:"fine,omitempty"`
}

// UpdateSubscription calls PUT /v3/subscriptions/{id} on the Asaas API.
func (c *Client) UpdateSubscription(subscriptionID string, req UpdateSubscriptionRequest) (int, []byte, error) {
	if c.BaseURL == "" {
		return 0, nil, fmt.Errorf("asaas baseURL is empty")
	}
	if c.Token == "" {
		return 0, nil, fmt.Errorf("asaas token is empty")
	}
	if subscriptionID == "" {
		return 0, nil, fmt.Errorf("subscriptionID is required")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.BaseURL + "/v3/subscriptions/" + subscriptionID
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
