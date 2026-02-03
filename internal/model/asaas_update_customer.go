package model

// AsaasUpdateCustomerRequest is the payload we accept from our API to update a customer in Asaas.
// Asaas reference: PUT /v3/customers/{id}
//
// We keep fields optional to allow partial updates.
type AsaasUpdateCustomerRequest struct {
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

	// Flags / enums
	PersonType AsaasPersonType `json:"personType,omitempty"`
	// Use pointers for booleans so "omit vs false" is preserved.
	NotificationDisabled *bool `json:"notificationDisabled,omitempty"`
	Company              *bool `json:"company,omitempty"`
	ForeignCustomer      *bool `json:"foreignCustomer,omitempty"`
}
