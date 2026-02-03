package model

// AsaasCustomerResponse is the response body returned by Asaas for customer operations.
// We model it now because we'll reuse it for GET by ID in the future.
type AsaasCustomerResponse struct {
	Object               string `json:"object"`
	ID                   string `json:"id"`
	DateCreated          string `json:"dateCreated"`
	Name                 string `json:"name"`
	Email                string `json:"email"`
	Phone                string `json:"phone"`
	MobilePhone          string `json:"mobilePhone"`
	Address              string `json:"address"`
	AddressNumber        string `json:"addressNumber"`
	Complement           string `json:"complement"`
	Province             string `json:"province"`
	City                 int32  `json:"city"`
	CityName             string `json:"cityName"`
	State                string `json:"state"`
	Country              string `json:"country"`
	PostalCode           string `json:"postalCode"`
	CpfCnpj              string `json:"cpfCnpj"`
	PersonType           string `json:"personType"` // JURIDICA | FISICA
	Deleted              bool   `json:"deleted"`
	AdditionalEmails     string `json:"additionalEmails"`
	ExternalReference    string `json:"externalReference"`
	NotificationDisabled bool   `json:"notificationDisabled"`
	Observations         string `json:"observations"`
	ForeignCustomer      bool   `json:"foreignCustomer"`
}
