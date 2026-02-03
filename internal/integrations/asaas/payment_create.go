package asaas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// PaymentDiscount mirrors Asaas discount object.
type PaymentDiscount struct {
	Value            *float64 `json:"value,omitempty"`
	DueDateLimitDays *int32   `json:"dueDateLimitDays,omitempty"`
	Type             *string  `json:"type,omitempty"` // FIXED | PERCENTAGE
}

// PaymentInterest mirrors Asaas interest object.
type PaymentInterest struct {
	Value *float64 `json:"value,omitempty"`
}

// PaymentFine mirrors Asaas fine object.
type PaymentFine struct {
	Value *float64 `json:"value,omitempty"`
	Type  *string  `json:"type,omitempty"` // FIXED | PERCENTAGE
}

// CreatePaymentRequest is the payload for creating a new charge (payment) in Asaas.
// https://docs.asaas.com/reference/criar-nova-cobranca
type CreatePaymentRequest struct {
	Customer                                   string  `json:"customer"`
	BillingType                                string  `json:"billingType"` // BOLETO | CREDIT_CARD | PIX | UNDEFINED
	Value                                      float64 `json:"value"`
	DueDate                                    string  `json:"dueDate"` // YYYY-MM-DD
	Description                                *string `json:"description,omitempty"`
	DaysAfterDueDateToRegistrationCancellation *int32  `json:"daysAfterDueDateToRegistrationCancellation,omitempty"`
	ExternalReference                          *string `json:"externalReference,omitempty"`

	InstallmentCount *int32   `json:"installmentCount,omitempty"`
	TotalValue       *float64 `json:"totalValue,omitempty"`
	InstallmentValue *float64 `json:"installmentValue,omitempty"`

	Discount      *PaymentDiscount `json:"discount,omitempty"`
	Interest      *PaymentInterest `json:"interest,omitempty"`
	Fine          *PaymentFine     `json:"fine,omitempty"`
	PostalService *bool            `json:"postalService,omitempty"`
}

// CreatePayment calls the Asaas API to create a new charge (payment).
func (c *Client) CreatePayment(req CreatePaymentRequest) (int, []byte, error) {
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
	if req.DueDate == "" {
		return 0, nil, fmt.Errorf("dueDate is required")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.BaseURL + "/v3/payments"
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
