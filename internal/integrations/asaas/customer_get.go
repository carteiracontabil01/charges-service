package asaas

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GetCustomer retrieves a single Asaas customer by id.
// Asaas reference: GET /v3/customers/{id}
func (c *Client) GetCustomer(customerID string) (int, []byte, error) {
	customerID = strings.TrimSpace(customerID)
	if customerID == "" {
		return 0, nil, fmt.Errorf("asaas customerID is empty")
	}
	if c.BaseURL == "" {
		return 0, nil, fmt.Errorf("asaas baseURL is empty")
	}
	if c.Token == "" {
		return 0, nil, fmt.Errorf("asaas token is empty")
	}

	url := c.BaseURL + "/v3/customers/" + customerID
	httpReq, err := http.NewRequest(http.MethodGet, url, nil)
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
