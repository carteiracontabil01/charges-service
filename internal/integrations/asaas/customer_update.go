package asaas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type UpdateCustomerRequest struct {
	// Identificação
	Name        string `json:"name,omitempty"`
	CpfCnpj     string `json:"cpfCnpj,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	MobilePhone string `json:"mobilePhone,omitempty"`

	// Endereço
	Address       string `json:"address,omitempty"`
	AddressNumber string `json:"addressNumber,omitempty"`
	Complement    string `json:"complement,omitempty"`
	Province      string `json:"province,omitempty"`
	City          *int32 `json:"city,omitempty"`
	State         string `json:"state,omitempty"`
	Country       string `json:"country,omitempty"`
	PostalCode    string `json:"postalCode,omitempty"`

	// Metadados
	AdditionalEmails  string `json:"additionalEmails,omitempty"`
	ExternalReference string `json:"externalReference,omitempty"`
	Observations      string `json:"observations,omitempty"`

	// Enums/flags
	PersonType           string `json:"personType,omitempty"`
	NotificationDisabled *bool  `json:"notificationDisabled,omitempty"`
	Company              *bool  `json:"company,omitempty"`
	ForeignCustomer      *bool  `json:"foreignCustomer,omitempty"`
}

func (c *Client) UpdateCustomer(customerID string, req UpdateCustomerRequest) (int, []byte, error) {
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

	payload, err := json.Marshal(req)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.BaseURL + "/v3/customers/" + customerID
	httpReq, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, fmt.Errorf("new request: %w", err)
	}

	// Asaas auth header (per docs)
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
