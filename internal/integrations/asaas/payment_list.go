package asaas

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ListPayments calls the Asaas API to list payments (charges) with optional filters.
// https://docs.asaas.com/reference/listar-cobrancas
func (c *Client) ListPayments(params url.Values) (int, []byte, error) {
	if c.BaseURL == "" {
		return 0, nil, fmt.Errorf("asaas baseURL is empty")
	}
	if c.Token == "" {
		return 0, nil, fmt.Errorf("asaas token is empty")
	}

	endpoint := c.BaseURL + "/v3/payments"
	if params != nil && len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

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
