package asaas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CreateSubscriptionRequest is the payload for creating a new subscription in Asaas.
// https://docs.asaas.com/reference/criar-nova-assinatura
type CreateSubscriptionRequest struct {
	Customer    string  `json:"customer"`
	BillingType string  `json:"billingType"` // BOLETO | CREDIT_CARD | PIX | UNDEFINED
	Value       float64 `json:"value"`
	NextDueDate string  `json:"nextDueDate"` // YYYY-MM-DD
	Cycle       string  `json:"cycle"`       // WEEKLY | BIWEEKLY | MONTHLY | BIMONTHLY | QUARTERLY | SEMIANNUALLY | YEARLY
	Description *string `json:"description,omitempty"`
	EndDate     *string `json:"endDate,omitempty"` // YYYY-MM-DD
	MaxPayments *int32  `json:"maxPayments,omitempty"`

	ExternalReference *string `json:"externalReference,omitempty"`

	Discount *PaymentDiscount `json:"discount,omitempty"`
	Interest *PaymentInterest `json:"interest,omitempty"`
	Fine     *PaymentFine     `json:"fine,omitempty"`
}

// CreateSubscription calls the Asaas API to create a subscription.
func (c *Client) CreateSubscription(req CreateSubscriptionRequest) (int, []byte, error) {
	if c.BaseURL == "" {
		return 0, nil, fmt.Errorf("asaas baseURL is empty")
	}
	if c.Token == "" {
		return 0, nil, fmt.Errorf("asaas token is empty")
	}
	if req.Customer == "" {
		return 0, nil, fmt.Errorf("customer is required")
	}
	if req.BillingType == "" {
		return 0, nil, fmt.Errorf("billingType is required")
	}
	if req.Value <= 0 {
		return 0, nil, fmt.Errorf("value must be > 0")
	}
	if req.NextDueDate == "" {
		return 0, nil, fmt.Errorf("nextDueDate is required")
	}
	if req.Cycle == "" {
		return 0, nil, fmt.Errorf("cycle is required")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.BaseURL + "/v3/subscriptions"
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
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

