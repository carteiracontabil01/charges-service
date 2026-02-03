package asaas

import (
	"fmt"
	"io"
	"net/http"
)

// DeletePayment deletes a payment (charge) from Asaas.
// Reference: https://docs.asaas.com/reference/excluir-cobranca
func (c *Client) DeletePayment(paymentID string) (int, []byte, error) {
	if c.BaseURL == "" {
		return 0, nil, fmt.Errorf("asaas baseURL is empty")
	}
	if c.Token == "" {
		return 0, nil, fmt.Errorf("asaas token is empty")
	}
	if paymentID == "" {
		return 0, nil, fmt.Errorf("paymentID is required")
	}

	url := fmt.Sprintf("%s/v3/payments/%s", c.BaseURL, paymentID)
	httpReq, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("new request: %w", err)
	}

	// Asaas auth header (per docs)
	httpReq.Header.Set("access_token", c.Token)
	// Also set Authorization for compatibility with proxies/tools and potential Asaas variants.
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
