package asaas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func NewClient(baseURL, token string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	token = strings.TrimSpace(token)
	// Some users paste the token with a Bearer prefix; Asaas expects the raw token.
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimPrefix(token, "bearer ")
	return &Client{
		BaseURL: baseURL,
		Token:   strings.TrimSpace(token),
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type CreateCustomerRequest struct {
	Name                 string `json:"name"`
	CpfCnpj              string `json:"cpfCnpj"`
	Email                string `json:"email,omitempty"`
	MobilePhone          string `json:"mobilePhone,omitempty"`
	NotificationDisabled bool   `json:"notificationDisabled"`
	Company              bool   `json:"company"`
}

func (c *Client) CreateCustomer(req CreateCustomerRequest) (int, []byte, error) {
	if c.BaseURL == "" {
		return 0, nil, fmt.Errorf("asaas baseURL is empty")
	}
	if c.Token == "" {
		return 0, nil, fmt.Errorf("asaas token is empty")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.BaseURL + "/v3/customers"
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, fmt.Errorf("new request: %w", err)
	}

	// Asaas auth header (per docs)
	httpReq.Header.Set("access_token", c.Token)
	// Also set Authorization for compatibility with proxies/tools and potential Asaas variants.
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
