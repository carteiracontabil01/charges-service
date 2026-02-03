package asaas

import (
	"fmt"
	"io"
	"net/http"
)

// GetPaymentIdentificationField fetches the boleto "linha digitável" for a payment.
// Asaas endpoint: GET /v3/payments/{id}/identificationField
func (c *Client) GetPaymentIdentificationField(paymentID string) (int, []byte, error) {
	if c.BaseURL == "" {
		return 0, nil, fmt.Errorf("asaas baseURL is empty")
	}
	if c.Token == "" {
		return 0, nil, fmt.Errorf("asaas token is empty")
	}
	if paymentID == "" {
		return 0, nil, fmt.Errorf("payment id is empty")
	}

	endpoint := c.BaseURL + "/v3/payments/" + paymentID + "/identificationField"
	httpReq, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("new request: %w", err)
	}

	httpReq.Header.Set("access_token", c.Token)
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body, nil
}
