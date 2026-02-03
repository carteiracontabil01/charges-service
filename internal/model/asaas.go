package model

// AsaasCreateCustomerRequest is the payload we accept from our API to create a customer in Asaas.
// It mirrors the subset requested by the product.
type AsaasCreateCustomerRequest struct {
	Name                 string `json:"name"`
	CpfCnpj              string `json:"cpfCnpj"`
	Email                string `json:"email,omitempty"`
	MobilePhone          string `json:"mobilePhone,omitempty"`
	NotificationDisabled bool   `json:"notificationDisabled"`
	Company              bool   `json:"company"`
}
